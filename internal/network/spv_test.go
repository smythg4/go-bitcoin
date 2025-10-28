package network

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"go-bitcoin/internal/address"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/encoding"
	"go-bitcoin/internal/script"
	"go-bitcoin/internal/transactions"
	"slices"
	"testing"
	"time"
)

func TestSPVFlowBIP37(t *testing.T) {
	// setup
	lastBlockHex := "0000000013e7e85518dac94d012d73253d3fdac5c30c4143b177f3086f129580" // block 57042 - right before pizza tx
	targetAddress := "17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ"                              // jercos - received 10,000 BTC for pizza in block 57043

	// decode address to get hash160
	h160, err := encoding.DecodeBase58(targetAddress)
	if err != nil {
		t.Fatal(err)
	}

	ip := "34.126.115.35" // known node that supports BIP37
	port := 8333
	t.Logf("Trying %s:%d...", ip, port)
	node, err := NewSimpleNode(ip, port, false, false) // testNet: false
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()
	if err := node.Handshake(); err != nil {
		t.Fatal(err)
	}

	// debugging
	node.OnMessage("inv", func(env NetworkEnvelope) {
		// First byte is count
		if len(env.Payload) > 0 {
			count := env.Payload[0]
			t.Logf("Inv with %d items: %x", count, env.Payload[:min(50, len(env.Payload))])
		}
	})

	// create bloom filter
	bf := NewBloomFilter(30, 5, 90210)
	// add address h160 to filter
	bf.Add(h160)

	// send filterload - nodes are rejecting them now that BIP 37 is discontinued
	filterload := &FilterLoadMessage{
		Filter: &bf,
		Flag:   byte(BLOOM_UPDATE_ALL),
	}
	if err := node.Send(filterload); err != nil {
		t.Fatal(err)
	}

	// request headers starting from last_block
	startBlock, err := hex.DecodeString(lastBlockHex)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(startBlock)
	var startBlockHash [32]byte
	copy(startBlockHash[:], startBlock)

	getheaders := NewGetHeadersMessage(70015, [][32]byte{startBlockHash}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Send(&getheaders); err != nil {
		t.Fatal(err)
	}
	t.Log("  Sent getheaders, waiting for response...")

	// receive headers
	headersEnv, err := node.Receive("headers")
	if err != nil {
		t.Fatal(err)
	}
	headers, err := ParseHeadersMessage(bytes.NewReader(headersEnv.Payload))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Received %d headers\n", len(headers.Blocks))

	// request merkleblocks
	getdata := NewGetDataMessage()
	for _, block := range headers.Blocks {
		if !block.CheckProofOfWork() {
			t.Fatal("invalid POW")
		}

		blockHash, _ := block.Hash()
		var hash [32]byte
		copy(hash[:], blockHash)

		getdata.AddData(DATA_TYPE_FILTERED_BLOCK, hash)
	}
	if err := node.Send(&getdata); err != nil {
		t.Fatal(err)
	}

	// receive merkleblocks and transactions
	found := false
	for !found {
		mbEnv, err := node.Receive("merkleblock")
		if err != nil {
			t.Fatal(err)
		}

		mb, err := ParseMerkleBlock(bytes.NewReader(mbEnv.Payload))
		if err != nil {
			t.Fatal(err)
		}

		// Calculate and log block hash for debugging
		blockHash := encoding.Hash256(mbEnv.Payload[:80])
		slices.Reverse(blockHash)
		t.Logf("Processing block: %x...", blockHash[:4])

		if !mb.IsValid() {
			t.Logf("Invalid merkle proof: NumTx=%d, NumHashes=%d, NumFlags=%d",
				mb.NumTransactions, mb.NumHashes, mb.NumFlags)
			continue
		}

		t.Logf("Valid merkleblock with %d matched transactions\n", mb.NumHashes)

		if mb.NumHashes == 0 {
			t.Log("No matching transactions in this block (false positive)")
			continue
		}
		if mb.NumTransactions == 1 {
			t.Log("Skipping block with only coinbase transaction")
			continue
		}

		// Log the matched transaction hashes
		for i, txHash := range mb.TxHashes {
			t.Logf("  Matched tx hash %d: %x...", i, txHash[:4])
		}

		// receive the matching transactions
		// one message per matched transaction
		// NOTE: nodes don't send coinbase transactions
		// If the only match is coinbase, we won't receive any tx messages
		// So we need to handle timeouts gracefully

		receivedAnyTx := false
		for i := uint64(0); i < mb.NumHashes; i++ {
			txEnv, err := node.Receive("tx")
			if err != nil {
				// Timeout likely means matched tx was coinbase
				t.Logf("Did not receive tx %d (likely coinbase): %v", i, err)
				break
			}
			receivedAnyTx = true

			// parse transaction
			tx, err := transactions.ParseTransaction(bytes.NewReader(txEnv.Payload))
			if err != nil {
				t.Fatal(err)
			}

			txID, _ := tx.Id()
			t.Logf("Received non-coinbase tx: %x... (checking %d outputs)", txID[:4], len(tx.Outputs))

			// check each output to see if it pays to our address
			for j, txOut := range tx.Outputs {
				// get address from ScriptPubKey
				addrObj, err := txOut.ScriptPubKey.AddressV2(address.MAINNET)
				if err != nil {
					// skip unparseable
					t.Logf("  Output %d: unparseable address (%v)", j, err)
					continue
				}

				//t.Logf("  Output %d: %s (amount: %d sat)", j, addrObj.String, txOut.Amount)
				if addrObj.String == targetAddress {
					txID, _ := tx.Id()
					t.Logf("SUCCESS! Found transaction paying to %s", targetAddress)
					t.Logf("  Transaction: %x", txID)
					t.Logf("  Output index: %d", j)
					t.Logf("  Amount: %d satoshis", txOut.Amount)
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !receivedAnyTx {
			t.Log("No tx messages received (matched coinbase only), continuing...")
			continue
		}
	}
	if !found {
		t.Error("Did not find expected transaction")
	}

}

func TestSPVFlowBIP158(t *testing.T) {
	// setup
	lastBlockHex := "0000000013e7e85518dac94d012d73253d3fdac5c30c4143b177f3086f129580" // block 57042 - right before pizza tx
	targetAddress := "17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ"                              // jercos - received 10,000 BTC for pizza in block 57043

	// decode address to get hash160
	h160, err := encoding.DecodeBase58(targetAddress)
	if err != nil {
		t.Fatal(err)
	}
	targetScript := script.P2pkhScript(h160)          // Creates the P2PKH scriptPubKey
	targetScriptBytes, err := targetScript.RawBytes() // Get raw bytes (no varint prefix)
	if err != nil {
		t.Errorf("failed to get script raw bytes: %v", err)
	}

	ip := "77.174.133.117"
	port := 8333
	t.Logf("Trying %s:%d...", ip, port)
	node, err := NewSimpleNode(ip, port, false, false) // testNet: false
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()
	if err := node.Handshake(); err != nil {
		t.Fatal(err)
	}

	const NODE_COMPACT_FILTERS = uint64(1 << 6) // Bit 6

	// Check if peer supports compact filters
	if node.PeerServices&NODE_COMPACT_FILTERS == 0 {
		t.Skipf("Peer does not support BIP 157 compact filters. Services: %d (binary: %064b)",
			node.PeerServices, node.PeerServices)
	}
	t.Logf("✓ Peer supports compact filters!")

	// debugging
	node.OnMessage("inv", func(env NetworkEnvelope) {
		// First byte is count
		if len(env.Payload) > 0 {
			count := env.Payload[0]
			t.Logf("Inv with %d items: %x", count, env.Payload[:min(50, len(env.Payload))])
		}
	})

	// request headers starting from last_block
	startBlock, err := hex.DecodeString(lastBlockHex)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(startBlock)
	var startBlockHash [32]byte
	copy(startBlockHash[:], startBlock)

	getheaders := NewGetHeadersMessage(70015, [][32]byte{startBlockHash}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Send(&getheaders); err != nil {
		t.Fatal(err)
	}
	t.Log("  Sent getheaders, waiting for response...")

	// receive compact filter
	headersEnv, err := node.Receive("headers")
	if err != nil {
		t.Fatal(err)
	}
	headers, err := ParseHeadersMessage(bytes.NewReader(headersEnv.Payload))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Received %d headers\n", len(headers.Blocks))

	startHeight := uint32(57042) // Known starting height
	found := false
	for i, thisBlock := range headers.Blocks {
		currentHeight := startHeight + uint32(i) + 1
		blockHash, err := thisBlock.Hash()
		if err != nil {
			t.Errorf("failed to calculate hash on block %d: %v", i, err)
		}
		var hash [32]byte
		copy(hash[:], blockHash)

		// request filter for this block
		getCFilter := &GetCFilterMessage{
			FType:       BASIC,
			StartHeight: currentHeight,
			StopHash:    hash,
		}
		if err := node.Send(getCFilter); err != nil {
			t.Errorf("failed to send getCFilter for block %d: %v", i, err)
			continue
		}

		// receive filter response
		cfilterEnv, err := node.ReceiveWithTimeout("cfilter", 30*time.Second)
		if err != nil {
			t.Errorf("failed to receive filter message on block %d: %v", i, err)
			continue
		}
		cfilter, err := ParseCFilterMessage(bytes.NewReader(cfilterEnv.Payload))
		if err != nil {
			t.Errorf("failed to parse filter message on block %d: %v", i, err)
			continue
		}

		// deserialize GCS filter
		gcs, err := ParseGCSFilter(bytes.NewReader(cfilter.FilterBytes))
		if err != nil {
			t.Errorf("failed to parse GCS filter: %v", err)
			continue
		}

		k0 := binary.LittleEndian.Uint64(blockHash[0:8])
		k1 := binary.LittleEndian.Uint64(blockHash[8:16])

		// Check if our target script matches the filter
		match, err := gcs.Match(targetScriptBytes, k0, k1)
		if err != nil {
			t.Logf("Match error: %v", err)
			continue
		}

		if !match {
			// Make a display-order copy for logging
			t.Logf("Block %x...: no match", blockHash[:4])
			continue
		}

		t.Logf("Block %x: FILTER MATCH! Requesting full block...", blockHash[:4])

		// Filter matched - request full block
		getdata := NewGetDataMessage()
		getdata.AddData(DATA_TYPE_BLOCK, hash) // Request full block (not merkleblock)
		if err := node.Send(&getdata); err != nil {
			t.Errorf("failed to send getdatamessage: %v", err)
			continue
		}
		// Receive full block
		blockEnv, _ := node.Receive("block")
		fullBlock, err := block.ParseFullBlock(bytes.NewReader(blockEnv.Payload))
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Received full block with %d transactions", len(fullBlock.Txs))

		// Search for our target address in the outputs
		for _, tx := range fullBlock.Txs {
			for j, txOut := range tx.Outputs {
				addrObj, err := txOut.ScriptPubKey.AddressV2(address.MAINNET)
				if err != nil {
					continue // Skip unparseable
				}

				if addrObj.String == targetAddress {
					txID, _ := tx.Id()
					t.Logf("SUCCESS! Found transaction paying to %s", targetAddress)
					t.Logf("  Transaction: %x", txID)
					t.Logf("  Output index: %d", j)
					t.Logf("  Amount: %d satoshis", txOut.Amount)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Error("Did not find expected transaction")
	}
}
