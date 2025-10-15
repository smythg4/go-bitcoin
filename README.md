# go-bitcoin

A Bitcoin implementation in Go, following **Programming Bitcoin** by Jimmy Song.

## Current Progress

**Chapter 3: Elliptic Curve Cryptography** ✅
**Chapter 4: Serialization** ✅
**Chapter 5: Transactions** (next)

## Features

### Finite Field Arithmetic (`internal/eccmath`)
- Field element operations over prime fields using `math/big.Int`
- Addition, subtraction, multiplication, division
- Modular exponentiation and multiplicative inverse
- Modular square root (for primes where p ≡ 3 mod 4)
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
- S256Point and S256Field types for secp256k1-specific operations
- Signature type with hex formatting and DER encoding

### ECDSA Signing & Verification (`internal/keys`)
- PrivateKey type with signing and serialization (WIF format)
- PublicKey type with verification and address generation
- Secure random k generation using `crypto/rand`
- Complete ECDSA signature creation and verification

### Serialization (`internal/eccmath`, `internal/encoding`)
- **SEC Format**: Standards for Efficient Cryptography point encoding
  - Uncompressed format: `04 || x || y` (65 bytes)
  - Compressed format: `02/03 || x` (33 bytes)
  - Deserialization with y-coordinate recovery from x
- **DER Encoding**: Distinguished Encoding Rules for signatures
  - Proper length encoding and high-bit handling
  - Bitcoin-compatible signature format
- **Base58**: Bitcoin's base58 encoding (no 0, O, I, l)
- **Base58Check**: Base58 with checksum for error detection
- **WIF Format**: Wallet Import Format for private keys
  - Mainnet/testnet support
  - Compressed/uncompressed variants

### Hashing (`internal/encoding`)
- **Hash256**: Double SHA-256 (used for checksums and block hashing)
- **Hash160**: SHA-256 followed by RIPEMD-160 (used for addresses)

### Bitcoin Address Generation (`internal/eccmath`)
- P2PKH (Pay-to-Public-Key-Hash) address generation
- Mainnet addresses (starts with `1`)
- Testnet addresses (starts with `m` or `n`)
- Support for both compressed and uncompressed public keys

## Example Usage

```go
package main

import (
    "fmt"
    "go-bitcoin/internal/keys"
    "math/big"
)

func main() {
    // Create a private key from a secret
    secret := big.NewInt(0xdeadbeef54321)
    privateKey := keys.NewPrivateKey(secret)

    // Generate public key
    publicKey := privateKey.PublicKey()

    // Generate Bitcoin addresses
    mainnetAddr := publicKey.Address(true, false)  // compressed, mainnet
    testnetAddr := publicKey.Address(true, true)   // compressed, testnet

    fmt.Printf("Mainnet address: %s\n", mainnetAddr)
    fmt.Printf("Testnet address: %s\n", testnetAddr)

    // Export private key in WIF format
    wif := privateKey.Serialize(true, false)  // compressed, mainnet
    fmt.Printf("Private key (WIF): %s\n", wif)

    // Sign a message
    z := big.NewInt(1234567890)
    sig, _ := privateKey.Sign(z)

    // Verify signature
    valid := publicKey.Verify(z, sig)
    fmt.Printf("Signature valid: %v\n", valid)

    // Serialize signature in DER format
    derSig := sig.Serialize()
    fmt.Printf("DER signature: %x\n", derSig)
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
    │   └── signature.go         # ECDSA signature with DER encoding
    ├── encoding/
    │   ├── base58.go            # Base58 and Base58Check encoding
    │   └── hash.go              # Hash256 and Hash160 functions
    └── keys/
        └── private_key.go       # Private/public key management
```

## Implementation Notes

- Uses Go's `math/big.Int` for arbitrary-precision arithmetic (256-bit operations)
- Cryptographically secure random number generation via `crypto/rand`
- All operations use big-endian byte order (Bitcoin standard)
- Follows idiomatic Go patterns (composition over inheritance)
- Implements Bitcoin's legacy P2PKH address format
- RIPEMD-160 via `golang.org/x/crypto` (legacy hash, required for Bitcoin)

## Standards Implemented

- **SEC (Standards for Efficient Cryptography)**: Public key serialization
- **DER (Distinguished Encoding Rules)**: Signature serialization
- **Base58Check**: Address encoding with checksum
- **WIF (Wallet Import Format)**: Private key serialization
- **BIP-13**: Pay-to-Script-Hash (P2SH) address format (coming soon)

## Next Steps

- Chapter 5: Transactions
  - Transaction structure and serialization
  - Input and output handling
  - Script evaluation
  - Transaction signing and verification

## Development

This is a learning project following "Programming Bitcoin" by Jimmy Song. The goal is to understand Bitcoin's cryptographic foundations by implementing them from scratch.

**⚠️ For production use**, consider battle-tested libraries like:
- `github.com/btcsuite/btcd` - Full Bitcoin implementation
- `github.com/btcsuite/btcutil` - Bitcoin utility functions

```bash
# Run the example
go run main.go

# Run tests (coming soon)
go test ./...

# Dependencies
go get golang.org/x/crypto/ripemd160
```

## Resources

- Book: [Programming Bitcoin](https://programmingbitcoin.com/) by Jimmy Song
- [Bitcoin Developer Reference](https://developer.bitcoin.org/reference/)
- [BIPs (Bitcoin Improvement Proposals)](https://github.com/bitcoin/bips)
- [SEC Format Specification](https://www.secg.org/sec1-v2.pdf)
- [Base58Check encoding](https://en.bitcoin.it/wiki/Base58Check_encoding)
