# go-bitcoin

A Bitcoin implementation in Go, following **Programming Bitcoin** by Jimmy Song.

## Current Progress

**Chapter 3: Elliptic Curve Cryptography** ✅
**Chapter 4: Serialization** ✅
**Chapter 5: Transactions** ✅
**Chapter 6: Script** (next)

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

### Variable-Length Integers (`internal/encoding`)
- **VarInt Encoding**: Compact integer encoding used throughout Bitcoin protocol
  - 1-byte for values < 0xfd
  - 3-byte for values < 0x10000 (0xfd prefix)
  - 5-byte for values < 0x100000000 (0xfe prefix)
  - 9-byte for larger values (0xff prefix)
- Efficient serialization/deserialization from io.Reader

### Bitcoin Script (`internal/script`)
- Script command parsing and serialization
- Support for data push operations (1-75 bytes, PUSHDATA1/2/4)
- Opcode definitions (OP_DUP, OP_HASH160, OP_EQUALVERIFY, OP_CHECKSIG, etc.)
- Clean separation of data elements vs opcodes using ScriptCommand type

### Transaction Handling (`internal/transactions`)
- **Transaction Structure**: Version, inputs, outputs, locktime
- **TxIn (Transaction Input)**:
  - Previous transaction hash (with endianness handling)
  - Previous output index
  - ScriptSig (signature script)
  - Sequence number
- **TxOut (Transaction Output)**:
  - Amount in satoshis
  - ScriptPubKey (locking script)
- **Transaction Serialization/Deserialization**: Full round-trip support
- **Transaction ID Calculation**: Hash256 with proper byte reversal
- **Transaction Fetching**: Download and parse real transactions from Blockstream API
- **SegWit Support**: Detects and strips SegWit marker bytes for legacy parsing
- **UTXO Chain Traversal**: Follow transaction inputs to previous outputs

## Example Usage

### Generating Keys and Addresses

```go
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
```

### Fetching and Parsing Transactions

```go
// Create transaction fetcher
fetcher := transactions.NewTxFetcher()

// Fetch a real testnet transaction
txId := "e0fc453aa494912627ca3d93c411e8b5f1c8e8d81d5a07af023d45426f224fab"
tx, err := fetcher.Fetch(txId, true, false)  // testnet, use cache
if err != nil {
    panic(err)
}

// Transaction implements String() for easy printing
fmt.Println(tx)
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
    │   ├── hash.go              # Hash256 and Hash160 functions
    │   └── varints.go           # Variable-length integer encoding
    ├── keys/
    │   └── private_key.go       # Private/public key management
    ├── script/
    │   └── script.go            # Bitcoin Script parsing and serialization
    └── transactions/
        ├── transaction.go       # Transaction structure and operations
        ├── txinputs.go          # TxIn and TxOut types
        └── fetch.go             # Transaction fetching from network
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

- Chapter 6: Script Evaluation
  - Execute Bitcoin Script opcodes
  - Stack-based operation engine
  - Implement common script patterns (P2PKH, P2SH)
- Chapter 7: Transaction Signing
  - Create transactions from scratch
  - Sign transaction inputs
  - Verify transaction signatures
- Chapter 8+: Advanced Topics
  - Bloom filters
  - Merkle trees
  - Network communication
  - Blockchain validation

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
