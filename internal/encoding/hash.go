package encoding

import (
	"crypto/sha256"

	"golang.org/x/crypto/ripemd160"
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
