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
**Chapter 11: Simplified Payment Verification (SPV)** ✅
**Chapter 12: Bloom Filters** ✅
**Chapter 13: Segregated Witness (SegWit)** ✅

**🎉 Book Complete!** All chapters from Programming Bitcoin have been successfully implemented and tested.

## Beyond the Book

After completing all chapters, this implementation has been extended with additional Bitcoin protocol features:

**BIP 152: Compact Block Relay** ✅
- Full compact block negotiation and transmission
- SipHash-2-4 shortID calculation for bandwidth optimization
- Mempool-based transaction matching (25-60% bandwidth savings)
- Automatic fallback with `getblocktxn`/`blocktxn` messages
- Support for both version 1 (txid) and version 2 (wtxid)
- Successfully tested against Bitcoin mainnet with real blocks

**BIP 173: Bech32 Address Encoding** ✅
- Complete bech32 encoding for native SegWit addresses
- P2WPKH address generation (bc1q... / tb1q...)
- P2WSH address generation (bc1q... for 32-byte witness programs)
- Polymod checksum calculation and validation
- 8-bit to 5-bit data conversion for witness programs
- Fully compliant with BIP 173 specification and test vectors

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
- **MurmurHash3**: 32-bit MurmurHash3 implementation for bloom filters (BIP 37)

### Bitcoin Address Generation (`internal/address`)
- **P2PKH (Pay-to-Public-Key-Hash)** address generation
  - Mainnet addresses (starts with `1`)
  - Testnet addresses (starts with `m` or `n`)
  - Support for both compressed and uncompressed public keys
  - Base58Check encoding
- **P2SH (Pay-to-Script-Hash)** address generation
  - Mainnet addresses (starts with `3`)
  - Testnet addresses (starts with `2`)
  - Generate addresses from arbitrary scripts (multisig, timelocks, etc.)
  - Base58Check encoding
- **P2WPKH (Pay-to-Witness-Public-Key-Hash)** address generation
  - Native SegWit (witness version 0)
  - Mainnet addresses (starts with `bc1q`)
  - Testnet addresses (starts with `tb1q`)
  - 20-byte pubkey hash witness programs
  - Bech32 encoding (BIP 173)
- **P2WSH (Pay-to-Witness-Script-Hash)** address generation
  - Native SegWit (witness version 0)
  - Mainnet addresses (starts with `bc1q`)
  - Testnet addresses (starts with `tb1q`)
  - 32-byte script hash witness programs
  - Bech32 encoding (BIP 173)

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
- **Dynamic Pattern Detection**: Runtime detection of P2SH, P2WPKH, and P2WSH patterns during execution
- **Data Push Operations**: Support for 1-75 byte inline push, OP_PUSHDATA1/2/4
- **Stack Operations**: OP_DUP, OP_2DUP, OP_DROP, OP_2DROP, OP_SWAP, OP_TOALTSTACK, OP_FROMALTSTACK
- **Arithmetic Operations**: OP_ADD, OP_SUB, OP_MUL with Bitcoin number encoding (little-endian signed)
- **Logical Operations**: OP_EQUAL, OP_EQUALVERIFY, OP_VERIFY, OP_NOT
- **Flow Control**: OP_IF, OP_NOTIF, OP_ELSE, OP_ENDIF with nested block support
- **Cryptographic Operations**:
  - OP_SHA1, OP_SHA256, OP_HASH160, OP_HASH256, OP_RIPEMD160
  - OP_CHECKSIG, OP_CHECKSIGVERIFY with full ECDSA verification
  - OP_CHECKMULTISIG with sliding window signature matching for m-of-n multisig
- **Numeric Constants**: OP_0 through OP_16, OP_1NEGATE
- **P2PKH Script Validation**: Complete Pay-to-Public-Key-Hash transaction verification
- **P2SH Script Validation**: Pay-to-Script-Hash (BIP 16) with two-phase execution
  - Automatic P2SH pattern detection during script execution
  - RedeemScript extraction and parsing from ScriptSig
  - Hash verification and redeemScript command injection
  - Full support for P2SH-wrapped multisig transactions
- **P2WPKH Script Validation**: Pay-to-Witness-Public-Key-Hash (BIP 141)
  - Stack-based witness program detection
  - Automatic witness data injection
  - Converts witness program to P2PKH equivalent script
- **P2WSH Script Validation**: Pay-to-Witness-Script-Hash (BIP 141)
  - Witness script hash validation using SHA256
  - Witness stack item injection
  - Dynamic witness script parsing and execution
- **Nested SegWit Support**: P2SH-wrapped witness programs (P2SH-P2WPKH, P2SH-P2WSH)
- **Script Combining**: Merge ScriptSig with ScriptPubKey for evaluation

### Transaction Handling (`internal/transactions`)
- **Transaction Structure**: Version, inputs, outputs, locktime
- **TxIn (Transaction Input)**:
  - Previous transaction hash (with endianness handling)
  - Previous output index
  - ScriptSig (signature script)
  - Sequence number
  - Witness data (for SegWit transactions)
- **TxOut (Transaction Output)**:
  - Amount in satoshis
  - ScriptPubKey (locking script)
- **Transaction Serialization/Deserialization**: Full round-trip support
  - Legacy format serialization
  - SegWit format serialization (BIP 144) with marker/flag bytes
  - Automatic format detection during parsing
- **Transaction ID Calculation**: Hash256 with proper byte reversal
  - Always uses legacy serialization (witness data excluded from txid)
- **Signature Hash (SigHash)**: Complete signature hash calculation for transaction signing/verification
  - Legacy sighash for pre-SegWit transactions
  - BIP 143 sighash for SegWit transactions
  - Automatic P2SH detection with redeemScript extraction
  - Uses redeemScript (not P2SH wrapper) for P2SH sighash calculation per BIP 16
  - **BIP 143 Optimizations**: Caches hashPrevouts, hashSequence, hashOutputs for efficiency
- **Transaction Fetching**: Download and parse real transactions from Blockstream API
  - Supports both legacy and SegWit transactions
  - Multi-block search capability
  - Caching for efficient repeated fetches
- **SegWit Support**: Full Segregated Witness implementation
  - Witness data parsing and serialization
  - Marker (0x00) and flag (0x01) byte handling
  - Witness item count and data extraction
  - BIP 141 (Segregated Witness structure)
  - BIP 143 (Signature verification for witness v0)
  - BIP 144 (Peer services for witness transactions)
- **UTXO Chain Traversal**: Follow transaction inputs to previous outputs
- **Transaction Verification**: Full transaction validation from the blockchain
  - P2PKH (Pay-to-Public-Key-Hash) validation
  - P2SH (Pay-to-Script-Hash) validation with multisig support
  - P2WPKH (Native SegWit pubkey hash) validation
  - P2WSH (Native SegWit script hash) validation
  - Nested SegWit (P2SH-wrapped witness programs) validation
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
- **Soft Fork Signaling Detection**:
  - BIP 9 version bits signaling
  - BIP 91 SegWit activation signaling
  - BIP 141 SegWit readiness signaling

### Networking (`internal/network`)
- **Bitcoin P2P Protocol**: Full network message handling
- **Network Envelope**: Magic bytes, command, payload length, checksum
- **Message Types**:
  - Version handshake (protocol version negotiation with NODE_WITNESS flag)
  - Verack acknowledgment
  - Ping/Pong keepalive
  - GetHeaders (request block headers)
  - Headers response (batch header delivery)
  - FilterLoad (bloom filter transmission - BIP 37)
  - GetData (request filtered blocks or transactions with MSG_WITNESS_TX support)
  - MerkleBlock (filtered block with merkle proof)
  - Tx (transaction data with witness support)
  - **SendCmpct** (compact block negotiation - BIP 152)
  - **CmpctBlock** (compact block transmission - BIP 152)
  - **GetBlockTxn** (request missing transactions - BIP 152)
  - **BlockTxn** (missing transaction response - BIP 152)
- **Connection Management**:
  - TCP connection handling with timeouts
  - Concurrent read/write loops with goroutines
  - Message routing with dedicated channels per message type
  - Buffered channels to prevent message loss
  - Graceful shutdown with sync.WaitGroup
- **Auto-responses**: Automatic ping/pong handling
- **Block Header Download**: Download and validate blockchain headers from peers
- **Compact Blocks (BIP 152)**:
  - Protocol version negotiation (supports v1/txid and v2/wtxid)
  - High-bandwidth and low-bandwidth modes
  - SipHash-2-4 shortID calculation with proper byte ordering
  - Mempool matching for bandwidth optimization
  - Differential encoding for prefilled transaction indexes
  - Automatic fallback to full transaction request
  - Successfully tested on mainnet with 25-60% bandwidth savings
  - Handles coinbase transactions correctly (arbitrary data in scriptSig)

### Merkle Trees & SPV (`internal/encoding`, `internal/network`)
- **Merkle Tree Construction**: Build complete merkle trees from transaction hashes
- **Merkle Root Calculation**: Recursive parent level computation with odd-node duplication
- **Tree Navigation**: Cursor-based tree traversal (up, left, right)
- **Merkle Block Parsing**: Parse merkleblock messages with partial merkle trees (BIP 37)
- **Merkle Proof Validation**: Reconstruct merkle trees from partial data and verify against block header
- **Bit Field Handling**: Convert compact flag bytes to bit arrays for tree reconstruction
- **Flag Bit Traversal**: Navigate merkle tree using flag bits to identify included transactions
- **Coinbase Transaction Handling**: Proper handling of coinbase-only matches (nodes don't send coinbase txs)

### Bloom Filters (`internal/network`)
- **Bloom Filter Creation**: Probabilistic data structure with configurable size and hash functions
- **MurmurHash3 Implementation**: BIP 37-compliant hashing for bloom filter population
- **Multi-Pattern Matching**: Support for addresses, transaction IDs, outpoints, and arbitrary data
- **Filter Transmission**: FilterLoad message creation and serialization
- **SPV Client**: Complete simplified payment verification implementation
  - Connect to BIP 37-enabled nodes
  - Filter transactions by address, txid, or outpoints
  - Receive and validate merkleblocks
  - Verify transactions without downloading full blockchain
  - Successfully tested against Bitcoin mainnet (found historic pizza transaction!)

### Mempool & ShortIDs (`internal/mempool`)
- **Transaction Pool**: In-memory transaction storage indexed by txid
- **Thread-Safe Operations**: Concurrent access with mutex protection
- **SipHash-2-4 Implementation**: Fast keyed hash function for shortID calculation
- **ShortID Matching**: Match compact block shortIDs to mempool transactions
  - Correct byte order handling (internal vs display order)
  - Support for both txid (v1) and wtxid (v2) matching
  - Efficient O(1) lookup using hash maps
- **Key Derivation**: Calculate SipHash keys from block header + nonce

## Example Usage

### Verifying SegWit Transactions

```go
// Create transaction fetcher
fetcher := transactions.NewTxFetcher()

// Fetch a real mainnet P2WPKH (native SegWit) transaction
txId := "7f5186d1b8d31fc8f083d51864a2a775ce25bd41a87e7ff4622ebbdc9cffe39e"
tx, err := fetcher.Fetch(txId, false, false)  // mainnet, use cache
if err != nil {
    panic(err)
}

fmt.Printf("Transaction ID: %s\n", txId)
fmt.Printf("Is SegWit: %v\n", tx.IsSegwit)
fmt.Printf("Number of inputs: %d\n", len(tx.Inputs))

// Check witness data
for i, input := range tx.Inputs {
    fmt.Printf("Input %d witness items: %d\n", i, len(input.Witness))
    for j, item := range input.Witness {
        fmt.Printf("  Item %d: %d bytes\n", j, len(item))
    }
}

// Verify the transaction
valid, err := tx.Verify()
if err != nil {
    panic(err)
}
fmt.Printf("Transaction valid: %v\n", valid)  // true!
```

### Verifying Nested SegWit (P2SH-Wrapped)

```go
// Fetch a P2SH-P2WPKH transaction
txId := "c586389e5e4b3acb9d6c8be1c19ae8ab2795397633176f5a6442a261bbdefc3a"
tx, err := fetcher.Fetch(txId, false, false)
if err != nil {
    panic(err)
}

// The scriptPubKey is P2SH, but the scriptSig contains a witness program
// The script engine automatically detects and handles the nested SegWit structure

valid, err := tx.Verify()
fmt.Printf("Nested SegWit valid: %v\n", valid)  // true!
```

### Generating Keys and Addresses

```go
import (
    "go-bitcoin/internal/address"
    "go-bitcoin/internal/keys"
)

// Create a private key from a secret
secret := big.NewInt(0xdeadbeef54321)
privateKey := keys.NewPrivateKey(secret)

// Generate public key
publicKey := privateKey.PublicKey()
pubkeyBytes := publicKey.Serialize(true) // compressed

// Generate P2PKH address (legacy)
p2pkhAddr, _ := address.FromPublicKey(pubkeyBytes, address.P2PKH, address.MAINNET)
fmt.Printf("P2PKH address: %s\n", p2pkhAddr.String) // Starts with "1"

// Generate P2WPKH address (native SegWit)
p2wpkhAddr, _ := address.FromPublicKey(pubkeyBytes, address.P2WPKH, address.MAINNET)
fmt.Printf("P2WPKH address: %s\n", p2wpkhAddr.String) // Starts with "bc1q"

// Generate testnet address
testnetAddr, _ := address.FromPublicKey(pubkeyBytes, address.P2PKH, address.TESTNET)
fmt.Printf("Testnet address: %s\n", testnetAddr.String) // Starts with "m" or "n"

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
    valid := combinedScript.Evaluate(z, input.Witness)
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
result := combined.Evaluate([]byte{}, nil)
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

### Using the SPV Client (Simplified Payment Verification)

```go
// Connect to a Bitcoin node that supports BIP 37 bloom filters
node, _ := network.NewSimpleNode("34.126.115.35", 8333, false, true)
defer node.Close()
node.Handshake()

// Watch for payments to a specific address
address := "17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ"
h160, _ := encoding.DecodeBase58(address)

// Create bloom filter
bf := network.NewBloomFilter(30, 5, 90210)
bf.Add(h160)  // Add address to filter

// Send bloom filter to node
filterload := &network.FilterLoadMessage{
    Filter: &bf,
    Flag:   byte(network.BLOOM_UPDATE_ALL),
}
node.Send(filterload)

// Request headers starting from a specific block
startBlock := getBlockHash(57042)  // Block before transaction
getheaders := network.NewGetHeadersMessage(70015, [][32]byte{startBlock}, nil)
node.Send(&getheaders)

// Receive headers
headersEnv, _ := node.Receive("headers")
headers, _ := network.ParseHeadersMessage(bytes.NewReader(headersEnv.Payload))

// Request filtered blocks
getdata := network.NewGetDataMessage()
for _, block := range headers.Blocks {
    blockHash, _ := block.Hash()
    var hash [32]byte
    copy(hash[:], blockHash)
    getdata.AddData(network.DATA_TYPE_FILTERED_BLOCK, hash)
}
node.Send(&getdata)

// Process merkleblocks and transactions
for {
    mbEnv, _ := node.Receive("merkleblock")
    mb, _ := network.ParseMerkleBlock(bytes.NewReader(mbEnv.Payload))

    if !mb.IsValid() {
        continue  // Invalid merkle proof
    }

    // Receive matching transactions
    for i := 0; i < mb.NumHashes; i++ {
        txEnv, err := node.Receive("tx")
        if err != nil {
            break  // Coinbase transaction (not sent)
        }

        tx, _ := transactions.ParseTransaction(bytes.NewReader(txEnv.Payload))

        // Check if transaction pays to our address
        for j, output := range tx.Outputs {
            addrObj, _ := output.ScriptPubKey.AddressV2(address.MAINNET)
            if addrObj.String == address {
                fmt.Printf("Found payment: %d satoshis\n", output.Amount)
                return
            }
        }
    }
}
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
    │   ├── hash.go              # Hash256, Hash160, MurmurHash3
    │   ├── varints.go           # Variable-length integer encoding
    │   ├── merkle.go            # Merkle tree construction and navigation
    │   └── merkle_test.go       # Merkle tree tests
    ├── keys/
    │   └── keys.go              # Private/public key management
    ├── script/
    │   ├── script.go            # Bitcoin Script parsing and serialization
    │   ├── scriptengine.go      # Script execution engine and opcodes
    │   └── exercise_test.go     # Script test cases (arithmetic, SHA-1 collision)
    ├── transactions/
    │   ├── transaction.go       # Transaction structure and SigHash
    │   ├── txinputs.go          # TxIn and TxOut types
    │   ├── fetch.go             # Transaction fetching with SegWit support
    │   ├── segwit_test.go       # SegWit transaction verification tests
    │   ├── bip143_test.go       # BIP 143 sighash test vectors
    │   └── bip143_manual_test.go # Manual BIP 143 preimage construction
    ├── block/
    │   └── block.go             # Block header parsing and proof of work
    ├── network/
    │   ├── node.go              # P2P node with concurrent message handling
    │   ├── network.go           # Network envelope parsing
    │   ├── version.go           # Version message with NODE_WITNESS flag
    │   ├── verack.go            # Verack message
    │   ├── pong.go              # Pong message
    │   ├── getheaders.go        # GetHeaders and Headers messages
    │   ├── bloomfilter.go       # Bloom filter creation and FilterLoad message
    │   ├── getdata.go           # GetData message (supports MSG_WITNESS_TX)
    │   ├── merkleblock.go       # MerkleBlock parsing and validation
    │   ├── compact.go           # BIP 152 compact block messages
    │   ├── compact_test.go      # Compact block integration test (mainnet)
    │   ├── generic.go           # Generic message types
    │   ├── bloom_test.go        # Bloom filter tests
    │   └── spv_test.go          # Full SPV client integration test
    └── mempool/
        ├── mempool.go           # Transaction pool and shortID matching
        ├── shortid.go           # SipHash-2-4 implementation
        └── shortid_test.go      # ShortID calculation tests
```

## Architecture Highlights

### Interface-Based Design
The networking layer uses clean interfaces for extensibility:
```go
type Message interface {
    Serialize() ([]byte, error)
    Command() string
}
```
All message types implement this interface, making it trivial to add new message types.

### Concurrent Network I/O
The `SimpleNode` uses three concurrent goroutines with channels for non-blocking I/O:
- `readLoop()` - Reads from network socket
- `sendLoop()` - Writes to network socket
- `messageLoop()` - Routes messages to handlers

### Dynamic Script Pattern Detection
The script engine detects P2SH, P2WPKH, and P2WSH patterns at runtime during execution, matching Bitcoin Core's approach. When a pattern is detected, the engine dynamically injects additional commands into the execution queue.

### BIP 143 Sighash Caching
Transaction type caches expensive hash calculations (`hashPrevouts`, `hashSequence`, `hashOutputs`) as private fields, providing significant performance optimization for multi-input transactions.

### Type Safety
Go's strong typing prevents entire classes of bugs:
- Compile-time validation of message implementations
- Type aliases for clarity (e.g., `MagicNum = uint32`)
- Clear distinction between data and opcodes in `ScriptCommand`

## Implementation Notes

- Uses Go's `math/big.Int` for arbitrary-precision arithmetic (256-bit operations)
- Cryptographically secure random number generation via `crypto/rand`
- All operations use correct byte order (Bitcoin standard)
- Follows idiomatic Go patterns (composition over inheritance)
- **Validates real Bitcoin transactions** from the blockchain using full Script evaluation
- **Connects to Bitcoin mainnet** and downloads/validates block headers
- Implements Bitcoin's legacy P2PKH (Pay-to-Public-Key-Hash) format
- Full SegWit support: P2WPKH, P2WSH, and nested SegWit (P2SH-wrapped)
- Stack-based Script VM with comprehensive opcode support
- Concurrent P2P networking with goroutines and channels
- RIPEMD-160 via `golang.org/x/crypto` (legacy hash, required for Bitcoin)
- **BIP 152 Compact Blocks**: Bandwidth optimization with mempool matching
  - Proper byte order handling (internal vs display order for txid/wtxid)
  - Coinbase transaction handling (arbitrary data in scriptSig)
  - Successfully tested on mainnet achieving 25-60% bandwidth savings
- Comprehensive test suite including:
  - Real mainnet SegWit transaction verification
  - Official BIP 143 test vectors
  - SHA-1 collision detection (SHAttered attack)
  - SPV client with bloom filters
  - Compact block negotiation and reconstruction with live mainnet nodes

## Standards Implemented

- **SEC (Standards for Efficient Cryptography)**: Public key serialization
- **DER (Distinguished Encoding Rules)**: Signature serialization
- **Base58Check**: Address encoding with checksum
- **WIF (Wallet Import Format)**: Private key serialization
- **BIP-13**: Pay-to-Script-Hash (P2SH) address format
- **BIP-16**: Pay-to-Script-Hash execution semantics
- **BIP-37**: Connection Bloom filtering (merkleblock parsing for SPV)
- **BIP-62**: Low-S signature enforcement for transaction malleability prevention
- **BIP-141**: Segregated Witness (consensus layer)
- **BIP-143**: Transaction signature verification for version 0 witness program
- **BIP-144**: Peer services for Segregated Witness
- **BIP-152**: Compact Block Relay (bandwidth optimization)

## Test Coverage

All major components have comprehensive test coverage with real blockchain data:

```bash
# Run SegWit transaction verification tests (real mainnet transactions)
go test -v ./internal/transactions/ -run TestP2wpkh
go test -v ./internal/transactions/ -run TestP2wsh
go test -v ./internal/transactions/ -run TestNestedSegwit

# Run BIP 143 sighash tests (official test vectors)
go test -v ./internal/transactions/ -run TestBIP143

# Run SPV client test (finds the Bitcoin Pizza transaction!)
go test -v ./internal/network/ -run TestSPVFlow

# Run Script tests (arithmetic, SHA-1 collision, number encoding)
go test -v ./internal/script/

# Run Merkle tree tests (tree construction, navigation)
go test -v ./internal/encoding/ -run TestMerkle

# Run bloom filter tests
go test -v ./internal/network/ -run TestBloom

# Run compact block tests (requires mainnet connection, can take 10-20 minutes)
go test -v ./internal/network/ -run TestCompactBlock -timeout 20m

# Run all tests
go test ./...
```

## Known Limitations

While this implementation successfully completes all chapters of Programming Bitcoin, there are some features present in production implementations that are not included:

- **RFC 6979 deterministic signatures** - Uses crypto/rand instead of deterministic k generation
- **Timelock opcodes** - OP_CHECKLOCKTIMEVERIFY (BIP 65) and OP_CHECKSEQUENCEVERIFY (BIP 112) not implemented
- **Additional opcodes** - ~50 opcodes not yet implemented (OP_OVER, OP_PICK, OP_ROLL, OP_MIN, OP_MAX, etc.)
- **Taproot** - Witness v1 (BIP 341, 342) not implemented

## Development

This is a learning project following "Programming Bitcoin" by Jimmy Song. The goal is to understand Bitcoin's cryptographic foundations by implementing them from scratch.

**⚠️ For production use**, consider battle-tested libraries like:
- `github.com/btcsuite/btcd` - Full Bitcoin implementation
- `github.com/btcsuite/btcutil` - Bitcoin utility functions

```bash
# Connect to Bitcoin mainnet and download block headers
go run main.go

# Dependencies
go get golang.org/x/crypto/ripemd160
```

## Resources

- Book: [Programming Bitcoin](https://programmingbitcoin.com/) by Jimmy Song
- [Bitcoin Developer Reference](https://developer.bitcoin.org/reference/)
- [BIPs (Bitcoin Improvement Proposals)](https://github.com/bitcoin/bips)
- [SEC Format Specification](https://www.secg.org/sec1-v2.pdf)
- [Base58Check encoding](https://en.bitcoin.it/wiki/Base58Check_encoding)
- [BIP 141: Segregated Witness](https://github.com/bitcoin/bips/blob/master/bip-0141.mediawiki)
- [BIP 143: Transaction Signature Verification for Version 0 Witness Program](https://github.com/bitcoin/bips/blob/master/bip-0143.mediawiki)
- [BIP 144: Peer Services](https://github.com/bitcoin/bips/blob/master/bip-0144.mediawiki)

## Acknowledgments

This implementation follows the excellent "Programming Bitcoin" book by Jimmy Song. The hands-on approach of building Bitcoin from first principles provides deep understanding of the protocol's cryptographic foundations.

**Key Achievement**: Successfully verifies real SegWit transactions from Bitcoin mainnet, including native P2WPKH, P2WSH, and nested SegWit (P2SH-wrapped) transactions. All BIP 143 test vectors pass.
