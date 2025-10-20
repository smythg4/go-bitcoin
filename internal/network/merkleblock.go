package network

import (
	"encoding/binary"
	"go-bitcoin/internal/encoding"
	"io"
)

type MerkleBlockMessage struct {
	Version         uint32     // 4 bytes LE
	PrevBlock       [32]byte   // LE
	MerkleRoot      [32]byte   // LE
	TimeStamp       uint32     // 4 bytes LE, Unix epoch seconds
	Bits            uint32     // 4 bytes LE, compact difficulty target
	Nonce           uint32     // 4 bytes LE, proof of work nonce
	NumTransactions uint32     // 4 bytes LE, total transactions in block
	NumHashes       uint64     // VarInt, number of hashes
	TxHashes        [][32]byte // partial merkle tree hashes
	NumFlags        uint64     // VarInt, number of flag bits
	FlagBits        []byte     // flag bits for tree reconstruction
}

func BytesToBitField(data []byte) []byte {
	flagBits := make([]byte, 0, len(data)*8)
	for _, b := range data {
		for i := 0; i < 8; i++ {
			flagBits = append(flagBits, b&1)
			b >>= 1
		}
	}
	return flagBits
}

func ParseMerkleBlockMessage(r io.Reader) (MerkleBlockMessage, error) {
	var mb MerkleBlockMessage
	buf := make([]byte, 4)
	var err error
	// Version (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlockMessage{}, err
	}
	mb.Version = binary.LittleEndian.Uint32(buf)

	// PrevBlock (32 bytes)
	if _, err := io.ReadFull(r, mb.PrevBlock[:]); err != nil {
		return MerkleBlockMessage{}, err
	}

	// MerkleRoot (32 bytes)
	if _, err := io.ReadFull(r, mb.MerkleRoot[:]); err != nil {
		return MerkleBlockMessage{}, err
	}

	// TimeStamp (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlockMessage{}, err
	}
	mb.TimeStamp = binary.LittleEndian.Uint32(buf)

	// Bits (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlockMessage{}, err
	}
	mb.Bits = binary.LittleEndian.Uint32(buf)

	// Nonce (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlockMessage{}, err
	}
	mb.Nonce = binary.LittleEndian.Uint32(buf)

	// NumTransactions (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlockMessage{}, err
	}
	mb.NumTransactions = binary.LittleEndian.Uint32(buf)

	// NumHashes (VarInt)
	mb.NumHashes, err = encoding.ReadVarInt(r)
	if err != nil {
		return MerkleBlockMessage{}, err
	}

	// TxHashes slice
	hashes := make([][32]byte, mb.NumHashes)
	for i := range hashes {
		if _, err := io.ReadFull(r, hashes[i][:]); err != nil {
			return MerkleBlockMessage{}, err
		}
	}
	mb.TxHashes = hashes

	// NumFlags (VarInt)
	mb.NumFlags, err = encoding.ReadVarInt(r)
	if err != nil {
		return MerkleBlockMessage{}, err
	}

	// FlagBits slice
	flagBytes := make([]byte, mb.NumFlags)
	if _, err := io.ReadFull(r, flagBytes); err != nil {
		return MerkleBlockMessage{}, err
	}
	mb.FlagBits = BytesToBitField(flagBytes)

	return mb, nil
}
