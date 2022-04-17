package jina

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"strings"
	"testing"

	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
)

var (
	usdc           = uint64(1) //10458941)
	collateral     = uint64(2)
	mng            = uint64(3)
	lqt            = uint64(4)
	jina           = uint64(6)
	jusd           = uint64(7)
	jna            = uint64(8)
	sandboxAddress = "http://localhost:4001"
	sandboxToken   = strings.Repeat("a", 64)
)

func TestConfigASA(t *testing.T) {
	//idempotencyKey := sha256.Sum256([]byte(fmt.Sprintf("%v",fields...)))
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	_, err = ConfigASA(algodClient, acct.PrivateKey, mng, jina, lqt, collateral)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestStart(t *testing.T) {
	// create USDC asset
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	_, err = Start(algodClient, acct)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCreateASA(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	err = CreateASA(algodClient, acct, 1000, 0, "LFT")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestOptinASA(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	err = OptinASA(algodClient, acct.PrivateKey, jusd)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestDeploy(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	//	algodClient, err := InitAlgodClient(AlgodAddressPurestake, AlgodTokenPurestake, "purestake")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	/*
		sk, err := mnemonic.ToPrivateKey(ThirdMn)
		if err != nil {
			t.Errorf("sk found error, %s", err)
		}
		acct, err := crypto.AccountFromPrivateKey(sk)
		if err != nil {
			t.Errorf("acct found error, %s", err)
		}
	*/

	mng, err = Deploy(algodClient, acct, usdc, "./abi/manager.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestUpdate(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = Update(algodClient, acct, "./abi/manager.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestFund(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]
	/*
		sk, err := mnemonic.ToPrivateKey(ToMn)
		if err != nil {
			t.Errorf("sk found error, %s", err)
		}
		acct, err := crypto.AccountFromPrivateKey(sk)
		if err != nil {
			t.Errorf("acct found error, %s", err)
		}
	*/

	err = Fund(algodClient, acct, mng, 10000000)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestCreateApps(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	lqtClear, err := CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearState found error, %s", err)
	}
	lqtApp, err := CompileSmartContractTeal(algodClient, "./teal/liquidatorApp.teal")
	if err != nil {
		log.Fatalf("liquidatorApp found error, %s", err)
	}
	jinaClear, err := CompileSmartContractTeal(algodClient, "./teal/jinaClear.teal")
	if err != nil {
		log.Fatalf("jinaClear found error, %s", err)
	}
	jinaApp, err := CompileSmartContractTeal(algodClient, "./teal/jinaApp.teal")
	if err != nil {
		log.Fatalf("jinaApp found error, %s", err)
	}

	_, err = CreateApps(algodClient, acct, usdc, lqtApp, lqtClear, jinaApp, jinaClear, "./abi/manager.json", "./abi/lqt.json", "./abi/jina.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestConfigureApps(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = ConfigureApps(algodClient, acct, lqt, jina, usdc, jusd, "./abi/manager.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestUsdc(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
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

	txn, _ := future.MakeAssetTransferTxn(acct.Address.String(), accts[2].Address.String(), 100000000, nil, txParams, "", usdc)
	signSendWait(algodClient, acct.PrivateKey, txn)
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestSendJusd(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	rec := crypto.GetApplicationAddress(jina)
	err = SendJusd(algodClient, acct, rec, jusd, "./abi/manager.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestChildUpdate(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[1]

	err = ChildUpdate(algodClient, acct, jina, "./teal/jinaApp.teal", "./teal/jinaClear.teal", "./abi/manager.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestOptin(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	err = Optin(algodClient, acct, mng, "./abi/jina.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}
}

func TestEarn(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	xids := []uint64{collateral, jusd, jna}
	aamt := uint64(100000000)
	lvr := uint64(172800) //+ uint64(txParams.FirstRoundValid)

	lsigArgs := make([][]byte, 4)
	var buf [4][8]byte
	binary.BigEndian.PutUint64(buf[0][:], usdc) // USDCa asset ID
	binary.BigEndian.PutUint64(buf[1][:], aamt) // loan available (50 USDCa)
	binary.BigEndian.PutUint64(buf[2][:], lvr)  // Expiring lifespan: 17280 rounds == 1 day
	binary.BigEndian.PutUint64(buf[3][:], jina) // jina appID
	lsigArgs[0] = buf[0][:]
	lsigArgs[1] = buf[1][:]
	lsigArgs[2] = buf[2][:]
	lsigArgs[3] = buf[3][:]

	lsaRaw := CompileToLsig(algodClient, lsigArgs, "./teal/logicSigDelegated.teal", "./codec/lender_lsig.codec", acct.PrivateKey)
	if lsaRaw.SigningKey == nil {
		t.Errorf("lsig is empty")
	}
	lsa := sha256.Sum256(lsaRaw.Lsig.Logic)

	err = Earn(algodClient, acct, xids, aamt, lvr, lsa[:4], "./abi/jina.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestClaim(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[0]

	amt := uint64(10000000)

	err = Claim(algodClient, acct, mng, amt, usdc, jusd, "./abi/jina.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestBorrow(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	xids := []uint64{collateral}
	camt := []uint64{20}
	lamt := []uint64{10000000}

	err = Borrow(algodClient, acct, accts[0], usdc, jusd, mng, lqt, xids, camt, lamt, "./codec/lender_lsig.codec", "./abi/jina.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}

func TestRepay(t *testing.T) {
	algodClient, err := InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	accts, err := GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	acct := accts[2]

	xids := []uint64{collateral}
	ramt := []uint64{10000000}

	err = Repay(algodClient, acct, mng, lqt, usdc, xids, ramt, "./abi/jina.json")
	if err != nil {
		t.Errorf("test found error, %s", err)
	}

}
