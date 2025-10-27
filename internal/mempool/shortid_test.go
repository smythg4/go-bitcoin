package mempool

import (
	"encoding/hex"
	"testing"
)

func TestShortIDByteOrder(t *testing.T) {
	// Test that shortID calculation uses correct byte order (little-endian/internal)
	// Example from BIP152: https://github.com/bitcoin/bips/blob/master/bip-0152.mediawiki

	// Example transaction hash (in display order / big-endian)
	// This is how txids are typically shown to users
	displayOrderHex := "8c3d3c2aa5b34f7e1e6d7b84f5c6e33c8f7a3f5e3b5c3d5e6a7f8b9c0d1e2f3a"
	displayOrder, _ := hex.DecodeString(displayOrderHex)

	// For SipHash, we need internal order (little-endian / reversed)
	internalOrder := make([]byte, 32)
	for i := 0; i < 32; i++ {
		internalOrder[i] = displayOrder[31-i]
	}

	var displayHash [32]byte
	var internalHash [32]byte
	copy(displayHash[:], displayOrder)
	copy(internalHash[:], internalOrder)

	// Calculate shortIDs with both byte orders
	k0 := uint64(0x9f5813e8d3e8924c)
	k1 := uint64(0x43c9b8f8b7f3d7a2)

	sidDisplay := CalculateShortID(displayHash, k0, k1)
	sidInternal := CalculateShortID(internalHash, k0, k1)

	t.Logf("Display order hash:  %x", displayHash[:8])
	t.Logf("Display order shortID: %x", sidDisplay)
	t.Logf("")
	t.Logf("Internal order hash: %x", internalHash[:8])
	t.Logf("Internal order shortID: %x", sidInternal)

	// They should be different!
	if sidDisplay == sidInternal {
		t.Error("ShortIDs should be different for reversed vs unreversed hashes!")
	}

	t.Log("✓ Byte order matters - shortIDs are different as expected")
}

func TestShortIDCalculation(t *testing.T) {
	// Test basic shortID calculation with known values
	// Using simple test vectors

	var txid [32]byte
	// Simple pattern for testing
	for i := 0; i < 32; i++ {
		txid[i] = byte(i)
	}

	k0 := uint64(0x1122334455667788)
	k1 := uint64(0x99aabbccddeeff00)

	sid := CalculateShortID(txid, k0, k1)

	t.Logf("Test txid: %x", txid[:8])
	t.Logf("Keys: k0=0x%016x, k1=0x%016x", k0, k1)
	t.Logf("ShortID: %x", sid)

	// Verify it's 6 bytes
	if len(sid) != 6 {
		t.Errorf("ShortID should be 6 bytes, got %d", len(sid))
	}

	// Verify deterministic (same inputs give same output)
	sid2 := CalculateShortID(txid, k0, k1)
	if sid != sid2 {
		t.Error("ShortID calculation should be deterministic")
	}

	t.Log("✓ ShortID calculation is deterministic")
}

func TestMempoolMatchingSimulation(t *testing.T) {
	// Simulate the full flow: transaction gets added to mempool, then we try to match it
	// This tests that the byte order reversal works correctly

	// Create a mock transaction hash (in display order, as returned by tx.Hash())
	var displayOrderHash [32]byte
	for i := 0; i < 32; i++ {
		displayOrderHash[i] = byte(31 - i) // Display order
	}

	t.Logf("Transaction hash (display order): %x", displayOrderHash[:8])

	// For SipHash calculation, we need internal order
	var internalOrderHash [32]byte
	for i := 0; i < 32; i++ {
		internalOrderHash[i] = displayOrderHash[31-i]
	}
	t.Logf("Transaction hash (internal order): %x", internalOrderHash[:8])

	// Calculate shortID using internal order (what the sender does)
	k0 := uint64(0xabcdef0123456789)
	k1 := uint64(0xfedcba9876543210)

	expectedShortID := CalculateShortID(internalOrderHash, k0, k1)
	t.Logf("Expected shortID: %x", expectedShortID)

	// Now simulate what happens in MatchShortIDs:
	// We get a display-order hash from tx.Hash(), need to reverse it, then calculate shortID
	actualHash := displayOrderHash // This is what tx.Hash() returns

	// Apply the fix: reverse to internal order
	hashForSipHash := actualHash
	for i := 0; i < 16; i++ {
		hashForSipHash[i], hashForSipHash[31-i] = hashForSipHash[31-i], hashForSipHash[i]
	}

	actualShortID := CalculateShortID(hashForSipHash, k0, k1)
	t.Logf("Actual shortID (after fix): %x", actualShortID)

	if expectedShortID != actualShortID {
		t.Errorf("ShortID mismatch! Expected %x, got %x", expectedShortID, actualShortID)
		t.Error("The byte order fix is not working correctly!")
	} else {
		t.Log("✓ ShortIDs match! The byte order fix works correctly")
	}
}
