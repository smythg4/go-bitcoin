package keys

import (
	"fmt"
	"go-bitcoin/internal/eccmath"
	"go-bitcoin/internal/encoding"
	"io"
	"math/big"
)

// WIF (Wallet Import Format) version bytes
const (
	WIF_PREFIX_MAINNET byte = 0x80 // Mainnet private key prefix
	WIF_PREFIX_TESTNET byte = 0xef // Testnet private key prefix
	WIF_COMPRESSED_SUFFIX byte = 0x01 // Compressed public key marker
)

type PublicKey = eccmath.S256Point

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

func (pk *PrivateKey) PublicKey() PublicKey {
	point := pk.group.ScalarBaseMultiply(pk.secret)
	return eccmath.NewS256Point(point, pk.group)
}

func (pk *PrivateKey) SignHash(hash []byte) (eccmath.Signature, error) {
	z := new(big.Int).SetBytes(hash)
	return pk.Sign(z)
}

func (pk *PrivateKey) Sign(z *big.Int) (eccmath.Signature, error) {
	return pk.group.Sign(pk.secret, z)
}

func (pk *PrivateKey) Serialize(compressed, testnet bool) string {
	// WIF format encoding for private keys
	secretBytes := make([]byte, 32)
	secret := pk.secret.Bytes()
	copy(secretBytes[32-len(secret):], secret) // right-align with zero padding

	prefix := WIF_PREFIX_MAINNET
	if testnet {
		prefix = WIF_PREFIX_TESTNET
	}

	// build result
	result := []byte{prefix}
	result = append(result, secretBytes...)

	// Add 0x01 suffix only if compressed
	if compressed {
		result = append(result, WIF_COMPRESSED_SUFFIX)
	}

	return encoding.EncodeBase58Checksum(result)
}

func ParsePublicKey(r io.Reader) (*PublicKey, error) {
	bc := eccmath.NewBitcoin()

	// create a temporary point
	tempPoint := eccmath.NewS256Point(bc.G, bc)

	// deserialize
	secBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	point, err := tempPoint.Deserialize(secBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SEC pubkey: %w", err)
	}

	return &point, nil
}
