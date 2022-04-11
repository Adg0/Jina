package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/algod"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
)

var (
	algodAddress = "http://localhost:4001"
	algodToken   = strings.Repeat("a", 64)
	usdc         = 1
)

func main() {
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		log.Fatalf("Failed to init client: %+v", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	// Manager contract calls
	f, err := os.Open("../abi/manager.json")
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

	signer := future.BasicAccountTransactionSigner{Account: accts[0]}

	txParams.Fee = 1000 // creation txn is min fee
	mcp := future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          accts[0].Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
		ApprovalProgram: app,
		ClearProgram:    clear,
		GlobalSchema:    types.StateSchema{NumUint: 6, NumByteSlice: 0},
		LocalSchema:     types.StateSchema{NumUint: 0, NumByteSlice: 0},
	}

	// Skipping error checks below during AddMethodCall and txn create
	var atc = future.AtomicTransactionComposer{}
	atc.AddMethodCall(combine(mcp, getMethod(contract, "create"), []interface{}{usdc}))
	_, err = atc.Execute(algodClient, context.Background(), 2)
	if err != nil {
		log.Fatalf("Failed to execute call: %+v", err)
	}

	txParams.Fee = 5000 // manage creates 4 txn
	mcp = future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          accts[0].Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
	}

	atc = future.AtomicTransactionComposer{}
	atc.AddMethodCall(combine(mcp, getMethod(contract, "manage"), []interface{}{usdc, lqtApproval, lqtClear, jinaApproval, jinaClear}))
	ret, err := atc.Execute(algodClient, context.Background(), 2)
	if err != nil {
		log.Fatalf("Failed to execute call: %+v", err)
	}
	for _, r := range ret.MethodResults {
		log.Printf("%s returned %+v", r.TxID, r.ReturnValue)
	}

	atc = future.AtomicTransactionComposer{}
	lqt := ret.MethodResults[0]
	jina := ret.MethodResults[1]
	jusd := ret.MethodResults[2]
	atc.AddMethodCall(combine(mcp, getMethod(contract, "config"), []interface{}{lqt, jina, usdc, jusd}))

	// Jina calls
	f, err = os.Open("../abi/jina.json")
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err = ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract = &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	// Optin
	signer = future.BasicAccountTransactionSigner{Account: accts[1]}
	txParams.Fee = 1000 // fee for optin
	mcp = future.AddMethodCallParams{
		AppID:           contract.Networks["default"].AppID,
		Sender:          accts[1].Address,
		SuggestedParams: txParams,
		OnComplete:      types.OptInOC,
		Signer:          signer,
	}
	atc = future.AtomicTransactionComposer{}
	atc.AddMethodCall(combine(mcp, getMethod(contract, "optin"), []interface{}{manager}))
	ret, err = atc.Execute(algodClient, context.Background(), 2)
	if err != nil {
		log.Fatalf("Failed to execute call: %+v", err)
	}
	for _, r := range ret.MethodResults {
		log.Printf("%s returned %+v", r.TxID, r.ReturnValue)
	}

	atc.AddMethodCall(combine(mcp, getMethod(contract, "earn"), []interface{}{xids, aamt, lvr, lsa}))
	atc.AddMethodCall(combine(mcp, getMethod(contract, "borrow"), []interface{}{xids, camt, lamt}))

	// // String arg/return
	atc.AddMethodCall(combine(mcp, getMethod(contract, "reverse"), []interface{}{"desrever yllufsseccus"}))

	// []string arg, string return
	atc.AddMethodCall(combine(mcp, getMethod(contract, "concat_strings"), []interface{}{[]string{"this", "string", "is", "joined"}}))

	// Txn arg, uint return
	txn, _ := future.MakePaymentTxn(acct.Address.String(), acct.Address.String(), 10000, nil, "", txParams)
	stxn := future.TransactionWithSigner{Txn: txn, Signer: signer}
	atc.AddMethodCall(combine(mcp, getMethod(contract, "txntest"), []interface{}{10000, stxn, 1000}))

	// >14 args, so they get tuple encoded automatically
	err = atc.AddMethodCall(combine(mcp, getMethod(contract, "manyargs"),
		[]interface{}{
			1, 1, 1, 1, 1,
			1, 1, 1, 1, 1,
			1, 1, 1, 1, 1,
			1, 1, 1, 1, 1,
		}))

	if err != nil {
		log.Fatalf("Failed to add method call: %+v", err)
	}

	ret, err = atc.Execute(algodClient, context.Background(), 2)
	if err != nil {
		log.Fatalf("Failed to execute call: %+v", err)
	}

	for _, r := range ret.MethodResults {
		log.Printf("%s returned %+v", r.TxID, r.ReturnValue)
	}
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
