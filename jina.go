package jina

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
)

func InitAlgodClient(algodAddress, algodToken, node string) (*algod.Client, error) {
	// Initialize an algodClient
	authHeader := "X-API-Key"
	if node == "local" {
		authHeader = "X-Algo-API-Token"
	}
	commonClient, err := common.MakeClient(algodAddress, authHeader, algodToken)
	if err != nil {
		log.Fatalf("Failed to make common client: %+v", err)
	}
	return (*algod.Client)(commonClient), nil
}

func debugAppCall(algodClient *algod.Client, atc future.AtomicTransactionComposer, dryrunDump, response string) []future.ABIMethodResult {
	// gather signatures
	stxns, _ := atc.GatherSignatures()
	stx := make([]types.SignedTxn, len(stxns))
	for i, sigTxns := range stxns {
		stxn := types.SignedTxn{}
		msgpack.Decode(sigTxns, &stxn)
		stx[i] = stxn
	}

	// Create the dryrun request object
	dryrunRequest, err := future.CreateDryrun(algodClient, stx, nil, context.Background())
	if err != nil {
		log.Fatalf("Failed creating dryrun: %+v", err)
	}

	// Pass dryrun request to algod server
	dryrunResponse, _ := algodClient.TealDryrun(dryrunRequest).Do(context.Background())

	// Inspect the response to check result
	os.WriteFile(dryrunDump, msgpack.Encode(dryrunRequest), 0666)
	drr, err := json.MarshalIndent(dryrunResponse, "", "")
	if err != nil {
		log.Fatalf("Failed JSON marshal indent: %+v", err)
	}
	os.WriteFile(response, drr, 0666)

	ret, err := atc.Execute(algodClient, context.Background(), 2)
	if err != nil {
		log.Fatalf("Failed to execute call: %+v", err)
	}
	for _, r := range ret.MethodResults {
		log.Printf("%s returned %+v", r.TxID, r.ReturnValue)
	}
	return ret.MethodResults
}

func ConfigASA(algodClient *algod.Client, sk ed25519.PrivateKey, mngID, jinaID, lqtID, assetID uint64) (err error) {
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting suggested tx params: %s\n", err)
	}
	manager, err := crypto.GenerateAddressFromSK(sk)
	if err != nil {
		log.Fatalf("Error recovering account address: %s\n", err)
	}
	// Make Contract admin for asset
	new_manager := crypto.GetApplicationAddress(mngID).String()
	// TODO make the reserve as is (don't change)
	new_reserve := manager.String()
	new_freeze := crypto.GetApplicationAddress(jinaID).String()
	new_clawback := crypto.GetApplicationAddress(lqtID).String() // liquidatorAddr
	strictEmptyAddressChecking := true
	note := []byte(nil)
	txn, err := future.MakeAssetConfigTxn(manager.String(), note, txParams, assetID, new_manager, new_reserve, new_freeze, new_clawback, strictEmptyAddressChecking)
	if err != nil {
		log.Fatalf("Failed to send transaction MakeAssetConfig Txn: %s\n", err)
	}

	// sign the transaction
	err = signSendWait(algodClient, sk, txn)
	return
}

func OptinASA(algodClient *algod.Client, sk ed25519.PrivateKey, assetID uint64) (err error) {
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting suggested tx params: %s\n", err)
	}
	sender, err := crypto.GenerateAddressFromSK(sk)
	if err != nil {
		log.Fatalf("Error recovering account address: %s\n", err)
	}
	txn, err := future.MakeAssetAcceptanceTxn(sender.String(), []byte(nil), txParams, assetID)
	if err != nil {
		log.Fatalf("Failed to send transaction MakeAssetAcceptance Txn: %s\n", err)
	}
	err = signSendWait(algodClient, sk, txn)
	return
}

func signSendWait(algodClient *algod.Client, sk ed25519.PrivateKey, txn types.Transaction) (err error) {
	// sign the transaction
	txid, stx, err := crypto.SignTransaction(sk, txn)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %s\n", err)
	}
	log.Printf("Transaction ID: %s\n", txid)

	// Broadcast the transaction to the network
	_, err = algodClient.SendRawTransaction(stx).Do(context.Background())
	if err != nil {
		log.Fatalf("failed to send transaction: %s\n", err)
	}

	// Wait for transaction to be confirmed
	waitForConfirmation(txid, algodClient)
	return
}

// Make Jina application call to earn USDCa at 3%
func Earn(algodClient *algod.Client, acct crypto.Account, xids []uint64, aamt, lvr uint64, lsa []byte, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "earn"), []interface{}{xids, aamt, lvr, lsa}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/earn.msgp", "./dryrun/response/earn.json")
	return
}

func Optin(algodClient *algod.Client, acct crypto.Account, app uint64, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.OptInOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "optin"), []interface{}{app}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/optin.msgp", "./dryrun/response/optin.json")
	return
}

// Make Jina application call to borrow against provided collateral
func Borrow(algodClient *algod.Client, acct, lender crypto.Account, usdc, jusd, mng, lqt uint64, xids, camt, lamt []uint64, lsigFile, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = types.MicroAlgos(4 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	txParams.Fee = 0
	txn, _ := future.MakeAssetTransferTxn(lender.Address.String(), acct.Address.String(), lamt[0], nil, txParams, "", usdc)
	lsa, err := FetchLsigFromFile(lsigFile)
	if err != nil {
		log.Fatalf("Failed to get lsa from file: %+v", err)
	}
	signerLsa := future.LogicSigAccountTransactionSigner{LogicSigAccount: lsa}
	//sig := future.BasicAccountTransactionSigner{Account: lender}
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signerLsa} //sig}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "borrow"), []interface{}{stxn, xids, camt, lamt, lender.Address, xids[0], jusd, mng, lqt}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/borrow.msgp", "./dryrun/response/borrow.json")
	return

}

// Make Jina application call to repay loan and unfreeze asset
func Repay(algodClient *algod.Client, acct crypto.Account, mng, lqt, usdc uint64, xids, ramt []uint64, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = types.MicroAlgos(3 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	jina := contract.Networks["default"].AppID
	mcp := future.AddMethodCallParams{
		AppID:           jina,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	txParams.Fee = 0
	txn, _ := future.MakeAssetTransferTxn(acct.Address.String(), crypto.GetApplicationAddress(jina).String(), ramt[0], nil, txParams, "", usdc)
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signer}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "repay"), []interface{}{stxn, xids, ramt, xids[0], mng, lqt}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/repay.msgp", "./dryrun/response/repay.json")
	return
}

// Make Jina application call to claim USDCa for JUSD
func Claim(algodClient *algod.Client, acct crypto.Account, mng, amt, usdc, jusd uint64, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = types.MicroAlgos(3 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	jina := contract.Networks["default"].AppID
	mcp := future.AddMethodCallParams{
		AppID:           jina,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	txParams.Fee = 0
	txn, _ := future.MakeAssetTransferTxn(acct.Address.String(), crypto.GetApplicationAddress(jina).String(), amt, nil, txParams, "", jusd)
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signer}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "claim"), []interface{}{stxn, usdc, mng}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/claim.msgp", "./dryrun/response/claim.json")
	return
}

func ConfigureApps(algodClient *algod.Client, acct crypto.Account, lqt, jina, usdc, jusd uint64, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = types.MicroAlgos(12 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	lqtAddress := crypto.GetApplicationAddress(lqt)
	jinaAddress := crypto.GetApplicationAddress(jina)
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "config"), []interface{}{lqt, jina, lqtAddress, jinaAddress, usdc, jusd}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/config.msgp", "./dryrun/response/config.json")
	return
}

// create sub-apps
func CreateApps(algodClient *algod.Client, acct crypto.Account, usdc uint64, lqtApproval, lqtClear, jinaApproval, jinaClear []byte, contract_json, lqt_contract, jina_contract string) (ids [4]uint64, err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = types.MicroAlgos(2 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	var atc2 future.AtomicTransactionComposer
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "create_liquidator"), []interface{}{lqtApproval, lqtClear}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall, create_liquidator: %+v", err)
	}
	lqt := uint64(0)
	jina := uint64(0)
	ret := debugAppCall(algodClient, atc, "./dryrun/create_liquidator.msgp", "./dryrun/response/create_liquidator.json")
	lqt = ret[0].ReturnValue.(uint64)

	txParams.Fee = types.MicroAlgos(4 * txParams.MinFee)
	mcp.SuggestedParams = txParams
	err = atc2.AddMethodCall(combine(mcp, getMethod(contract, "create_child"), []interface{}{usdc, jinaApproval, jinaClear, lqt}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall, create_child: %+v", err)
	}

	ret_j := debugAppCall(algodClient, atc2, "./dryrun/create_child.msgp", "./dryrun/response/create_child.json")
	var v []interface{} = ret_j[0].ReturnValue.([]interface{})
	jina = v[0].(uint64)
	ids[0] = lqt
	ids[1] = jina
	ids[2] = v[1].(uint64)
	ids[3] = v[2].(uint64)
	updateABI(algodClient, lqt_contract, lqt)
	updateABI(algodClient, jina_contract, jina)
	return
}

// Fund app
func Fund(algodClient *algod.Client, acct crypto.Account, app, amt uint64) (err error) {
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	addr := acct.Address.String()
	txn, err := future.MakePaymentTxn(addr, crypto.GetApplicationAddress(app).String(), amt, []byte(""), "", txParams)
	if err != nil {
		log.Fatalf("Failed creating asset: %+v", err)
	}
	signSendWait(algodClient, acct.PrivateKey, txn)
	return
}

// Update smart contract
func Update(algodClient *algod.Client, acct crypto.Account, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}

	signer := future.BasicAccountTransactionSigner{Account: acct}

	// get approval and clearState as []byte
	clear, err := CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearState found error, %s", err)
	}
	app, err := CompileSmartContractTeal(algodClient, "./teal/managerApp.teal")
	if err != nil {
		log.Fatalf("approval found error, %s", err)
	}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.UpdateApplicationOC,
		Signer:          signer,
		ApprovalProgram: app,
		ClearProgram:    clear,
	}

	var atc future.AtomicTransactionComposer
	err = atc.AddMethodCall(mcp)
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/update.msgp", "./dryrun/response/update.json")
	return
}

func SendJusd(algodClient *algod.Client, acct crypto.Account, rec types.Address, jusd uint64, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = 2000

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "fund"), []interface{}{rec, jusd}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/fund.msgp", "./dryrun/response/fund.json")
	return
}

// Update child smart contract
func ChildUpdate(algodClient *algod.Client, acct crypto.Account, appID uint64, app, clear, contract_json string) (err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	txParams.FlatFee = true
	txParams.Fee = 2000

	signer := future.BasicAccountTransactionSigner{Account: acct}

	// get approval and clearState as []byte
	clearState, err := CompileSmartContractTeal(algodClient, clear)
	if err != nil {
		log.Fatalf("clearState found error, %s", err)
	}
	approval, err := CompileSmartContractTeal(algodClient, app)
	if err != nil {
		log.Fatalf("approval found error, %s", err)
	}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc future.AtomicTransactionComposer
	atc.AddMethodCall(combine(mcp, getMethod(contract, "update_child_app"), []interface{}{appID, approval, clearState}))

	debugAppCall(algodClient, atc, "./dryrun/update_child.msgp", "./dryrun/response/update_child.json")
	return
}

// Deploy smart contract
func Deploy(algodClient *algod.Client, acct crypto.Account, usdc uint64, contract_json string) (newApp uint64, err error) {
	f, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}

	signer := future.BasicAccountTransactionSigner{Account: acct}

	// get approval and clearState as []byte
	clear, err := CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearState found error, %s", err)
	}
	app, err := CompileSmartContractTeal(algodClient, "./teal/managerApp.teal")
	if err != nil {
		log.Fatalf("approval found error, %s", err)
	}

	mcp := future.AddMethodCallParams{
		AppID:           0,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
		ApprovalProgram: app,
		ClearProgram:    clear,
		GlobalSchema:    types.StateSchema{NumUint: 6, NumByteSlice: 0},
		LocalSchema:     types.StateSchema{NumUint: 0, NumByteSlice: 0},
	}

	var atc future.AtomicTransactionComposer
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "create"), []interface{}{usdc}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/create.msgp", "./dryrun/response/create.json")

	// get the created appID
	acctInfo, err := algodClient.AccountInformation(acct.Address.String()).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to fetch Account information: %+v", err)
	}
	newApp = acctInfo.CreatedApps[len(acctInfo.CreatedApps)-1].Id

	updateABI(algodClient, contract_json, newApp)
	return
}

func CreateASA(algodClient *algod.Client, acct crypto.Account, amt uint64, dec uint32, name, url string) (assetID uint64, err error) {
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	addr := acct.Address.String()
	txn, err := future.MakeAssetCreateTxn(addr, []byte(""), txParams, amt, dec, false, addr, addr, addr, addr, name, name, url, "")
	if err != nil {
		log.Fatalf("Failed creating asset: %+v", err)
	}
	signSendWait(algodClient, acct.PrivateKey, txn)
	// get the created assetID
	acctInfo, err := algodClient.AccountInformation(acct.Address.String()).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to fetch Account information: %+v", err)
	}
	assetID = acctInfo.CreatedAssets[len(acctInfo.CreatedAssets)-1].Index
	return
}

// Start sandbox and create USDCa and other NFTs for testing purpose
func Start(algodClient *algod.Client, acct crypto.Account) (assetID uint64, err error) {
	assetID, err = CreateASA(algodClient, acct, 18446744073709551615, 6, "USDC", "https://circle.com/")
	return
}

func getMethod(c *abi.Contract, name string) (m abi.Method) {
	for _, m = range c.Methods {
		if m.Name == name {
			return
		}
	}
	log.Fatalf("No method named: %s", name)
	return
}

func combine(mcp future.AddMethodCallParams, m abi.Method, a []interface{}) future.AddMethodCallParams {
	mcp.Method = m
	mcp.MethodArgs = a
	return mcp
}

func getContract(file string) (err error) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}
	return
}

func updateABI(algodClient *algod.Client, contract_json string, newApp uint64) {
	file, err := os.Open(contract_json)
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	// update appID of contract
	if network, ok := contract.Networks["default"]; ok {
		network.AppID = newApp
		contract.Networks["default"] = network
	}

	out, err := json.MarshalIndent(contract, "", "    ")
	if err != nil {
		log.Fatalf("Failed to marshal: %+v", err)
	}
	err = ioutil.WriteFile(contract_json, out, 0666)
	if err != nil {
		log.Fatalf("Failed to write file: %+v", err)
	}
}

func CompileSmartContractTeal(algodClient *algod.Client, osTealFile string) (compiledProgram []byte, err error) {
	file, err := os.Open(osTealFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	tealFile, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("failed to read file: %s\n", err)
	}
	compileResponse, err := algodClient.TealCompile(tealFile).Do(context.Background())
	if err != nil {
		log.Fatalf("Issue with compile: %s\n", err)
	}
	compiledProgram, _ = base64.StdEncoding.DecodeString(compileResponse.Result)
	log.Printf("%s size: %v\n", osTealFile, len(compiledProgram))
	return
}
