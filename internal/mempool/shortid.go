package mempool

import (
	"crypto/sha256"
	"encoding/binary"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/encoding"
)

func CalcShortIDKeys(header *block.Block, nonce uint64) (k0, k1 uint64, err error) {
	k0, k1 = 0, 0

	ser, err := header.Serialize()
	if err != nil {
		return k0, k1, err
	}

	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, nonce)
	fullBytes := append(ser, nonceBytes...)

	hashBytes := sha256.Sum256(fullBytes)

	k0 = binary.LittleEndian.Uint64(hashBytes[0:8])
	k1 = binary.LittleEndian.Uint64(hashBytes[8:16])

	return k0, k1, nil
}

func CalculateShortID(txid [32]byte, k0, k1 uint64) [6]byte {
	hash := encoding.SipHash24(k0, k1, txid[:])
	var result [6]byte
	result[0] = byte(hash)
	result[1] = byte(hash >> 8)
	result[2] = byte(hash >> 16)
	result[3] = byte(hash >> 24)
	result[4] = byte(hash >> 32)
	result[5] = byte(hash >> 40)
	return result
}
