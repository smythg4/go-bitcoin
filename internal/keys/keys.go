package keys

import (
	"fmt"
	"go-bitcoin/internal/eccmath"
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
