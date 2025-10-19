package encoding

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
)

const BASE58_ALPHABET string = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func EncodeBase58(data []byte) string {
	// Base58 will be replaced by Bech32 standard
	count := 0
	for _, b := range data {
		if b == 0 {
			count++
		} else {
			break
		}
	}

	num := new(big.Int).SetBytes(data)

	// encode to base58
	result := ""
	zero := big.NewInt(0)
	fiftyEight := big.NewInt(58)
	mod := new(big.Int)

	for num.Cmp(zero) > 0 {
		num.DivMod(num, fiftyEight, mod)
		result = string(BASE58_ALPHABET[mod.Int64()]) + result
	}

	// Add '1' for each leading zero byte
	prefix := strings.Repeat("1", count)

	return prefix + result
}

func EncodeBase58Checksum(data []byte) string {
	return EncodeBase58(append(data, Hash256(data)[:4]...))
}

func getIndex(c byte) int {
	for i := 0; i < len(BASE58_ALPHABET); i++ {
		if BASE58_ALPHABET[i] == c {
			return i
		}
	}
	return -1
}

func DecodeBase58(base58 string) ([]byte, error) {
	// 1. Count leading '1's
	count := 0
	for _, c := range base58 {
		if c == '1' {
			count++
		} else {
			break
		}
	}

	// 2. Decode the numeric part
	num := big.NewInt(0)
	fiftyEight := big.NewInt(58)

	for _, c := range base58 {
		num.Mul(num, fiftyEight)
		index := getIndex(byte(c)) // gotta implement
		if index == -1 {
			return nil, fmt.Errorf("invalid character: %c", c)
		}
		bigIndex := big.NewInt(int64(index))
		num.Add(num, bigIndex)
	}

	// 3. Convert to bytes and prepend zeros
	combined := num.Bytes()
	combined = append(make([]byte, count), combined...)

	// 4. Split value and checksum (last 4 bytes)
	if len(combined) < 4 {
		return nil, errors.New("decoded data too short")
	}
	valueWithVersion := combined[:len(combined)-4]
	checksum := combined[len(combined)-4:] // last 4 bytes are the checksum

	// 5. Verify checksum
	hashedValue := Hash256(valueWithVersion)
	hashCheckSum := hashedValue[:4]

	if !slices.Equal(hashCheckSum, checksum) {
		return nil, fmt.Errorf("bad checksum: %x, %x", hashCheckSum, checksum)
	}
	return valueWithVersion[1:], nil
}
