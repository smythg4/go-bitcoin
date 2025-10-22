package encoding

import (
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/ripemd160"
)

const BIP37_CONSTANT uint32 = 0xfba4c795
const SIGHASH_ALL uint32 = 0x01000000

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
	c1 := uint32(0xcc9e2d51)
	c2 := uint32(0x1b873593)

	length := len(data)
	h1 := uint32(seed)
	roundedEnd := length & 0xfffffffc // round down to 4 byte block

	// process 4-byte blocks
	for i := 0; i < roundedEnd; i += 4 {
		k1 := uint32(data[i]&0xff) |
			(uint32(data[i+1]&0xff) << 8) |
			(uint32(data[i+2]&0xff) << 16) |
			(uint32(data[i+3]) << 24)
		k1 *= c1
		k1 = (k1 << 15) | (k1 >> 17) // ROTL32(k1, 15)
		k1 *= c2

		h1 ^= k1
		h1 = (h1 << 13) | (h1 >> 19) // ROTL32(h1, 13)
		h1 = h1*5 + 0xe6546b64
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
		k1 *= c1
		k1 = (k1 << 15) | (k1 >> 17) // ROTL(k1, 15)
		k1 *= c2
		h1 ^= k1
	}

	// finalization
	h1 ^= uint32(length)

	// fmix(h1)
	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
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
