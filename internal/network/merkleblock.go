package network

import (
	"bytes"
	"encoding/binary"
	"go-bitcoin/internal/encoding"
	"io"
)

type MerkleBlock struct {
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

func ParseMerkleBlock(r io.Reader) (MerkleBlock, error) {
	var mb MerkleBlock
	buf := make([]byte, 4)
	var err error
	// Version (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlock{}, err
	}
	mb.Version = binary.LittleEndian.Uint32(buf)

	// PrevBlock (32 bytes)
	if _, err := io.ReadFull(r, mb.PrevBlock[:]); err != nil {
		return MerkleBlock{}, err
	}

	// MerkleRoot (32 bytes)
	if _, err := io.ReadFull(r, mb.MerkleRoot[:]); err != nil {
		return MerkleBlock{}, err
	}

	// TimeStamp (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlock{}, err
	}
	mb.TimeStamp = binary.LittleEndian.Uint32(buf)

	// Bits (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlock{}, err
	}
	mb.Bits = binary.LittleEndian.Uint32(buf)

	// Nonce (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlock{}, err
	}
	mb.Nonce = binary.LittleEndian.Uint32(buf)

	// NumTransactions (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return MerkleBlock{}, err
	}
	mb.NumTransactions = binary.LittleEndian.Uint32(buf)

	// NumHashes (VarInt)
	mb.NumHashes, err = encoding.ReadVarInt(r)
	if err != nil {
		return MerkleBlock{}, err
	}

	// TxHashes slice
	hashes := make([][32]byte, mb.NumHashes)
	for i := range hashes {
		if _, err := io.ReadFull(r, hashes[i][:]); err != nil {
			return MerkleBlock{}, err
		}
	}
	mb.TxHashes = hashes

	// NumFlags (VarInt)
	mb.NumFlags, err = encoding.ReadVarInt(r)
	if err != nil {
		return MerkleBlock{}, err
	}

	// FlagBits slice
	flagBytes := make([]byte, mb.NumFlags)
	if _, err := io.ReadFull(r, flagBytes); err != nil {
		return MerkleBlock{}, err
	}
	mb.FlagBits = encoding.BytesToBitField(flagBytes)

	return mb, nil
}

func (mb *MerkleBlock) IsValid() bool {
	mt, err := encoding.NewEmptyMerkleTree(int(mb.NumTransactions))
	if err != nil {
		return false
	}

	if err := mt.PopulateTree(mb.FlagBits, mb.TxHashes); err != nil {
		return false
	}

	return bytes.Equal(mt.Root(), mb.MerkleRoot[:])
}
