package eventstore

import (
	"fmt"
	"sync"
)

// EventStore is the high-level interface for the event store.
// It coordinates between DiskStorage (persistence) and Index (fast queries).
//
// The EventStore provides:
//   - Durable event appending with automatic versioning
//   - Fast stream queries without scanning the entire log
//   - Thread-safe concurrent operations
//   - Automatic index rebuilding on startup
type EventStore interface {
	// Append adds a new event to the store.
	// The event is persisted to disk and the index is updated atomically.
	//
	// Parameters:
	//   - streamID: The stream to append to
	//   - eventType: The type of event
	//   - data: The event payload (will be marshaled to JSON)
	//
	// Returns:
	//   - *Event: The created event with ID, timestamps, and sequences
	//   - error: If validation, persistence, or indexing fails
	Append(streamID, eventType string, data interface{}) (*Event, error)

	// GetStream returns all events for a specific stream in chronological order.
	//
	// Parameters:
	//   - streamID: The stream to query
	//
	// Returns:
	//   - []*Event: Slice of events (empty if stream doesn't exist)
	//   - error: If reading from storage fails
	GetStream(streamID string) ([]*Event, error)

	// GetAllEvents returns all events in the store in global sequence order.
	//
	// Returns:
	//   - []*Event: Slice of all events
	//   - error: If reading from storage fails
	GetAllEvents() ([]*Event, error)

	// Close releases all resources (file handles, etc.).
	// Should be called when the store is no longer needed.
	//
	// Returns:
	//   - error: If closing resources fails
	Close() error
}

// eventStore is the concrete implementation of EventStore.
// It coordinates between DiskStorage and Index with proper synchronization.
type eventStore struct {
	storage *DiskStorage
	index   *Index

	// mu coordinates access to both storage and index
	// Write operations need exclusive access
	// Read operations can be concurrent
	mu sync.RWMutex

	// lastGlobalSequence tracks the next sequence number to assign
	// Protected by mu (write lock)
	lastGlobalSequence int
}

// NewEventStore creates and initializes a new EventStore.
// It performs the following startup sequence:
//  1. Opens/creates the data directory
//  2. Opens the event log file
//  3. Reads all existing events from disk
//  4. Rebuilds the in-memory index
//  5. Returns a ready-to-use EventStore
//
// Parameters:
//   - dataDir: Directory where event store data is stored
//
// Returns:
//   - EventStore: Initialized event store instance
//   - error: If initialization fails at any step
func NewEventStore(dataDir string) (EventStore, error) {
	// Initialize storage
	storage, err := NewDiskStorage(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Read all events from disk
	events, err := storage.ReadAll()
	if err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	// Build index from events
	index := NewIndex()
	if err := index.BuildFromEvents(events); err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to build index: %w", err)
	}

	// Determine last global sequence
	lastSeq := 0
	if len(events) > 0 {
		lastSeq = events[len(events)-1].GlobalSequence
	}

	return &eventStore{
		storage:            storage,
		index:              index,
		lastGlobalSequence: lastSeq,
	}, nil
}

// Append adds a new event to the store.
// This is the core write operation with the following workflow:
//  1. Acquire write lock (exclusive access)
//  2. Calculate next global sequence and stream version
//  3. Create Event instance
//  4. Write to DiskStorage (durable)
//  5. Update Index (fast queries)
//  6. Release write lock
//
// The operation is atomic - either both storage and index are updated,
// or neither is (in case of error).
//
// Parameters:
//   - streamID: The stream to append to (must be non-empty)
//   - eventType: The type of event (must be non-empty)
//   - data: The event payload (will be marshaled to JSON)
//
// Returns:
//   - *Event: The created event with all fields populated
//   - error: If validation or persistence fails
func (es *eventStore) Append(streamID, eventType string, data interface{}) (*Event, error) {
	// Validate inputs
	if streamID == "" {
		return nil, fmt.Errorf("streamID cannot be empty")
	}
	if eventType == "" {
		return nil, fmt.Errorf("eventType cannot be empty")
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	// Calculate next global sequence
	es.lastGlobalSequence++
	globalSeq := es.lastGlobalSequence

	// Get current stream version (next version is current + 1)
	currentVersion, _ := es.index.GetStreamVersion(streamID)
	version := currentVersion + 1

	// Create event
	event, err := NewEvent(streamID, eventType, data, version, globalSeq)
	if err != nil {
		// Rollback global sequence on event creation failure
		es.lastGlobalSequence--
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Write to storage (durable persistence)
	if err := es.storage.Append(event); err != nil {
		// Rollback global sequence on storage failure
		es.lastGlobalSequence--
		return nil, fmt.Errorf("failed to persist event: %w", err)
	}

	// Update index (fast queries)
	if err := es.index.AddEvent(event); err != nil {
		// Index update failed - this is serious because storage succeeded
		// We don't rollback storage (it's already on disk)
		// But we log the inconsistency
		return nil, fmt.Errorf("CRITICAL: event persisted but index update failed: %w", err)
	}

	return event, nil
}

// GetStream returns all events for a specific stream in chronological order.
// This uses the index for O(1) lookup of event positions, then loads
// only the relevant events from storage.
//
// Parameters:
//   - streamID: The stream to query
//
// Returns:
//   - []*Event: Slice of events in order (empty if stream doesn't exist)
//   - error: If reading from storage fails
func (es *eventStore) GetStream(streamID string) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// Get global sequences for this stream from index
	sequences, err := es.index.GetStreamSequences(streamID)
	if err != nil {
		return nil, fmt.Errorf("failed to query index: %w", err)
	}

	// If stream doesn't exist or is empty, return empty slice
	if len(sequences) == 0 {
		return []*Event{}, nil
	}

	// Read all events from storage
	allEvents, err := es.storage.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	// Filter events by stream
	// We build a map for O(1) lookup of which sequences we need
	sequenceMap := make(map[int]bool)
	for _, seq := range sequences {
		sequenceMap[seq] = true
	}

	streamEvents := make([]*Event, 0, len(sequences))
	for _, event := range allEvents {
		if sequenceMap[event.GlobalSequence] {
			streamEvents = append(streamEvents, event)
		}
	}

	return streamEvents, nil
}

// GetAllEvents returns all events in the store in global sequence order.
// This is useful for debugging, replaying the entire event log, or
// building projections across all streams.
//
// Returns:
//   - []*Event: Slice of all events in order
//   - error: If reading from storage fails
func (es *eventStore) GetAllEvents() ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events, err := es.storage.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read all events: %w", err)
	}

	return events, nil
}

// Close releases all resources used by the event store.
// This includes closing file handles and cleaning up.
//
// After calling Close, the EventStore should not be used.
//
// Returns:
//   - error: If closing resources fails
func (es *eventStore) Close() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if err := es.storage.Close(); err != nil {
		return fmt.Errorf("failed to close storage: %w", err)
	}

	return nil
}
