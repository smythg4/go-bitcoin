package script

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestSimpleArithmeticScript(t *testing.T) {
	// Test: OP_2 OP_DUP OP_DUP OP_MUL OP_ADD OP_6 OP_EQUAL
	// Should evaluate to: 2^2 + 2 = 6, then 6 == 6 -> true

	// ScriptSig: just OP_2
	scriptSigHex := []byte{0x01, 0x52} // varint length + OP_2
	scriptSig, err := ParseScript(bytes.NewReader(scriptSigHex))
	if err != nil {
		t.Fatalf("Error parsing scriptSig: %v", err)
	}

	// ScriptPubKey: OP_DUP OP_DUP OP_MUL OP_ADD OP_6 OP_EQUAL
	scriptPubKeyHex := []byte{0x06, 0x76, 0x76, 0x95, 0x93, 0x56, 0x87}
	scriptPubKey, err := ParseScript(bytes.NewReader(scriptPubKeyHex))
	if err != nil {
		t.Fatalf("Error parsing scriptPubKey: %v", err)
	}

	// Combine and evaluate
	combined := scriptSig.Combine(scriptPubKey)
	result := combined.Evaluate([]byte{})

	if !result {
		t.Errorf("Simple arithmetic script failed, expected true")
	}
}

func TestSHA1CollisionScriptWithIdenticalValues(t *testing.T) {
	// Test: OP_2DUP OP_EQUAL OP_NOT OP_VERIFY OP_SHA1 OP_SWAP OP_SHA1 OP_EQUAL
	// With x=y, should fail at OP_NOT OP_VERIFY (same values not allowed)

	x := []byte("hello")
	y := []byte("hello") // same value - should fail

	scriptSig := NewScript([]ScriptCommand{
		{Data: x, IsData: true},
		{Data: y, IsData: true},
	})

	// ScriptPubKey: 6e879169a77ca787
	scriptPubKeyHex := []byte{0x08, 0x6e, 0x87, 0x91, 0x69, 0xa7, 0x7c, 0xa7, 0x87}
	scriptPubKey, err := ParseScript(bytes.NewReader(scriptPubKeyHex))
	if err != nil {
		t.Fatalf("Error parsing scriptPubKey: %v", err)
	}

	combined := scriptSig.Combine(scriptPubKey)
	result := combined.Evaluate([]byte{})

	if result {
		t.Errorf("SHA-1 collision script with identical values should fail (x != y required)")
	}
}

func TestSHA1CollisionScriptWithDifferentValues(t *testing.T) {
	// Test: OP_2DUP OP_EQUAL OP_NOT OP_VERIFY OP_SHA1 OP_SWAP OP_SHA1 OP_EQUAL
	// With different values but no collision, should fail at final OP_EQUAL

	x := []byte("hello")
	y := []byte("world") // different values, different hashes

	scriptSig := NewScript([]ScriptCommand{
		{Data: x, IsData: true},
		{Data: y, IsData: true},
	})

	// ScriptPubKey: 6e879169a77ca787
	scriptPubKeyHex := []byte{0x08, 0x6e, 0x87, 0x91, 0x69, 0xa7, 0x7c, 0xa7, 0x87}
	scriptPubKey, err := ParseScript(bytes.NewReader(scriptPubKeyHex))
	if err != nil {
		t.Fatalf("Error parsing scriptPubKey: %v", err)
	}

	combined := scriptSig.Combine(scriptPubKey)
	result := combined.Evaluate([]byte{})

	if result {
		t.Errorf("SHA-1 collision script with different values/hashes should fail (need actual collision)")
	}
}

func TestNumberEncoding(t *testing.T) {
	tests := []struct {
		num      int64
		expected []byte
	}{
		{0, []byte{}},
		{1, []byte{0x01}},
		{2, []byte{0x02}},
		{127, []byte{0x7f}},
		{128, []byte{0x80, 0x00}},    // needs extra byte because 0x80 has sign bit set
		{255, []byte{0xff, 0x00}},    // needs extra byte
		{256, []byte{0x00, 0x01}},    // little-endian
		{-1, []byte{0x81}},           // sign bit set
		{-127, []byte{0xff}},         // 0x7f with sign bit = 0xff
		{-128, []byte{0x80, 0x80}},   // needs extra byte
	}

	for _, test := range tests {
		encoded := encodeNum(test.num)
		if !bytes.Equal(encoded, test.expected) {
			t.Errorf("encodeNum(%d) = %x, expected %x", test.num, encoded, test.expected)
		}

		// Test round-trip
		decoded := decodeNum(encoded)
		if decoded != test.num {
			t.Errorf("decodeNum(encodeNum(%d)) = %d, expected %d", test.num, decoded, test.num)
		}
	}
}

func TestNumberDecoding(t *testing.T) {
	tests := []struct {
		data     []byte
		expected int64
	}{
		{[]byte{}, 0},
		{[]byte{0x01}, 1},
		{[]byte{0x02}, 2},
		{[]byte{0x7f}, 127},
		{[]byte{0x80, 0x00}, 128},
		{[]byte{0x00, 0x01}, 256}, // little-endian
		{[]byte{0x81}, -1},
		{[]byte{0xff}, -127},
		{[]byte{0x80, 0x80}, -128},
	}

	for _, test := range tests {
		decoded := decodeNum(test.data)
		if decoded != test.expected {
			t.Errorf("decodeNum(%x) = %d, expected %d", test.data, decoded, test.expected)
		}
	}
}

func TestSHA1CollisionScriptWithActualCollision(t *testing.T) {
	// Test with actual SHAttered collision from Google/CWI (2017)
	// These two different blocks produce the same SHA-1 hash

	// Collision block 1
	prefix1Hex := "255044462d312e330a25e2e3cfd30a0a0a312030206f626a0a3c3c2f57696474682032203020522f4865696768742033203020522f547970652034203020522f537562747970652035203020522f46696c7465722036203020522f436f6c6f7253706163652037203020522f4c656e6774682038203020522f42697473506572436f6d706f6e656e7420383e3e0a73747265616d0affd8fffe00245348412d3120697320646561642121212121852fec092339759c39b1a1c63c4c97e1fffe017346dc9166b67e118f029ab621b2560ff9ca67cca8c7f85ba84c79030c2b3de218f86db3a90901d5df45c14f26fedfb3dc38e96ac22fe7bd728f0e45bce046d23c570feb141398bb552ef5a0a82be331fea48037b8b5d71f0e332edf93ac3500eb4ddc0decc1a864790c782c76215660dd309791d06bd0af3f98cda4bc4629b1"
	prefix1, err := hex.DecodeString(prefix1Hex)
	if err != nil {
		t.Fatalf("Failed to decode collision block 1: %v", err)
	}

	// Collision block 2 (different bytes, same SHA-1)
	prefix2Hex := "255044462d312e330a25e2e3cfd30a0a0a312030206f626a0a3c3c2f57696474682032203020522f4865696768742033203020522f547970652034203020522f537562747970652035203020522f46696c7465722036203020522f436f6c6f7253706163652037203020522f4c656e6774682038203020522f42697473506572436f6d706f6e656e7420383e3e0a73747265616d0affd8fffe00245348412d3120697320646561642121212121852fec092339759c39b1a1c63c4c97e1fffe017f46dc93a6b67e013b029aaa1db2560b45ca67d688c7f84b8c4c791fe02b3df614f86db1690901c56b45c1530afedfb76038e972722fe7ad728f0e4904e046c230570fe9d41398abe12ef5bc942be33542a4802d98b5d70f2a332ec37fac3514e74ddc0f2cc1a874cd0c78305a21566461309789606bd0bf3f98cda8044629a1"
	prefix2, err := hex.DecodeString(prefix2Hex)
	if err != nil {
		t.Fatalf("Failed to decode collision block 2: %v", err)
	}

	// Verify they're different
	if bytes.Equal(prefix1, prefix2) {
		t.Fatal("Collision blocks should be different")
	}

	scriptSig := NewScript([]ScriptCommand{
		{Data: prefix1, IsData: true},
		{Data: prefix2, IsData: true},
	})

	// ScriptPubKey: 6e879169a77ca787
	// OP_2DUP OP_EQUAL OP_NOT OP_VERIFY OP_SHA1 OP_SWAP OP_SHA1 OP_EQUAL
	scriptPubKeyHex := []byte{0x08, 0x6e, 0x87, 0x91, 0x69, 0xa7, 0x7c, 0xa7, 0x87}
	scriptPubKey, err := ParseScript(bytes.NewReader(scriptPubKeyHex))
	if err != nil {
		t.Fatalf("Error parsing scriptPubKey: %v", err)
	}

	combined := scriptSig.Combine(scriptPubKey)
	result := combined.Evaluate([]byte{})

	if !result {
		t.Errorf("SHA-1 collision script with actual collision should pass!")
	}

	t.Log("✓ Successfully unlocked SHA-1 collision script with SHAttered collision data!")
}
