package transactions

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
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
