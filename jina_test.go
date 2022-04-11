package jina

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"testing"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
)

func TestConfigASA(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	err = ConfigASA(algodClient, acct.PrivateKey, 2, 55, 54, 86)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestStart(t *testing.T) {
	// create USDC asset
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	err = Start(algodClient, acct)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestOptinASA(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	err = OptinASA(algodClient, acct.PrivateKey, 56)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestDeploy(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = Deploy(algodClient, acct, 1, "./abi/manager.json", "./dryrun/app.msgp", "./dryrun/response/app.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestUpdate(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = Update(algodClient, acct, "./abi/manager.json", "./dryrun/update.msgp", "./dryrun/response/update.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestFund(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	err = Fund(algodClient, acct, 2, 100000000)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCreateApps(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]
	// get approvalProg and clearProg as []byte
	lqtClear, err := CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearState found error, %s", err)
	}
	lqtApp, err := CompileSmartContractTeal(algodClient, "./teal/liquidatorProg.teal")
	if err != nil {
		log.Fatalf("liquidatorProg found error, %s", err)
	}
	jinaClear, err := CompileSmartContractTeal(algodClient, "./teal/jinaClear.teal")
	if err != nil {
		log.Fatalf("jinaClear found error, %s", err)
	}
	jinaApp, err := CompileSmartContractTeal(algodClient, "./teal/approvalProg.teal")
	if err != nil {
		log.Fatalf("approvalProg found error, %s", err)
	}
	fmt.Printf("address: %s\n", crypto.GetApplicationAddress(2).String())
	err = CreateApps(algodClient, acct, 1, lqtApp, lqtClear, jinaApp, jinaClear, "./abi/manager.json", "./dryrun/create_apps.msgp", "./dryrun/response/create_apps.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestConfigureApps(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = ConfigureApps(algodClient, acct, 54, 55, 1, 56, "./abi/manager.json", "./dryrun/config.msgp", "./dryrun/response/config.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestUsdc(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggeted params: %+v", err)
	}

	rec := "Z5PCDU5SNKRFJIIOIZDY2PUUQ4RTV4RZYVKHJPPYKCQLGNGGEGCZD5PDT4"
	//rec := "U2PARS7KSZ3XIY6OU45Q43UHRCYYCRS7A32RRYFQER2YCCWE4ARKYB2WXQ"
	txn, _ := future.MakeAssetTransferTxn(acct.Address.String(), rec, 100000000, nil, txParams, "", 1)
	signSendWait(algodClient, acct.PrivateKey, txn)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestSendJusd(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	rec := crypto.GetApplicationAddress(55)
	err = SendJusd(algodClient, acct, rec, 56, "./abi/manager.json", "./dryrun/sendJusd.msgp", "./dryrun/response/sendJusd.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestChildUpdate(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = ChildUpdate(algodClient, acct, 55, "./teal/approvalProg.teal", "./teal/jinaClear.teal", "./abi/manager.json", "./dryrun/update_child.msgp", "./dryrun/response/update_child.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestOptin(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	err = Optin(algodClient, acct, 2, "./abi/jina.json", "./dryrun/optin.msgp", "./dryrun/response/optin.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestEarn(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	xids := []uint64{86, 56, 57}
	aamt := uint64(100000000)
	lvr := uint64(172800) //+ uint64(txParams.FirstRoundValid)

	lsigArgs := make([][]byte, 4)
	var buf [4][8]byte
	binary.BigEndian.PutUint64(buf[0][:], 1)    // USDCa asset ID
	binary.BigEndian.PutUint64(buf[1][:], aamt) // loan available (50 USDCa)
	binary.BigEndian.PutUint64(buf[2][:], lvr)  // Expiring lifespan: 17280 rounds == 1 day
	binary.BigEndian.PutUint64(buf[3][:], 55)   // jina appID
	lsigArgs[0] = buf[0][:]
	lsigArgs[1] = buf[1][:]
	lsigArgs[2] = buf[2][:]
	lsigArgs[3] = buf[3][:]

	lsaRaw := CompileToLsig(algodClient, lsigArgs, "./teal/logicSigDelegated.teal", "./codec/lender_lsig.codec", acct.PrivateKey)
	if lsaRaw.SigningKey == nil {
		t.Errorf("lsig is empty")
	}
	lsa := sha256.Sum256(lsaRaw.Lsig.Logic)

	err = Earn(algodClient, acct, xids, aamt, lvr, lsa[:4], "./abi/jina.json", "./dryrun/earn.msgp", "./dryrun/response/earn.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestClaim(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	amt := uint64(100000000)

	err = Claim(algodClient, acct, 55, amt, 1, 56, "./abi/jina.json", "./dryrun/claim.msgp", "./dryrun/response/claim.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestBorrow(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	xids := []uint64{86}
	camt := []uint64{20}
	lamt := []uint64{10000000}

	err = Borrow(algodClient, acct, accts[0], 1, xids, camt, lamt, "./codec/lender_lsig.codec", "./abi/jina.json", "./dryrun/borrow.msgp", "./dryrun/response/borrow.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestRepay(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	xids := []uint64{86}
	ramt := []uint64{1000000}

	err = Repay(algodClient, acct, 55, 1, xids, ramt, "./abi/jina.json", "./dryrun/repay.msgp", "./dryrun/response/repay.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}
