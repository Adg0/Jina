package main

import (
	"log"
	"strings"

	"github.com/Adg0/Jina"
)

var (
	usdc           = uint64(10458941)
	sandboxAddress = "http://localhost:4001"
	sandboxToken   = strings.Repeat("a", 64)
)

func main() {
	algodClient, err := Jina.InitAlgodClient(sandboxAddress, sandboxToken, "local")
	if err != nil {
		log.Fatalf("algodClient found error: %s", err)
	}
	// Three accounts
	// accts[0] is creator of dapp
	// accts[1] is manager of NFT collateral (default holder of collateral)
	// accts[2] is liquidity provider
	// any account that holds collateral NFT configured to have appropriate admin addresses can be borrower after optin to jina app
	accts, err := Jina.GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}
	// Create USDC asset for sandbox
	usdc, err = Jina.Start(algodClient, accts[0])
	if err != nil {
		log.Fatalf("Start found error: %s", err)
	}
	// Create NFT for sandbox
	collateral, err := Jina.CreateASA(algodClient, accts[1], 1000, 0, "LFT", "https://")
	if err != nil {
		log.Fatalf("Create NFT found error: %s", err)
	}

	// Deploy manager contract
	mng, err := Jina.Deploy(algodClient, accts[0], usdc, "./abi/manager.json")
	if err != nil {
		log.Fatalf("Deploying found error: %s", err)
	}
	err = Jina.Fund(algodClient, accts[0], mng, 10000000)
	if err != nil {
		log.Fatalf("Funding contract found error: %s", err)
	}

	// Create child apps
	lqtClear, err := Jina.CompileSmartContractTeal(algodClient, "./teal/clearState.teal")
	if err != nil {
		log.Fatalf("clearState found error, %s", err)
	}
	lqtApp, err := Jina.CompileSmartContractTeal(algodClient, "./teal/liquidatorApp.teal")
	if err != nil {
		log.Fatalf("liquidatorApp found error, %s", err)
	}
	jinaClear, err := Jina.CompileSmartContractTeal(algodClient, "./teal/jinaClear.teal")
	if err != nil {
		log.Fatalf("jinaClear found error, %s", err)
	}
	jinaApp, err := Jina.CompileSmartContractTeal(algodClient, "./teal/jinaApp.teal")
	if err != nil {
		log.Fatalf("jinaApp found error, %s", err)
	}
	ids, err := Jina.CreateApps(algodClient, accts[0], usdc, lqtApp, lqtClear, jinaApp, jinaClear, "./abi/manager.json", "./abi/lqt.json", "./abi/jina.json")
	lqt := ids[0]
	jina := ids[1]
	jusd := ids[2]
	jna := ids[3]
	err = Jina.ConfigureApps(algodClient, accts[0], lqt, jina, usdc, jusd, "./abi/manager.json")
	if err != nil {
		log.Fatalf("Configuring created apps found error: %s", err)
	}
	err = Jina.ConfigASA(algodClient, accts[1].PrivateKey, mng, jina, lqt, collateral)
	if err != nil {
		log.Fatalf("Configuring NFT found error: %s", err)
	}
}
