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
