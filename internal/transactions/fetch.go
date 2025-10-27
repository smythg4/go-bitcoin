package transactions

import (
	"bytes"
	"context"
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

	r := bytes.NewBuffer(rawBytes)
	tx, err := ParseTransaction(r)
	if err != nil {
		return nil, err
	}

	// verify txids match
	fetchId, err := tx.Id()
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

// FetchRecentTxIds fetches up to maxCount recent transaction IDs from the blockchain
// with a timeout. Checks multiple recent blocks (excluding coinbase transactions).
func (tf *TxFetcher) FetchRecentTxIds(testNet bool, maxCount int, maxCheckPerBlock int, maxBlocks int, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	txIds := []string{}

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
	for blockNum := 0; blockNum < maxBlocks && len(txIds) < maxCount; blockNum++ {
		// Check timeout
		select {
		case <-ctx.Done():
			return txIds, nil
		default:
		}

		// Get transaction IDs from this block
		url = fmt.Sprintf("%s/block/%s/txids", tf.GetUrl(testNet), currentBlockHash)
		resp, err = http.Get(url)
		if err != nil {
			break
		}

		var blockTxIds []string
		if err := json.NewDecoder(resp.Body).Decode(&blockTxIds); err != nil {
			resp.Body.Close()
			break
		}
		resp.Body.Close()

		// Skip coinbase (index 0) and check up to maxCheckPerBlock transactions
		maxToCheck := maxCheckPerBlock
		if maxToCheck > len(blockTxIds)-1 {
			maxToCheck = len(blockTxIds) - 1
		}

		for i := 1; i <= maxToCheck && len(txIds) < maxCount; i++ {
			// Check timeout
			select {
			case <-ctx.Done():
				return txIds, nil
			default:
			}

			txId := blockTxIds[i]
			txIds = append(txIds, txId)
		}

		// Get previous block hash for next iteration
		url = fmt.Sprintf("%s/block/%s", tf.GetUrl(testNet), currentBlockHash)
		resp, err = http.Get(url)
		if err != nil {
			break
		}

		var blockInfo struct {
			PreviousBlockHash string `json:"previousblockhash"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&blockInfo); err != nil {
			resp.Body.Close()
			break
		}
		resp.Body.Close()

		if blockInfo.PreviousBlockHash == "" {
			// Reached genesis block
			break
		}

		currentBlockHash = blockInfo.PreviousBlockHash
	}

	return txIds, nil
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
