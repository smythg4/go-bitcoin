# go-bitcoin

A Bitcoin implementation in Go, following **Programming Bitcoin** by Jimmy Song.

## Current Progress

**Chapter 3: Elliptic Curve Cryptography** ✅
**Chapter 4: Serialization** (in progress)

## Features

### Finite Field Arithmetic (`internal/eccmath`)
- Field element operations over prime fields using `math/big.Int`
- Addition, subtraction, multiplication, division
- Modular exponentiation and multiplicative inverse
- Modular square root (for secp256k1's prime)
- Proper handling of negative numbers in modular arithmetic

### Elliptic Curve Operations (`internal/eccmath`)
- Point representation on elliptic curves (y² = x³ + ax + b)
- Point validation (curve equation verification)
- Point at infinity handling
- Point addition (general case, vertical line case, point doubling)
- Optimized scalar multiplication using binary expansion (double-and-add)

### secp256k1 Implementation (`internal/eccmath`)
- Bitcoin's secp256k1 curve (y² = x³ + 7 over F_p)
- Generator point G and order N
- S256Point and S256Field types
- Signature type with hex formatting

### ECDSA Signing & Verification (`internal/keys`)
- PrivateKey type with signing capability
- PublicKey type with verification capability
- Secure random k generation using `crypto/rand`
- Complete ECDSA signature creation and verification

### Serialization (`internal/eccmath`)
- **SEC Format**: Standards for Efficient Cryptography point encoding
  - Uncompressed format: `04 || x || y` (65 bytes)
  - Compressed format: `02/03 || x` (33 bytes)
  - Deserialization with y-coordinate recovery from x

## Example Usage

```go
package main

import (
    "fmt"
    "go-bitcoin/internal/keys"
    "math/big"
)

func main() {
    // Create a private key
    secret := big.NewInt(0xdeadbeef54321)
    privateKey := keys.NewPrivateKey(secret)

    // Generate public key
    publicKey := privateKey.PublicKey()

    // Serialize public key (compressed or uncompressed)
    compressed := publicKey.SecSerialize(true)
    fmt.Printf("Compressed public key: %x\n", compressed)

    // Sign a message
    z := new(big.Int).SetBytes([]byte("message hash"))
    sig, _ := privateKey.Sign(z)

    // Verify signature
    valid := publicKey.Verify(z, sig)
    fmt.Printf("Signature valid: %v\n", valid)
}
```

## Project Structure

```
go-bitcoin/
├── main.go
├── go.mod
├── README.md
└── internal/
    ├── eccmath/
    │   ├── elliptic_curve.go   # Generic elliptic curve operations
    │   ├── field_elements.go    # Finite field arithmetic
    │   ├── secp256k1.go        # Bitcoin's secp256k1 curve
    │   └── signature.go         # ECDSA signature type
    └── keys/
        └── private_key.go       # Private/public key management
```

## Implementation Notes

- Uses Go's `math/big.Int` for arbitrary-precision arithmetic
- Cryptographically secure random number generation via `crypto/rand`
- All operations use big-endian byte order (Bitcoin standard)
- Follows idiomatic Go patterns (composition over inheritance)

## Next Steps

- DER signature encoding
- Base58Check encoding
- Bitcoin address generation
- Chapter 5: Transactions

## Development

This is a learning project following "Programming Bitcoin" by Jimmy Song. The goal is to understand Bitcoin's cryptographic foundations by implementing them from scratch. For production use, consider battle-tested libraries like `btcsuite/btcd`.

```bash
# Run the example
go run main.go

# Run tests (coming soon)
go test ./...
```

## Resources

- Book: [Programming Bitcoin](https://programmingbitcoin.com/) by Jimmy Song
- Bitcoin Standards: [BIP 340](https://github.com/bitcoin/bips/blob/master/bip-0340.mediawiki), [SEC Format](https://www.secg.org/sec1-v2.pdf)
