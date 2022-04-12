package jina

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
		fmt.Printf("failed to make common client: %s\n", err)
		return nil, err
	}
	return (*algod.Client)(commonClient), nil
}

func debugAppCall(algodClient *algod.Client, atc future.AtomicTransactionComposer, dryrunDump, response string) {
	// gather signatures
	stxns, _ := atc.GatherSignatures()
	fmt.Printf("len:%v\n", len(stxns))
	stx := make([]types.SignedTxn, len(stxns))
	for _, sigTxns := range stxns {
		stxn := types.SignedTxn{}
		msgpack.Decode(sigTxns, &stxn)
		stx = append(stx, stxn)
	}

	// Create the dryrun request object
	dryrunRequest, _ := future.CreateDryrun(algodClient, stx, nil, context.Background())

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

}

func ConfigASA(algodClient *algod.Client, sk ed25519.PrivateKey, mngID, jinaID, lqtID, assetID uint64) (err error) {
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting suggested tx params: %s\n", err)
		return
	}
	manager, err := crypto.GenerateAddressFromSK(sk)
	if err != nil {
		fmt.Printf("Error recovering account address: %s\n", err)
		return
	}
	// Make Contract admin for asset
	new_manager := crypto.GetApplicationAddress(mngID).String()
	new_reserve := manager.String()
	new_freeze := crypto.GetApplicationAddress(jinaID).String()
	new_clawback := crypto.GetApplicationAddress(lqtID).String() // liquidatorAddr
	strictEmptyAddressChecking := true
	note := []byte(nil)
	txn, err := future.MakeAssetConfigTxn(manager.String(), note, txParams, assetID, new_manager, new_reserve, new_freeze, new_clawback, strictEmptyAddressChecking)
	if err != nil {
		fmt.Printf("Failed to send transaction MakeAssetConfig Txn: %s\n", err)
		return
	}

	// sign the transaction
	err = signSendWait(algodClient, sk, txn)
	return
}

func OptinASA(algodClient *algod.Client, sk ed25519.PrivateKey, assetID uint64) (err error) {
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting suggested tx params: %s\n", err)
		return
	}
	sender, err := crypto.GenerateAddressFromSK(sk)
	if err != nil {
		fmt.Printf("Error recovering account address: %s\n", err)
		return
	}
	txn, err := future.MakeAssetAcceptanceTxn(sender.String(), []byte(nil), txParams, assetID)
	if err != nil {
		fmt.Printf("Failed to send transaction MakeAssetAcceptance Txn: %s\n", err)
		return
	}
	err = signSendWait(algodClient, sk, txn)
	return
}

func signSendWait(algodClient *algod.Client, sk ed25519.PrivateKey, txn types.Transaction) (err error) {

	// sign the transaction
	txid, stx, err := crypto.SignTransaction(sk, txn)
	if err != nil {
		fmt.Printf("Failed to sign transaction: %s\n", err)
		return
	}
	fmt.Printf("Transaction ID: %s\n", txid)

	// Broadcast the transaction to the network
	_, err = algodClient.SendRawTransaction(stx).Do(context.Background())
	if err != nil {
		fmt.Printf("failed to send transaction: %s\n", err)
		return
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

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "earn"), []interface{}{xids, aamt, lvr, lsa}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/earn.msgp", "./dryrun/response/earn.json")
	return
}

func Optin(algodClient *algod.Client, acct crypto.Account, app uint64, contract_json string) (err error) {
	fmt.Println("inside Optin")
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

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "optin"), []interface{}{app}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/optin.msgp", "./dryrun/response/optin.json")
	return
}

// Make Jina application call to borrow against provided collateral
func Borrow(algodClient *algod.Client, acct, lender crypto.Account, usdc uint64, xids, camt, lamt []uint64, lsigFile, contract_json string) (err error) {
	fmt.Println("inside Borrow")
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

	var atc = future.AtomicTransactionComposer{}
	txParams.Fee = 0
	txn, _ := future.MakeAssetTransferTxn(lender.Address.String(), acct.Address.String(), lamt[0], nil, txParams, "", usdc)
	lsa, err := FetchLsigFromFile(lsigFile)
	if err != nil {
		log.Fatalf("Failed to get lsa from file: %+v", err)
	}
	signerLsa := future.LogicSigAccountTransactionSigner{LogicSigAccount: lsa}
	//sig := future.BasicAccountTransactionSigner{Account: lender}
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signerLsa} //sig}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "borrow"), []interface{}{stxn, xids, camt, lamt, lender.Address, 56, xids[0], 2, 54}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/borrow.msgp", "./dryrun/response/borrow.json")
	return

}

// Make Jina application call to repay loan and unfreeze asset
func Repay(algodClient *algod.Client, acct crypto.Account, jina, usdc uint64, xids, ramt []uint64, contract_json string) (err error) {
	fmt.Println("inside Repay")
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

	var atc = future.AtomicTransactionComposer{}
	txParams.Fee = 0
	txn, _ := future.MakeAssetTransferTxn(acct.Address.String(), crypto.GetApplicationAddress(jina).String(), ramt[0], nil, txParams, "", usdc)
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signer}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "repay"), []interface{}{stxn, xids, ramt, xids[0]}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/repay.msgp", "./dryrun/response/repay.json")
	return
}

// Make Jina application call to claim USDCa for JUSD
func Claim(algodClient *algod.Client, acct crypto.Account, jina, amt, usdc, jusd uint64, contract_json string) (err error) {
	fmt.Println("inside Claim")
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

	var atc = future.AtomicTransactionComposer{}
	txParams.Fee = 0
	txn, _ := future.MakeAssetTransferTxn(acct.Address.String(), crypto.GetApplicationAddress(jina).String(), amt, nil, txParams, "", jusd)
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signer}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "claim"), []interface{}{stxn, usdc}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/claim.msgp", "./dryrun/response/claim.json")
	return
}

func ConfigureApps(algodClient *algod.Client, acct crypto.Account, lqt, jina, usdc, jusd uint64, contract_json string) (err error) {
	fmt.Println("inside ConfigureApps")
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
	txParams.Fee = types.MicroAlgos(8 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "config"), []interface{}{lqt, jina, usdc, jusd}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/config.msgp", "./dryrun/response/config.json")
	return
}

// create sub-apps
func CreateApps(algodClient *algod.Client, acct crypto.Account, usdc uint64, lqtApproval, lqtClear, jinaApproval, jinaClear []byte, contract_json string) (err error) {
	fmt.Println("inside CreateApps")
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
	txParams.Fee = types.MicroAlgos(5 * txParams.MinFee)

	signer := future.BasicAccountTransactionSigner{Account: acct}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "manage"), []interface{}{usdc, lqtApproval, lqtClear, jinaApproval, jinaClear}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/manage.msgp", "./dryrun/response/manage.json")
	return
}

// Fund app
func Fund(algodClient *algod.Client, acct crypto.Account, app, amt uint64) (err error) {
	fmt.Println("inside Fund")
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
	fmt.Println("inside Update")
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

	// get approvalProg and clearProg as []byte
	clear, err := CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearProg found error, %s", err)
	}
	app, err := CompileSmartContractTeal(algodClient, "./teal/managerProg.teal")
	if err != nil {
		log.Fatalf("approvalProg found error, %s", err)
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

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(mcp)
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/update.msgp", "./dryrun/response/update.json")
	return
}

func SendJusd(algodClient *algod.Client, acct crypto.Account, rec types.Address, jusd uint64, contract_json string) (err error) {
	fmt.Println("inside SendJusd")
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

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "fund"), []interface{}{rec, jusd}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/fund.msgp", "./dryrun/response/fund.json")
	return
}

// Update child smart contract
func ChildUpdate(algodClient *algod.Client, acct crypto.Account, appID uint64, app, clear, contract_json string) (err error) {
	fmt.Println("inside UpdateChild")
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

	// get approvalProg and clearProg as []byte
	clearState, err := CompileSmartContractTeal(algodClient, clear)
	if err != nil {
		log.Fatalf("clearProg found error, %s", err)
	}
	approval, err := CompileSmartContractTeal(algodClient, app)
	if err != nil {
		log.Fatalf("approvalProg found error, %s", err)
	}

	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          acct.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}
	// Skipping error checks below during AddMethodCall and txn create
	var atc = future.AtomicTransactionComposer{}
	atc.AddMethodCall(combine(mcp, getMethod(contract, "update_child_app"), []interface{}{appID, approval, clearState}))

	debugAppCall(algodClient, atc, "./dryrun/update_child.msgp", "./dryrun/response/update_child.json")
	return
}

// Deploy smart contract
func Deploy(algodClient *algod.Client, acct crypto.Account, usdc uint64, contract_json string) (err error) {
	fmt.Println("inside Deploy")
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

	// get approvalProg and clearProg as []byte
	clear, err := CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearProg found error, %s", err)
	}
	app, err := CompileSmartContractTeal(algodClient, "./teal/managerProg.teal")
	if err != nil {
		log.Fatalf("approvalProg found error, %s", err)
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

	var atc = future.AtomicTransactionComposer{}
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "create"), []interface{}{usdc}))
	if err != nil {
		log.Fatalf("Failed to AddMethodCall: %+v", err)
	}

	debugAppCall(algodClient, atc, "./dryrun/create.msgp", "./dryrun/response/create.json")
	return
}

// Start sandbox and create USDCa and other NFTs for testing purpose
func Start(algodClient *algod.Client, acct crypto.Account) (err error) {
	fmt.Println("inside Start")
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}
	addr := acct.Address.String()
	txn, err := future.MakeAssetCreateTxn(addr, []byte(""), txParams, 18446744073709551615, 6, false, addr, addr, addr, addr, "USDC", "USDC", "https://circle.com/", "")
	if err != nil {
		log.Fatalf("Failed creating asset: %+v", err)
	}
	signSendWait(algodClient, acct.PrivateKey, txn)
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

func CompileSmartContractTeal(algodClient *algod.Client, osTealFile string) (compiledProgram []byte, err error) {
	file, err := os.Open(osTealFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	tealFile, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("failed to read file: %s\n", err)
		return
	}
	compileResponse, err := algodClient.TealCompile(tealFile).Do(context.Background())
	if err != nil {
		fmt.Printf("Issue with compile: %s\n", err)
		return
	}
	compiledProgram, _ = base64.StdEncoding.DecodeString(compileResponse.Result)
	fmt.Printf("%s size: %v\n", osTealFile, len(compiledProgram))
	return
}
