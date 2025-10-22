package transactions_test

import (
	"go-bitcoin/internal/transactions"
	"testing"
)

func TestP2wpkhVerification(t *testing.T) {
	// Known P2WPKH (native SegWit) transaction
	// This is a real mainnet transaction spending from bc1q (bech32) address
	// Transaction: spending native SegWit outputs
	txHash := "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4"

	fetcher := transactions.NewTxFetcher()
	tx, err := fetcher.Fetch(txHash, false, false) // mainnet, not fresh
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Transaction: %s", tx)
	t.Logf("IsSegwit: %v", tx.IsSegwit)
	t.Logf("Number of inputs: %d", len(tx.Inputs))

	// Check if it's actually SegWit
	if !tx.IsSegwit {
		t.Fatal("Expected SegWit transaction")
	}

	// Verify each input
	for i, txin := range tx.Inputs {
		t.Logf("Input %d:", i)
		t.Logf("  Witness items: %d", len(txin.Witness))
		for j, item := range txin.Witness {
			t.Logf("    Item %d: %d bytes", j, len(item))
		}

		// Get the scriptPubKey being spent
		scriptPubKey, err := txin.ScriptPubKey(false)
		if err != nil {
			t.Fatalf("Error fetching scriptPubKey for input %d: %v", i, err)
		}

		t.Logf("  ScriptPubKey type:")
		if scriptPubKey.IsP2wpkhScriptPubKey() {
			t.Logf("    P2WPKH (native SegWit)")
		} else if scriptPubKey.IsP2shScriptPubKey() {
			t.Logf("    P2SH (possibly nested SegWit)")
		} else {
			t.Logf("    Other type")
		}
	}

	// Try to verify
	valid, err := tx.Verify()
	if err != nil {
		t.Fatalf("Verification error: %v", err)
	}

	if !valid {
		t.Fatal("Transaction verification failed")
	}

	t.Log("SUCCESS! SegWit transaction verified")
}

func TestNestedSegwitVerification(t *testing.T) {
	txHash := "c586389e5e4b3acb9d6c8be1c19ae8ab2795397633176f5a6442a261bbdefc3a"

	fetcher := transactions.NewTxFetcher()
	tx, err := fetcher.Fetch(txHash, false, false)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Transaction: %s", tx)
	t.Logf("IsSegwit: %v", tx.IsSegwit)

	// Log witness data
	for i, txin := range tx.Inputs {
		t.Logf("Input %d witness items: %d", i, len(txin.Witness))
		scriptPubKey, _ := txin.ScriptPubKey(false)
		t.Logf("  ScriptPubKey type: P2SH=%v P2WPKH=%v",
			scriptPubKey.IsP2shScriptPubKey(),
			scriptPubKey.IsP2wpkhScriptPubKey())
		t.Logf("  ScriptSig commands: %d", len(txin.ScriptSig.CommandStack))
	}

	// Try verification and log detailed error
	for i := range tx.Inputs {
		valid, err := tx.Verify()
		if err != nil {
			t.Fatalf("Input %d error: %v", i, err)
		}
		if !valid {
			t.Fatalf("Input %d verification returned false", i)
		}
	}

	t.Log("SUCCESS!")
}
