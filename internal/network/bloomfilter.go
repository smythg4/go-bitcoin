package network

import (
	"bytes"
	"encoding/binary"
	"go-bitcoin/internal/encoding"
)

const BLOOM_UPDATE_ALL int = 0

type FilterLoadMessage struct {
	Filter *BloomFilter
	Flag   byte
}

type BloomFilter struct {
	Size          uint32
	BitField      []byte
	FunctionCount int
	Tweak         uint32
}

func NewBloomFilter(size, funcCount, tweak int) BloomFilter {
	return BloomFilter{
		Size:          uint32(size),
		BitField:      make([]byte, size*8),
		FunctionCount: funcCount,
		Tweak:         uint32(tweak),
	}
}

func (bf *BloomFilter) Add(item []byte) {
	for i := 0; i < bf.FunctionCount; i++ {
		seed := uint32(i)*encoding.BIP37_CONSTANT + bf.Tweak
		h := encoding.MurmurHash3(item, seed)
		bit := h % (bf.Size * 8)
		bf.BitField[bit] = 1
	}
}

func (bf *BloomFilter) FilterBytes() ([]byte, error) {
	return encoding.BitFieldToBytes(bf.BitField)
}

func (f *FilterLoadMessage) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Filter size (varint)
	sizeBytes, err := encoding.EncodeVarInt(uint64(f.Filter.Size))
	if err != nil {
		return nil, err
	}
	buf.Write(sizeBytes)

	// Filter data
	filterBytes, err := f.Filter.FilterBytes()
	if err != nil {
		return nil, err
	}
	buf.Write(filterBytes)

	// Number of hash functions (4 bytes LE)
	binary.Write(buf, binary.LittleEndian, uint32(f.Filter.FunctionCount))

	// Tweak (4 bytes LE)
	binary.Write(buf, binary.LittleEndian, f.Filter.Tweak)

	// Flag (1 byte)
	buf.WriteByte(f.Flag)

	return buf.Bytes(), nil
}

func (f *FilterLoadMessage) Command() string {
	return "filterload"
}
