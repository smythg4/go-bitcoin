package mempool

import (
	"go-bitcoin/internal/transactions"
	"sync"
)

type Mempool struct {
	txs map[[32]byte]*transactions.Transaction // txid -> transaction
	mu  sync.Mutex
}

func New() *Mempool {
	return &Mempool{
		txs: make(map[[32]byte]*transactions.Transaction),
	}
}

func (m *Mempool) Add(tx *transactions.Transaction) error {
	txid, err := tx.Hash()
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.txs[txid] = tx
	m.mu.Unlock()
	return nil
}

func (m *Mempool) Get(txid [32]byte) (*transactions.Transaction, bool) {
	m.mu.Lock()
	tx, exists := m.txs[txid]
	m.mu.Unlock()
	return tx, exists
}

func (m *Mempool) Remove(txid [32]byte) {
	m.mu.Lock()
	delete(m.txs, txid)
	m.mu.Unlock()
}

func (m *Mempool) All() []*transactions.Transaction {
	result := make([]*transactions.Transaction, 0, len(m.txs))
	m.mu.Lock()
	for _, tx := range m.txs {
		result = append(result, tx)
	}
	m.mu.Unlock()
	return result
}

func (m *Mempool) MatchShortIDs(shortids [][6]byte, k0, k1 uint64, useWtxid bool) map[[6]byte]*transactions.Transaction {
	requested := make(map[[6]byte]bool, len(shortids))
	for _, sid := range shortids {
		requested[sid] = true
	}

	m.mu.Lock()
	matches := make(map[[6]byte]*transactions.Transaction)

	for _, tx := range m.txs {
		var hash [32]byte
		var err error
		if useWtxid {
			hash, err = tx.WitnessHash()
		} else {
			hash, err = tx.Hash()
		}
		if err != nil {
			continue
		}

		// CRITICAL FIX: Hash() and WitnessHash() return reversed (display order) hashes,
		// but BIP152 requires non-reversed (internal little-endian) hashes for SipHash.
		// We need to reverse it back to its internal representation.
		hashForSipHash := hash
		for i := 0; i < 16; i++ {
			hashForSipHash[i], hashForSipHash[31-i] = hashForSipHash[31-i], hashForSipHash[i]
		}

		sid := CalculateShortID(hashForSipHash, k0, k1)

		if requested[sid] {
			matches[sid] = tx
		}
	}
	m.mu.Unlock()
	return matches
}
