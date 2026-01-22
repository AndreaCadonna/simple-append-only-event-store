# Event Store Implementation - Task Tracker

**Project**: Append-Only Event Store with Bank Domain Example
**Started**: 2026-01-22
**Status**: In Progress

---

## Progress Overview

- **Wave 0 (Setup)**: ✅ 1/1 Complete
- **Wave 1 (Foundations)**: ⬜ 0/3 Complete
- **Wave 2 (Core Components)**: ⬜ 0/2 Complete
- **Wave 3 (Integration)**: ⬜ 0/1 Complete
- **Wave 4 (Domain Logic)**: ⬜ 0/2 Complete
- **Wave 5 (User Interface)**: ⬜ 0/1 Complete
- **Wave 6 (Testing & Docs)**: ⬜ 0/2 Complete

**Overall Progress**: 1/12 tasks complete (8%)

---

## Wave 0: Setup

### ✅ Task 0: Project Initialization
**Complexity**: Trivial
**Dependencies**: None
**Can Start**: Immediately

**Deliverables**:
- [x] Initialize Go module
- [x] Create directory structure (`cmd/eventstore/`, `pkg/eventstore/`, `pkg/domain/bank/`, `data/`)
- [x] Create `.gitignore` with `data/` and `*.log` entries
- [x] Create empty `README.md` placeholder
- [x] Initialize Git repository
- [x] Push to GitHub

**Files Created**:
- `go.mod`
- `.gitignore`
- `README.md`

**Status**: Completed
**Assigned To**: Claude
**Completed**: 2026-01-22

---

## Wave 1: Foundations (Parallel)

### ⬜ Task 1: Event Structure
**Complexity**: Low
**Dependencies**: Task 0
**Can Start**: After Task 0

**Deliverables**:
- [ ] Define `Event` struct with all fields
- [ ] Implement `NewEvent()` constructor with validation
- [ ] Implement JSON marshaling/unmarshaling
- [ ] Write unit tests

**Files Created**:
- `pkg/eventstore/event.go`
- `pkg/eventstore/event_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

### ⬜ Task 2: Bank Event Definitions
**Complexity**: Low
**Dependencies**: Task 0
**Can Start**: After Task 0 (parallel with Task 1)

**Deliverables**:
- [ ] Define event type constants (AccountOpened, MoneyDeposited, MoneyWithdrawn)
- [ ] Define event data structs
- [ ] Write JSON marshaling tests

**Files Created**:
- `pkg/domain/bank/events.go`
- `pkg/domain/bank/events_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

### ⬜ Task 3: CLI Skeleton
**Complexity**: Low
**Dependencies**: Task 0
**Can Start**: After Task 0 (parallel with Tasks 1, 2)

**Deliverables**:
- [ ] Create CLI entry point with command parsing
- [ ] Define command structure (append, get-stream, get-all, bank)
- [ ] Add stub handlers that print "Not implemented"
- [ ] Add `--data-dir` flag

**Files Created**:
- `cmd/eventstore/main.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

## Wave 2: Core Components (Parallel)

### ⬜ Task 4: DiskStorage Implementation
**Complexity**: Medium
**Dependencies**: Task 1
**Can Start**: After Task 1 completes

**Deliverables**:
- [ ] Implement `DiskStorage` struct
- [ ] Implement `Append()` with file format [4-byte length][JSON]
- [ ] Implement `ReadAll()` to parse entire log file
- [ ] Add mutex for thread safety
- [ ] Add `file.Sync()` for durability
- [ ] Write unit tests

**Files Created**:
- `pkg/eventstore/storage.go`
- `pkg/eventstore/storage_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

### ⬜ Task 5: Index Implementation
**Complexity**: Medium
**Dependencies**: Task 1
**Can Start**: After Task 1 completes (parallel with Task 4)

**Deliverables**:
- [ ] Implement `Index` struct with `map[string][]int`
- [ ] Implement `BuildFromEvents()`
- [ ] Implement `AddEvent()`
- [ ] Implement `GetStreamSequences()` and `GetStreamVersion()`
- [ ] Add RWMutex for concurrent access
- [ ] Write unit tests

**Files Created**:
- `pkg/eventstore/index.go`
- `pkg/eventstore/index_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

## Wave 3: Integration Layer

### ⬜ Task 6: EventStore Coordinator
**Complexity**: High
**Dependencies**: Tasks 1, 4, 5
**Can Start**: After Tasks 1, 4, 5 complete

**Deliverables**:
- [ ] Define `EventStore` interface
- [ ] Implement coordinator struct with DiskStorage and Index
- [ ] Implement `NewEventStore()` with initialization and reload
- [ ] Implement `Append()` with concurrency control
- [ ] Implement `GetStream()` and `GetAllEvents()`
- [ ] Write unit tests and integration tests

**Files Created**:
- `pkg/eventstore/store.go`
- `pkg/eventstore/store_test.go`
- `pkg/eventstore/integration_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

## Wave 4: Domain Logic (Parallel)

### ⬜ Task 7: Bank Account Aggregate
**Complexity**: Medium
**Dependencies**: Tasks 2, 6
**Can Start**: After Tasks 2 and 6 complete

**Deliverables**:
- [ ] Define `BankAccount` struct
- [ ] Implement `ApplyEvent()` with event type switching
- [ ] Implement `ReplayAccount()` to rebuild state
- [ ] Write unit tests

**Files Created**:
- `pkg/domain/bank/account.go`
- `pkg/domain/bank/account_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

### ⬜ Task 8: Bank Commands (Business Logic)
**Complexity**: Medium
**Dependencies**: Tasks 2, 7
**Can Start**: After Tasks 2, 7 complete

**Deliverables**:
- [ ] Implement `OpenAccount()` with validation
- [ ] Implement `Deposit()` with validation
- [ ] Implement `Withdraw()` with balance check
- [ ] Implement `GetBalance()`
- [ ] Write unit tests

**Files Created**:
- `pkg/domain/bank/commands.go`
- `pkg/domain/bank/commands_test.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

## Wave 5: User Interface

### ⬜ Task 9: CLI Implementation
**Complexity**: Medium
**Dependencies**: Tasks 6, 8
**Can Start**: After Tasks 6, 8 complete

**Deliverables**:
- [ ] Initialize EventStore in main()
- [ ] Implement all command handlers (append, get-stream, get-all, bank)
- [ ] Add output formatting for events and balances
- [ ] Add error handling
- [ ] Test manual workflow

**Files Updated**:
- `cmd/eventstore/main.go`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

## Wave 6: Testing & Documentation

### ⬜ Task 10: Integration Testing
**Complexity**: Medium
**Dependencies**: Task 9
**Can Start**: After Task 9 completes

**Deliverables**:
- [ ] Write EventStore integration tests
- [ ] Write Bank domain integration tests
- [ ] Create CLI test bash script
- [ ] Verify persistence across restarts
- [ ] Test concurrent access

**Files Created**:
- `pkg/eventstore/integration_test.go` (if not done in Task 6)
- `pkg/domain/bank/integration_test.go`
- `test_cli.sh`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

### ⬜ Task 11: Documentation
**Complexity**: Low
**Dependencies**: Task 9
**Can Start**: After Task 9 completes (parallel with Task 10)

**Deliverables**:
- [ ] Update README.md with full documentation
- [ ] Create docs/ARCHITECTURE.md
- [ ] Create docs/ADR.md (Architecture Decision Record)
- [ ] Add code examples to README
- [ ] Document limitations

**Files Created/Updated**:
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/ADR.md`

**Status**: Not Started
**Assigned To**: -
**Completed**: -

---

## Critical Integration Points

### ✅ Integration Checklist

**Integration 1: EventStore → DiskStorage & Index**
- [ ] Unit tests with mocked components pass
- [ ] Integration tests with real components pass

**Integration 2: Bank Commands → EventStore**
- [ ] Unit tests with mocked EventStore pass
- [ ] Integration tests with real EventStore pass

**Integration 3: CLI → EventStore & Bank**
- [ ] Manual CLI testing passes
- [ ] Shell script testing passes

**Integration 4: BankAccount → Event Replay**
- [ ] Unit tests with fake events pass
- [ ] Integration tests with real EventStore pass

---

## Verification Checklist

### After Wave 3 (EventStore complete)
- [ ] Can append events via Go code
- [ ] Can query events by stream
- [ ] Events persist across restarts
- [ ] No race conditions (run with `-race` flag)

### After Wave 4 (Bank domain complete)
- [ ] Can open bank account
- [ ] Can deposit money
- [ ] Can withdraw money
- [ ] Cannot overdraw account
- [ ] Balance is correct after replay

### After Wave 5 (CLI complete)
- [ ] All CLI commands work
- [ ] Can build executable
- [ ] CLI test script passes

### After Wave 6 (Complete)
- [ ] All integration tests pass
- [ ] Documentation is accurate
- [ ] README examples work
- [ ] Project is ready for demo

---

## Success Criteria

### Functional
- [ ] Can append 1000+ events without issues
- [ ] Can query events by stream in <10ms
- [ ] Events survive process restart (durable)
- [ ] Bank account demo works end-to-end
- [ ] Concurrent reads don't block each other

### Code Quality
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] No external dependencies (stdlib only)
- [ ] Code is well-documented
- [ ] No race conditions (verified with `-race`)

### Documentation
- [ ] README explains project clearly
- [ ] Architecture document is accurate
- [ ] ADR captures key decisions
- [ ] Code comments explain non-obvious logic

---

## Notes & Issues

### Blockers
*None currently*

### Questions
*None currently*

### Decisions
*Document any changes to the original plan here*

---

## How to Use This Tracker

1. **Mark tasks complete** by changing `⬜` to `✅` in the task header
2. **Check off deliverables** as you complete them within each task
3. **Update status field** (Not Started → In Progress → Completed)
4. **Add notes** in the Notes & Issues section for blockers or questions
5. **Update progress overview** at the top after completing tasks
6. **Track integration points** as you connect components together

---

**Last Updated**: 2026-01-22
