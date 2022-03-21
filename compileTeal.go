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

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
)

func CompileToLsig(algodClient *algod.Client, args [][]byte, osTealFile, codecFile string, sk ed25519.PrivateKey) (lsa crypto.LogicSigAccount) {

	// the Teal program to compile
	file, err := os.Open(osTealFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	tealFile, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("failed to read teal file: %s\n", err)
		return
	}

	// compile teal program
	response, err := algodClient.TealCompile(tealFile).Do(context.Background())
	// print response
	fmt.Printf("Hash = %s\n", response.Hash)
	fmt.Printf("Result = %s\n", response.Result)

	program, err := base64.StdEncoding.DecodeString(response.Result)
	if err != nil {
		fmt.Printf("Error decoding program: %s\n", err)
		return
	}

	// Signing the TEAL
	lsa, err = crypto.MakeLogicSigAccountDelegated(program, args, sk)
	if err != nil {
		fmt.Printf("Error making Delegated logicSig account: %s\n", err)
		return
	}
	fileL, _ := json.MarshalIndent(lsa, "", "")
	err = ioutil.WriteFile(codecFile, fileL, 0644)
	if err != nil {
		fmt.Printf("failed to write file: %s\n", err)
		return
	}
	return
}
