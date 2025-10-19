package transactions

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TxFetcher struct {
	Cache map[string]*Transaction
}

func NewTxFetcher() TxFetcher {
	return TxFetcher{
		Cache: make(map[string]*Transaction, 1),
	}
}

func (tf *TxFetcher) GetUrl(testNet bool) string {
	baseURL := "https://blockstream.info/api"
	if testNet {
		baseURL = "https://blockstream.info/testnet/api"
	}
	return baseURL
}

func (tf *TxFetcher) Fetch(txId string, testNet, fresh bool) (*Transaction, error) {
	if !fresh {
		if tx, exists := tf.Cache[txId]; exists {
			return tx, nil
		}
	}

	url := fmt.Sprintf("%s/tx/%s/hex", tf.GetUrl(testNet), txId)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	hexData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// decode hex string to raw bytes
	rawBytes, err := hex.DecodeString(string(hexData))
	if err != nil {
		return nil, err
	}

	var tx Transaction

	if rawBytes[4] == 0 {
		// special SegWit handling
		stripped := make([]byte, 0, len(rawBytes)-2)
		stripped = append(stripped, rawBytes[:4]...)
		stripped = append(stripped, rawBytes[6:]...)
		r := bytes.NewBuffer(stripped)
		tx, err = ParseTransaction(r)
		if err != nil {
			return nil, err
		}
		tx.Locktime = binary.LittleEndian.Uint32(rawBytes[len(rawBytes)-4:])
	} else {
		r := bytes.NewBuffer(rawBytes)
		tx, err = ParseTransaction(r)
		if err != nil {
			return nil, err
		}
	}

	// verify txids match
	fetchId, err := tx.id()
	if err != nil {
		return nil, err
	}
	if fetchId != txId {
		return nil, fmt.Errorf("Transaction IDs don't match. Got: %s, expected: %s", fetchId, txId)
	}

	// cache the transaction for future use
	tx.IsTestnet = testNet
	tf.Cache[txId] = &tx

	return &tx, nil
}

// isLegacyTransaction checks if a transaction uses legacy P2PKH (not SegWit)
// Uses a fast heuristic: SegWit transactions have empty/short ScriptSigs
func (tf *TxFetcher) isLegacyTransaction(txId string, testNet bool) (bool, error) {
	tx, err := tf.Fetch(txId, testNet, false)
	if err != nil {
		return false, err
	}

	// Fast heuristic: SegWit transactions have empty or very short ScriptSigs
	// Legacy P2PKH ScriptSigs are typically 100+ bytes (sig + pubkey)
	for _, input := range tx.Inputs {
		scriptSigLen := len(input.ScriptSig.CommandStack)

		// SegWit typically has 0 commands in ScriptSig (witness is separate)
		// Legacy P2PKH has 2 commands: <sig> <pubkey>
		if scriptSigLen < 2 {
			return false, nil
		}

		// Double check: if we have commands, first should be data (signature)
		if !input.ScriptSig.CommandStack[0].IsData {
			return false, nil
		}
	}

	return true, nil
}

// FetchRecentLegacyTxIds fetches up to maxCount recent legacy transaction IDs
// with a timeout. Checks multiple recent blocks and skips SegWit transactions.
func (tf *TxFetcher) FetchRecentLegacyTxIds(testNet bool, maxCount int, maxCheckPerBlock int, maxBlocks int, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	legacyTxIds := []string{}

	// Get the latest block hash
	url := fmt.Sprintf("%s/blocks/tip/hash", tf.GetUrl(testNet))
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest block hash: %w", err)
	}
	defer resp.Body.Close()

	blockHash, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read block hash: %w", err)
	}

	currentBlockHash := string(blockHash)

	// Check multiple recent blocks
	for blockNum := 0; blockNum < maxBlocks && len(legacyTxIds) < maxCount; blockNum++ {
		// Check timeout
		select {
		case <-ctx.Done():
			fmt.Printf("Timeout reached\n")
			return legacyTxIds, nil
		default:
		}

		fmt.Printf("\nChecking block %d: %s\n", blockNum+1, currentBlockHash)

		// Get transaction IDs from this block
		url = fmt.Sprintf("%s/block/%s/txids", tf.GetUrl(testNet), currentBlockHash)
		resp, err = http.Get(url)
		if err != nil {
			fmt.Printf("Error fetching block txids: %v\n", err)
			break
		}

		var txids []string
		if err := json.NewDecoder(resp.Body).Decode(&txids); err != nil {
			resp.Body.Close()
			fmt.Printf("Error decoding txids: %v\n", err)
			break
		}
		resp.Body.Close()

		fmt.Printf("Found %d transactions in block\n", len(txids))

		// Skip coinbase (index 0) and check up to maxCheckPerBlock transactions
		maxToCheck := maxCheckPerBlock
		if maxToCheck > len(txids)-1 {
			maxToCheck = len(txids) - 1
		}

		for i := 1; i <= maxToCheck && len(legacyTxIds) < maxCount; i++ {
			// Check timeout
			select {
			case <-ctx.Done():
				fmt.Printf("Timeout reached\n")
				return legacyTxIds, nil
			default:
			}

			txId := txids[i]
			fmt.Printf("  Checking tx %d/%d... ", i, maxToCheck)

			isLegacy, err := tf.isLegacyTransaction(txId, testNet)
			if err != nil {
				fmt.Printf("error (skipping)\n")
				continue
			}

			if isLegacy {
				fmt.Printf("✓ legacy\n")
				legacyTxIds = append(legacyTxIds, txId)
			} else {
				fmt.Printf("SegWit (skipping)\n")
			}
		}

		// Get previous block hash for next iteration
		url = fmt.Sprintf("%s/block/%s", tf.GetUrl(testNet), currentBlockHash)
		resp, err = http.Get(url)
		if err != nil {
			fmt.Printf("Error fetching block info: %v\n", err)
			break
		}

		var blockInfo struct {
			PreviousBlockHash string `json:"previousblockhash"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&blockInfo); err != nil {
			resp.Body.Close()
			fmt.Printf("Error decoding block info: %v\n", err)
			break
		}
		resp.Body.Close()

		if blockInfo.PreviousBlockHash == "" {
			// Reached genesis block
			break
		}

		currentBlockHash = blockInfo.PreviousBlockHash
	}

	return legacyTxIds, nil
}

// FetchAddressTransactions fetches all transaction IDs for a given address
func (tf *TxFetcher) FetchAddressTransactions(address string, testNet bool) ([]string, error) {
	url := fmt.Sprintf("%s/address/%s/txs", tf.GetUrl(testNet), address)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions for address: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// The API returns an array of transaction objects
	var txs []struct {
		TxID string `json:"txid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	txIds := make([]string, len(txs))
	for i, tx := range txs {
		txIds[i] = tx.TxID
	}

	return txIds, nil
}
