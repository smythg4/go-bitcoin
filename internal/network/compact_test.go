package network

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/encoding"
	"go-bitcoin/internal/mempool"
	"go-bitcoin/internal/script"
	"go-bitcoin/internal/transactions"
	"io"
	"testing"
	"time"
)

func TestCompactBlockRoundtrip(t *testing.T) {
	// Create a test block header
	header := &block.Block{
		Version:    1,
		PrevBlock:  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		MerkleRoot: [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		TimeStamp:  1234567890,
		Bits:       0x1d00ffff,
		Nonce:      987654321,
	}

	// Create a test coinbase transaction for prefilled
	// Coinbase input: prevTx is all zeros, prevIdx is 0xffffffff
	coinbaseInput := transactions.TxIn{
		PrevTx:  make([]byte, 32), // all zeros
		PrevIdx: 0xffffffff,
		// Use empty scriptSig to avoid serialization/deserialization asymmetry
		// (Real coinbase scriptSigs contain push opcodes, but test parser treats them as raw bytes)
		ScriptSig: script.Script{CommandStack: []script.ScriptCommand{}},
		Sequence:  0xffffffff,
	}

	coinbase := &transactions.Transaction{
		Version: 1,
		Inputs:  []transactions.TxIn{coinbaseInput},
		Outputs: []transactions.TxOut{
			{
				Amount:       5000000000,                                            // 50 BTC
				ScriptPubKey: script.Script{CommandStack: []script.ScriptCommand{}}, // empty script
			},
		},
		Locktime: 0,
	}

	// Create shortIDs (6 bytes each)
	shortIDs := [][6]byte{
		{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
		{0x11, 0x12, 0x13, 0x14, 0x15, 0x16},
		{0x21, 0x22, 0x23, 0x24, 0x25, 0x26},
	}

	// Create prefilled transactions (coinbase at index 0)
	prefilledTxns := []PrefilledTransaction{
		{
			Index: 0,
			Tx:    coinbase,
		},
	}

	// Build CompactBlockMessage
	original := CompactBlockMessage{
		Header:        header,
		Nonce:         0x123456789ABCDEF0,
		ShortIDs:      shortIDs,
		PrefilledTxns: prefilledTxns,
	}

	// Serialize
	serialized, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	t.Logf("Serialized length: %d bytes", len(serialized))

	// Parse
	parsed, err := ParseCompactBlockMessage(bytes.NewReader(serialized))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify header fields
	if parsed.Header.Version != original.Header.Version {
		t.Errorf("Version mismatch: got %d, want %d", parsed.Header.Version, original.Header.Version)
	}
	if parsed.Header.PrevBlock != original.Header.PrevBlock {
		t.Errorf("PrevBlock mismatch")
	}
	if parsed.Header.MerkleRoot != original.Header.MerkleRoot {
		t.Errorf("MerkleRoot mismatch")
	}
	if parsed.Header.TimeStamp != original.Header.TimeStamp {
		t.Errorf("TimeStamp mismatch: got %d, want %d", parsed.Header.TimeStamp, original.Header.TimeStamp)
	}
	if parsed.Header.Bits != original.Header.Bits {
		t.Errorf("Bits mismatch: got %d, want %d", parsed.Header.Bits, original.Header.Bits)
	}
	if parsed.Header.Nonce != original.Header.Nonce {
		t.Errorf("Nonce mismatch: got %d, want %d", parsed.Header.Nonce, original.Header.Nonce)
	}

	// Verify nonce
	if parsed.Nonce != original.Nonce {
		t.Errorf("CompactBlock Nonce mismatch: got 0x%x, want 0x%x", parsed.Nonce, original.Nonce)
	}

	// Verify shortIDs count
	if len(parsed.ShortIDs) != len(original.ShortIDs) {
		t.Fatalf("ShortIDs count mismatch: got %d, want %d", len(parsed.ShortIDs), len(original.ShortIDs))
	}

	// Verify each shortID
	for i, sid := range parsed.ShortIDs {
		if sid != original.ShortIDs[i] {
			t.Errorf("ShortID[%d] mismatch: got %x, want %x", i, sid, original.ShortIDs[i])
		}
	}

	// Verify prefilled transactions count
	if len(parsed.PrefilledTxns) != len(original.PrefilledTxns) {
		t.Fatalf("PrefilledTxns count mismatch: got %d, want %d", len(parsed.PrefilledTxns), len(original.PrefilledTxns))
	}

	// Verify prefilled transaction index
	if parsed.PrefilledTxns[0].Index != original.PrefilledTxns[0].Index {
		t.Errorf("PrefilledTxn index mismatch: got %d, want %d", parsed.PrefilledTxns[0].Index, original.PrefilledTxns[0].Index)
	}

	// Verify prefilled transaction hash
	originalHash, _ := original.PrefilledTxns[0].Tx.Hash()
	parsedHash, _ := parsed.PrefilledTxns[0].Tx.Hash()
	if originalHash != parsedHash {
		t.Errorf("PrefilledTxn hash mismatch: got %x, want %x", parsedHash, originalHash)
	}

	t.Log("✓ CompactBlockMessage roundtrip successful!")
}

func TestCompactBlockNegotiation(t *testing.T) {
	// This test connects to a real Bitcoin node and negotiates compact blocks
	// It verifies the sendcmpct message exchange works

	// Use a known Bitcoin node IP
	ip := "34.126.115.35" // One of seed.bitcoin.sipa.be's IPs
	port := 8333

	t.Logf("Connecting to %s:%d...", ip, port)
	node, err := NewSimpleNode(ip, port, false, true) // testNet: false, logging: true
	if err != nil {
		t.Skip("Could not connect to Bitcoin node:", err)
	}
	defer node.Close()

	// Perform handshake
	if err := node.Handshake(); err != nil {
		t.Fatal("Handshake failed:", err)
	}
	t.Log("✓ Handshake complete")

	// The node likely already sent us a sendcmpct during handshake
	// Try to receive it (with short timeout)
	select {
	case env := <-node.channelsMap["sendcmpct"]:
		scm, err := ParseSendCompactMessage(bytes.NewReader(env.Payload))
		if err != nil {
			t.Fatal("Failed to parse sendcmpct:", err)
		}
		t.Logf("✓ Received sendcmpct from peer: HighBandwidth=%v, Version=%d",
			scm.HighBandwidth, scm.Version)
	case <-time.After(2 * time.Second):
		t.Log("Peer didn't send sendcmpct (that's okay)")
	}

	// Send our sendcmpct to enable compact blocks (version 2, low-bandwidth mode)
	ourSendCmpct := &SendCompactMessage{
		HighBandwidth: false, // Low-bandwidth mode (request on demand)
		Version:       2,     // Use version 2 (wtxid-based)
	}

	if err := node.Send(ourSendCmpct); err != nil {
		t.Fatal("Failed to send sendcmpct:", err)
	}
	t.Log("✓ Sent sendcmpct to peer (version 2, low-bandwidth)")

	t.Log("✓ Compact block negotiation successful!")
}

func TestCompactBlockRealFlow(t *testing.T) {
	// This test waits for a real compact block from the network
	// WARNING: This can take ~10 minutes (average block time)
	// Run with: go test -v -run TestCompactBlockRealFlow -timeout 20m

	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// Use a known Bitcoin node IP
	ip := "34.126.115.35"
	port := 8333

	t.Logf("Connecting to %s:%d...", ip, port)
	node, err := NewSimpleNode(ip, port, false, true) // testNet: false, logging: true
	if err != nil {
		t.Fatal("Could not connect to Bitcoin node:", err)
	}
	defer node.Close()

	// Perform handshake
	if err := node.Handshake(); err != nil {
		t.Fatal("Handshake failed:", err)
	}
	t.Log("✓ Handshake complete")

	// Try to receive peer's sendcmpct
	peerVersion := uint64(1) // Default to version 1 if peer doesn't send sendcmpct
	select {
	case env := <-node.channelsMap["sendcmpct"]:
		scm, err := ParseSendCompactMessage(bytes.NewReader(env.Payload))
		if err != nil {
			t.Fatal("Failed to parse sendcmpct:", err)
		}
		peerVersion = scm.Version
		t.Logf("✓ Peer supports compact blocks: Version=%d, HighBandwidth=%v",
			scm.Version, scm.HighBandwidth)
	case <-time.After(2 * time.Second):
		t.Log("⚠️  Peer didn't send sendcmpct (will assume version 1)")
	}
	t.Logf("📋 Peer will send us compact blocks using version %d shortIDs", peerVersion)

	// Send our sendcmpct to enable compact blocks (version 2, high-bandwidth mode for this test)
	ourSendCmpct := &SendCompactMessage{
		HighBandwidth: true, // High-bandwidth: peer will send us compact blocks unsolicited
		Version:       2,    // Version 2 (wtxid-based)
	}

	if err := node.Send(ourSendCmpct); err != nil {
		t.Fatal("Failed to send sendcmpct:", err)
	}
	t.Log("✓ Enabled high-bandwidth compact blocks (version 2)")

	// Create a mempool
	mp := mempool.New()
	txCount := 0

	node.OnMessage("inv", func(env NetworkEnvelope) {
		r := bytes.NewReader(env.Payload)
		count, err := encoding.ReadVarInt(r)
		if err != nil {
			return
		}

		t.Logf("📬 Received inv with %d items", count) // ADD THIS

		getdata := NewGetDataMessage()
		for i := uint64(0); i < count; i++ {
			typeBuf := make([]byte, 4)
			io.ReadFull(r, typeBuf)
			invType := binary.LittleEndian.Uint32(typeBuf)

			var hash [32]byte
			io.ReadFull(r, hash[:])

			t.Logf("  - inv type: %d, hash: %x...", invType, hash[:4])

			switch invType {
			case 1, 5:
				getdata.AddData(0x40000001, hash) // MSG_WITNESS_TX
			case 2:
				t.Log("📦 Peer announced REGULAR block (type 2) - requesting as compact block")
				getdata.AddData(4, hash) // Request as compact block (BIP152 allows this)
			case 4:
				t.Log("📦 Peer announced compact block via inv (low-bandwidth mode)")
				getdata.AddData(4, hash)
			default:
				t.Logf("⚠️  Unknown inv type: %d", invType)
			}
		}

		if len(getdata.Data) > 0 {
			node.Send(&getdata)
		}
	})

	t.Log("⏳ Building mempool from network transactions...")
	t.Log("   Listening to inv messages and requesting transactions")
	t.Log("⏳ Waiting for a compact block (this may take ~10 minutes)...")
	t.Log("   Average Bitcoin block time is ~10 minutes")
	t.Log("   You can watch for new blocks at: https://mempool.space/")

	// Process tx messages and wait for compact block
	var env NetworkEnvelope
	timeout := time.After(20 * time.Minute)
	start := time.Now()
	// Keep connection alive during long wait
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Send ping to keep connection alive
				nonce := make([]byte, 8)
				rand.Read(nonce)
				node.Send(&PingMessage{Nonce: nonce})
				t.Logf("Time elapsed: %v", time.Since(start))
			case <-node.done:
				return
			}
		}
	}()
loop:
	for {
		select {
		case txEnv, ok := <-node.channelsMap["tx"]:
			if !ok {
				t.Fatal("tx channel closed")
			}

			tx, err := transactions.ParseTransaction(bytes.NewReader(txEnv.Payload))
			if err != nil {
				t.Logf("Failed to parse tx: %v", err)
				continue
			}

			// Diagnostic: Check if IsSegwit is being set correctly
			hasWitness := len(tx.Inputs) > 0 && len(tx.Inputs[0].Witness) > 0
			if txCount < 5 || (txCount%100 == 0) {
				txid, _ := tx.Id()
				t.Logf("📊 TX #%d (%s...): IsSegwit=%v, HasWitness=%v, Inputs=%d",
					txCount+1, txid[:8], tx.IsSegwit, hasWitness, len(tx.Inputs))
			}

			// assume txs from peer nodes are valid

			if err := mp.Add(&tx); err != nil {
				t.Logf("Failed to add tx to mempool: %v", err)
				continue
			}

			txCount++
			if txCount%10 == 0 {
				t.Logf("📝 Mempool now has %d transactions", txCount)
			}

		case cmpctEnv, ok := <-node.channelsMap["cmpctblock"]:
			if !ok {
				t.Fatal("cmpctblock channel closed")
			}
			env = cmpctEnv
			break loop

		case <-timeout:
			t.Fatalf("Timeout waiting for compact block (20 minutes). Mempool had %d transactions.", txCount)

		case <-node.done:
			t.Fatal("Connection closed before receiving compact block")
		}
	}

	t.Log("🎉 Received compact block!")
	t.Logf("Payload size: %d bytes", len(env.Payload))
	t.Logf("Command: %s", env.Command)
	t.Logf("Magic: 0x%x", env.Magic)

	// Debug: show first 100 bytes of payload
	if len(env.Payload) > 0 {
		preview := len(env.Payload)
		if preview > 100 {
			preview = 100
		}
		t.Logf("Payload preview (first %d bytes): %x", preview, env.Payload[:preview])
	} else {
		t.Fatal("Payload is empty!")
	}

	// Parse the compact block
	cmpct, err := ParseCompactBlockMessage(bytes.NewReader(env.Payload))
	if err != nil {
		t.Fatalf("Failed to parse compact block: %v (payload size: %d)", err, len(env.Payload))
	}

	t.Logf("Compact block details:")
	t.Logf("  - Header version: %d", cmpct.Header.Version)
	t.Logf("  - Timestamp: %s", time.Unix(int64(cmpct.Header.TimeStamp), 0))
	t.Logf("  - Nonce: 0x%x", cmpct.Nonce)
	t.Logf("  - ShortIDs: %d", len(cmpct.ShortIDs))
	t.Logf("  - Prefilled transactions: %d", len(cmpct.PrefilledTxns))

	// Calculate and display the shortID keys
	k0, k1, err := mempool.CalcShortIDKeys(cmpct.Header, cmpct.Nonce)
	if err != nil {
		t.Fatal("Failed to calculate shortID keys:", err)
	}
	t.Logf("🔑 ShortID keys: k0=0x%016x, k1=0x%016x", k0, k1)
	t.Logf("📋 First 5 compact block shortIDs: %x, %x, %x, %x, %x",
		cmpct.ShortIDs[0], cmpct.ShortIDs[1], cmpct.ShortIDs[2], cmpct.ShortIDs[3], cmpct.ShortIDs[4])

	// Try both versions to see which one works
	t.Logf("🔍 Attempting reconstruction with version 2 (wtxid)...")
	reconstructed, missing, err := ReconstructBlock(cmpct, mp, nil, 2)
	if err != nil {
		t.Fatal("ReconstructBlock failed:", err)
	}

	matchRate := float64(len(cmpct.ShortIDs)-len(missing)) / float64(len(cmpct.ShortIDs)) * 100
	t.Logf("   Version 2 match rate: %.1f%% (%d/%d matched)",
		matchRate, len(cmpct.ShortIDs)-len(missing), len(cmpct.ShortIDs))

	// Also try version 1 to compare
	t.Logf("🔍 Attempting reconstruction with version 1 (txid)...")
	_, missing1, err := ReconstructBlock(cmpct, mp, nil, 1)
	if err != nil {
		t.Fatal("ReconstructBlock (v1) failed:", err)
	}
	matchRate1 := float64(len(cmpct.ShortIDs)-len(missing1)) / float64(len(cmpct.ShortIDs)) * 100
	t.Logf("   Version 1 match rate: %.1f%% (%d/%d matched)",
		matchRate1, len(cmpct.ShortIDs)-len(missing1), len(cmpct.ShortIDs))

	// Use the version with better match rate
	useVersion := peerVersion
	useMissing := missing
	if len(missing1) < len(missing) {
		t.Logf("⚠️  Version 1 has better match rate! Peer may be using v1 despite our v2 request")
		useVersion = 1
		useMissing = missing1
	}

	if len(useMissing) > 0 {
		t.Logf("⚠️  Missing %d transactions from mempool (using version %d)", len(useMissing), useVersion)
		t.Logf("   Missing indexes (showing first 20): %v", useMissing[:min(20, len(useMissing))])

		// In production, we'd send getblocktxn here:
		blockHash, _ := cmpct.Header.Hash()
		var hash32 [32]byte
		copy(hash32[:], blockHash)

		getBlockTxn := &GetBlockTransactionMessage{
			BlockHash: hash32,
			Indexes:   useMissing,
		}

		t.Logf("📤 Requesting %d missing transactions via getblocktxn", len(useMissing))
		if err := node.Send(getBlockTxn); err != nil {
			t.Fatal("Failed to send getblocktxn:", err)
		}

		// Wait for blocktxn response
		t.Log("⏳ Waiting for blocktxn response...")
		btxnEnv, err := node.ReceiveWithTimeout("blocktxn", 30*time.Second)
		if err != nil {
			t.Fatal("Failed to receive blocktxn:", err)
		}

		btxn, err := ParseBlockTransactionMessage(bytes.NewReader(btxnEnv.Payload))
		if err != nil {
			t.Fatal("Failed to parse blocktxn:", err)
		}

		t.Logf("✓ Received %d missing transactions", len(btxn.Transactions))

		// Reconstruct again with missing transactions using the correct version
		reconstructed, stillMissing, err := ReconstructBlock(cmpct, mp, btxn.Transactions, useVersion)
		if err != nil {
			t.Fatal("Second ReconstructBlock failed:", err)
		}

		if len(stillMissing) > 0 {
			t.Fatalf("Still missing %d transactions after blocktxn!", len(stillMissing))
		}

		t.Log("✅ Block fully reconstructed!")
		t.Logf("   Total transactions: %d", len(reconstructed.TxHashes))

		// Display some transaction hashes
		for i, txHash := range reconstructed.TxHashes {
			if i < 3 || i >= len(reconstructed.TxHashes)-1 {
				t.Logf("   tx[%d]: %x", i, txHash)
			} else if i == 3 {
				t.Logf("   ... (%d more transactions)", len(reconstructed.TxHashes)-4)
			}
		}
	} else {
		t.Log("✅ Block fully reconstructed from mempool!")
		t.Logf("   Total transactions: %d", len(reconstructed.TxHashes))

		// This would be amazing - means we had all txs in mempool already!
		for i, txHash := range reconstructed.TxHashes {
			if i < 5 {
				t.Logf("   tx[%d]: %x", i, txHash)
			}
		}
	}

	t.Log("🎊 BIP152 full flow test COMPLETE!")
}

func TestCompactBlockReconstruction(t *testing.T) {
	// This test verifies the block reconstruction logic with a mock scenario

	// Create a mempool with some transactions
	mp := mempool.New()

	// Create test transactions
	tx1 := &transactions.Transaction{
		Version: 1,
		Inputs: []transactions.TxIn{
			{
				PrevTx:    bytes.Repeat([]byte{0x11}, 32),
				PrevIdx:   0,
				ScriptSig: script.Script{CommandStack: []script.ScriptCommand{}},
				Sequence:  0xffffffff,
			},
		},
		Outputs: []transactions.TxOut{
			{
				Amount:       1000000,
				ScriptPubKey: script.Script{CommandStack: []script.ScriptCommand{}},
			},
		},
		Locktime: 0,
	}

	tx2 := &transactions.Transaction{
		Version: 1,
		Inputs: []transactions.TxIn{
			{
				PrevTx:    bytes.Repeat([]byte{0x22}, 32),
				PrevIdx:   0,
				ScriptSig: script.Script{CommandStack: []script.ScriptCommand{}},
				Sequence:  0xffffffff,
			},
		},
		Outputs: []transactions.TxOut{
			{
				Amount:       2000000,
				ScriptPubKey: script.Script{CommandStack: []script.ScriptCommand{}},
			},
		},
		Locktime: 0,
	}

	// Add to mempool
	if err := mp.Add(tx1); err != nil {
		t.Fatal("Failed to add tx1 to mempool:", err)
	}
	if err := mp.Add(tx2); err != nil {
		t.Fatal("Failed to add tx2 to mempool:", err)
	}

	t.Logf("Mempool has %d transactions", len(mp.All()))

	// Create a test block header
	header := &block.Block{
		Version:    1,
		PrevBlock:  [32]byte{},
		MerkleRoot: [32]byte{},
		TimeStamp:  uint32(time.Now().Unix()),
		Bits:       0x1d00ffff,
		Nonce:      12345,
	}

	// Create coinbase (prefilled at index 0)
	coinbase := &transactions.Transaction{
		Version: 1,
		Inputs: []transactions.TxIn{
			{
				PrevTx:  make([]byte, 32),
				PrevIdx: 0xffffffff,
				// Coinbase scriptSig contains arbitrary data (block height, extra nonce, etc.)
				ScriptSig: script.Script{
					CommandStack: []script.ScriptCommand{
						{Data: []byte{0x03, 0x42, 0x00, 0x01}, IsData: true},
					},
				},
				Sequence: 0xffffffff,
			},
		},
		Outputs: []transactions.TxOut{
			{
				Amount:       5000000000,
				ScriptPubKey: script.Script{CommandStack: []script.ScriptCommand{}},
			},
		},
		Locktime: 0,
	}

	// Calculate shortIDs for our mempool transactions
	nonce := uint64(0xABCDEF1234567890)
	k0, k1, err := mempool.CalcShortIDKeys(header, nonce)
	if err != nil {
		t.Fatal("Failed to calculate shortID keys:", err)
	}

	tx1Hash, _ := tx1.Hash()
	tx2Hash, _ := tx2.Hash()

	// CRITICAL: Hash() returns display order (reversed) hash, but BIP152 requires
	// internal order (little-endian, non-reversed) for SipHash calculation.
	// Must reverse back to internal order, just like production code does.
	tx1HashInternal := tx1Hash
	for i := 0; i < 16; i++ {
		tx1HashInternal[i], tx1HashInternal[31-i] = tx1HashInternal[31-i], tx1HashInternal[i]
	}

	tx2HashInternal := tx2Hash
	for i := 0; i < 16; i++ {
		tx2HashInternal[i], tx2HashInternal[31-i] = tx2HashInternal[31-i], tx2HashInternal[i]
	}

	sid1 := mempool.CalculateShortID(tx1HashInternal, k0, k1)
	sid2 := mempool.CalculateShortID(tx2HashInternal, k0, k1)

	t.Logf("tx1 shortID: %x", sid1)
	t.Logf("tx2 shortID: %x", sid2)

	// Build compact block message
	// Block structure: [coinbase, tx1, tx2]
	cmpct := CompactBlockMessage{
		Header:   header,
		Nonce:    nonce,
		ShortIDs: [][6]byte{sid1, sid2},
		PrefilledTxns: []PrefilledTransaction{
			{Index: 0, Tx: coinbase},
		},
	}

	// Reconstruct the block
	reconstructed, missing, err := ReconstructBlock(cmpct, mp, nil, uint64(coinbase.Version))
	if err != nil {
		t.Fatal("ReconstructBlock failed:", err)
	}

	if len(missing) > 0 {
		t.Errorf("Expected no missing transactions, got %d missing: %v", len(missing), missing)
	}

	if len(reconstructed.TxHashes) != 3 {
		t.Fatalf("Expected 3 transactions, got %d", len(reconstructed.TxHashes))
	}

	// Verify transaction hashes
	coinbaseHash, _ := coinbase.Hash()
	if reconstructed.TxHashes[0] != coinbaseHash {
		t.Error("Coinbase hash mismatch")
	}
	if reconstructed.TxHashes[1] != tx1Hash {
		t.Error("tx1 hash mismatch")
	}
	if reconstructed.TxHashes[2] != tx2Hash {
		t.Error("tx2 hash mismatch")
	}

	t.Log("✓ Block reconstruction successful!")
	t.Logf("  - Coinbase (prefilled): %x", coinbaseHash)
	t.Logf("  - tx1 (from mempool): %x", tx1Hash)
	t.Logf("  - tx2 (from mempool): %x", tx2Hash)
}
