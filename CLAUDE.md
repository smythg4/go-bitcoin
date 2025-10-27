# Project Status

## Recently Completed

### Bech32 Address Encoding (BIP 173) ✅
**Status**: Fully implemented and tested

Implemented complete bech32 encoding for native SegWit addresses:
- P2WPKH addresses (bc1q... for mainnet, tb1q... for testnet)
- P2WSH addresses (bc1q... with 32-byte witness programs)
- Full BIP 173 compliance with official test vectors
- Polymod checksum calculation
- 8-bit to 5-bit data conversion
- HRP (human-readable part) expansion

**Key Bug Fix**: Witness version is already 5-bit compatible (0-16), so it's added directly to the data array rather than being converted from 8-bit.

**Files**:
- `internal/address/bech32.go` - Core bech32 encoding implementation
- `internal/address/address.go` - Address type and generation functions
- `internal/address/address_test.go` - BIP 173 test vectors

### Address API Refactoring ✅
**Status**: Complete migration across codebase

Refactored address generation to use a unified `internal/address` package:

**Old API** (deprecated):
```go
addr := script.Address(testnet bool) string  // Returns string, boolean flags
addr := publicKey.Address(compressed, testnet bool) string
```

**New API** (current):
```go
addrObj, err := script.AddressV2(network address.Network) (*address.Address, error)
addrObj, err := address.FromPublicKey(pubkey []byte, addrType address.AddrType, network address.Network)
addrObj, err := address.FromHash160(hash160 []byte, addrType address.AddrType, network address.Network)
addrObj, err := address.FromWitnessProgram(version byte, program []byte, network address.Network)
```

**Benefits**:
- Type-safe `address.Network` enum (MAINNET, TESTNET) instead of boolean flags
- Supports all address types: P2PKH, P2SH, P2WPKH, P2WSH
- Centralized address logic in `internal/address` package
- Returns structured `Address` object with `.String` property
- Proper error handling

**Migration Complete**:
- ✅ All `script.Address()` calls replaced with `script.AddressV2()`
- ✅ Code examples in README.md updated
- ✅ No remaining deprecated API usage in codebase

# TODO

## Concurrency Improvements (High Priority)

### 1. Fix Unbounded Goroutine Spawning (CRITICAL)
**Issue**: `node.go:231` spawns unlimited goroutines for message handlers
```go
go handler(env)  // ⚠️ Could spawn 1000+ concurrent goroutines under load
```

**Solution**: Implement worker pool pattern with semaphore:
```go
type SimpleNode struct {
    // ...
    workerPool chan struct{} // Limit concurrent handlers
}

// In NewSimpleNode:
sn.workerPool = make(chan struct{}, 10) // Max 10 concurrent handlers

// In messageLoop:
if handler, ok := sn.handlers[env.Command]; ok {
    sn.workerPool <- struct{}{} // Acquire slot (blocks if full)
    go func(e NetworkEnvelope) {
        defer func() { <-sn.workerPool }() // Release slot
        handler(e)
    }(env)
}
```

**Impact**: Prevents memory exhaustion during traffic bursts

### 2. Add Context-Based Cancellation
**Issue**: Using bare `done` channel instead of modern `context.Context`

**Solution**: Migrate to context:
```go
func NewSimpleNodeWithContext(ctx context.Context, ...) (*SimpleNode, error)
func (sn *SimpleNode) ReceiveWithTimeout(ctx context.Context, commands []string, timeout time.Duration)
```

**Benefits**:
- Hierarchical cancellation (shutdown cascades properly)
- Deadline propagation
- Standard Go pattern for 2024+

### 3. Propagate Goroutine Errors
**Issue**: `readLoop`, `sendLoop`, `messageLoop` die silently on errors (node.go:134-138)

**Solution**: Use `errgroup` pattern:
```go
import "golang.org/x/sync/errgroup"

type SimpleNode struct {
    // ...
    eg     *errgroup.Group
    egCtx  context.Context
}

// In loops:
if err != nil {
    return err // errgroup captures and returns first error
}

// Callers can check:
if err := node.Wait(); err != nil {
    log.Printf("Node failed: %v", err)
}
```

**Impact**: Easier debugging, proper error handling

### 4. Add Observability
**Issue**: No metrics for dropped messages, queue depths, goroutine counts

**Solution**: Add basic metrics:
```go
type NodeMetrics struct {
    MessagesDropped   atomic.Uint64
    ActiveGoroutines  atomic.Int32
    QueueDepth        map[string]func() int
}

func (sn *SimpleNode) Metrics() NodeMetrics
```

**Impact**: Visibility into production issues

## Network Layer Improvements

### ReceiveWithTimeout - Support Multiple Commands
Update `SimpleNode.ReceiveWithTimeout()` to accept multiple commands and wait on any of them:

```go
func (sn *SimpleNode) ReceiveWithTimeout(commands []string, timeout time.Duration) (NetworkEnvelope, error)
```

Usage:
```go
env, err := node.ReceiveWithTimeout([]string{"tx", "cmpctblock"}, 20*time.Minute)
if err != nil {
    // handle error
}

switch env.Command {
case "tx":
    // handle transaction
case "cmpctblock":
    // handle compact block
}
```

Implementation approach:
- Use `reflect.Select()` to dynamically wait on multiple channels
- Return the envelope (caller checks `env.Command` to determine type)
- Maintains clean API without needing to return which command matched
