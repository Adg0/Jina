package jina

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
		authHeader = "X-API-Key"
		//authHeader = "X-Algo-API-Token"
	}
	commonClient, err := common.MakeClient(algodAddress, authHeader, algodToken)
	if err != nil {
		fmt.Printf("failed to make common client: %s\n", err)
		return nil, err
	}
	return (*algod.Client)(commonClient), nil
}

func ConfigASA(algodClient *algod.Client, sk ed25519.PrivateKey, assetID uint64) (err error) {
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
	new_manager := crypto.GetApplicationAddress(AppID).String()
	new_reserve := crypto.GetApplicationAddress(AppID).String()
	new_freeze := crypto.GetApplicationAddress(AppID).String()
	new_clawback := crypto.GetApplicationAddress(LqtID).String() // liquidatorAddr
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

func DeploySmartContract(algodClient *algod.Client, args ApplicationTxnArgs, globalInts, localInts, globalBytes, localBytes uint64, clear, approve string) (appId uint64, err error) {
	// define schema
	globalSchema := types.StateSchema{NumUint: globalInts, NumByteSlice: globalBytes}
	localSchema := types.StateSchema{NumUint: localInts, NumByteSlice: localBytes}

	// get approvalProg and clearProg as []byte
	clearProg, err := CompileSmartContractTeal(algodClient, clear)
	if err != nil {
		return
	}
	approvalProg, err := CompileSmartContractTeal(algodClient, approve)
	if err != nil {
		return
	}

	// get suggested transaction parameters
	if args.txParams.FlatFee == false {
		args.txParams, err = algodClient.SuggestedParams().Do(context.Background())
		if err != nil {
			log.Fatalf("Error getting suggested tx params: %s\n", err)
			return
		}
	}

	// make transaction
	txn, err := future.MakeApplicationCreateTx(true, approvalProg, clearProg, globalSchema, localSchema, args.appArgs, args.accounts, args.foreignApps, args.foreignAssets, args.txParams, args.sender, args.note, args.group, args.lease, args.rekeyTo)
	if err != nil {
		fmt.Printf("Building transaction failed with %v", err)
		return
	}

	// sign send await
	txID := signSendAwait(algodClient, txn, args.sk, "./dryrun/deploy.msgp")

	// display results
	confirmedTxn, _, _ := algodClient.PendingTransactionInformation(txID).Do(context.Background())
	appId = confirmedTxn.ApplicationIndex
	fmt.Printf("Created new app-id: %d\n", appId)
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
	return
}

func LenderCallSmartContract(algodClient *algod.Client, appArgs ApplicationTxnArgs, lsigFile, dryRun string) (txID string, err error) {
	// get suggested transaction parameters
	if appArgs.txParams.FlatFee == false {
		appArgs.txParams, err = algodClient.SuggestedParams().Do(context.Background())
		if err != nil {
			log.Fatalf("Error getting suggested tx params: %s\n", err)
			return
		}
	}

	// Setting parameters for lender
	lsigArgs := make([][]byte, 5)
	var buf [6][8]byte
	binary.BigEndian.PutUint64(buf[0][:], USDCa)                                           // USDCa asset ID
	binary.BigEndian.PutUint64(buf[1][:], 2000000)                                         // loan available (50 USDCa)
	binary.BigEndian.PutUint64(buf[2][:], 172800+uint64(appArgs.txParams.FirstRoundValid)) // Expiring lifespan: 17280 rounds == 1 day
	binary.BigEndian.PutUint64(buf[3][:], AppID)                                           // jina appID
	binary.BigEndian.PutUint64(buf[4][:], LFT_jina)                                        // LFT-jina asset ID
	binary.BigEndian.PutUint64(buf[5][:], JUSD)                                            // JUSD asset ID
	lsigArgs[0] = buf[0][:]
	lsigArgs[1] = buf[1][:]
	lsigArgs[2] = buf[2][:]
	lsigArgs[3] = buf[3][:]

	// Creating delegated logicSig
	lsaRaw := CompileToLsig(algodClient, lsigArgs, "./teal/logicSigDelegated.teal", lsigFile, appArgs.sk)
	if lsaRaw.SigningKey == nil {
		err = fmt.Errorf("lsig is empty")
		return
	}

	file, err := os.Open(lsigFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	lsa, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("failed to read file: %s\n", err)
		return
	}
	lsaHash := sha256.Sum256(lsa)
	var assets_allowed []byte
	for i := 4; i < len(buf); i++ {
		assets_allowed = append(assets_allowed, buf[i][:]...)
	}
	appArgs.appArgs[1] = assets_allowed
	appArgs.appArgs[2] = lsigArgs[1]
	appArgs.appArgs[3] = lsaHash[:8]
	appArgs.appArgs[4] = lsigArgs[2]

	txID, err = CallSmartContract(algodClient, appArgs, nil, nil, dryRun)
	return
}

func CallSmartContract(algodClient *algod.Client, args ApplicationTxnArgs, clearProg, approvalProg []byte, filename string) (txID string, err error) {

	// get suggested transaction parameters
	if args.txParams.FlatFee == false {
		args.txParams, err = algodClient.SuggestedParams().Do(context.Background())
		if err != nil {
			log.Fatalf("Error getting suggested tx params: %s\n", err)
			return
		}
	}

	// create unsigned transaction
	txn, err := future.MakeApplicationCallTx(args.appID, args.appArgs, args.accounts, args.foreignApps, args.foreignAssets, args.onCompletion, approvalProg, clearProg, types.StateSchema{}, types.StateSchema{}, args.txParams, args.sender, args.note, args.group, args.lease, args.rekeyTo)
	if err != nil {
		fmt.Printf("Building transaction failed with %v", err)
		return
	}

	// sign send await
	txID = signSendAwait(algodClient, txn, args.sk, filename)

	// display results
	confirmedTxn, _, _ := algodClient.PendingTransactionInformation(txID).Do(context.Background())
	fmt.Printf("Called to app-id: %d\n", confirmedTxn.Transaction.Txn.ApplicationID)
	return
}

func signSendAwait(algodClient *algod.Client, txn types.Transaction, sk ed25519.PrivateKey, filename string) (txID string) {
	// Sign transaction
	txID, stx, err := crypto.SignTransaction(sk, txn)
	if err != nil {
		fmt.Printf("Signing failed with %v", err)
		return
	}
	fmt.Printf("Signed tx: %v\n", txID)

	// -->
	s_app_txn := types.SignedTxn{}
	msgpack.Decode(stx, &s_app_txn)
	drr, err := future.CreateDryrun(algodClient, []types.SignedTxn{s_app_txn}, nil, context.Background())
	if err != nil {
		fmt.Printf("Failed creating dryrun: %v", err)
		log.Fatalf("Failed to create dryrun: %+v", err)
		return
	}
	os.WriteFile(filename, msgpack.Encode(drr), 0666)
	// <--

	// Submit the raw transaction to network
	sendResponse, err := algodClient.SendRawTransaction(stx).Do(context.Background())
	if err != nil {
		fmt.Printf("Sending failed with %v\n", err)
		return
	}

	confirmedTxn, err := future.WaitForConfirmation(algodClient, txID, 4, context.Background())
	if err != nil {
		fmt.Printf("Error waiting for confirmation on txID: %s\n", txID)
		return
	}
	fmt.Printf("Confirmed Transaction: %s in Round %d\n", sendResponse, confirmedTxn.ConfirmedRound)
	return txID
}

func BorrowCallSmartContract(algodClient *algod.Client, args ApplicationTxnArgs, lenders []Lender, filename string) (err error) {
	// get suggested transaction parameters
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting suggested tx params: %s\n", err)
		return
	}
	txParams.FlatFee = true
	txParams.Fee = 0

	// create unsigned transactions
	var txnGroup []types.Transaction

	borrowerAddr := args.sender.String() // address that prompted this calls
	//liquidator := crypto.GetApplicationAddress(args.appID).String()
	note := []byte(nil)
	closeRemainderTo := ""

	// first transaction is app call
	var txn1 types.Transaction
	txnGroup = append(txnGroup, txn1)

	for i, lender := range lenders {
		args.accounts = append(args.accounts, lender.Address)
		txn, err := future.MakeAssetTransferTxn(lender.Address, borrowerAddr, lender.Amount, note, txParams, closeRemainderTo, USDCa)
		if err != nil {
			log.Fatalf("Failed to make transaction MakeAssetTransfer Txn-%d: %s\n", i, err)
			break
		}
		txnGroup = append(txnGroup, txn)
	}

	txn1, err = future.MakeApplicationCallTx(args.appID, args.appArgs, args.accounts, args.foreignApps, args.foreignAssets, types.NoOpOC, nil, nil, types.StateSchema{}, types.StateSchema{}, args.txParams, args.sender, args.note, args.group, args.lease, args.rekeyTo)
	if err != nil {
		log.Fatalf("Building transaction failed with: %s", err)
		return
	}
	txnGroup[0] = txn1
	gid, err := crypto.ComputeGroupID(txnGroup)
	fmt.Println("...computed groupId: ", gid)
	txn1.Group = gid

	fmt.Println("Assembling transaction group...")
	var signedGroup []byte
	// Sign transaction appl
	txID, stx, err := crypto.SignTransaction(args.sk, txn1)
	if err != nil {
		log.Fatalf("Signing appl failed with %s", err)
		return
	}
	fmt.Printf("Signed tx appl: %v\n", txID)
	signedGroup = append(signedGroup, stx...)
	// -->
	var signedTxns []types.SignedTxn
	s_app_txn := types.SignedTxn{}
	msgpack.Decode(stx, &s_app_txn)
	signedTxns = append(signedTxns, s_app_txn)
	// <--

	for i, lender := range lenders {
		txnGroup[i+1].Group = gid
		txid, stx, err := crypto.SignLogicSigAccountTransaction(lender.Lsa, txnGroup[i+1])
		if err != nil {
			log.Fatalf("Failed to sign transaction: %s\n", err)
			break
		}
		fmt.Printf("...LogicSig signed txn%d: %s\n", i, txid)
		signedGroup = append(signedGroup, stx...)
		// -->
		msgpack.Decode(stx, &s_app_txn)
		signedTxns = append(signedTxns, s_app_txn)
		// <--
	}

	// -->
	drr, err := future.CreateDryrun(algodClient, signedTxns, nil, context.Background())
	if err != nil {
		log.Fatalf("Failed to create dryrun: %+v", err)
	}
	os.WriteFile(filename, msgpack.Encode(drr), 0666)
	// <--

	// Broadcast transactions
	fmt.Println("Sending transaction group...")
	pendingTxID, err := algodClient.SendRawTransaction(signedGroup).Do(context.Background())
	if err != nil {
		fmt.Printf("Failed to send transaction: %s\n", err)
		return
	}

	// altrenate waiting for confirmation
	confirmedTxn, err := future.WaitForConfirmation(algodClient, txID, 4, context.Background())
	if err != nil {
		fmt.Printf("Error waiting for confirmation on txID: %s\n", txID)
		return
	}
	fmt.Printf("Confirmed Transaction: %s in Round %d\n", pendingTxID, confirmedTxn.ConfirmedRound)
	return
}

func RepayClaimCallSmartContract(algodClient *algod.Client, args ApplicationTxnArgs, assetID, assetAmount uint64, filename string) (err error) {
	// create unsigned transactions
	var txnGroup []types.Transaction

	callerAddr := args.sender.String() // address that prompted this calls
	jinaAddr := crypto.GetApplicationAddress(AppID).String()
	note := []byte(nil)
	closeRemainderTo := ""

	// first transaction is app call
	txn1, err := future.MakeApplicationCallTx(args.appID, args.appArgs, args.accounts, args.foreignApps, args.foreignAssets, types.NoOpOC, nil, nil, types.StateSchema{}, types.StateSchema{}, args.txParams, args.sender, args.note, args.group, args.lease, args.rekeyTo)
	if err != nil {
		log.Fatalf("Building transaction failed with: %s", err)
		return
	}
	txnGroup = append(txnGroup, txn1)

	// next to app call: repay loan transaction
	args.txParams.Fee = 0
	txn, err := future.MakeAssetTransferTxn(callerAddr, jinaAddr, assetAmount, note, args.txParams, closeRemainderTo, assetID)
	if err != nil {
		log.Fatalf("Building transaction failed with: %s", err)
		return
	}
	txnGroup = append(txnGroup, txn)

	gid, err := crypto.ComputeGroupID(txnGroup)
	fmt.Println("...computed groupId: ", gid)
	txn1.Group = gid
	txn.Group = gid

	fmt.Println("Assembling transaction group...")
	var signedGroup []byte
	// Sign transaction appl
	txID1, stx1, err := crypto.SignTransaction(args.sk, txn1)
	if err != nil {
		log.Fatalf("Signing appl failed with %s", err)
		return
	}
	fmt.Printf("Signed tx appl: %v\n", txID1)
	signedGroup = append(signedGroup, stx1...)
	// Sign transaction repay
	txID, stx, err := crypto.SignTransaction(args.sk, txn)
	if err != nil {
		log.Fatalf("Signing appl failed with %s", err)
		return
	}
	fmt.Printf("Signed tx: %v\n", txID)
	signedGroup = append(signedGroup, stx...)
	// -->
	var signedTxns []types.SignedTxn
	s_app_txn := types.SignedTxn{}
	msgpack.Decode(stx1, &s_app_txn)
	signedTxns = append(signedTxns, s_app_txn)
	// -->
	msgpack.Decode(stx, &s_app_txn)
	signedTxns = append(signedTxns, s_app_txn)
	// <--

	// -->
	drr, err := future.CreateDryrun(algodClient, signedTxns, nil, context.Background())
	if err != nil {
		log.Fatalf("Failed to create dryrun: %+v", err)
	}
	os.WriteFile(filename, msgpack.Encode(drr), 0666)
	// <--

	// Broadcast transactions
	fmt.Println("Sending transaction group...")
	pendingTxID, err := algodClient.SendRawTransaction(signedGroup).Do(context.Background())
	if err != nil {
		fmt.Printf("Failed to send transaction: %s\n", err)
		return
	}

	// altrenate waiting for confirmation
	confirmedTxn, err := future.WaitForConfirmation(algodClient, txID, 4, context.Background())
	if err != nil {
		fmt.Printf("Error waiting for confirmation on txID: %s\n", txID)
		return
	}
	fmt.Printf("Confirmed Transaction: %s in Round %d\n", pendingTxID, confirmedTxn.ConfirmedRound)
	return
}

type Lender struct {
	Address string
	Amount  uint64
	AssetID uint64
	Lsa     crypto.LogicSigAccount
}

type ApplicationTxnArgs struct {
	appID         uint64
	appArgs       [][]byte
	accounts      []string
	foreignApps   []uint64
	foreignAssets []uint64
	txParams      types.SuggestedParams
	sender        types.Address
	note          []byte
	group         types.Digest
	lease         [32]byte
	rekeyTo       types.Address
	sk            ed25519.PrivateKey
	onCompletion  types.OnCompletion
}
