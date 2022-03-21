package jina

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"testing"

	"github.com/algorand/go-algorand-sdk/mnemonic"
)

func TestCompileToLsig(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting suggested tx params: %s\n", err)
		return
	}

	sk, err := mnemonic.ToPrivateKey(ToMn) // signer account
	if err != nil {
		fmt.Printf("Error recovering account: %s\n", err)
		return
	}

	lsigArgs := make([][]byte, 4)
	var buf [4][8]byte
	binary.BigEndian.PutUint64(buf[0][:], USDCa)                                   // USDCa asset ID
	binary.BigEndian.PutUint64(buf[1][:], 50000000)                                // loan available (50 USDCa)
	binary.BigEndian.PutUint64(buf[2][:], 172800+uint64(txParams.FirstRoundValid)) // Expiring lifespan: 17280 rounds == 1 day
	binary.BigEndian.PutUint64(buf[3][:], AppID)                                   // LFT-jina asset ID
	lsigArgs[0] = buf[0][:]
	lsigArgs[1] = buf[1][:]
	lsigArgs[2] = buf[2][:]
	lsigArgs[3] = buf[3][:]

	lsa := CompileToLsig(algodClient, lsigArgs, "./teal/logicSigDelegated.teal", "./codec/lender_lsig_To.codec", sk)
	if lsa.SigningKey == nil {
		t.Errorf("lsig is empty")
	}
}

func TestCompileToLsigDispenser(t *testing.T) {
	algodClient, err := InitAlgodClient(AlgodAddressSandbox, AlgodTokenSandbox, "local")
	if err != nil {
		t.Errorf("algodClient found error, %s", err)
	}

	sk, err := mnemonic.ToPrivateKey(ReserveMn) // signer account
	if err != nil {
		fmt.Printf("Error recovering account: %s\n", err)
		return
	}

	lsigArgs := make([][]byte, 2)
	var buf [2][8]byte
	binary.BigEndian.PutUint64(buf[0][:], LFT_jina) // JUSD asset ID
	binary.BigEndian.PutUint64(buf[1][:], 4)        // maximum one time dispense
	lsigArgs[0] = buf[0][:]
	lsigArgs[1] = buf[1][:]

	lsa := CompileToLsig(algodClient, lsigArgs, "./teal/dispense.teal", "./codec/dispenserLFT.codec", sk)
	if lsa.SigningKey == nil {
		t.Errorf("lsig is empty")
	}
}
