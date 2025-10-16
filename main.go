package main

import (
	"fmt"
	"go-bitcoin/internal/transactions"
	"log"
	"time"
)

func main() {
	fmt.Println("=== Verifying Real Bitcoin Transactions ===")
	fetcher := transactions.NewTxFetcher()

	// Fetch up to 3 recent legacy transactions from testnet
	// Check 15 tx per block, check last 10 blocks, 60 second timeout
	fmt.Println("Fetching recent legacy transactions from testnet (skipping SegWit)...")
	txIds, err := fetcher.FetchRecentLegacyTxIds(true, 3, 15, 10, 60*time.Second)
	if err != nil {
		fmt.Printf("Error fetching recent txIds: %v\n", err)
		log.Fatal()
	}

	if len(txIds) == 0 {
		fmt.Println("\nNo legacy transactions found in recent blocks (all SegWit)")
		fmt.Println("Legacy P2PKH transactions are rare on modern testnet. Try again later or use mainnet.")
		return
	}

	fmt.Printf("Found %d legacy transaction(s)\n\n", len(txIds))

	// Verify each transaction
	successCount := 0
	failCount := 0

	for txNum, txId := range txIds {
		fmt.Printf("=== Transaction %d/%d ===\n", txNum+1, len(txIds))
		fmt.Printf("TxID: %s\n", txId)

		tx, err := fetcher.Fetch(txId, true, false)
		if err != nil {
			fmt.Printf("Error fetching transaction: %v\n\n", err)
			failCount++
			continue
		}

		// Verify each input
		allValid := true
		for i, input := range tx.Inputs {
			// fetch the previous transaction
			prevTxId := fmt.Sprintf("%x", input.PrevTx)
			prevTx, err := fetcher.Fetch(prevTxId, true, false)
			if err != nil {
				fmt.Printf("  Input %d: Error fetching previous tx: %v\n", i, err)
				allValid = false
				continue
			}

			// get the previous output's ScriptPubKey
			prevOutput := prevTx.Outputs[input.PrevIdx]

			// calculate signature hash for this input
			z := tx.SigHash(i, prevOutput.ScriptPubKey)

			// combine ScriptSig + ScriptPubKey
			combinedScript := input.ScriptSig.Combine(prevOutput.ScriptPubKey)

			// evaluate
			valid := combinedScript.Evaluate(z)
			if valid {
				fmt.Printf("  Input %d: ✓ VALID\n", i)
			} else {
				fmt.Printf("  Input %d: ✗ INVALID\n", i)
				allValid = false
			}
		}

		if allValid {
			fmt.Printf("Result: ✓ All inputs verified\n\n")
			successCount++
		} else {
			fmt.Printf("Result: ✗ Verification failed\n\n")
			failCount++
		}
	}

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("Total: %d transactions\n", len(txIds))
	fmt.Printf("Verified: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
}
