# Architecture Decision Records (ADR)

This document captures the key architectural decisions made during the design and implementation of the append-only event store, along with their context, rationale, and tradeoffs.

## Table of Contents

1. [ADR-001: Zero External Dependencies](#adr-001-zero-external-dependencies)
2. [ADR-002: JSON for Event Serialization](#adr-002-json-for-event-serialization)
3. [ADR-003: Single Log File vs Per-Stream Files](#adr-003-single-log-file-vs-per-stream-files)
4. [ADR-004: Length-Prefixed Binary Format](#adr-004-length-prefixed-binary-format)
5. [ADR-005: In-Memory Index (Not Persisted)](#adr-005-in-memory-index-not-persisted)
6. [ADR-006: Sync After Every Write](#adr-006-sync-after-every-write)
7. [ADR-007: UUID for Event IDs](#adr-007-uuid-for-event-ids)
8. [ADR-008: Money as Integer Cents](#adr-008-money-as-integer-cents)
9. [ADR-009: Single-Process Only (No File Locking)](#adr-009-single-process-only-no-file-locking)
10. [ADR-010: Mutex-Based Concurrency](#adr-010-mutex-based-concurrency)
11. [ADR-011: Validation in Commands, Not Aggregates](#adr-011-validation-in-commands-not-aggregates)
12. [ADR-012: No Snapshots](#adr-012-no-snapshots)

---

## ADR-001: Zero External Dependencies

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Event stores can leverage existing libraries for serialization (Protobuf, MessagePack), logging, testing, and storage. However, dependencies increase complexity and make the codebase harder to understand as a learning resource.

### Decision

Use **only Go's standard library** with no external dependencies.

### Rationale

**Pros:**
- **Simplicity**: Easier to understand without learning third-party APIs
- **Portability**: Works anywhere Go works, no dependency management
- **Educational**: Forces understanding of low-level concepts (file I/O, concurrency, serialization)
- **Maintainability**: No breaking changes from dependency updates
- **Build Speed**: Faster compilation without external packages

**Cons:**
- **Reinventing Wheels**: Must implement features available in libraries
- **Less Performant**: stdlib solutions may be slower than optimized libraries
- **More Code**: More lines to achieve same functionality

### Consequences

- Must implement own UUID generation (using `crypto/rand`)
- Use JSON from stdlib instead of Protobuf or MessagePack
- Implement own file format instead of using database libraries
- All testing with stdlib `testing` package only

---

## ADR-002: JSON for Event Serialization

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Events need to be serialized for storage. Options include:
- JSON (human-readable, verbose)
- Protobuf (compact, requires schema)
- MessagePack (compact, binary)
- Go's gob encoding (Go-specific)

### Decision

Use **JSON** for event serialization.

### Rationale

**Pros:**
- **Human-Readable**: Can inspect `events.log` with `cat` or text editors
- **Debuggable**: Easy to troubleshoot issues by reading raw files
- **Language-Agnostic**: Other languages can read events
- **Schema-Flexible**: No schema compilation required
- **Stdlib Support**: Native `encoding/json` package

**Cons:**
- **Larger Size**: ~2-3x larger than binary formats
- **Slower**: Parsing overhead compared to binary
- **Type Safety**: Weak typing can lead to runtime errors

### Consequences

- Event log files are larger (acceptable for <10K events)
- Events can be inspected with standard text tools
- JSON validation required on append
- Must use `json.RawMessage` to preserve exact event data

---

## ADR-003: Single Log File vs Per-Stream Files

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Events can be stored in:
1. **Single log file**: All streams in one `events.log`
2. **Per-stream files**: Each stream gets its own file (e.g., `account-123.log`)
3. **Partitioned logs**: Multiple log files partitioned by time or size

### Decision

Use a **single log file** (`events.log`) for all events.

### Rationale

**Pros:**
- **Simpler Implementation**: One file handle, one write path
- **Global Ordering**: Natural total order across all streams
- **Easier Replay**: Single sequential read for all events
- **Fewer File Handles**: No OS limits on open files

**Cons:**
- **No Parallel Reads**: Can't read different streams concurrently from disk
- **Large File**: Single file grows indefinitely
- **No Stream Isolation**: Can't delete one stream's data independently

**Alternatives Considered:**

| Approach | Pros | Cons |
|----------|------|------|
| Per-stream files | Parallel reads, stream isolation | Complex file management, no global order |
| Partitioned logs | Bounded file sizes | Complex partitioning logic |

### Consequences

- All events written to single `data/events.log`
- Global sequence number is naturally the file position
- Cannot delete individual streams without rewriting entire log
- Index needed to find stream events (O(n) scan otherwise)

---

## ADR-004: Length-Prefixed Binary Format

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Within the log file, individual events need delimiters. Options:
1. **Length-prefixed**: `[4-byte length][JSON data]`
2. **Newline-delimited**: `{...}\n{...}\n`
3. **Custom delimiter**: `{...}|||{...}`

### Decision

Use **length-prefixed format**: 4-byte length header + JSON payload.

### Rationale

**Pros:**
- **Robust**: Works even if JSON contains newlines
- **Efficient**: Know exact bytes to read without parsing
- **Forward-Only**: Can scan file without backtracking
- **Standard**: Common in binary protocols (e.g., Protobuf, Kafka)

**Cons:**
- **Not Line-Oriented**: Can't use `grep` directly on log file
- **Binary Header**: Slightly less human-readable

**Alternatives Considered:**

| Format | Pros | Cons |
|--------|------|------|
| Newline-delimited | Simpler, `grep`-able | Fails with multiline JSON |
| Fixed-size records | Fast seeking | Wastes space, limits event size |

### Consequences

- File format: `[uint32 length][JSON bytes][uint32 length][JSON bytes]...`
- Must read length before reading payload
- Cannot use `cat events.log | grep` for debugging (use CLI instead)
- Reading is straightforward: loop reading length + payload

---

## ADR-005: In-Memory Index (Not Persisted)

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

To query events by stream efficiently, an index is needed. Options:
1. **In-memory only**: Rebuild from log on startup
2. **Persisted index**: Write index to separate file
3. **Embedded database**: Use BoltDB, SQLite, etc.

### Decision

Use an **in-memory index** that is rebuilt on every startup.

### Rationale

**Pros:**
- **Simpler**: No index persistence logic, no serialization
- **Self-Healing**: Index always consistent with log (can't desync)
- **No Corruption**: Index corruption impossible (just rebuild)
- **Fast Queries**: O(1) stream lookups once built

**Cons:**
- **Slow Startup**: Must read entire log on startup (O(n) events)
- **Memory Usage**: Index consumes RAM proportional to event count
- **No Persistence**: Rebuild cost paid on every restart

**Performance Impact:**
- 1,000 events: ~10ms rebuild
- 10,000 events: ~100ms rebuild
- 100,000 events: ~1s rebuild (approaching limit)

### Consequences

- Startup time grows linearly with event count
- Memory usage: ~100 bytes per event for index
- Suitable for <10,000 events (our target scale)
- Could add persisted index as future enhancement if needed

---

## ADR-006: Sync After Every Write

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

After writing events to the log file, we can:
1. **Sync immediately**: Call `file.Sync()` after each write
2. **Batch sync**: Sync periodically (e.g., every 100ms)
3. **OS buffering**: Rely on OS to flush (no explicit sync)

### Decision

Call **`file.Sync()` after every write** to guarantee durability.

### Rationale

**Pros:**
- **Durability**: No data loss on crash or power failure
- **Correctness**: Events are durable before returning to caller
- **Simpler**: No batching or background syncing logic
- **Predictable**: Every event either fully written or not

**Cons:**
- **Slow Writes**: ~1ms per sync on typical SSD (~1000 writes/sec max)
- **Throughput**: Can't batch for higher throughput
- **Latency**: Every write pays fsync cost

**Performance vs Correctness:**

| Approach | Throughput | Durability | Complexity |
|----------|------------|------------|------------|
| Sync every write | ~1K/sec | Perfect | Simple |
| Batch sync (100ms) | ~10K/sec | 100ms window | Medium |
| No sync | ~100K/sec | OS-dependent | Simple |

### Consequences

- Write throughput limited to ~1,000 events/sec
- Production systems would use batching for performance
- Acceptable for educational/demo purposes
- Every successful write is crash-safe

---

## ADR-007: UUID for Event IDs

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Events need unique identifiers. Options:
1. **UUID/GUID**: Globally unique random ID
2. **Auto-increment**: Sequential integer (1, 2, 3...)
3. **Timestamp + counter**: Hybrid approach

### Decision

Use **UUID (128-bit random hex string)** for event IDs.

### Rationale

**Pros:**
- **Distributed-Ready**: No coordination needed across processes
- **Globally Unique**: Collision probability negligible
- **Future-Proof**: Works if we add multi-process support later
- **Standard**: Well-understood, used by EventStoreDB, Kafka

**Cons:**
- **Larger**: 32 chars vs 8 chars for int64
- **Non-Sequential**: Can't sort events by ID
- **Randomness**: Need good entropy source (`crypto/rand`)

**Alternatives Considered:**

| ID Type | Size | Sortable | Distributed | Complexity |
|---------|------|----------|-------------|------------|
| UUID | 36 bytes | No | Yes | Low |
| Auto-increment | 8 bytes | Yes | No | Medium |
| ULID | 26 bytes | Yes | Yes | High |

### Consequences

- Event IDs are 32-character hex strings (e.g., `d936ad6dd740c18ea8744ec1cd4f5869`)
- Must use GlobalSequence for ordering, not ID
- IDs generated with `crypto/rand` (cryptographically secure)
- Could support distributed event generation in future

---

## ADR-008: Money as Integer Cents

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Bank account balances need to represent money. Options:
1. **Integer cents**: Store 10050 for $100.50
2. **float64**: Store 100.50 directly
3. **Decimal library**: Use third-party decimal type

### Decision

Store money as **integer cents** (e.g., 10050 = $100.50).

### Rationale

**Pros:**
- **Precision**: No floating-point rounding errors
- **Correctness**: Financial calculations always exact
- **Simple**: Standard `int` type, no special handling
- **Fast**: Integer arithmetic faster than float

**Cons:**
- **UI Conversion**: Must divide by 100 for display
- **Less Intuitive**: Users think in dollars, not cents
- **Range Limit**: int64 max = $92 quadrillion (acceptable)

**Example of Float Problem:**
```go
balance := 0.1 + 0.2  // equals 0.30000000000000004 (not 0.3!)
```

**Alternatives Considered:**

| Type | Precision | Performance | Dependencies |
|------|-----------|-------------|--------------|
| int cents | Exact | Fast | None |
| float64 | Approximate | Fast | None |
| decimal.Decimal | Exact | Slow | External lib |

### Consequences

- All amounts in cents (100 = $1.00)
- CLI displays as cents and dollars: `10000 cents ($100.00)`
- Deposit/withdraw amounts input as cents
- No floating-point arithmetic anywhere in business logic

---

## ADR-009: Single-Process Only (No File Locking)

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Event store could support:
1. **Single process**: Only one process accesses files
2. **Multi-process**: File locking to coordinate access
3. **Client-server**: Central server, multiple clients

### Decision

**Single-process only** - no file locking implemented.

### Rationale

**Pros:**
- **Simpler**: No lock files, no coordination logic
- **Fewer Edge Cases**: No deadlocks, stale locks, etc.
- **Educational Focus**: Teaches core concepts without distributed systems complexity
- **Sufficient**: Adequate for learning and demo purposes

**Cons:**
- **Limited**: Can't run multiple CLI commands concurrently
- **Production Blocker**: Real systems need multi-process support
- **No Protection**: Concurrent access causes corruption

**Alternatives Considered:**

| Approach | Complexity | Safety | Scalability |
|----------|------------|--------|-------------|
| No locking | Low | Low | Single process |
| File locks (flock) | Medium | Medium | Multi-process, single node |
| Client-server | High | High | Multi-node |

### Consequences

- Only one process can access event store at a time
- Multiple concurrent CLI commands will cause data corruption
- Documented as intentional limitation in README
- Could add file locking as future enhancement

---

## ADR-010: Mutex-Based Concurrency

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Within a single process, concurrent goroutines need coordination. Options:
1. **Mutex (sync.RWMutex)**: Lock-based synchronization
2. **Channels**: Message passing for coordination
3. **No synchronization**: Serial execution only

### Decision

Use **`sync.RWMutex`** for read/write concurrency control.

### Rationale

**Pros:**
- **Simple**: Standard Go pattern, well-understood
- **Read Concurrency**: Multiple readers can access simultaneously
- **Write Safety**: Exclusive write lock prevents races
- **Standard Library**: No external dependencies

**Cons:**
- **Lock Contention**: Heavy write load blocks readers
- **Deadlock Risk**: Requires careful lock ordering
- **Not Lock-Free**: Blocking vs non-blocking tradeoffs

**Concurrency Model:**
- **Reads** (GetStream, GetAllEvents): Shared lock (concurrent)
- **Writes** (Append): Exclusive lock (serialized)
- **Index**: RWMutex for concurrent reads
- **Storage**: Mutex for exclusive writes

### Consequences

- Multiple goroutines can read simultaneously
- Writes are serialized (one at a time)
- Verified with `go test -race` (no data races)
- Simple to understand and maintain

---

## ADR-011: Validation in Commands, Not Aggregates

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

Business rule validation can happen at:
1. **Command layer**: Validate before creating events
2. **Aggregate layer**: Validate during event application
3. **Both layers**: Redundant validation

### Decision

**Validate in commands** (e.g., `Deposit`, `Withdraw`), not in aggregates.

### Rationale

**Pros:**
- **Events are Facts**: Events represent what happened (already validated)
- **Faster Replay**: No validation overhead during event replay
- **Single Responsibility**: Commands validate, aggregates just apply
- **Idempotent Replay**: Same events always produce same state

**Cons:**
- **Trust Required**: Must trust that stored events are valid
- **Historical Issues**: Can't retroactively fix invalid events
- **Corruption Risk**: Direct event log manipulation bypasses validation

**Example:**
```go
// ✅ CORRECT: Validate in command
func Withdraw(store EventStore, accountID string, amount int) error {
    if amount <= 0 {
        return fmt.Errorf("amount must be positive")  // Validate HERE
    }
    account, _ := ReplayAccount(store, accountID)
    if account.Balance < amount {
        return fmt.Errorf("insufficient funds")  // Validate HERE
    }
    return store.Append(accountID, "MoneyWithdrawn", ...)
}

// ✅ CORRECT: No validation in aggregate
func (a *BankAccount) ApplyEvent(event *Event) error {
    switch event.EventType {
    case "MoneyWithdrawn":
        a.Balance -= data.Amount  // Just apply, don't validate
    }
}
```

### Consequences

- Commands enforce: amount > 0, balance >= withdrawal
- Aggregates trust all events are valid
- Invalid events cannot be created through API
- Direct log manipulation could create invalid events (documented risk)

---

## ADR-012: No Snapshots

**Status**: Accepted
**Date**: 2026-01-22
**Deciders**: Claude, Andrea Cadonna

### Context

To rebuild aggregate state, we can:
1. **Always replay from event 1**: Pure event sourcing
2. **Snapshots**: Periodic state checkpoints to speed up replay
3. **CQRS**: Separate read models (projections)

### Decision

**No snapshots** - always replay from the first event.

### Rationale

**Pros:**
- **Conceptually Pure**: Demonstrates event sourcing without shortcuts
- **Simpler**: No snapshot storage, no snapshot logic
- **Consistent**: Always the same code path (replay)
- **Educational**: Forces understanding of event replay

**Cons:**
- **Slow Replay**: Long streams (1000+ events) are slow to rebuild
- **Scalability Limit**: Not suitable for high-volume streams
- **Production Blocker**: Real systems need snapshots

**Performance Impact:**
- 10 events: <1ms replay
- 100 events: ~5ms replay
- 1,000 events: ~50ms replay (getting slow)
- 10,000 events: ~500ms replay (too slow for production)

**Alternatives Considered:**

| Approach | Replay Speed | Complexity | Correctness |
|----------|--------------|------------|-------------|
| No snapshots | O(n events) | Low | Simple |
| Periodic snapshots | O(n since snapshot) | Medium | Must validate |
| CQRS projections | O(1) | High | Eventually consistent |

### Consequences

- Every balance query replays all account events
- Suitable for streams with <100 events
- Could add snapshots as enhancement if needed
- Simple to understand and debug

---

## Summary of Tradeoffs

| Decision | Chose | Over | Why |
|----------|-------|------|-----|
| Dependencies | None | Libraries | Educational clarity |
| Serialization | JSON | Binary | Debuggability |
| File Structure | Single log | Per-stream | Simplicity |
| Format | Length-prefixed | Newline | Robustness |
| Index | In-memory | Persisted | Self-healing |
| Durability | Sync every write | Batching | Correctness |
| Event ID | UUID | Sequential | Distributed-ready |
| Money | Integer cents | Float | Precision |
| Processes | Single | Multi | Complexity |
| Concurrency | Mutex | Channels | Simplicity |
| Validation | Commands | Aggregates | Event purity |
| Performance | Snapshots omitted | Included | Educational focus |

---

## Future Considerations

These decisions are appropriate for an educational project (<10K events). Production systems would need:

1. **Snapshots**: For streams with 100+ events
2. **Batched Writes**: For throughput >1K events/sec
3. **Persisted Index**: For startup times <100ms with large logs
4. **Protobuf**: For compact storage (2-3x smaller)
5. **File Locking**: For multi-process safety
6. **Partitioned Logs**: For unbounded growth
7. **CQRS Projections**: For complex queries

The current design prioritizes **clarity and correctness** over performance, making it ideal for learning event sourcing fundamentals.

---

**Document Status**: Complete
**Last Updated**: 2026-01-23
**Review**: Before adding new features, revisit relevant ADRs to ensure consistency
