package eventstore

import (
	"sync"
	"testing"
)

func TestNewIndex(t *testing.T) {
	idx := NewIndex()

	if idx == nil {
		t.Fatal("NewIndex() returned nil")
	}

	if idx.streamIndex == nil {
		t.Error("streamIndex map is nil")
	}

	// Should be empty initially
	if len(idx.streamIndex) != 0 {
		t.Errorf("Expected empty index, got %d streams", len(idx.streamIndex))
	}
}

func TestIndex_BuildFromEvents_SingleStream(t *testing.T) {
	idx := NewIndex()

	// Create events for a single stream
	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event1", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 2, EventType: "Event2", Version: 2},
		{StreamID: "stream-1", GlobalSequence: 3, EventType: "Event3", Version: 3},
	}

	err := idx.BuildFromEvents(events)
	if err != nil {
		t.Fatalf("BuildFromEvents failed: %v", err)
	}

	// Check that stream was indexed
	sequences, err := idx.GetStreamSequences("stream-1")
	if err != nil {
		t.Fatalf("GetStreamSequences failed: %v", err)
	}

	if len(sequences) != 3 {
		t.Fatalf("Expected 3 sequences, got %d", len(sequences))
	}

	// Verify sequences are correct
	expected := []int{1, 2, 3}
	for i, seq := range sequences {
		if seq != expected[i] {
			t.Errorf("Sequence %d: expected %d, got %d", i, expected[i], seq)
		}
	}
}

func TestIndex_BuildFromEvents_MultipleStreams(t *testing.T) {
	idx := NewIndex()

	// Create events for multiple streams (interleaved)
	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-2", GlobalSequence: 2, EventType: "Event", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 3, EventType: "Event", Version: 2},
		{StreamID: "stream-3", GlobalSequence: 4, EventType: "Event", Version: 1},
		{StreamID: "stream-2", GlobalSequence: 5, EventType: "Event", Version: 2},
	}

	err := idx.BuildFromEvents(events)
	if err != nil {
		t.Fatalf("BuildFromEvents failed: %v", err)
	}

	// Check stream-1
	seq1, _ := idx.GetStreamSequences("stream-1")
	if len(seq1) != 2 || seq1[0] != 1 || seq1[1] != 3 {
		t.Errorf("stream-1 sequences incorrect: %v", seq1)
	}

	// Check stream-2
	seq2, _ := idx.GetStreamSequences("stream-2")
	if len(seq2) != 2 || seq2[0] != 2 || seq2[1] != 5 {
		t.Errorf("stream-2 sequences incorrect: %v", seq2)
	}

	// Check stream-3
	seq3, _ := idx.GetStreamSequences("stream-3")
	if len(seq3) != 1 || seq3[0] != 4 {
		t.Errorf("stream-3 sequences incorrect: %v", seq3)
	}
}

func TestIndex_BuildFromEvents_EmptySlice(t *testing.T) {
	idx := NewIndex()

	err := idx.BuildFromEvents([]*Event{})
	if err != nil {
		t.Fatalf("BuildFromEvents with empty slice failed: %v", err)
	}

	// Should have empty index
	if len(idx.streamIndex) != 0 {
		t.Errorf("Expected empty index, got %d streams", len(idx.streamIndex))
	}
}

func TestIndex_BuildFromEvents_NilEvent(t *testing.T) {
	idx := NewIndex()

	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		nil, // Nil event
		{StreamID: "stream-1", GlobalSequence: 2, EventType: "Event", Version: 2},
	}

	err := idx.BuildFromEvents(events)
	if err == nil {
		t.Fatal("Expected error for nil event, got nil")
	}
}

func TestIndex_BuildFromEvents_EmptyStreamID(t *testing.T) {
	idx := NewIndex()

	events := []*Event{
		{StreamID: "", GlobalSequence: 1, EventType: "Event", Version: 1}, // Empty stream ID
	}

	err := idx.BuildFromEvents(events)
	if err == nil {
		t.Fatal("Expected error for empty StreamID, got nil")
	}
}

func TestIndex_BuildFromEvents_InvalidGlobalSequence(t *testing.T) {
	idx := NewIndex()

	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 0, EventType: "Event", Version: 1}, // Invalid sequence
	}

	err := idx.BuildFromEvents(events)
	if err == nil {
		t.Fatal("Expected error for invalid GlobalSequence, got nil")
	}
}

func TestIndex_BuildFromEvents_DuplicateGlobalSequence(t *testing.T) {
	idx := NewIndex()

	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-2", GlobalSequence: 1, EventType: "Event", Version: 1}, // Duplicate sequence
	}

	err := idx.BuildFromEvents(events)
	if err == nil {
		t.Fatal("Expected error for duplicate GlobalSequence, got nil")
	}
}

func TestIndex_BuildFromEvents_Rebuild(t *testing.T) {
	idx := NewIndex()

	// Build index first time
	events1 := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 2, EventType: "Event", Version: 2},
	}
	idx.BuildFromEvents(events1)

	// Rebuild with different events (should replace old index)
	events2 := []*Event{
		{StreamID: "stream-2", GlobalSequence: 1, EventType: "Event", Version: 1},
	}
	err := idx.BuildFromEvents(events2)
	if err != nil {
		t.Fatalf("Rebuild failed: %v", err)
	}

	// stream-1 should no longer exist
	seq1, _ := idx.GetStreamSequences("stream-1")
	if len(seq1) != 0 {
		t.Errorf("stream-1 should be gone after rebuild, got %d events", len(seq1))
	}

	// stream-2 should exist
	seq2, _ := idx.GetStreamSequences("stream-2")
	if len(seq2) != 1 {
		t.Errorf("stream-2 should have 1 event after rebuild, got %d", len(seq2))
	}
}

func TestIndex_AddEvent(t *testing.T) {
	idx := NewIndex()

	event := &Event{
		StreamID:       "stream-1",
		GlobalSequence: 1,
		EventType:      "TestEvent",
		Version:        1,
	}

	err := idx.AddEvent(event)
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	// Verify event was added
	sequences, _ := idx.GetStreamSequences("stream-1")
	if len(sequences) != 1 {
		t.Fatalf("Expected 1 sequence, got %d", len(sequences))
	}

	if sequences[0] != 1 {
		t.Errorf("Expected sequence 1, got %d", sequences[0])
	}
}

func TestIndex_AddEvent_MultipleEvents(t *testing.T) {
	idx := NewIndex()

	// Add events one by one
	for i := 1; i <= 5; i++ {
		event := &Event{
			StreamID:       "stream-1",
			GlobalSequence: i,
			EventType:      "TestEvent",
			Version:        i,
		}
		idx.AddEvent(event)
	}

	// Verify all events were added
	sequences, _ := idx.GetStreamSequences("stream-1")
	if len(sequences) != 5 {
		t.Fatalf("Expected 5 sequences, got %d", len(sequences))
	}

	for i := 0; i < 5; i++ {
		if sequences[i] != i+1 {
			t.Errorf("Sequence %d: expected %d, got %d", i, i+1, sequences[i])
		}
	}
}

func TestIndex_AddEvent_NilEvent(t *testing.T) {
	idx := NewIndex()

	err := idx.AddEvent(nil)
	if err == nil {
		t.Fatal("Expected error for nil event, got nil")
	}
}

func TestIndex_AddEvent_EmptyStreamID(t *testing.T) {
	idx := NewIndex()

	event := &Event{
		StreamID:       "",
		GlobalSequence: 1,
		EventType:      "TestEvent",
		Version:        1,
	}

	err := idx.AddEvent(event)
	if err == nil {
		t.Fatal("Expected error for empty StreamID, got nil")
	}
}

func TestIndex_AddEvent_InvalidGlobalSequence(t *testing.T) {
	idx := NewIndex()

	event := &Event{
		StreamID:       "stream-1",
		GlobalSequence: 0,
		EventType:      "TestEvent",
		Version:        1,
	}

	err := idx.AddEvent(event)
	if err == nil {
		t.Fatal("Expected error for invalid GlobalSequence, got nil")
	}
}

func TestIndex_GetStreamSequences_NonExistentStream(t *testing.T) {
	idx := NewIndex()

	sequences, err := idx.GetStreamSequences("non-existent")
	if err != nil {
		t.Fatalf("GetStreamSequences failed: %v", err)
	}

	// Should return empty slice, not nil
	if sequences == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(sequences) != 0 {
		t.Errorf("Expected 0 sequences, got %d", len(sequences))
	}
}

func TestIndex_GetStreamSequences_ReturnsACopy(t *testing.T) {
	idx := NewIndex()

	event := &Event{
		StreamID:       "stream-1",
		GlobalSequence: 1,
		EventType:      "TestEvent",
		Version:        1,
	}
	idx.AddEvent(event)

	// Get sequences
	sequences1, _ := idx.GetStreamSequences("stream-1")

	// Modify the returned slice
	sequences1[0] = 999

	// Get sequences again
	sequences2, _ := idx.GetStreamSequences("stream-1")

	// Original should be unchanged
	if sequences2[0] != 1 {
		t.Errorf("Expected original sequence 1, got %d (returned slice was not a copy)", sequences2[0])
	}
}

func TestIndex_GetStreamVersion(t *testing.T) {
	idx := NewIndex()

	// Add events to stream
	for i := 1; i <= 3; i++ {
		event := &Event{
			StreamID:       "stream-1",
			GlobalSequence: i,
			EventType:      "TestEvent",
			Version:        i,
		}
		idx.AddEvent(event)
	}

	version, err := idx.GetStreamVersion("stream-1")
	if err != nil {
		t.Fatalf("GetStreamVersion failed: %v", err)
	}

	if version != 3 {
		t.Errorf("Expected version 3, got %d", version)
	}
}

func TestIndex_GetStreamVersion_NonExistentStream(t *testing.T) {
	idx := NewIndex()

	version, err := idx.GetStreamVersion("non-existent")
	if err != nil {
		t.Fatalf("GetStreamVersion failed: %v", err)
	}

	if version != 0 {
		t.Errorf("Expected version 0 for non-existent stream, got %d", version)
	}
}

func TestIndex_GetAllStreams(t *testing.T) {
	idx := NewIndex()

	// Add events to multiple streams
	streams := []string{"stream-1", "stream-2", "stream-3"}
	for i, streamID := range streams {
		event := &Event{
			StreamID:       streamID,
			GlobalSequence: i + 1,
			EventType:      "TestEvent",
			Version:        1,
		}
		idx.AddEvent(event)
	}

	allStreams := idx.GetAllStreams()

	if len(allStreams) != 3 {
		t.Fatalf("Expected 3 streams, got %d", len(allStreams))
	}

	// Verify all expected streams are present
	streamMap := make(map[string]bool)
	for _, s := range allStreams {
		streamMap[s] = true
	}

	for _, expectedStream := range streams {
		if !streamMap[expectedStream] {
			t.Errorf("Expected stream %s not found in GetAllStreams()", expectedStream)
		}
	}
}

func TestIndex_GetAllStreams_Empty(t *testing.T) {
	idx := NewIndex()

	allStreams := idx.GetAllStreams()

	// Should return empty slice, not nil
	if allStreams == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(allStreams) != 0 {
		t.Errorf("Expected 0 streams, got %d", len(allStreams))
	}
}

func TestIndex_GetEventCount(t *testing.T) {
	idx := NewIndex()

	// Add events to multiple streams
	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 2, EventType: "Event", Version: 2},
		{StreamID: "stream-2", GlobalSequence: 3, EventType: "Event", Version: 1},
		{StreamID: "stream-3", GlobalSequence: 4, EventType: "Event", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 5, EventType: "Event", Version: 3},
	}

	idx.BuildFromEvents(events)

	count := idx.GetEventCount()
	if count != 5 {
		t.Errorf("Expected event count 5, got %d", count)
	}
}

func TestIndex_GetEventCount_Empty(t *testing.T) {
	idx := NewIndex()

	count := idx.GetEventCount()
	if count != 0 {
		t.Errorf("Expected event count 0, got %d", count)
	}
}

func TestIndex_Clear(t *testing.T) {
	idx := NewIndex()

	// Add some events
	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-2", GlobalSequence: 2, EventType: "Event", Version: 1},
	}
	idx.BuildFromEvents(events)

	// Verify events exist
	if idx.GetEventCount() != 2 {
		t.Fatal("Setup failed: expected 2 events")
	}

	// Clear the index
	idx.Clear()

	// Verify index is empty
	if idx.GetEventCount() != 0 {
		t.Errorf("Expected event count 0 after Clear(), got %d", idx.GetEventCount())
	}

	allStreams := idx.GetAllStreams()
	if len(allStreams) != 0 {
		t.Errorf("Expected 0 streams after Clear(), got %d", len(allStreams))
	}
}

func TestIndex_ConcurrentReads(t *testing.T) {
	idx := NewIndex()

	// Build index with some events
	events := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 2, EventType: "Event", Version: 2},
		{StreamID: "stream-2", GlobalSequence: 3, EventType: "Event", Version: 1},
	}
	idx.BuildFromEvents(events)

	// Perform concurrent reads
	concurrency := 50
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Perform multiple read operations
			for j := 0; j < 100; j++ {
				if goroutineID%2 == 0 {
					idx.GetStreamSequences("stream-1")
				} else {
					idx.GetStreamVersion("stream-2")
				}
				idx.GetAllStreams()
				idx.GetEventCount()
			}
		}(i)
	}

	wg.Wait()

	// Verify index is still correct after concurrent reads
	seq1, _ := idx.GetStreamSequences("stream-1")
	if len(seq1) != 2 {
		t.Errorf("Concurrent reads corrupted index: expected 2 events in stream-1, got %d", len(seq1))
	}
}

func TestIndex_ConcurrentWrites(t *testing.T) {
	idx := NewIndex()

	// Add events concurrently
	concurrency := 10
	eventsPerGoroutine := 10
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				event := &Event{
					StreamID:       "stream-1",
					GlobalSequence: goroutineID*eventsPerGoroutine + j + 1,
					EventType:      "TestEvent",
					Version:        j + 1,
				}
				idx.AddEvent(event)
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were added
	sequences, _ := idx.GetStreamSequences("stream-1")
	expectedCount := concurrency * eventsPerGoroutine

	if len(sequences) != expectedCount {
		t.Errorf("Expected %d events after concurrent writes, got %d", expectedCount, len(sequences))
	}
}

func TestIndex_ConcurrentMixedOperations(t *testing.T) {
	idx := NewIndex()

	// Start with some events
	initialEvents := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-2", GlobalSequence: 2, EventType: "Event", Version: 1},
	}
	idx.BuildFromEvents(initialEvents)

	concurrency := 20
	var wg sync.WaitGroup

	// Mix of readers and writers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < 50; j++ {
				if goroutineID%3 == 0 {
					// Writer
					event := &Event{
						StreamID:       "stream-3",
						GlobalSequence: goroutineID*100 + j + 100,
						EventType:      "TestEvent",
						Version:        j + 1,
					}
					idx.AddEvent(event)
				} else {
					// Reader
					idx.GetStreamSequences("stream-1")
					idx.GetStreamVersion("stream-2")
					idx.GetEventCount()
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify index is still functional
	count := idx.GetEventCount()
	if count < 2 {
		t.Errorf("Index corrupted: expected at least 2 events, got %d", count)
	}
}

func TestIndex_BuildThenAdd(t *testing.T) {
	idx := NewIndex()

	// Build from initial events
	initialEvents := []*Event{
		{StreamID: "stream-1", GlobalSequence: 1, EventType: "Event", Version: 1},
		{StreamID: "stream-1", GlobalSequence: 2, EventType: "Event", Version: 2},
	}
	idx.BuildFromEvents(initialEvents)

	// Add more events incrementally
	newEvent := &Event{
		StreamID:       "stream-1",
		GlobalSequence: 3,
		EventType:      "Event",
		Version:        3,
	}
	idx.AddEvent(newEvent)

	// Verify all events are present
	sequences, _ := idx.GetStreamSequences("stream-1")
	if len(sequences) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(sequences))
	}

	expected := []int{1, 2, 3}
	for i, seq := range sequences {
		if seq != expected[i] {
			t.Errorf("Sequence %d: expected %d, got %d", i, expected[i], seq)
		}
	}
}

func TestIndex_LargeNumberOfStreams(t *testing.T) {
	idx := NewIndex()

	// Create many streams with one event each
	streamCount := 1000
	events := make([]*Event, streamCount)

	for i := 0; i < streamCount; i++ {
		events[i] = &Event{
			StreamID:       string(rune('a' + i%26)) + string(rune('0'+i/26)),
			GlobalSequence: i + 1,
			EventType:      "Event",
			Version:        1,
		}
	}

	err := idx.BuildFromEvents(events)
	if err != nil {
		t.Fatalf("BuildFromEvents failed with %d streams: %v", streamCount, err)
	}

	// Verify count
	allStreams := idx.GetAllStreams()
	if len(allStreams) != streamCount {
		t.Errorf("Expected %d streams, got %d", streamCount, len(allStreams))
	}

	totalEvents := idx.GetEventCount()
	if totalEvents != streamCount {
		t.Errorf("Expected %d total events, got %d", streamCount, totalEvents)
	}
}

func TestIndex_LargeNumberOfEventsInSingleStream(t *testing.T) {
	idx := NewIndex()

	// Create one stream with many events
	eventCount := 1000
	events := make([]*Event, eventCount)

	for i := 0; i < eventCount; i++ {
		events[i] = &Event{
			StreamID:       "stream-1",
			GlobalSequence: i + 1,
			EventType:      "Event",
			Version:        i + 1,
		}
	}

	err := idx.BuildFromEvents(events)
	if err != nil {
		t.Fatalf("BuildFromEvents failed with %d events: %v", eventCount, err)
	}

	// Verify count
	sequences, _ := idx.GetStreamSequences("stream-1")
	if len(sequences) != eventCount {
		t.Errorf("Expected %d sequences, got %d", eventCount, len(sequences))
	}

	version, _ := idx.GetStreamVersion("stream-1")
	if version != eventCount {
		t.Errorf("Expected version %d, got %d", eventCount, version)
	}
}
