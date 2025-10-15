package encoding

import (
	"math/big"
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
