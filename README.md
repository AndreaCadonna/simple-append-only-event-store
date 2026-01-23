# Append-Only Event Store

A simple, educational implementation of an append-only event store in Go with zero external dependencies. Demonstrates event sourcing principles through a practical bank account domain example.

**Status**: ✅ Complete - Ready for demonstration and learning

## Features

- **Pure Go Implementation**: Zero external dependencies (stdlib only)
- **Append-Only Log**: Immutable events persisted to disk with durability guarantees
- **In-Memory Indexing**: Fast O(1) stream lookups with automatic rebuild on startup
- **Event Sourcing**: Full demonstration of event replay and state reconstruction
- **Bank Domain Example**: Complete working example with business logic and validation
- **CLI Interface**: Command-line tool for all operations
- **Thread-Safe**: Concurrent reads with serialized writes, verified with race detector
- **Comprehensive Tests**: 75+ unit and integration tests, all passing

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/AndreaCadonna/simple-append-only-event-store.git
cd simple-append-only-event-store

# Build the CLI
go build -o eventstore ./cmd/eventstore

# Verify installation
./eventstore help
```

### Basic Usage

```bash
# Open a bank account
./eventstore bank open acc-alice "Alice Smith"

# Deposit money (amounts in cents)
./eventstore bank deposit acc-alice 10000  # $100.00

# Withdraw money
./eventstore bank withdraw acc-alice 2500  # $25.00

# Check balance
./eventstore bank balance acc-alice
# Output: Balance: 7500 cents ($75.00)

# View all events for an account
./eventstore get-stream acc-alice

# View all events in the store
./eventstore get-all
```

### Generic Event Operations

```bash
# Append a generic event
./eventstore append order-123 OrderCreated '{"item":"laptop","price":99900}'

# Query events by stream
./eventstore get-stream order-123

# Query all events
./eventstore get-all
```

## Architecture Overview

The system consists of five core components:

```
┌─────────────────────────────────────────────────┐
│                 CLI Interface                    │
└──────────────┬──────────────────────────────────┘
               │
               ├─────────────────────────┐
               │                         │
    ┌──────────▼────────┐    ┌──────────▼──────────┐
    │   EventStore      │    │   Bank Commands     │
    │   (Coordinator)   │    │   (Domain Logic)    │
    └────┬────────┬─────┘    └─────────────────────┘
         │        │
    ┌────▼────┐  │
    │  Index  │  │
    │(In-Mem) │  │
    └─────────┘  │
                 │
            ┌────▼─────────┐
            │ DiskStorage  │
            │(Append-Only) │
            └──────────────┘
                 │
            [events.log]
```

### Key Design Decisions

- **Single Log File**: All events in one `events.log` file (simpler, preserves global ordering)
- **Length-Prefixed Format**: `[4-byte length][JSON event]` for efficient reading
- **JSON Serialization**: Human-readable, debuggable, no external dependencies
- **Index Rebuild**: Index reconstructed on startup (self-healing, always consistent)
- **Money as Cents**: Integer math avoids floating-point precision issues
- **Sync After Write**: Durability over speed (every write is flushed to disk)

See [architecture.md](architecture.md) for detailed design documentation.

## CLI Commands Reference

### Bank Commands

```bash
# Open a new account
eventstore bank open <accountId> <owner>

# Deposit money (amount in cents)
eventstore bank deposit <accountId> <amount>

# Withdraw money (amount in cents)
eventstore bank withdraw <accountId> <amount>

# Get account balance
eventstore bank balance <accountId>
```

### Generic Event Store Commands

```bash
# Append an event to a stream
eventstore append <streamId> <eventType> <jsonData>

# Get all events for a stream
eventstore get-stream <streamId>

# Get all events in the store
eventstore get-all
```

### Options

```bash
# Specify custom data directory
eventstore --data-dir=/path/to/data bank open acc-1 "Alice"

# Default data directory is ./data
```

## Using as a Library

You can use the event store directly in your Go code:

```go
package main

import (
    "fmt"
    "github.com/cadonna/append-only-event-store/pkg/eventstore"
    "github.com/cadonna/append-only-event-store/pkg/domain/bank"
)

func main() {
    // Create event store
    store, err := eventstore.NewEventStore("./data")
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Open a bank account
    _, err = bank.OpenAccount(store, "acc-123", "Alice Smith")
    if err != nil {
        panic(err)
    }

    // Deposit money
    _, err = bank.Deposit(store, "acc-123", 10000) // $100.00
    if err != nil {
        panic(err)
    }

    // Withdraw money
    _, err = bank.Withdraw(store, "acc-123", 2500) // $25.00
    if err != nil {
        panic(err)
    }

    // Get balance (via event replay)
    balance, err := bank.GetBalance(store, "acc-123")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Balance: %d cents ($%.2f)\n", balance, float64(balance)/100.0)
    // Output: Balance: 7500 cents ($75.00)
}
```

### Appending Generic Events

```go
// Append a custom event
event, err := store.Append("order-123", "OrderCreated", map[string]interface{}{
    "orderId": "order-123",
    "item":    "laptop",
    "price":   99900, // $999.00
})

// Query events by stream
events, err := store.GetStream("order-123")

// Get all events
allEvents, err := store.GetAllEvents()
```

### Creating Your Own Domain

```go
type OrderAggregate struct {
    OrderID  string
    Items    []string
    Total    int
    Status   string
    Version  int
}

func (o *OrderAggregate) ApplyEvent(event *eventstore.Event) error {
    switch event.EventType {
    case "OrderCreated":
        // Parse event data and update state
        o.Status = "created"
        o.Version++
    case "ItemAdded":
        // Handle item added
        o.Version++
    // ... more event types
    }
    return nil
}
```

## Testing

### Run All Tests

```bash
# Unit and integration tests
go test ./...

# With verbose output
go test -v ./...

# With race detector
go test -race ./...

# With coverage
go test -cover ./...
```

### Run CLI Integration Tests

```bash
# Automated end-to-end tests (63 tests)
./test_cli.sh
```

### Test Coverage

- **Unit Tests**: 43+ tests covering all components
- **Integration Tests**: 12 tests for EventStore and Bank domain
- **CLI Tests**: 63 end-to-end tests via bash script
- **Concurrency Tests**: Verified with `-race` flag (no data races)
- **Stress Tests**: 500+ transaction tests, 10,000+ event tests

## Project Structure

```
append-only-event-store/
├── cmd/
│   └── eventstore/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── eventstore/
│   │   ├── event.go             # Event struct + methods
│   │   ├── storage.go           # DiskStorage (append-only log)
│   │   ├── index.go             # In-memory index
│   │   ├── store.go             # EventStore coordinator
│   │   ├── *_test.go            # Unit tests
│   │   └── integration_test.go  # Integration tests
│   └── domain/
│       └── bank/
│           ├── events.go        # Bank event types
│           ├── account.go       # BankAccount aggregate
│           ├── commands.go      # Business logic
│           ├── *_test.go        # Unit tests
│           └── integration_test.go  # Integration tests
├── data/                        # Event store data (gitignored)
│   └── events.log
├── test_cli.sh                  # CLI integration tests
├── architecture.md              # Detailed architecture documentation
├── ADR.md                       # Architecture Decision Records
├── TASK_TRACKER.md              # Implementation progress
├── go.mod
└── README.md                    # This file
```

## Limitations

This is an **educational project** designed for learning event sourcing concepts. It has intentional limitations:

### By Design

1. **Single Process Only**: No file locking, not safe for multi-process access
2. **Scale**: Optimized for <10,000 events (index rebuilt on every startup)
3. **No Deletions**: True append-only (no GDPR "right to be forgotten")
4. **No Snapshots**: Always replay from event 1 (can be slow for long streams)
5. **No Subscriptions**: Cannot notify on new events (polling only)
6. **No Schema Evolution**: Event structure changes require manual migration
7. **Single Log File**: All streams in one file (simpler but less flexible)

### Future Enhancements (Out of Scope)

- Snapshots for faster state reconstruction
- Projections for materialized views
- Event subscriptions for real-time notifications
- Optimistic locking for concurrent writes
- Multi-process safety with file locks
- Event upcasting for schema evolution

See [ADR.md](ADR.md) for detailed rationale behind these decisions.

## Learning Resources

This project is designed for learning. Here are the key concepts demonstrated:

- **Event Sourcing**: Storing state changes as immutable events
- **Event Replay**: Reconstructing current state by replaying events
- **Aggregates**: Domain objects that apply events to build state
- **Commands**: Business logic that validates and produces events
- **Stream-per-Aggregate**: Each entity (account) has its own event stream
- **Optimistic Concurrency**: Per-stream version numbers (foundation for conflict detection)
- **Durability**: Sync-after-write for crash recovery guarantees

### Recommended Reading Order

1. [README.md](README.md) - This file (overview and usage)
2. [architecture.md](architecture.md) - Detailed technical architecture
3. [ADR.md](ADR.md) - Architecture decision rationale
4. [pkg/eventstore/event.go](pkg/eventstore/event.go) - Start with the Event struct
5. [pkg/domain/bank/account.go](pkg/domain/bank/account.go) - See event replay in action
6. [pkg/domain/bank/commands.go](pkg/domain/bank/commands.go) - Understand business logic

## Performance Characteristics

Measured on a typical development machine:

- **Write Throughput**: ~1,000 events/sec (limited by `fsync`)
- **Read Throughput**: ~50,000 events/sec (in-memory index)
- **Stream Query**: <1ms for streams with <1,000 events
- **Index Rebuild**: ~10ms for 1,000 events, ~100ms for 10,000 events
- **Startup Time**: Dominated by index rebuild from log

## Contributing

This is an educational project. Contributions are welcome, especially:

- Bug fixes
- Additional test cases
- Documentation improvements
- Example domain models beyond banking

Please maintain the zero-dependency philosophy and focus on clarity over performance.

## License

MIT License - See LICENSE file for details

## Acknowledgments

This project was built as a learning exercise to understand:
- Event sourcing fundamentals
- Append-only data structures
- Domain-Driven Design patterns
- Go concurrency primitives

Inspired by production event stores like EventStoreDB, but intentionally simplified for educational purposes.

## Support

- **Issues**: Report bugs or ask questions via GitHub Issues
- **Architecture Questions**: See [architecture.md](architecture.md) and [ADR.md](ADR.md)
- **Usage Examples**: See [test_cli.sh](test_cli.sh) for comprehensive examples

---

**Built with ❤️ for learning event sourcing**
