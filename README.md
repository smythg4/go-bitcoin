# go-bitcoin

A Bitcoin implementation in Go, following **Programming Bitcoin** by Jimmy Song.

## Current Progress

**Chapter 3: Elliptic Curve Cryptography** ✅
**Chapter 4: Serialization** ✅
**Chapter 5: Transactions** ✅
**Chapter 6: Script** ✅
**Chapter 7: Transaction Creation and Validation** ✅
**Chapter 8: Pay-to-Script-Hash (P2SH)** ✅
**Chapter 9: Blocks** ✅
**Chapter 10: Networking** ✅
**Chapter 11: Simplified Payment Verification (SPV)** (in progress)

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

### Bitcoin Address Generation (`internal/eccmath`, `internal/script`)
- **P2PKH (Pay-to-Public-Key-Hash)** address generation
  - Mainnet addresses (starts with `1`)
  - Testnet addresses (starts with `m` or `n`)
  - Support for both compressed and uncompressed public keys
- **P2SH (Pay-to-Script-Hash)** address generation
  - Mainnet addresses (starts with `3`)
  - Testnet addresses (starts with `2`)
  - Generate addresses from arbitrary scripts (multisig, timelocks, etc.)

### Variable-Length Integers (`internal/encoding`)
- **VarInt Encoding**: Compact integer encoding used throughout Bitcoin protocol
  - 1-byte for values < 0xfd
  - 3-byte for values < 0x10000 (0xfd prefix)
  - 5-byte for values < 0x100000000 (0xfe prefix)
  - 9-byte for larger values (0xff prefix)
- Efficient serialization/deserialization from io.Reader

### Bitcoin Script (`internal/script`)
- **Script Parsing & Serialization**: Full Script parsing and serialization with varint encoding
- **Script Execution Engine**: Stack-based virtual machine for executing Bitcoin scripts
- **Data Push Operations**: Support for 1-75 byte inline push, OP_PUSHDATA1/2/4
- **Stack Operations**: OP_DUP, OP_2DUP, OP_DROP, OP_2DROP, OP_SWAP, OP_TOALTSTACK, OP_FROMALTSTACK
- **Arithmetic Operations**: OP_ADD, OP_SUB, OP_MUL with Bitcoin number encoding (little-endian signed)
- **Logical Operations**: OP_EQUAL, OP_EQUALVERIFY, OP_VERIFY, OP_NOT
- **Flow Control**: OP_IF, OP_NOTIF, OP_ELSE, OP_ENDIF with nested block support
- **Cryptographic Operations**:
  - OP_SHA1, OP_SHA256, OP_HASH160, OP_HASH256
  - OP_CHECKSIG, OP_CHECKSIGVERIFY with full ECDSA verification
  - OP_CHECKMULTISIG with sliding window signature matching for m-of-n multisig
- **Numeric Constants**: OP_0 through OP_16, OP_1NEGATE
- **P2PKH Script Validation**: Complete Pay-to-Public-Key-Hash transaction verification
- **P2SH Script Validation**: Pay-to-Script-Hash (BIP 16) with two-phase execution
  - Automatic P2SH pattern detection during script execution
  - RedeemScript extraction and parsing from ScriptSig
  - Hash verification and redeemScript command injection
  - Full support for P2SH-wrapped multisig transactions
- **Script Combining**: Merge ScriptSig with ScriptPubKey for evaluation

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
- **Signature Hash (SigHash)**: Complete signature hash calculation for transaction signing/verification
  - Automatic P2SH detection with redeemScript extraction
  - Uses redeemScript (not P2SH wrapper) for P2SH sighash calculation per BIP 16
  - Standard P2PKH sighash for non-P2SH transactions
- **Transaction Fetching**: Download and parse real transactions from Blockstream API
  - Automatic legacy transaction detection (filters SegWit)
  - Multi-block search capability
  - Caching for efficient repeated fetches
- **SegWit Support**: Detects and strips SegWit marker bytes for legacy parsing
- **UTXO Chain Traversal**: Follow transaction inputs to previous outputs
- **Transaction Verification**: Full transaction validation from the blockchain
  - P2PKH (Pay-to-Public-Key-Hash) validation
  - P2SH (Pay-to-Script-Hash) validation with multisig support
  - Low-S signature enforcement (BIP 62) for transaction malleability prevention

### Block Handling (`internal/block`)
- **Block Structure**: Version, previous block hash, merkle root, timestamp, bits, nonce
- **Block Parsing/Serialization**: Full block header parsing from binary format
- **Proof of Work Verification**: Validates block hash meets difficulty target
- **Difficulty Calculation**:
  - Bits field to target conversion (compact format)
  - Target to difficulty conversion
  - Difficulty adjustment every 2016 blocks
  - New bits calculation based on time differential
- **Block ID Generation**: Double SHA-256 hash with proper byte reversal
- **Genesis Block Support**: Mainnet and testnet genesis blocks
- **Block Chain Validation**: Validates proof of work and difficulty adjustments

### Networking (`internal/network`)
- **Bitcoin P2P Protocol**: Full network message handling
- **Network Envelope**: Magic bytes, command, payload length, checksum
- **Message Types**:
  - Version handshake (protocol version negotiation)
  - Verack acknowledgment
  - Ping/Pong keepalive
  - GetHeaders (request block headers)
  - Headers response (batch header delivery)
- **Connection Management**:
  - TCP connection handling with timeouts
  - Concurrent read/write loops with goroutines
  - Message routing with dedicated channels
  - Graceful shutdown with sync.WaitGroup
- **Auto-responses**: Automatic ping/pong and version/verack handling
- **Block Header Download**: Download and validate blockchain headers from mainnet peers
- **DNS Seed Support**: Peer discovery via DNS seeds

### Merkle Trees (`internal/encoding`)
- **Merkle Tree Construction**: Build complete merkle trees from transaction hashes
- **Merkle Root Calculation**: Recursive parent level computation with odd-node duplication
- **Tree Navigation**: Cursor-based tree traversal (up, left, right)
- **Merkle Block Parsing**: Parse merkleblock messages for SPV (BIP 37)
- **Bit Field Expansion**: Convert compact flag bytes to bit arrays for tree reconstruction
- **SPV Proof Support**: Infrastructure for simplified payment verification (in progress)

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

### Verifying Bitcoin Transactions

```go
// Create transaction fetcher
fetcher := transactions.NewTxFetcher()

// Fetch a real testnet transaction
txId := "d1f473ab9845130ca3cf1c4880ac093af87720b4df0de1f344c701a5d07ecaa3"
tx, err := fetcher.Fetch(txId, true, false)  // testnet, use cache
if err != nil {
    panic(err)
}

// Verify each input
for i, input := range tx.Inputs {
    // Fetch the previous transaction
    prevTxId := fmt.Sprintf("%x", input.PrevTx)
    prevTx, err := fetcher.Fetch(prevTxId, true, false)
    if err != nil {
        panic(err)
    }

    // Get the previous output's ScriptPubKey
    prevOutput := prevTx.Outputs[input.PrevIdx]

    // Calculate signature hash for this input
    z := tx.SigHash(i, prevOutput.ScriptPubKey)

    // Combine ScriptSig + ScriptPubKey
    combinedScript := input.ScriptSig.Combine(prevOutput.ScriptPubKey)

    // Evaluate the script!
    valid := combinedScript.Evaluate(z)
    fmt.Printf("Input %d: %v\n", i, valid)
}
```

### Evaluating Bitcoin Scripts

```go
// Simple arithmetic script: OP_2 OP_DUP OP_DUP OP_MUL OP_ADD OP_6 OP_EQUAL
// Tests: 2^2 + 2 = 6

scriptSigHex := []byte{0x01, 0x52}  // OP_2
scriptSig, _ := script.ParseScript(bytes.NewReader(scriptSigHex))

scriptPubKeyHex := []byte{0x06, 0x76, 0x76, 0x95, 0x93, 0x56, 0x87}
scriptPubKey, _ := script.ParseScript(bytes.NewReader(scriptPubKeyHex))

combined := scriptSig.Combine(scriptPubKey)
result := combined.Evaluate([]byte{})
fmt.Printf("Result: %v\n", result)  // true
```

### Verifying P2SH Multisig Transactions

```go
// Verify a real 2-of-3 multisig P2SH transaction from testnet
fetcher := transactions.NewTxFetcher()

// This transaction spends from a P2SH address
txId := "fa65bc5fa0ee39e012282701a4ce378474183a330487e839cd1b65b398d7646e"
tx, err := fetcher.Fetch(txId, true, false)
if err != nil {
    panic(err)
}

// Verify the transaction - automatically handles P2SH
valid, err := tx.Verify()
if err != nil {
    panic(err)
}

fmt.Printf("P2SH Transaction Valid: %v\n", valid)  // true
// The redeemScript (2-of-3 multisig) is automatically extracted and executed
```

## Project Structure

```
go-bitcoin/
├── main.go                      # Network demo (header download & validation)
├── go.mod
├── README.md
└── internal/
    ├── eccmath/
    │   ├── elliptic_curve.go   # Generic elliptic curve operations
    │   ├── field_elements.go    # Finite field arithmetic
    │   ├── secp256k1.go        # Bitcoin's secp256k1 curve
    │   └── signature.go         # ECDSA signature with DER/SEC parsing
    ├── encoding/
    │   ├── base58.go            # Base58 and Base58Check encoding
    │   ├── hash.go              # Hash256 and Hash160 functions
    │   ├── varints.go           # Variable-length integer encoding
    │   ├── merkle.go            # Merkle tree construction and navigation
    │   └── merkle_test.go       # Merkle tree tests
    ├── keys/
    │   └── keys.go              # Private/public key management
    ├── script/
    │   ├── script.go            # Bitcoin Script parsing and serialization
    │   ├── opcodes.go           # Script execution engine and opcodes
    │   └── exercise_test.go     # Script test cases (arithmetic, SHA-1 collision)
    ├── transactions/
    │   ├── transaction.go       # Transaction structure and SigHash
    │   ├── txinputs.go          # TxIn and TxOut types
    │   └── fetch.go             # Transaction fetching and legacy detection
    ├── block/
    │   └── block.go             # Block header parsing and proof of work
    └── network/
        ├── node.go              # P2P node with concurrent message handling
        ├── network.go           # Network envelope parsing
        ├── version.go           # Version message
        ├── verack.go            # Verack message
        ├── pong.go              # Pong message
        └── blockheader.go       # GetHeaders and Headers messages
```

## Implementation Notes

- Uses Go's `math/big.Int` for arbitrary-precision arithmetic (256-bit operations)
- Cryptographically secure random number generation via `crypto/rand`
- All operations use big-endian byte order (Bitcoin standard)
- Follows idiomatic Go patterns (composition over inheritance)
- **Validates real Bitcoin transactions** from the blockchain using full Script evaluation
- **Connects to Bitcoin mainnet** and downloads/validates block headers
- Implements Bitcoin's legacy P2PKH (Pay-to-Public-Key-Hash) format
- Stack-based Script VM with complete opcode support
- Concurrent P2P networking with goroutines and channels
- RIPEMD-160 via `golang.org/x/crypto` (legacy hash, required for Bitcoin)
- Comprehensive test suite including SHA-1 collision detection (SHAttered attack)

## Standards Implemented

- **SEC (Standards for Efficient Cryptography)**: Public key serialization
- **DER (Distinguished Encoding Rules)**: Signature serialization
- **Base58Check**: Address encoding with checksum
- **WIF (Wallet Import Format)**: Private key serialization
- **BIP-13**: Pay-to-Script-Hash (P2SH) address format
- **BIP-16**: Pay-to-Script-Hash execution semantics
- **BIP-37**: Connection Bloom filtering (merkleblock parsing for SPV)
- **BIP-62**: Low-S signature enforcement for transaction malleability prevention

## Next Steps

- Chapter 11: Simplified Payment Verification (SPV) - **IN PROGRESS**
  - Merkle path generation
  - Merkle proof verification
  - Light client implementation
- Chapter 12: Bloom Filters
  - Bloom filter creation and testing
  - Filtered block retrieval (BIP 37)
  - Privacy-preserving SPV
- Chapter 13: SegWit
  - Segregated Witness implementation (BIP 141, 143, 144)
  - Witness data handling
  - Native SegWit address support
- Chapter 14: Advanced Topics
  - Payment channels
  - Advanced scripting

## Development

This is a learning project following "Programming Bitcoin" by Jimmy Song. The goal is to understand Bitcoin's cryptographic foundations by implementing them from scratch.

**⚠️ For production use**, consider battle-tested libraries like:
- `github.com/btcsuite/btcd` - Full Bitcoin implementation
- `github.com/btcsuite/btcutil` - Bitcoin utility functions

```bash
# Connect to Bitcoin mainnet and download block headers
go run main.go

# Run Script tests (arithmetic, SHA-1 collision, number encoding)
go test -v ./internal/script/

# Run Merkle tree tests (tree construction, navigation)
go test -v ./internal/encoding/ -run TestMerkle

# Run all tests
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
