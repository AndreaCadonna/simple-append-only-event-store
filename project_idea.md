# Append-Only Event Store

## What Is This?

A learning project to build a simple append-only event store - a database that only allows writing new records (events) to the end, never updating or deleting existing records. Think of it like a ledger or log file that grows forever.

## Why This Exists

This is an educational project focused on:
- Understanding what event stores are and why they exist
- Learning core architectural patterns (event sourcing, immutability, audit trails)
- Gaining broad knowledge across multiple topics: storage, concurrency, querying patterns
- Building something concrete that runs locally and can be tested

**Not aiming for**: Production-ready system or deep dive into any single topic.

## What Problem Does It Solve?

Event stores solve problems that traditional CRUD databases don't:
- **Complete audit trail**: Every change is recorded as an event
- **Time travel**: Replay events to see state at any point in time
- **Debugging**: Trace how you got to current state
- **Multiple views**: Build different read models from same events
- **Event-driven architectures**: Other systems can subscribe to event streams

**Common use cases**: Banking/financial systems, e-commerce order processing, collaborative software, systems requiring compliance/audit trails.

## Success Criteria

### 1. Core Functionality
- ✅ Can append events to the store
- ✅ Can read events back in order
- ✅ Events are immutable (can't modify/delete after writing)
- ✅ Can query events by stream (e.g., "all events for user-123")

### 2. Basic Performance Understanding
- ✅ Can handle 1,000 events without issues
- ✅ Can measure: write latency, read latency, replay speed

### 3. Observability
- ✅ Can see what's happening (logs, basic stats)
- ✅ Can verify data integrity (events aren't corrupted)

### 4. Practical Application
- ✅ Has at least one real example use case (e.g., bank account or order processing)
- ✅ Can demonstrate event replay (rebuild state from events)

### 5. Learning Documentation
- ✅ README explaining: what it is, why it exists, key design decisions
- ✅ ADR (Architecture Decision Record) for major choices made

## Requirements

### Functional Requirements

**Event Writing**
- Accept events with: stream_id, event_type, data (JSON), timestamp
- Return: event_id (unique), sequence_number
- Guarantee: events are persisted durably

**Event Reading**
- Query all events in a stream (by stream_id)
- Query all events globally (in order)
- Query events from a specific sequence number onwards

**Example Domain**
- Implement one concrete use case (e.g., bank account or shopping cart)
- Show how to: write events, replay to get current state

### Non-Functional Requirements

**Storage**
- Persist to disk (survives restart)
- Single node (no distributed complexity)

**Concurrency**
- Handle concurrent writes without corruption
- Keep it simple (don't optimize for high concurrency)

**Query Performance**
- Should be acceptable for 1,000 events (not optimizing for millions)

**Technology**
- Runnable locally without complex setup
- Minimal external dependencies
- Implementation language: **Go**

## Scope Boundaries

### In Scope
- Single-node event store
- Basic write/read operations
- One example use case
- Correctness over performance

### Out of Scope
- ❌ Distributed systems (replication, consensus, partitioning)
- ❌ High-performance optimization
- ❌ Advanced features (snapshots, projections, subscriptions)
- ❌ Production hardening (auth, encryption, monitoring dashboards)

## Testing & Validation Plan

### Manual Testing
- CLI tool to append events, query events, and replay state
- Example commands to interact with the store

### Automated Tests
- Unit tests for core logic
- Integration test: write 100 events, read them back, verify order

### Demo Scenario
- Real use case demonstrated end-to-end
- Example: Create account → Deposit money → Withdraw money → Show final balance by replaying events

## Learning Goals

By the end of this project, should be able to:
- Explain what append-only event stores are and when to use them
- Understand tradeoffs between event sourcing and traditional CRUD
- Know how events are stored, queried, and replayed
- Have hands-on experience with persistence, concurrency, and data integrity
- Document architectural decisions with clear reasoning