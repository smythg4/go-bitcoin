package encoding

import (
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/ripemd160"
)

const BIP37_CONSTANT uint32 = 0xfba4c795
const SIGHASH_ALL uint32 = 1

// MurmurHash3 constants
const (
	MURMUR3_C1         uint32 = 0xcc9e2d51
	MURMUR3_C2         uint32 = 0x1b873593
	MURMUR3_SEED       uint32 = 0xe6546b64
	MURMUR3_FMIX_C1    uint32 = 0x85ebca6b
	MURMUR3_FMIX_C2    uint32 = 0xc2b2ae35
)

// SipHash-2-4 initialization constants
const (
	SIPHASH_INIT_V0 uint64 = 0x736f6d6570736575
	SIPHASH_INIT_V1 uint64 = 0x646f72616e646f6d
	SIPHASH_INIT_V2 uint64 = 0x6c7967656e657261
	SIPHASH_INIT_V3 uint64 = 0x7465646279746573
	SIPHASH_FINAL   uint64 = 0xff
)

func Hash256(data []byte) []byte {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return second[:]
}

func Hash160(data []byte) []byte {
	h1 := sha256.Sum256(data)

	hasher := ripemd160.New()
	hasher.Write(h1[:])
	return hasher.Sum(nil)
}

func MurmurHash3(data []byte, seed uint32) uint32 {
	length := len(data)
	h1 := uint32(seed)
	roundedEnd := length & 0xfffffffc // round down to 4 byte block

	// process 4-byte blocks
	for i := 0; i < roundedEnd; i += 4 {
		k1 := uint32(data[i]&0xff) |
			(uint32(data[i+1]&0xff) << 8) |
			(uint32(data[i+2]&0xff) << 16) |
			(uint32(data[i+3]) << 24)
		k1 *= MURMUR3_C1
		k1 = (k1 << 15) | (k1 >> 17) // ROTL32(k1, 15)
		k1 *= MURMUR3_C2

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19) // ROTL32(h1, 13)
		h1 = h1*5 + MURMUR3_SEED
	}

	// tail (remaining 1-3 bytes)
	k1 := uint32(0)
	val := length & 0x03

	if val == 3 {
		k1 = uint32(data[roundedEnd+2]&0xff) << 16
	}
	if val >= 2 {
		k1 |= uint32(data[roundedEnd+1]&0xff) << 8
	}
	if val >= 1 {
		k1 |= uint32(data[roundedEnd] & 0xff)
		k1 *= MURMUR3_C1
		k1 = (k1 << 15) | (k1 >> 17) // ROTL(k1, 15)
		k1 *= MURMUR3_C2
		h1 ^= k1
	}

	// finalization
	h1 ^= uint32(length)

	// fmix(h1)
	h1 ^= h1 >> 16
	h1 *= MURMUR3_FMIX_C1
	h1 ^= h1 >> 13
	h1 *= MURMUR3_FMIX_C2
	h1 ^= h1 >> 16

	return h1
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

func BitFieldToBytes(bitField []byte) ([]byte, error) {
	if len(bitField)%8 != 0 {
		return nil, errors.New("bitField does not have a length divisible by 8")
	}
	result := make([]byte, len(bitField)/8)
	for i, bit := range bitField {
		byteIndex := i / 8
		bitIndex := i % 8
		if bit != byte(0x00) {
			result[byteIndex] |= (1 << byte(bitIndex))
		}
	}
	return result, nil
}

func SipHash24(key0, key1 uint64, data []byte) uint64 {
	// Initialize state with keys
	v0 := key0 ^ SIPHASH_INIT_V0
	v1 := key1 ^ SIPHASH_INIT_V1
	v2 := key0 ^ SIPHASH_INIT_V2
	v3 := key1 ^ SIPHASH_INIT_V3

	// Process full 8-byte blocks
	length := len(data)
	left := length & 7 // remaining bytes after 8-byte blocks
	blocks := length - left

	for i := 0; i < blocks; i += 8 {
		// Read 8 bytes as little-endian uint64
		m := uint64(data[i]) |
			uint64(data[i+1])<<8 |
			uint64(data[i+2])<<16 |
			uint64(data[i+3])<<24 |
			uint64(data[i+4])<<32 |
			uint64(data[i+5])<<40 |
			uint64(data[i+6])<<48 |
			uint64(data[i+7])<<56

		// Compression: 2 rounds
		v3 ^= m
		sipRound(&v0, &v1, &v2, &v3)
		sipRound(&v0, &v1, &v2, &v3)
		v0 ^= m
	}

	// Process remaining bytes (0-7 bytes)
	var b uint64 = uint64(length) << 56
	switch left {
	case 7:
		b |= uint64(data[blocks+6]) << 48
		fallthrough
	case 6:
		b |= uint64(data[blocks+5]) << 40
		fallthrough
	case 5:
		b |= uint64(data[blocks+4]) << 32
		fallthrough
	case 4:
		b |= uint64(data[blocks+3]) << 24
		fallthrough
	case 3:
		b |= uint64(data[blocks+2]) << 16
		fallthrough
	case 2:
		b |= uint64(data[blocks+1]) << 8
		fallthrough
	case 1:
		b |= uint64(data[blocks])
	}

	// Final compression
	v3 ^= b
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)
	v0 ^= b

	// Finalization: XOR 0xff into v2, then 4 rounds
	v2 ^= SIPHASH_FINAL
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)

	return v0 ^ v1 ^ v2 ^ v3
}

func sipRound(v0, v1, v2, v3 *uint64) {
	*v0 += *v1
	*v1 = (*v1 << 13) | (*v1 >> (64 - 13))
	*v1 ^= *v0
	*v0 = (*v0 << 32) | (*v0 >> (64 - 32))

	*v2 += *v3
	*v3 = (*v3 << 16) | (*v3 >> (64 - 16))
	*v3 ^= *v2

	*v0 += *v3
	*v3 = (*v3 << 21) | (*v3 >> (64 - 21))
	*v3 ^= *v0

	*v2 += *v1
	*v1 = (*v1 << 17) | (*v1 >> (64 - 17))
	*v1 ^= *v2
	*v2 = (*v2 << 32) | (*v2 >> (64 - 32))
}
