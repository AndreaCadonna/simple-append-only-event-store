package eventstore

import (
	"fmt"
	"sync"
)

// Index provides fast in-memory lookups for events by stream ID.
// It maps stream IDs to the global sequences of events in that stream,
// enabling O(1) stream queries instead of O(n) log scans.
//
// The index is rebuilt on startup from all events in the log file.
// It stays synchronized with new appends through the AddEvent method.
//
// Example:
//
//	{
//	  "account-123": [1, 5, 8, 15],    // Events at global positions 1, 5, 8, 15
//	  "account-456": [2, 9, 12]
//	}
type Index struct {
	// streamIndex maps streamID -> slice of global sequences
	streamIndex map[string][]int

	// mu protects concurrent access to the index
	// Uses RWMutex to allow concurrent reads but exclusive writes
	mu sync.RWMutex
}

// NewIndex creates a new empty Index.
//
// Returns:
//   - *Index: Initialized index instance
func NewIndex() *Index {
	return &Index{
		streamIndex: make(map[string][]int),
	}
}

// BuildFromEvents constructs the index from a slice of events.
// This is called on startup after loading all events from disk.
//
// The method clears any existing index data and rebuilds from scratch.
//
// Parameters:
//   - events: Slice of events to index (typically from DiskStorage.ReadAll())
//
// Returns:
//   - error: If validation fails (duplicate global sequences, invalid data)
func (idx *Index) BuildFromEvents(events []*Event) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear existing index
	idx.streamIndex = make(map[string][]int)

	// Track seen global sequences to detect duplicates
	seenSequences := make(map[int]bool)

	for i, event := range events {
		if event == nil {
			return fmt.Errorf("event at position %d is nil", i)
		}

		// Validate global sequence
		if event.GlobalSequence < 1 {
			return fmt.Errorf("event at position %d has invalid GlobalSequence: %d", i, event.GlobalSequence)
		}

		// Check for duplicate global sequences
		if seenSequences[event.GlobalSequence] {
			return fmt.Errorf("duplicate GlobalSequence %d found", event.GlobalSequence)
		}
		seenSequences[event.GlobalSequence] = true

		// Validate stream ID
		if event.StreamID == "" {
			return fmt.Errorf("event at position %d has empty StreamID", i)
		}

		// Add to index
		idx.streamIndex[event.StreamID] = append(idx.streamIndex[event.StreamID], event.GlobalSequence)
	}

	return nil
}

// AddEvent incrementally updates the index when a new event is appended.
// This is called after successfully writing an event to disk.
//
// Parameters:
//   - event: The event that was just appended
//
// Returns:
//   - error: If validation fails
func (idx *Index) AddEvent(event *Event) error {
	if event == nil {
		return fmt.Errorf("cannot add nil event to index")
	}

	if event.StreamID == "" {
		return fmt.Errorf("event has empty StreamID")
	}

	if event.GlobalSequence < 1 {
		return fmt.Errorf("event has invalid GlobalSequence: %d", event.GlobalSequence)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Append global sequence to the stream's sequence list
	idx.streamIndex[event.StreamID] = append(idx.streamIndex[event.StreamID], event.GlobalSequence)

	return nil
}

// GetStreamSequences returns the global sequences of all events in a stream.
// The sequences are returned in the order they were added (chronological order).
//
// This method is thread-safe and allows concurrent reads.
//
// Parameters:
//   - streamID: The stream to query
//
// Returns:
//   - []int: Slice of global sequences (empty slice if stream doesn't exist)
//   - error: Currently always nil, reserved for future use
func (idx *Index) GetStreamSequences(streamID string) ([]int, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sequences, exists := idx.streamIndex[streamID]
	if !exists {
		// Return empty slice (not nil) for non-existent streams
		return []int{}, nil
	}

	// Return a copy to prevent external modification
	result := make([]int, len(sequences))
	copy(result, sequences)

	return result, nil
}

// GetStreamVersion returns the current version (count) of events in a stream.
// This represents the position of the next event that would be appended to this stream.
//
// For example:
//   - Version 0: Stream doesn't exist or has no events
//   - Version 3: Stream has 3 events, next event would be version 4
//
// This method is thread-safe and allows concurrent reads.
//
// Parameters:
//   - streamID: The stream to query
//
// Returns:
//   - int: Current version (count of events in the stream)
//   - error: Currently always nil, reserved for future use
func (idx *Index) GetStreamVersion(streamID string) (int, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sequences, exists := idx.streamIndex[streamID]
	if !exists {
		return 0, nil
	}

	return len(sequences), nil
}

// GetAllStreams returns a list of all stream IDs that have events.
// This is useful for debugging and understanding what streams exist in the store.
//
// This method is thread-safe and allows concurrent reads.
//
// Returns:
//   - []string: Slice of stream IDs (empty slice if no streams exist)
func (idx *Index) GetAllStreams() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.streamIndex) == 0 {
		return []string{}
	}

	// Collect all stream IDs
	streams := make([]string, 0, len(idx.streamIndex))
	for streamID := range idx.streamIndex {
		streams = append(streams, streamID)
	}

	return streams
}

// GetEventCount returns the total number of events indexed across all streams.
// This is useful for statistics and monitoring.
//
// This method is thread-safe and allows concurrent reads.
//
// Returns:
//   - int: Total count of events in the index
func (idx *Index) GetEventCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	count := 0
	for _, sequences := range idx.streamIndex {
		count += len(sequences)
	}

	return count
}

// Clear removes all data from the index.
// This is primarily useful for testing.
//
// This method is thread-safe.
func (idx *Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.streamIndex = make(map[string][]int)
}
