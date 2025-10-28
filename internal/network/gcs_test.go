package network

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/encoding"
	"os"
	"slices"
	"testing"
)

// Test vector structure from BIP 158
type BIP158TestVector struct {
	BlockHeight           int      `json:"blockHeight"`
	BlockHash             string   `json:"blockHash"`
	Block                 string   `json:"block"`
	PreviousOutputScripts []string `json:"previousOutputScripts"`
	PreviousBasicHeader   string   `json:"previousBasicHeader"`
	BasicFilter           string   `json:"basicFilter"`
	BasicHeader           string   `json:"basicHeader"`
	Notes                 string   `json:"notes"`
}

func TestBIP158Vectors(t *testing.T) {
	// Read test vectors from JSON file
	data, err := os.ReadFile("testdata/bip158-vectors.json")
	if err != nil {
		t.Skip("Test vectors not found. Download from: https://raw.githubusercontent.com/bitcoin/bips/master/bip-0158/testnet-19.json")
		return
	}

	var vectors []BIP158TestVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("Failed to parse test vectors: %v", err)
	}

	for _, vec := range vectors {
		t.Run(vec.Notes, func(t *testing.T) {
			// Decode expected filter
			expectedFilter, err := hex.DecodeString(vec.BasicFilter)
			if err != nil {
				t.Fatalf("Failed to decode expected filter: %v", err)
			}

			// parse block hex
			blockBytes, _ := hex.DecodeString(vec.Block)
			fullBlock, err := block.ParseFullBlock(bytes.NewReader(blockBytes))
			if err != nil {
				t.Fatalf("Failed to parse block: %v", err)
			}

			// parse previous output scripts
			prevScripts := make([][]byte, len(vec.PreviousOutputScripts))
			for i, scriptHex := range vec.PreviousOutputScripts {
				prevScripts[i], err = hex.DecodeString(scriptHex)
				if err != nil {
					t.Fatalf("Failed to decode prev script %d: %v", i, err)
				}
			}

			// Extract filter items
			items := fullBlock.ExtractBasicFilterItems(prevScripts)
			// build filter (using block hash as SipHash key)
			blockHashBytes, err := hex.DecodeString(vec.BlockHash)
			if err != nil {
				t.Fatalf("Failed to decode block hash: %v", err)
			}
			// Reverse to get internal byte order
			slices.Reverse(blockHashBytes)
			k0 := binary.LittleEndian.Uint64(blockHashBytes[0:8])
			k1 := binary.LittleEndian.Uint64(blockHashBytes[8:16])

			gcs, err := NewGCS(items, k0, k1)
			if err != nil {
				t.Fatalf("Failed to create GCS: %v", err)
			}
			// compare filter bytes
			filterBytes, err := gcs.Serialize()
			if err != nil {
				t.Errorf("Failed to serialize GCS: %v", err)
			}
			if !bytes.Equal(filterBytes, expectedFilter) {
				t.Errorf("Filter mismatch!\n  Got:      %x\n  Expected: %x", filterBytes, expectedFilter)
			}
		})
	}
}

func TestGCSBasic(t *testing.T) {
	// Simple test to verify GCS construction and matching
	items := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("bitcoin"),
	}

	k0 := uint64(0) // In real usage, derive from block hash
	k1 := uint64(0)

	// Construct filter
	gcs, err := NewGCS(items, k0, k1)
	if err != nil {
		t.Fatalf("Failed to create GCS: %v", err)
	}

	t.Logf("Created GCS with N=%d, P=%d, M=%d", gcs.NumItems, gcs.P, gcs.M)
	t.Logf("Filter size: %d bytes", len(gcs.data))

	// Test matching - should find items in filter
	for _, item := range items {
		match, err := gcs.Match(item, k0, k1)
		if err != nil {
			t.Errorf("Match failed for %s: %v", item, err)
		}
		if !match {
			t.Errorf("Expected to find %s in filter", item)
		}
	}

	// Test non-matching item - might get false positive
	notInFilter := []byte("ethereum")
	match, err := gcs.Match(notInFilter, k0, k1)
	if err != nil {
		t.Errorf("Match failed: %v", err)
	}
	if match {
		t.Logf("False positive for 'ethereum' (expected with P=19)")
	}
}

func TestGolombEncoding(t *testing.T) {
	// Test Golomb encoding/decoding round-trip
	testCases := []struct {
		value int
		p     int
	}{
		{0, 19},
		{1, 19},
		{100, 19},
		{524287, 19}, // 2^19 - 1
		{524288, 19}, // 2^19
	}

	for _, tc := range testCases {
		stream := encoding.NewBitStream()
		golombEncode(&stream, tc.value, tc.p)

		readStream := encoding.NewBitStreamFromSlice(stream.Bytes())
		decoded, err := golombDecode(&readStream, tc.p)
		if err != nil {
			t.Errorf("Failed to decode value %d: %v", tc.value, err)
		}

		if decoded != uint64(tc.value) {
			t.Errorf("Encode/decode mismatch: encoded %d, decoded %d", tc.value, decoded)
		}
	}
}

func TestHashToRange(t *testing.T) {
	// Test that hash-to-range produces values in expected range
	item := []byte("test")
	n := uint64(100)
	m := uint64(784931)
	k0 := uint64(0x123456789abcdef0)
	k1 := uint64(0xfedcba9876543210)

	hash, err := hashToRange(item, n, m, k0, k1)
	if err != nil {
		t.Fatalf("hashToRange failed: %v", err)
	}

	maxRange := n * m
	if hash >= maxRange {
		t.Errorf("Hash value %d exceeds range [0, %d)", hash, maxRange)
	}
}

func TestEmptyFilter(t *testing.T) {
	// Test creating filter with no items (like block 1414221 in test vectors)
	items := [][]byte{}

	k0 := uint64(0)
	k1 := uint64(0)

	gcs, err := NewGCS(items, k0, k1)
	if err != nil {
		t.Fatalf("Failed to create empty GCS: %v", err)
	}

	if gcs.NumItems != 0 {
		t.Errorf("Expected N=0 for empty filter, got %d", gcs.NumItems)
	}

	// Empty filter should not match anything
	match, err := gcs.Match([]byte("anything"), k0, k1)
	if err != nil {
		t.Errorf("Match on empty filter failed: %v", err)
	}
	if match {
		t.Error("Empty filter should not match anything")
	}
}
