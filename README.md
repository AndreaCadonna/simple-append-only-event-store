# Append-Only Event Store

A simple event store implementation in Go to learn event sourcing principles. Uses a bank account example to demonstrate how current state is rebuilt from immutable events.

## What is this?

An educational project showing how event sourcing works:
- Events are stored in an append-only log (never modified or deleted)
- Current state is calculated by replaying all events
- Example: account balance = sum of all deposits minus all withdrawals

## Goal

Learn event sourcing fundamentals through a working implementation with zero dependencies (stdlib only).

## Setup & Run

```bash
# Clone the repository
git clone https://github.com/AndreaCadonna/simple-append-only-event-store.git
cd simple-append-only-event-store

# Build
go build -o eventstore ./cmd/eventstore

# Run example commands
./eventstore bank open acc-1 "Alice"
./eventstore bank deposit acc-1 10000
./eventstore bank withdraw acc-1 2500
./eventstore bank balance acc-1
```

## Commands

```bash
# Bank operations
./eventstore bank open <accountId> <owner>
./eventstore bank deposit <accountId> <amount-in-cents>
./eventstore bank withdraw <accountId> <amount-in-cents>
./eventstore bank balance <accountId>

# View events
./eventstore get-stream <streamId>
./eventstore get-all

# Generic events
./eventstore append <streamId> <eventType> '{"json":"data"}'
```

## Run Tests

```bash
go test ./...              # All tests
./test_cli.sh              # CLI integration tests
```

## Architecture

- **Event**: Immutable record of something that happened
- **DiskStorage**: Append-only file (`data/events.log`)
- **Index**: In-memory map for fast stream lookups
- **EventStore**: Coordinates storage and queries
- **Bank Domain**: Example showing event replay to calculate balance

See `architecture.md` and `ADR.md` for detailed design decisions.

## Key Concepts Demonstrated

- Event sourcing (state from events, not database updates)
- Event replay (rebuild state by applying events in order)
- Append-only storage (immutability)
- Stream-per-aggregate pattern
- Command/event separation

## Limitations

Educational project - not for production:
- Single process only (no file locking)
- Optimized for <10K events
- No snapshots (always replay from beginning)
- No subscriptions or real-time notifications

## License

MIT
