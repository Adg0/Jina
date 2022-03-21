package jina

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"

	"github.com/algorand/go-algorand-sdk/future"
)

func waitForConfirmation(txID string, client *algod.Client) {
	status, err := client.Status().Do(context.Background())
	if err != nil {
		fmt.Printf("error getting algod status: %s\n", err)
		return
	}
	lastRound := status.LastRound
	for {
		pt, _, err := client.PendingTransactionInformation(txID).Do(context.Background())
		if err != nil {
			fmt.Printf("error getting pending transaction: %s\n", err)
			return
		}
		if pt.ConfirmedRound > 0 {
			fmt.Printf("Transaction "+txID+" confirmed in round %d\n", pt.ConfirmedRound)
			break
		}
		fmt.Printf("waiting for confirmation\n")
		lastRound++
		status, err = client.StatusAfterBlock(lastRound).Do(context.Background())
	}
}

// fetch LogicSig from file
func FetchLsigFromFile(filename string) (lsa crypto.LogicSigAccount, err error) {

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	lsigJSON, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("failed to read file: %s\n", err)
		return
	}
	err = json.Unmarshal(lsigJSON, &lsa)
	if err != nil {
		fmt.Printf("failed to json unmarshal: %s\n", err)
	}
	return
}

// function that dispenses set amount of asset, from a signed Delegated LogicSig
func DispenseAsset(algodClient *algod.Client, reserve, recipient string, assetamount, assetID uint64, codecFile string) (err error) {

	// Get network-related transaction parameters and assign
	txParams, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		fmt.Printf("Error getting suggested tx params: %s\n", err)
		return
	}
	// comment out the next two (2) lines to use suggested fees
	txParams.FlatFee = true
	txParams.Fee = 1000

	// TRANSFER ASSET
	closeRemainderTo := ""
	note := []byte(nil)
	txn, err := future.MakeAssetTransferTxn(reserve, recipient, assetamount, note, txParams, closeRemainderTo, assetID)
	if err != nil {
		fmt.Printf("Failed to send transaction MakeAssetTransfer Txn: %s\n", err)
		return
	}

	// fetch Delegated LogicSig from file
	lsa, err := FetchLsigFromFile(codecFile)
	if err != nil {
		fmt.Printf("Failed to fetch LogicSigAccount from file: %s\n", err)
		return
	}

	// sign the transaction
	txid, stx, err := crypto.SignLogicSigAccountTransaction(lsa, txn)
	if err != nil {
		fmt.Printf("Failed to sign transaction: %s\n", err)
		return
	}
	fmt.Printf("Transaction ID: %s\n", txid)

	// Broadcast the transaction to the network
	sendResponse, err := algodClient.SendRawTransaction(stx).Do(context.Background())
	if err != nil {
		fmt.Printf("failed to send transaction: %s\n", err)
		return
	}
	fmt.Printf("Submitted transaction %s\n", sendResponse)

	// Wait for transaction to be confirmed
	waitForConfirmation(txid, algodClient)
	return
}
