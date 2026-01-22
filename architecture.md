# Event Store Architecture

## Overview

An append-only event store built in Go with zero external dependencies. The system persists immutable events to disk, maintains an in-memory index for fast queries, and demonstrates event sourcing principles through a bank account domain example.

## Core Building Blocks

1. **Event** - The fundamental data structure representing something that happened
2. **DiskStorage** - Append-only log file for persistent storage
3. **Index** - In-memory data structure for fast event lookups
4. **EventStore** - High-level coordinator that orchestrates storage and indexing
5. **Domain Example (Bank)** - Demonstrates event sourcing with a practical use case
6. **CLI** - Command-line interface for interaction

---

## 1. Event

### Purpose
Immutable record of a state change that occurred in the system. Once written, never modified or deleted.

### Responsibilities
- Represent a domain event with all necessary context
- Serialize to/from JSON for storage
- Generate unique identifiers
- Track ordering (version within stream, global sequence)

### Key Attributes
- **ID**: Globally unique identifier (UUID)
- **StreamID**: Logical grouping (e.g., account-123)
- **EventType**: What happened (e.g., "AccountOpened")
- **Data**: JSON payload with event details
- **Timestamp**: When the event was created
- **Version**: Position within the stream (1, 2, 3...)
- **GlobalSequence**: Position in the global event log (1, 2, 3...)

### Relationships
- Written by: **EventStore**
- Stored by: **DiskStorage**
- Indexed by: **Index**
- Replayed by: **Domain Aggregates**

### Design Decisions
- **UUID vs Sequential ID**: UUID chosen for distributed-system readiness, no coordination needed
- **JSON Data**: Human-readable, debuggable, language-agnostic (vs Protobuf)
- **Immutability**: All fields set at creation, no setters

---

## 2. DiskStorage

### Purpose
Provides durable, append-only persistence of events to disk. Guarantees that events are never lost after successful write.

### Responsibilities
- Append events to a single log file
- Read all events from disk (for rebuilding index on startup)
- Ensure durability through sync operations
- Handle file I/O errors gracefully

### Storage Format
```
[4 bytes: length][N bytes: JSON event]
[4 bytes: length][N bytes: JSON event]
...
```

Each entry is length-prefixed to enable sequential reading without parsing JSON first.

### Operations
- **Append(event)**: Write event to end of log, sync to disk
- **ReadAll()**: Read all events from log file
- **Close()**: Flush and close file handle

### Relationships
- Used by: **EventStore**
- Stores: **Event** instances
- Rebuilt by: **Index** on startup

### Design Decisions
- **Single Log File**: One `events.log` file instead of per-stream files
  - Pro: Simpler implementation, preserves global ordering
  - Con: All streams in one file (acceptable for learning project)
- **Sync After Every Write**: Prioritize durability over performance
  - Pro: No data loss on crash
  - Con: Slower writes (acceptable for this scale)
- **Length-Prefixed Format**: Enables forward-only reading without JSON parsing
  - Alternative: Newline-delimited JSON (simpler but fragile with multiline JSON)

---

## 3. Index

### Purpose
Provides fast in-memory lookups for events without scanning the entire log file.

### Responsibilities
- Map stream IDs to their event sequences
- Enable O(1) stream lookups instead of O(n) log scans
- Rebuild itself on startup from all events
- Stay synchronized with new appends

### Data Structure
```
streamIndex: map[StreamID][]GlobalSequence

Example:
{
  "account-123": [1, 5, 8, 15],    // Events at global positions 1, 5, 8, 15
  "account-456": [2, 9, 12]
}
```

Optional enhancement:
```
offsetIndex: map[GlobalSequence]FileOffset
```
(Enables seeking to specific events without reading entire log)

### Operations
- **BuildFromEvents(events)**: Construct index from event list
- **GetStreamSequences(streamID)**: Return global sequences for a stream
- **AddEvent(event)**: Incrementally update index on new append

### Relationships
- Built by: **EventStore** on startup
- Queried by: **EventStore** for stream reads
- Updated by: **EventStore** on every append

### Design Decisions
- **Rebuild on Startup**: Index is not persisted to disk
  - Pro: Simpler implementation, self-healing (always consistent with log)
  - Con: Slower startup (acceptable for <10K events)
  - Alternative: Persist index to separate file (more complexity, risk of desync)
- **In-Memory Only**: Assumes log fits in memory for queries
  - Limitation: Not suitable for massive event logs (millions of events)

---

## 4. EventStore

### Purpose
High-level API that coordinates storage, indexing, and queries. This is the main interface for application code.

### Responsibilities
- Initialize storage and index on startup
- Append events with proper versioning and sequencing
- Query events by stream or globally
- Coordinate concurrency control
- Provide clean API abstraction over low-level details

### Operations
- **NewEventStore(dataDir)**: Initialize, load log, build index
- **Append(streamID, eventType, data)**: Write new event
- **GetStream(streamID)**: Return all events in a stream (in order)
- **GetAllEvents()**: Return all events globally (in order)
- **Close()**: Cleanup resources

### Internal Workflow: Append
1. Acquire write lock
2. Calculate next GlobalSequence (increment last)
3. Calculate next Version for stream (from index)
4. Create Event instance
5. Write to DiskStorage
6. Update Index
7. Release write lock

### Internal Workflow: GetStream
1. Acquire read lock
2. Query Index for stream's global sequences
3. Load events from storage at those positions
4. Release read lock
5. Return events in order

### Relationships
- Uses: **DiskStorage** for persistence
- Uses: **Index** for queries
- Creates: **Event** instances
- Used by: **CLI** and **Domain** packages

### Design Decisions
- **Single Transaction Per Append**: No batching
  - Pro: Simpler implementation, immediate durability
  - Con: Lower throughput (acceptable for learning project)
- **Concurrency Model**: Mutex-based synchronization
  - Write operations: Exclusive lock (serialize all writes)
  - Read operations: Shared lock (concurrent reads allowed)
  - Limitation: Single-process only, not multi-process safe
- **Versioning Strategy**: Per-stream version numbering
  - Stream version starts at 1, increments with each event
  - Enables optimistic concurrency control (future enhancement)

---

## 5. Domain Example: Bank Account

### Purpose
Demonstrates event sourcing in practice with a concrete business domain. Shows how to:
- Model state changes as events
- Rebuild aggregate state from event history
- Enforce business rules using events

### Components

#### Events
- **AccountOpened**: `{accountID, owner}`
- **MoneyDeposited**: `{accountID, amount}`
- **MoneyWithdrawn**: `{accountID, amount}`

#### Aggregate: BankAccount
- **Current State**: `{accountID, owner, balance}`
- **ApplyEvent(event)**: Mutate state based on event type

#### Commands (Business Logic)
- **OpenAccount(accountID, owner)**: Create new account
- **Deposit(accountID, amount)**: Add money (validation: amount > 0)
- **Withdraw(accountID, amount)**: Remove money (validation: amount > 0 AND balance >= amount)
- **GetBalance(accountID)**: Replay events to calculate current balance

### Event Sourcing Flow
1. Load all events for stream (account ID)
2. Create empty aggregate
3. Apply each event in order
4. Result: current state of the account

### Relationships
- Uses: **EventStore** to append and read events
- Demonstrates: Core event sourcing pattern
- Used by: **CLI** for bank commands

### Design Decisions
- **Money as Cents (int)**: Avoid floating-point precision issues
  - $100.50 stored as 10050 cents
  - Alternative: Decimal library (adds dependency)
- **Validation in Commands**: Check business rules before appending events
  - Events represent facts, should always be valid
  - No retroactive validation during replay
- **No State Persistence**: Aggregate state always rebuilt from events
  - Shows pure event sourcing (no shortcuts)
  - Future enhancement: Snapshots for performance

---

## 6. CLI (Command-Line Interface)

### Purpose
Provides human-friendly way to interact with the event store. Used for testing, debugging, and demonstration.

### Commands

#### Generic Event Operations
- `eventstore append <stream-id> <event-type> <json-data>`
- `eventstore get-stream <stream-id>`
- `eventstore get-all`

#### Bank Domain Commands
- `eventstore bank open <account-id> <owner>`
- `eventstore bank deposit <account-id> <amount-cents>`
- `eventstore bank withdraw <account-id> <amount-cents>`
- `eventstore bank balance <account-id>`

### Relationships
- Initializes: **EventStore** with data directory
- Invokes: **Bank domain commands**
- Formats: Output for human readability

### Design Decisions
- **Human-Readable Output**: Pretty-printed JSON for events, formatted balance for bank
- **Amount in Cents**: CLI accepts cents (10050 = $100.50) to avoid decimal confusion
- **Data Directory**: Configurable via flag or environment variable
  - Default: `./data/`

---

## System Interactions

### Startup Flow
1. CLI initializes EventStore with data directory
2. EventStore creates/opens DiskStorage (events.log)
3. DiskStorage reads all events from log file
4. EventStore builds Index from loaded events
5. System ready for commands

### Write Flow (Append Event)
```
CLI Command
  → EventStore.Append()
    → Calculate Version (from Index)
    → Create Event instance
    → DiskStorage.Append() [disk write + sync]
    → Index.AddEvent() [update in-memory map]
  ← Return success
```

### Read Flow (Get Stream)
```
CLI Command
  → EventStore.GetStream(streamID)
    → Index.GetStreamSequences(streamID)
    → DiskStorage.ReadAll() [load from disk]
    → Filter events by sequences
  ← Return ordered events
```

### Replay Flow (Get Balance)
```
CLI Command
  → Bank.GetBalance(accountID)
    → EventStore.GetStream(accountID)
    → Create empty BankAccount
    → For each event:
      → BankAccount.ApplyEvent(event)
    → Return current balance
```

---

## Concurrency Model

### Protection Mechanisms
- **DiskStorage**: Mutex protects file writes (single writer at a time)
- **Index**: RWMutex allows concurrent reads, exclusive writes
- **EventStore**: Coordinates locks between storage and index

### Concurrency Guarantees
- ✅ Multiple readers can query simultaneously
- ✅ Writes are serialized (one at a time)
- ✅ Readers see consistent snapshots
- ❌ Multi-process safety NOT guaranteed
  - No file locking mechanism
  - Document as single-process limitation

### Future Enhancement: Optimistic Locking
```
AppendWithVersion(streamID, expectedVersion, event)
  → Check current stream version
  → If version matches: append
  → If version differs: return ConcurrencyError
```

---

## Project Structure

```
append-only-event-store/
├── cmd/
│   └── eventstore/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── eventstore/
│   │   ├── event.go             # Event struct + methods
│   │   ├── storage.go           # DiskStorage implementation
│   │   ├── index.go             # Index implementation
│   │   └── store.go             # EventStore implementation
│   └── domain/
│       └── bank/
│           ├── events.go        # Bank event types
│           ├── account.go       # BankAccount aggregate
│           └── commands.go      # Business logic
├── data/                        # Event store data (gitignored)
│   └── events.log
├── architecture.md              # This document
├── go.mod
└── README.md
```

---

## Key Design Tradeoffs

| Aspect | Choice | Alternative | Rationale |
|--------|--------|-------------|-----------|
| **Event ID** | UUID string | Auto-increment int64 | Future-proof for distributed systems |
| **Serialization** | JSON | Protobuf/Binary | Human-readable, debuggable, no dependencies |
| **File Structure** | Single log | Per-stream files | Simpler, preserves global order |
| **Durability** | Sync every write | Batch + periodic sync | Correctness over speed |
| **Index Persistence** | Rebuild on startup | Persist to disk | Simpler, self-healing |
| **Concurrency** | Mutex (single process) | File locks (multi-process) | Adequate for learning project |
| **Stream Queries** | Load all + filter | Seek by offset | Simpler implementation first |

---

## Limitations (By Design)

1. **Single Process Only**: No multi-process coordination (file locks not implemented)
2. **Scale**: Optimized for <10K events, rebuilding index on every startup
3. **No Deletions**: True append-only (no GDPR "right to be forgotten")
4. **No Snapshots**: Always replay from event 1 (could be slow for long streams)
5. **No Subscriptions**: Cannot notify on new events (polling only)
6. **No Schema Evolution**: Event structure changes require migration logic

---

## Future Enhancements (Out of Scope)

- **Snapshots**: Periodic state checkpoints to speed up replay
- **Projections**: Materialized views for common queries
- **Subscriptions**: Real-time event notifications
- **Optimistic Locking**: Prevent concurrent modification conflicts
- **Offset Index**: Seek to specific events without full scan
- **Multi-Process Safety**: File locking or SQLite backend
- **Event Upcasting**: Handle schema evolution gracefully

---


