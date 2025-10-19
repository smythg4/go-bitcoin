package main

import (
	"fmt"
	"go-bitcoin/internal/transactions"
	"time"
)

func main() {

	fetcher := transactions.NewTxFetcher()
	txIds, err := fetcher.FetchAddressTransactions("2MtmgCyjRyf8QcpLiU9BpQLrmEvFKRMFCG9", true)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d transactions for address\n", len(txIds))

	// Verify each transaction
	for i, txId := range txIds {
		fmt.Printf("\n=== Transaction %d/%d ===\n", i+1, len(txIds))
		fmt.Printf("TxID: %s\n", txId)
		time.Sleep(2 * time.Second)

		tx, err := fetcher.Fetch(txId, true, false)
		if err != nil {
			fmt.Printf("Error fetching: %v\n", err)
			continue
		}

		valid, err := tx.Verify()
		if err != nil {
			fmt.Printf("Error verifying: %v\n", err)
			continue
		}

		if valid {
			fmt.Println("✓ Transaction is VALID!")
			// Add this to see what type it is
			for i, input := range tx.Inputs {
				scriptPubKey, _ := input.ScriptPubKey(true)
				fmt.Printf("  Input %d ScriptPubKey commands: %d\n", i,
					len(scriptPubKey.CommandStack))
				fmt.Printf("  Input %d ScriptSig commands: %d\n", i,
					len(input.ScriptSig.CommandStack))
			}
		} else {
			fmt.Println("✗ Transaction is INVALID")
			// Add: let's see what the scripts look like
			for i, input := range tx.Inputs {
				scriptPubKey, _ := input.ScriptPubKey(true)
				fmt.Printf("  Input %d ScriptPubKey commands: %d\n", i,
					len(scriptPubKey.CommandStack))
				fmt.Printf("  Input %d ScriptSig commands: %d\n", i,
					len(input.ScriptSig.CommandStack))
			}
		}
	}
}
