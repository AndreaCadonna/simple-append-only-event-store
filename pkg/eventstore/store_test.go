package eventstore

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Helper to create temp directory for testing
func createTestEventStore(t *testing.T) (EventStore, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "eventstore-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := NewEventStore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create event store: %v", err)
	}

	return store, tempDir
}

// Helper to cleanup test event store
func cleanupTestEventStore(t *testing.T, store EventStore, dir string) {
	t.Helper()
	if store != nil {
		store.Close()
	}
	if dir != "" {
		os.RemoveAll(dir)
	}
}

func TestNewEventStore_EmptyStore(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// New store should have no events
	events, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events in new store, got %d", len(events))
	}
}

func TestEventStore_AppendSingleEvent(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append an event
	data := map[string]string{"key": "value"}
	event, err := store.Append("stream-1", "TestEvent", data)

	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify event fields
	if event.StreamID != "stream-1" {
		t.Errorf("Expected StreamID 'stream-1', got '%s'", event.StreamID)
	}
	if event.EventType != "TestEvent" {
		t.Errorf("Expected EventType 'TestEvent', got '%s'", event.EventType)
	}
	if event.Version != 1 {
		t.Errorf("Expected Version 1, got %d", event.Version)
	}
	if event.GlobalSequence != 1 {
		t.Errorf("Expected GlobalSequence 1, got %d", event.GlobalSequence)
	}
	if event.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestEventStore_AppendMultipleEvents(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append multiple events
	for i := 1; i <= 5; i++ {
		data := map[string]int{"index": i}
		event, err := store.Append("stream-1", "TestEvent", data)

		if err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}

		if event.Version != i {
			t.Errorf("Event %d: expected Version %d, got %d", i, i, event.Version)
		}
		if event.GlobalSequence != i {
			t.Errorf("Event %d: expected GlobalSequence %d, got %d", i, i, event.GlobalSequence)
		}
	}

	// Verify all events are stored
	events, _ := store.GetAllEvents()
	if len(events) != 5 {
		t.Fatalf("Expected 5 events, got %d", len(events))
	}
}

func TestEventStore_AppendToMultipleStreams(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append events to different streams (interleaved)
	event1, _ := store.Append("stream-1", "Event", nil)
	event2, _ := store.Append("stream-2", "Event", nil)
	event3, _ := store.Append("stream-1", "Event", nil)
	event4, _ := store.Append("stream-3", "Event", nil)
	event5, _ := store.Append("stream-2", "Event", nil)

	// Verify global sequences are sequential
	if event1.GlobalSequence != 1 || event2.GlobalSequence != 2 ||
		event3.GlobalSequence != 3 || event4.GlobalSequence != 4 ||
		event5.GlobalSequence != 5 {
		t.Error("Global sequences not sequential")
	}

	// Verify stream versions are correct
	if event1.Version != 1 || event3.Version != 2 {
		t.Error("stream-1 versions incorrect")
	}
	if event2.Version != 1 || event5.Version != 2 {
		t.Error("stream-2 versions incorrect")
	}
	if event4.Version != 1 {
		t.Error("stream-3 version incorrect")
	}
}

func TestEventStore_GetStream(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append events to multiple streams
	store.Append("stream-1", "Event1", nil)
	store.Append("stream-2", "Event2", nil)
	store.Append("stream-1", "Event3", nil)
	store.Append("stream-1", "Event4", nil)
	store.Append("stream-2", "Event5", nil)

	// Get stream-1 events
	stream1Events, err := store.GetStream("stream-1")
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if len(stream1Events) != 3 {
		t.Fatalf("Expected 3 events in stream-1, got %d", len(stream1Events))
	}

	// Verify events are in order
	if stream1Events[0].EventType != "Event1" ||
		stream1Events[1].EventType != "Event3" ||
		stream1Events[2].EventType != "Event4" {
		t.Error("stream-1 events not in correct order")
	}

	// Get stream-2 events
	stream2Events, err := store.GetStream("stream-2")
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if len(stream2Events) != 2 {
		t.Fatalf("Expected 2 events in stream-2, got %d", len(stream2Events))
	}
}

func TestEventStore_GetStream_NonExistent(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Get non-existent stream
	events, err := store.GetStream("non-existent")
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	// Should return empty slice, not nil
	if events == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

func TestEventStore_GetAllEvents(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append events to multiple streams
	store.Append("stream-1", "Event1", nil)
	store.Append("stream-2", "Event2", nil)
	store.Append("stream-1", "Event3", nil)

	// Get all events
	allEvents, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	if len(allEvents) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(allEvents))
	}

	// Verify events are in global order
	if allEvents[0].GlobalSequence != 1 ||
		allEvents[1].GlobalSequence != 2 ||
		allEvents[2].GlobalSequence != 3 {
		t.Error("Events not in global sequence order")
	}
}

func TestEventStore_PersistenceAcrossReopening(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "eventstore-persistence-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// First store - write events
	store1, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	store1.Append("stream-1", "Event1", map[string]string{"data": "one"})
	store1.Append("stream-2", "Event2", map[string]string{"data": "two"})
	store1.Append("stream-1", "Event3", map[string]string{"data": "three"})
	store1.Close()

	// Second store - read events
	store2, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}
	defer store2.Close()

	// Verify all events persisted
	allEvents, err := store2.GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	if len(allEvents) != 3 {
		t.Fatalf("Expected 3 events after reopening, got %d", len(allEvents))
	}

	// Verify stream queries work
	stream1Events, _ := store2.GetStream("stream-1")
	if len(stream1Events) != 2 {
		t.Errorf("Expected 2 events in stream-1 after reopening, got %d", len(stream1Events))
	}

	// Append more events to verify store is functional
	event4, err := store2.Append("stream-1", "Event4", nil)
	if err != nil {
		t.Fatalf("Append after reopening failed: %v", err)
	}

	// Verify versioning continues correctly
	if event4.Version != 3 {
		t.Errorf("Expected Version 3 after reopening, got %d", event4.Version)
	}
	if event4.GlobalSequence != 4 {
		t.Errorf("Expected GlobalSequence 4 after reopening, got %d", event4.GlobalSequence)
	}
}

func TestEventStore_Append_EmptyStreamID(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	_, err := store.Append("", "EventType", nil)
	if err == nil {
		t.Fatal("Expected error for empty streamID, got nil")
	}
}

func TestEventStore_Append_EmptyEventType(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	_, err := store.Append("stream-1", "", nil)
	if err == nil {
		t.Fatal("Expected error for empty eventType, got nil")
	}
}

func TestEventStore_ConcurrentAppends(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append events concurrently
	concurrency := 10
	eventsPerGoroutine := 10
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				data := map[string]int{"goroutine": goroutineID, "index": j}
				_, err := store.Append("stream-1", "TestEvent", data)
				if err != nil {
					t.Errorf("Concurrent append failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were appended
	events, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	expectedCount := concurrency * eventsPerGoroutine
	if len(events) != expectedCount {
		t.Errorf("Expected %d events after concurrent appends, got %d", expectedCount, len(events))
	}

	// Verify global sequences are unique and sequential
	seenSequences := make(map[int]bool)
	for _, event := range events {
		if seenSequences[event.GlobalSequence] {
			t.Errorf("Duplicate GlobalSequence: %d", event.GlobalSequence)
		}
		seenSequences[event.GlobalSequence] = true
	}

	// Verify we have sequences 1 through expectedCount
	for i := 1; i <= expectedCount; i++ {
		if !seenSequences[i] {
			t.Errorf("Missing GlobalSequence: %d", i)
		}
	}
}

func TestEventStore_ConcurrentReads(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append some events first
	for i := 1; i <= 10; i++ {
		store.Append("stream-1", "Event", map[string]int{"index": i})
	}

	// Perform concurrent reads
	concurrency := 50
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				store.GetStream("stream-1")
				store.GetAllEvents()
			}
		}()
	}

	wg.Wait()

	// Verify data is still correct after concurrent reads
	events, _ := store.GetStream("stream-1")
	if len(events) != 10 {
		t.Errorf("Concurrent reads corrupted data: expected 10 events, got %d", len(events))
	}
}

func TestEventStore_ConcurrentMixedOperations(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	concurrency := 20
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < 50; j++ {
				if goroutineID%3 == 0 {
					// Writer
					store.Append("stream-1", "Event", map[string]int{"g": goroutineID, "i": j})
				} else {
					// Reader
					store.GetStream("stream-1")
					store.GetAllEvents()
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify store is still functional
	events, err := store.GetAllEvents()
	if err != nil {
		t.Errorf("GetAllEvents failed after concurrent operations: %v", err)
	}

	// Should have events from writers (goroutines 0, 3, 6, 9, 12, 15, 18 = 7 writers * 50 events)
	writerCount := (concurrency + 2) / 3 // ceiling division
	expectedMin := writerCount * 50
	if len(events) < expectedMin {
		t.Errorf("Expected at least %d events, got %d", expectedMin, len(events))
	}
}

func TestEventStore_LargeNumberOfEvents(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append many events
	eventCount := 1000
	for i := 1; i <= eventCount; i++ {
		_, err := store.Append("stream-1", "Event", map[string]int{"index": i})
		if err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	// Verify all events stored
	events, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	if len(events) != eventCount {
		t.Errorf("Expected %d events, got %d", eventCount, len(events))
	}

	// Verify stream query works
	streamEvents, err := store.GetStream("stream-1")
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if len(streamEvents) != eventCount {
		t.Errorf("Expected %d events in stream, got %d", eventCount, len(streamEvents))
	}
}

func TestEventStore_MultipleStreamsWithManyEvents(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Create 10 streams with 100 events each
	streamCount := 10
	eventsPerStream := 100

	for s := 1; s <= streamCount; s++ {
		streamID := string(rune('a' + s - 1))
		for e := 1; e <= eventsPerStream; e++ {
			_, err := store.Append(streamID, "Event", map[string]int{"stream": s, "index": e})
			if err != nil {
				t.Fatalf("Append failed: %v", err)
			}
		}
	}

	// Verify total event count
	allEvents, _ := store.GetAllEvents()
	expectedTotal := streamCount * eventsPerStream
	if len(allEvents) != expectedTotal {
		t.Errorf("Expected %d total events, got %d", expectedTotal, len(allEvents))
	}

	// Verify each stream has correct count
	for s := 1; s <= streamCount; s++ {
		streamID := string(rune('a' + s - 1))
		streamEvents, _ := store.GetStream(streamID)
		if len(streamEvents) != eventsPerStream {
			t.Errorf("Stream %s: expected %d events, got %d", streamID, eventsPerStream, len(streamEvents))
		}
	}
}

func TestEventStore_Close(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer os.RemoveAll(dir)

	// Append an event
	store.Append("stream-1", "Event", nil)

	// Close store
	err := store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestEventStore_ReopenAndContinue(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "eventstore-reopen-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Cycle 1: Create store, add events, close
	store1, _ := NewEventStore(tempDir)
	store1.Append("stream-1", "Event1", nil)
	store1.Append("stream-1", "Event2", nil)
	store1.Close()

	// Cycle 2: Reopen, add more events, close
	store2, _ := NewEventStore(tempDir)
	store2.Append("stream-1", "Event3", nil)
	store2.Close()

	// Cycle 3: Reopen and verify
	store3, _ := NewEventStore(tempDir)
	defer store3.Close()

	events, _ := store3.GetStream("stream-1")
	if len(events) != 3 {
		t.Fatalf("Expected 3 events after multiple reopen cycles, got %d", len(events))
	}

	// Verify versions
	if events[0].Version != 1 || events[1].Version != 2 || events[2].Version != 3 {
		t.Error("Event versions not correct after reopen cycles")
	}

	// Verify global sequences
	if events[0].GlobalSequence != 1 || events[1].GlobalSequence != 2 || events[2].GlobalSequence != 3 {
		t.Error("Global sequences not correct after reopen cycles")
	}
}

func TestEventStore_DataIntegrity(t *testing.T) {
	store, dir := createTestEventStore(t)
	defer cleanupTestEventStore(t, store, dir)

	// Append event with specific data
	originalData := map[string]interface{}{
		"accountId": "acc-123",
		"owner":     "Alice",
		"balance":   10000,
	}

	event, err := store.Append("stream-1", "AccountOpened", originalData)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Read back and verify data is intact
	events, _ := store.GetStream("stream-1")
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	readEvent := events[0]

	// Verify all fields match
	if readEvent.ID != event.ID {
		t.Error("ID mismatch")
	}
	if readEvent.StreamID != event.StreamID {
		t.Error("StreamID mismatch")
	}
	if readEvent.EventType != event.EventType {
		t.Error("EventType mismatch")
	}
	if readEvent.Version != event.Version {
		t.Error("Version mismatch")
	}
	if readEvent.GlobalSequence != event.GlobalSequence {
		t.Error("GlobalSequence mismatch")
	}
	if string(readEvent.Data) != string(event.Data) {
		t.Errorf("Data mismatch:\nExpected: %s\nGot: %s", string(event.Data), string(readEvent.Data))
	}
}

func TestEventStore_FilesCreated(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "eventstore-files-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("NewEventStore failed: %v", err)
	}
	defer store.Close()

	// Append an event to ensure file is created
	store.Append("stream-1", "Event", nil)

	// Verify events.log file exists
	logPath := filepath.Join(tempDir, EventLogFileName)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Event log file not created: %s", logPath)
	}
}
