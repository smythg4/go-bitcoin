package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/encoding"
	"go-bitcoin/internal/mempool"
	"go-bitcoin/internal/transactions"
	"io"
)

type PrefilledTransaction struct {
	Index int
	Tx    *transactions.Transaction
}

type CompactBlockMessage struct {
	// Headers and Short IDs
	Header        *block.Block
	Nonce         uint64 // 8 bytes LE
	ShortIDs      [][6]byte
	PrefilledTxns []PrefilledTransaction
}

func ParseCompactBlockMessage(r io.Reader) (CompactBlockMessage, error) {
	// parse header
	header, err := block.ParseBlock(r)
	if err != nil {
		return CompactBlockMessage{}, err
	}

	// parse nonce
	nonceBytes := make([]byte, 8)
	if _, err := io.ReadFull(r, nonceBytes); err != nil {
		return CompactBlockMessage{}, err
	}
	nonce := binary.LittleEndian.Uint64(nonceBytes)

	// parse short id len
	sidlen, err := encoding.ReadVarInt(r)
	if err != nil {
		return CompactBlockMessage{}, err
	}

	// parse short ids
	shortIds := make([][6]byte, sidlen)
	for i := uint64(0); i < sidlen; i++ {
		if _, err := io.ReadFull(r, shortIds[i][:]); err != nil {
			return CompactBlockMessage{}, err
		}
	}

	// parse prefilled txns len
	pfTxnsLen, err := encoding.ReadVarInt(r)
	if err != nil {
		return CompactBlockMessage{}, err
	}

	// parse prefilled transactions
	pfTxns := make([]PrefilledTransaction, pfTxnsLen)
	prevIndex := -1
	for i := uint64(0); i < pfTxnsLen; i++ {
		// read differential value
		diff, err := encoding.ReadVarInt(r)
		if err != nil {
			return CompactBlockMessage{}, err
		}

		// calculate actual index
		actualIndex := prevIndex + int(diff) + 1

		// parse transaction
		tx, err := transactions.ParseTransaction(r)
		if err != nil {
			return CompactBlockMessage{}, fmt.Errorf("prefilled tx %d (index %d): %w", i, actualIndex, err)
		}

		pfTxns[i] = PrefilledTransaction{
			Index: actualIndex,
			Tx:    &tx,
		}

		prevIndex = actualIndex
	}

	return CompactBlockMessage{
		Header:        &header,
		Nonce:         nonce,
		ShortIDs:      shortIds,
		PrefilledTxns: pfTxns,
	}, nil
}

func (cb *CompactBlockMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)
	// write header bytes
	headerBytes, err := cb.Header.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(headerBytes); err != nil {
		return nil, err
	}
	// write nonce
	buf8 := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf8, cb.Nonce)
	if _, err := result.Write(buf8); err != nil {
		return nil, err
	}

	// write shortIdsLen
	shortIdLen, err := encoding.EncodeVarInt(uint64(len(cb.ShortIDs)))
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(shortIdLen); err != nil {
		return nil, err
	}
	// write ShortIds
	for i := 0; i < len(cb.ShortIDs); i++ {
		if _, err := result.Write(cb.ShortIDs[i][:]); err != nil {
			return nil, err
		}
	}

	// write prefilledtxnslen
	prefilledLen, err := encoding.EncodeVarInt(uint64(len(cb.PrefilledTxns)))
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(prefilledLen); err != nil {
		return nil, err
	}
	// write ShortIds
	prevIndex := -1
	for i := 0; i < len(cb.PrefilledTxns); i++ {
		// calculate differential
		diff := int(cb.PrefilledTxns[i].Index) - prevIndex - 1
		diffBytes, err := encoding.EncodeVarInt(uint64(diff))
		if err != nil {
			return nil, err
		}
		if _, err := result.Write(diffBytes); err != nil {
			return nil, err
		}

		// serialize transaction
		txBytes, err := cb.PrefilledTxns[i].Tx.Serialize()
		if err != nil {
			return nil, err
		}
		if _, err := result.Write(txBytes); err != nil {
			return nil, err
		}

		prevIndex = int(cb.PrefilledTxns[i].Index)
	}
	return result.Bytes(), nil
}

func (cb CompactBlockMessage) Command() string {
	return "cmpctblock"
}

type GetBlockTransactionMessage struct {
	// Block Transaction Request
	BlockHash [32]byte // output from double-SHA256 of the block header
	Indexes   []int
}

func ParseGetBlockTransactionMessage(r io.Reader) (GetBlockTransactionMessage, error) {
	// parse block hash
	var bh [32]byte
	if _, err := io.ReadFull(r, bh[:]); err != nil {
		return GetBlockTransactionMessage{}, err
	}
	// parse indexLen
	idxLen, err := encoding.ReadVarInt(r)
	if err != nil {
		return GetBlockTransactionMessage{}, err
	}
	// parse indexes
	idxs := make([]int, idxLen)
	prevIndex := -1
	for i := uint64(0); i < idxLen; i++ {
		// read the differential values
		diff, err := encoding.ReadVarInt(r)
		if err != nil {
			return GetBlockTransactionMessage{}, err
		}

		// calculate actual index
		actualIndex := prevIndex + int(diff) + 1

		idxs[i] = actualIndex

		prevIndex = actualIndex
	}

	return GetBlockTransactionMessage{
		BlockHash: bh,
		Indexes:   idxs,
	}, nil
}

func (btm *GetBlockTransactionMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)
	// write blockhash
	if _, err := result.Write(btm.BlockHash[:]); err != nil {
		return nil, err
	}

	// write indexLen
	idxLen, err := encoding.EncodeVarInt(uint64(len(btm.Indexes)))
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(idxLen); err != nil {
		return nil, err
	}

	// write indexes
	prevIndex := -1
	for i := 0; i < len(btm.Indexes); i++ {
		// calculate differential
		diff := int(btm.Indexes[i]) - prevIndex - 1
		diffBytes, err := encoding.EncodeVarInt(uint64(diff))
		if err != nil {
			return nil, err
		}
		if _, err := result.Write(diffBytes); err != nil {
			return nil, err
		}

		prevIndex = int(btm.Indexes[i])
	}

	return result.Bytes(), nil
}

func (btm GetBlockTransactionMessage) Command() string {
	return "getblocktxn"
}

type BlockTransactionMessage struct {
	// Block Transactions
	BlockHash    [32]byte
	Transactions []*transactions.Transaction
}

func ParseBlockTransactionMessage(r io.Reader) (BlockTransactionMessage, error) {
	// parse block hash
	var bh [32]byte
	if _, err := io.ReadFull(r, bh[:]); err != nil {
		return BlockTransactionMessage{}, err
	}

	// parse transactions len
	txLen, err := encoding.ReadVarInt(r)
	if err != nil {
		return BlockTransactionMessage{}, err
	}

	// parse transactions
	txns := make([]*transactions.Transaction, txLen)
	for i := uint64(0); i < txLen; i++ {
		tx, err := transactions.ParseTransaction(r)
		if err != nil {
			return BlockTransactionMessage{}, err
		}
		txns[i] = &tx
	}

	return BlockTransactionMessage{
		BlockHash:    bh,
		Transactions: txns,
	}, nil
}

func (bt *BlockTransactionMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)
	// write blockhash
	if _, err := result.Write(bt.BlockHash[:]); err != nil {
		return nil, err
	}
	// write transaction length
	txLenBytes, err := encoding.EncodeVarInt(uint64(len(bt.Transactions)))
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(txLenBytes); err != nil {
		return nil, err
	}
	// write transactions
	for i := 0; i < len(bt.Transactions); i++ {
		txBytes, err := bt.Transactions[i].Serialize()
		if err != nil {
			return nil, err
		}
		if _, err := result.Write(txBytes); err != nil {
			return nil, err
		}
	}
	return result.Bytes(), nil
}

func (bt BlockTransactionMessage) Command() string {
	return "blocktxn"
}

type SendCompactMessage struct {
	HighBandwidth bool
	Version       uint64 // LE
}

func ParseSendCompactMessage(r io.Reader) (SendCompactMessage, error) {
	highBandwidthBytes := make([]byte, 1)
	if _, err := io.ReadFull(r, highBandwidthBytes); err != nil {
		return SendCompactMessage{}, err
	}
	hbw := false
	if highBandwidthBytes[0] == 1 {
		hbw = true
	}
	var versBytes [8]byte
	if _, err := io.ReadFull(r, versBytes[:]); err != nil {
		return SendCompactMessage{}, err
	}
	version := binary.LittleEndian.Uint64(versBytes[:])

	return SendCompactMessage{
		HighBandwidth: hbw,
		Version:       version,
	}, nil
}

func (cm *SendCompactMessage) Serialize() ([]byte, error) {
	result := make([]byte, 9)
	if cm.HighBandwidth {
		result[0] = 1
	}
	binary.LittleEndian.PutUint64(result[1:9], cm.Version)
	return result, nil
}

func (cm SendCompactMessage) Command() string {
	return "sendcmpct"
}

func ReconstructBlock(msg CompactBlockMessage, pool *mempool.Mempool, missingTxns []*transactions.Transaction, version uint64) (*block.Block, []int, error) {
	// return (reconstructed block, missing tx indexes, error)

	// calc short ids
	k0, k1, err := mempool.CalcShortIDKeys(msg.Header, msg.Nonce)
	if err != nil {
		return nil, nil, err
	}

	// match shortids to mempool
	// BIP152 version 2 uses wtxid, version 1 uses txid
	useWtxid := (version == 2)
	matches := pool.MatchShortIDs(msg.ShortIDs, k0, k1, useWtxid)

	// build full transaction list
	totalTxns := len(msg.ShortIDs) + len(msg.PrefilledTxns)
	txns := make([]*transactions.Transaction, totalTxns)

	// place prefilled txns
	for _, pf := range msg.PrefilledTxns {
		txns[pf.Index] = pf.Tx
	}

	// fill in match transactions
	shortIDIdx := 0
	missing := []int{}
	for i := 0; i < totalTxns; i++ {
		if txns[i] != nil {
			continue // was prefilled
		}

		sid := msg.ShortIDs[shortIDIdx]
		if tx, found := matches[sid]; found {
			txns[i] = tx
		} else {
			missing = append(missing, i)
		}
		shortIDIdx++
	}

	// if we have missing txns, fill them in
	if missingTxns != nil {
		missIdx := 0
		for _, idx := range missing {
			if missIdx < len(missingTxns) {
				txns[idx] = missingTxns[missIdx]
				missIdx++
			}
		}
		missing = missing[missIdx:]
	}

	// convert to block
	reconstructed := msg.Header
	reconstructed.TxHashes = make([][32]byte, len(txns))
	for i, tx := range txns {
		if tx != nil {
			hash, _ := tx.Hash()
			reconstructed.TxHashes[i] = hash
		}
	}

	return reconstructed, missing, nil
}
