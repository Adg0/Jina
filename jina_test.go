package jina

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/mnemonic"
	"github.com/algorand/go-algorand-sdk/types"
)

func TestOptinASA(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	sk, err := mnemonic.ToPrivateKey(ToMn) // mnemonic of address that wants to optin
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	err = OptinASA(algodClient, sk, JUSD) // assetID to optin to
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestConfigASA(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	sk, err := mnemonic.ToPrivateKey(ReserveMn) // mnemonic of admin address (manager address)
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	err = ConfigASA(algodClient, sk, JUSD)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestOptinSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.OptInOC
	app.appID = AppID
	app.sk, err = mnemonic.ToPrivateKey(ToMn) // mnemonic of address that wants to optin
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}
	txID, err := CallSmartContract(algodClient, app, nil, nil, "./dryrun/optin.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCallSmartContractLender(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	dryrunFile := "./dryrun/lender.msgp"
	app.appID = AppID
	arg := make([][]byte, 5)
	arg[0] = []byte("lend")
	app.appArgs = arg
	app.sk, err = mnemonic.ToPrivateKey(ToMn)  // mnemonic of address that wants to call app
	lsigFile := "./codec/lender_lsig_To.codec" // codec file path
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}
	txID, err := LenderCallSmartContract(algodClient, app, lsigFile, dryrunFile)
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCallSmartContractUpdateBorrow(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	app.appID = AppID
	arg := make([][]byte, 3)
	var buf [2][8]byte
	binary.BigEndian.PutUint64(buf[0][:], LFT_jina)  // LFT-jina asset ID
	binary.BigEndian.PutUint64(buf[1][:], uint64(1)) // collateral amount
	arg[0] = []byte("update_borrow")
	arg[1] = buf[0][:] // collateral_assets
	arg[2] = buf[1][:] // collateral_amount
	app.appArgs = arg
	app.foreignApps = append(app.foreignApps, LqtID)        // jina appID
	app.foreignAssets = append(app.foreignAssets, LFT_jina) // collateral assetID
	app.sk, err = mnemonic.ToPrivateKey(BonusMn)            // mnemonic of address that wants to call app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	txID, err := CallSmartContract(algodClient, app, nil, nil, "./dryrun/update_borrow.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestBorrowCallSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	var app ApplicationTxnArgs
	app.appID = AppID // first smart contract, that users interact with
	arg := make([][]byte, 3)
	var buf [2][8]byte
	binary.BigEndian.PutUint64(buf[0][:], LFT_jina)  // LFT-jina asset ID
	binary.BigEndian.PutUint64(buf[1][:], uint64(2)) // collateral amount
	arg[0] = []byte("borrow")
	arg[1] = buf[0][:] // collateral_assets
	arg[2] = buf[1][:] // collateral_amount
	app.appArgs = arg
	app.sk, err = mnemonic.ToPrivateKey(BonusMn) // mnemonic of address that wants to borrow
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}
	app.txParams, err = algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		t.Errorf("Error getting suggested tx params: %s\n", err)
	}
	app.txParams.FlatFee = true

	lenderz := make([]Lender, 2)
	app.txParams.Fee = types.MicroAlgos(uint64((len(lenderz)*2)+2) * app.txParams.MinFee)
	lenderz[0].Address = ToAddr
	lenderz[0].Amount = uint64(10000000)
	lenderz[0].Lsa, err = FetchLsigFromFile("./codec/lender_lsig_To.codec")
	if err != nil {
		fmt.Printf("FetchLsig found error, %s", err)
		return
	}
	lenderz[1].Address = ThirdAddr
	lenderz[1].Amount = uint64(10000000)
	lenderz[1].Lsa, err = FetchLsigFromFile("./codec/lender_lsig_Third.codec")
	if err != nil {
		fmt.Printf("FetchLsig found error, %s", err)
		return
	}

	app.foreignApps = append(app.foreignApps, LqtID)        // jina appID
	app.foreignAssets = append(app.foreignAssets, LFT_jina) // collateral assetID
	app.foreignAssets = append(app.foreignAssets, JUSD)     // jUSD assetID
	err = BorrowCallSmartContract(algodClient, app, lenderz, "./dryrun/borrow.msgp")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestLiquidate(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	app.appID = LqtID
	arg := make([][]byte, 1)
	arg[0] = []byte(ReserveAddr)
	app.appArgs = arg
	assetAmount := uint64(93200000)
	app.foreignAssets = append(app.foreignAssets, LFT_jina) // asset to be liquidated
	app.accounts = append(app.accounts, BonusAddr)          // account to liquidated
	app.accounts = append(app.accounts, ReserveAddr)        // account that recieves liquidation asset
	app.foreignApps = append(app.foreignApps, AppID)        // jina appID
	// get suggested transaction parameters
	app.txParams, err = algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		t.Errorf("Error getting suggested tx params: %s\n", err)
	}
	app.txParams.FlatFee = true
	app.txParams.Fee = 4000
	app.sk, err = mnemonic.ToPrivateKey(ToMn) // mnemonic of address that initiates liquidation
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	err = RepayClaimCallSmartContract(algodClient, app, USDCa, assetAmount, "./dryrun/liquidate.msgp")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestRepayCallSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	app.appID = AppID
	arg := make([][]byte, 1)
	arg[0] = []byte("repay")
	app.appArgs = arg
	app.foreignAssets = append(app.foreignAssets, LFT_jina)
	// get suggested transaction parameters
	app.txParams, err = algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		t.Errorf("Error getting suggested tx params: %s\n", err)
	}
	app.txParams.FlatFee = true
	app.txParams.Fee = 3000
	app.sk, err = mnemonic.ToPrivateKey(BonusMn) // mnemonic of address that wants to repay and unfreeze collateral
	assetAmount := uint64(10000000)
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	err = RepayClaimCallSmartContract(algodClient, app, USDCa, assetAmount, "./dryrun/repay.msgp")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestClaimCallSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	app.appID = AppID
	arg := make([][]byte, 1)
	arg[0] = []byte("claim")
	app.appArgs = arg
	app.foreignAssets = append(app.foreignAssets, USDCa)
	// get suggested transaction parameters
	app.txParams, err = algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		t.Errorf("Error getting suggested tx params: %s\n", err)
	}
	app.txParams.FlatFee = true
	app.txParams.Fee = 3000
	app.sk, err = mnemonic.ToPrivateKey(ThirdMn) // mnemonic of address that wants to call app
	assetAmount := uint64(10000000)
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	err = RepayClaimCallSmartContract(algodClient, app, JUSD, assetAmount, "./dryrun/claim.msgp")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCloseOutSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.CloseOutOC
	app.appID = AppID
	arg := make([][]byte, 1)
	arg[0] = []byte("close_out")
	app.appArgs = arg
	app.foreignAssets = append(app.foreignAssets, LFT_jina) // assets to unfreeze
	app.foreignAssets = append(app.foreignAssets, JUSD)     // assets to unfreeze
	app.sk, err = mnemonic.ToPrivateKey(ToMn)               // mnemonic of address that wants to call app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	txID, err := CallSmartContract(algodClient, app, nil, nil, "./dryrun/closeout.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestClearSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.ClearStateOC
	app.appID = AppID
	arg := make([][]byte, 1)
	arg[0] = []byte("clear")
	app.appArgs = arg
	app.foreignAssets = append(app.foreignAssets, LFT_jina) // assets to unfreeze
	app.foreignAssets = append(app.foreignAssets, JUSD)     // assets to unfreeze
	app.sk, err = mnemonic.ToPrivateKey(BonusMn)            // mnemonic of address that wants to call app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	app.txParams, err = algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		t.Errorf("Error getting suggested tx params: %s\n", err)
	}
	app.txParams.FlatFee = true
	app.txParams.Fee = 2000
	txID, err := CallSmartContract(algodClient, app, nil, nil, "./dryrun/clear.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCallSmartContractCreate(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	app.appID = AppID
	arg := make([][]byte, 2)
	var buf [1][8]byte
	binary.BigEndian.PutUint64(buf[0][:], LqtID) // Lquidator appID
	arg[0] = []byte("create")
	arg[1] = buf[0][:]
	app.appArgs = arg
	app.foreignAssets = append(app.foreignAssets, USDCa)
	app.foreignAssets = append(app.foreignAssets, JUSD)
	app.foreignApps = append(app.foreignApps, LqtID)
	app.accounts = append(app.accounts, ReserveAddr)
	app.txParams, err = algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		t.Errorf("Error getting suggested tx params: %s\n", err)
	}
	app.txParams.FlatFee = true
	app.txParams.Fee = 3000
	app.sk, err = mnemonic.ToPrivateKey(BorrowMn) // mnemonic of address that wants to call app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	txID, err := CallSmartContract(algodClient, app, nil, nil, "./dryrun/create.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestLiquidateChangeGlobalStates(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.NoOpOC
	app.appID = LqtID
	arg := make([][]byte, 2)
	var buf [2][8]byte
	binary.BigEndian.PutUint64(buf[0][:], AppID) // Lquidator appID
	binary.BigEndian.PutUint64(buf[1][:], AppID) // Lquidator appID
	arg[0] = buf[0][:]
	arg[1] = buf[1][:]
	app.appArgs = arg
	app.sk, err = mnemonic.ToPrivateKey(BorrowMn) // mnemonic of address that wants to update app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	txID, err := CallSmartContract(algodClient, app, nil, nil, "./dryrun/liquidate_jinaID.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestUpdateLiquidatorSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.UpdateApplicationOC
	app.appID = LqtID

	/*
		arg := make([][]byte, 2)
		var buf [2][8]byte
		binary.BigEndian.PutUint64(buf[0][:], AppID) // Lquidator appID
		binary.BigEndian.PutUint64(buf[1][:], AppID) // Lquidator appID
		arg[0] = buf[0][:]
		arg[1] = buf[1][:]
		app.appArgs = arg
	*/

	app.sk, err = mnemonic.ToPrivateKey(BorrowMn) // mnemonic of address that wants to update app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	// get approvalProg and clearProg as []byte
	clearProg, err := CompileSmartContractTeal(algodClient, "./teal/liquidatorClear.teal")
	if err != nil {
		t.Errorf("clearProg found error, %s", err)
	}
	approvalProg, err := CompileSmartContractTeal(algodClient, "./teal/liquidatorProg.teal")
	if err != nil {
		t.Errorf("approvalProg found error, %s", err)
	}
	txID, err := CallSmartContract(algodClient, app, clearProg, approvalProg, "./dryrun/update_liquidator.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestUpdateSmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	var app ApplicationTxnArgs
	app.onCompletion = types.UpdateApplicationOC
	app.appID = AppID
	app.sk, err = mnemonic.ToPrivateKey(BorrowMn) // mnemonic of address that wants to update app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}

	// get approvalProg and clearProg as []byte
	clearProg, err := CompileSmartContractTeal(algodClient, "./teal/clearProg.teal")
	if err != nil {
		t.Errorf("clearProg found error, %s", err)
	}
	approvalProg, err := CompileSmartContractTeal(algodClient, "./teal/approvalProg.teal")
	if err != nil {
		t.Errorf("approvalProg found error, %s", err)
	}
	txID, err := CallSmartContract(algodClient, app, clearProg, approvalProg, "./dryrun/update.msgp")
	t.Logf("txID = %s\n", txID)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCompileSmartContractTeal(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	compiledByte, err := CompileSmartContractTeal(algodClient, "./teal/clearProg.teal")
	t.Logf("compiledByte = %v\n", compiledByte)
	if err != nil {
		t.Errorf("clear.teal found error, %s", err)
	}
	compiledByte, err = CompileSmartContractTeal(algodClient, "./teal/approvalProg.teal")
	t.Logf("compiledByte = %v\n", compiledByte)
	if err != nil {
		t.Errorf("approval.teal found error, %s", err)
	}
}

func TestDeploySmartContract(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	var app ApplicationTxnArgs
	globalInts := uint64(2)
	globalBytes := uint64(0) // Liquidator address
	localInts := uint64(2)   // lender(assets allowed array, amount) borrower(collateral asset, amount, loan)
	localBytes := uint64(4)  // lsig of lender
	arg := make([][]byte, 2)
	var buf [1][8]byte
	binary.BigEndian.PutUint64(buf[0][:], LqtID) // Lquidator appID
	arg[0] = []byte("create")
	arg[1] = buf[0][:]
	app.appArgs = arg
	app.sk, err = mnemonic.ToPrivateKey(BorrowMn) // mnemonic of address that will create the app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}
	appId, err := DeploySmartContract(algodClient, app, globalInts, localInts, globalBytes, localBytes, "./teal/clearProg.teal", "./teal/approvalProg.teal")
	t.Logf("appId = %d\n", appId)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestDeploySmartContractLiquidator(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	var app ApplicationTxnArgs
	globalInts := uint64(2)
	globalBytes := uint64(0)
	localInts := uint64(0)
	localBytes := uint64(0)
	arg := make([][]byte, 2)
	//arg[0] = []byte("create")
	var buf [2][8]byte
	binary.BigEndian.PutUint64(buf[0][:], AppID) // jina
	binary.BigEndian.PutUint64(buf[1][:], AppID) // oracle
	arg[0] = buf[0][:]
	arg[1] = buf[1][:]
	app.appArgs = arg
	app.sk, err = mnemonic.ToPrivateKey(BorrowMn) // mnemonic of address that will create the app
	if err != nil {
		t.Errorf("Error recovering account key: %s\n", err)
	}
	app.sender, err = crypto.GenerateAddressFromSK(app.sk)
	if err != nil {
		t.Errorf("Error recovering account address: %s\n", err)
	}
	appId, err := DeploySmartContract(algodClient, app, globalInts, localInts, globalBytes, localBytes, "./teal/liquidatorClear.teal", "./teal/liquidatorProg.teal")
	t.Logf("appId = %d\n", appId)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}
