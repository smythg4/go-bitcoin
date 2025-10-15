package keys

import (
	"fmt"
	"go-bitcoin/internal/eccmath"
	"go-bitcoin/internal/encoding"
	"math/big"
)

type PrivateKey struct {
	secret *big.Int
	group  *eccmath.Secp256k1Group
}

func (pk PrivateKey) String() string {
	return fmt.Sprintf("%x", pk.secret)
}

func NewPrivateKey(secret *big.Int) *PrivateKey {
	bc := eccmath.NewBitcoin()
	return &PrivateKey{
		secret: secret,
		group:  bc,
	}
}

func (pk *PrivateKey) PublicKey() eccmath.S256Point {
	point := pk.group.ScalarBaseMultiply(pk.secret)
	return eccmath.NewS256Point(point, pk.group)
}

func (pk *PrivateKey) Sign(z *big.Int) (eccmath.Signature, error) {
	return pk.group.Sign(pk.secret, z)
}

func (pk *PrivateKey) Serialize(compressed, testnet bool) string {
	// WIF format encoding for private keys
	secretBytes := make([]byte, 32)
	secret := pk.secret.Bytes()
	copy(secretBytes[32-len(secret):], secret) // right-align with zero padding

	prefix := byte(0x80)
	if testnet {
		prefix = 0xef
	}

	// build result
	result := []byte{prefix}
	result = append(result, secretBytes...)

	// Add 0x01 suffix only if compressed
	if compressed {
		result = append(result, 0x01)
	}

	return encoding.EncodeBase58Checksum(result)
}
