package eventstore

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Helper function to create a temporary directory for testing
func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "eventstore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// Helper function to clean up temporary directory
func cleanupDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

func TestNewDiskStorage_CreatesDirectory(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "eventstore-test-new-dir")
	defer os.RemoveAll(tempDir)

	// Ensure directory doesn't exist
	os.RemoveAll(tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Verify directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Data directory was not created")
	}

	// Verify file path is correct
	expectedPath := filepath.Join(tempDir, EventLogFileName)
	if storage.FilePath() != expectedPath {
		t.Errorf("Expected file path %s, got %s", expectedPath, storage.FilePath())
	}
}

func TestDiskStorage_AppendAndReadSingleEvent(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create and append an event
	event, err := NewEvent("stream-1", "TestEvent", map[string]string{"key": "value"}, 1, 1)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	if err := storage.Append(event); err != nil {
		t.Fatalf("Failed to append event: %v", err)
	}

	// Read events back
	events, err := storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}

	// Verify we got one event back
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	// Verify event contents
	readEvent := events[0]
	if readEvent.ID != event.ID {
		t.Errorf("ID mismatch: expected %s, got %s", event.ID, readEvent.ID)
	}
	if readEvent.StreamID != event.StreamID {
		t.Errorf("StreamID mismatch: expected %s, got %s", event.StreamID, readEvent.StreamID)
	}
	if readEvent.EventType != event.EventType {
		t.Errorf("EventType mismatch: expected %s, got %s", event.EventType, readEvent.EventType)
	}
	if readEvent.Version != event.Version {
		t.Errorf("Version mismatch: expected %d, got %d", event.Version, readEvent.Version)
	}
	if readEvent.GlobalSequence != event.GlobalSequence {
		t.Errorf("GlobalSequence mismatch: expected %d, got %d", event.GlobalSequence, readEvent.GlobalSequence)
	}
}

func TestDiskStorage_AppendMultipleEvents(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create and append multiple events
	eventCount := 10
	originalEvents := make([]*Event, eventCount)

	for i := 0; i < eventCount; i++ {
		event, err := NewEvent("stream-1", "TestEvent", map[string]int{"index": i}, i+1, i+1)
		if err != nil {
			t.Fatalf("Failed to create event %d: %v", i, err)
		}
		originalEvents[i] = event

		if err := storage.Append(event); err != nil {
			t.Fatalf("Failed to append event %d: %v", i, err)
		}
	}

	// Read events back
	events, err := storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}

	// Verify count
	if len(events) != eventCount {
		t.Fatalf("Expected %d events, got %d", eventCount, len(events))
	}

	// Verify all events are correct and in order
	for i, event := range events {
		if event.ID != originalEvents[i].ID {
			t.Errorf("Event %d: ID mismatch", i)
		}
		if event.GlobalSequence != i+1 {
			t.Errorf("Event %d: expected GlobalSequence %d, got %d", i, i+1, event.GlobalSequence)
		}
	}
}

func TestDiskStorage_PersistenceAcrossReopening(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	// First storage instance - write events
	storage1, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	event1, _ := NewEvent("stream-1", "Event1", nil, 1, 1)
	event2, _ := NewEvent("stream-2", "Event2", nil, 1, 2)

	storage1.Append(event1)
	storage1.Append(event2)
	storage1.Close()

	// Second storage instance - read events
	storage2, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen storage: %v", err)
	}
	defer storage2.Close()

	events, err := storage2.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}

	// Verify events persisted
	if len(events) != 2 {
		t.Fatalf("Expected 2 events after reopening, got %d", len(events))
	}

	if events[0].ID != event1.ID {
		t.Error("First event doesn't match after reopening")
	}
	if events[1].ID != event2.ID {
		t.Error("Second event doesn't match after reopening")
	}
}

func TestDiskStorage_EmptyFile(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Read from empty file
	events, err := storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read from empty file: %v", err)
	}

	// Should return empty slice, not nil
	if events == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events in empty file, got %d", len(events))
	}
}

func TestDiskStorage_CorruptedFile_IncompleteLength(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Write a valid event first
	event, _ := NewEvent("stream-1", "TestEvent", nil, 1, 1)
	storage.Append(event)
	storage.Close()

	// Corrupt the file by appending incomplete length prefix
	filePath := storage.FilePath()
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for corruption: %v", err)
	}
	file.Write([]byte{0x00, 0x01}) // Only 2 bytes instead of 4
	file.Close()

	// Try to read - should fail
	storage2, _ := NewDiskStorage(tempDir)
	defer storage2.Close()

	_, err = storage2.ReadAll()
	if err == nil {
		t.Fatal("Expected error when reading corrupted file, got nil")
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

func TestDiskStorage_CorruptedFile_IncompleteData(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	storage.Close()

	// Manually write corrupted data (length says 100 bytes, but we write less)
	filePath := storage.FilePath()
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, 100) // Claim 100 bytes
	file.Write(lengthBytes)
	file.Write([]byte("{\"partial\": \"data\"}")) // But only write partial data
	file.Close()

	// Try to read - should fail
	storage2, _ := NewDiskStorage(tempDir)
	defer storage2.Close()

	_, err = storage2.ReadAll()
	if err == nil {
		t.Fatal("Expected error when reading corrupted file, got nil")
	}
}

func TestDiskStorage_CorruptedFile_InvalidJSON(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	storage.Close()

	// Write invalid JSON
	filePath := storage.FilePath()
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	invalidJSON := []byte("{not valid json}")
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(invalidJSON)))
	file.Write(lengthBytes)
	file.Write(invalidJSON)
	file.Close()

	// Try to read - should fail
	storage2, _ := NewDiskStorage(tempDir)
	defer storage2.Close()

	_, err = storage2.ReadAll()
	if err == nil {
		t.Fatal("Expected error when reading invalid JSON, got nil")
	}
}

func TestDiskStorage_ConcurrentAppends(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Append events concurrently
	concurrency := 10
	eventsPerGoroutine := 10
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				event, err := NewEvent(
					"stream-1",
					"TestEvent",
					map[string]int{"goroutine": goroutineID, "index": j},
					j+1,
					goroutineID*eventsPerGoroutine+j+1,
				)
				if err != nil {
					t.Errorf("Failed to create event: %v", err)
					return
				}

				if err := storage.Append(event); err != nil {
					t.Errorf("Failed to append event: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Read and verify all events were written
	events, err := storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}

	expectedCount := concurrency * eventsPerGoroutine
	if len(events) != expectedCount {
		t.Errorf("Expected %d events, got %d", expectedCount, len(events))
	}

	// Verify all events are valid (have IDs, can unmarshal data, etc.)
	for i, event := range events {
		if event.ID == "" {
			t.Errorf("Event %d has empty ID", i)
		}
		if event.StreamID == "" {
			t.Errorf("Event %d has empty StreamID", i)
		}
	}
}

func TestDiskStorage_AppendAfterRead(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Append event
	event1, _ := NewEvent("stream-1", "Event1", nil, 1, 1)
	storage.Append(event1)

	// Read events
	events, _ := storage.ReadAll()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event after first read, got %d", len(events))
	}

	// Append another event
	event2, _ := NewEvent("stream-1", "Event2", nil, 2, 2)
	storage.Append(event2)

	// Read again - should get both events
	events, err = storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("Expected 2 events after second read, got %d", len(events))
	}

	if events[0].ID != event1.ID || events[1].ID != event2.ID {
		t.Error("Events not in correct order")
	}
}

func TestDiskStorage_LargeEvent(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create event with large data payload (but under 10MB limit)
	largeData := make(map[string]string)
	for i := 0; i < 100; i++ {
		largeData[string(rune('a'+i%26))] = string(make([]byte, 1000)) // 100 * 1KB = 100KB
	}

	event, err := NewEvent("stream-1", "LargeEvent", largeData, 1, 1)
	if err != nil {
		t.Fatalf("Failed to create large event: %v", err)
	}

	// Append large event
	if err := storage.Append(event); err != nil {
		t.Fatalf("Failed to append large event: %v", err)
	}

	// Read back
	events, err := storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read large event: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	// Verify the large event was stored correctly
	if events[0].ID != event.ID {
		t.Error("Large event ID mismatch")
	}
}

func TestDiskStorage_MultipleStreams(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Append events from different streams
	event1, _ := NewEvent("stream-1", "Event", nil, 1, 1)
	event2, _ := NewEvent("stream-2", "Event", nil, 1, 2)
	event3, _ := NewEvent("stream-1", "Event", nil, 2, 3)
	event4, _ := NewEvent("stream-3", "Event", nil, 1, 4)

	storage.Append(event1)
	storage.Append(event2)
	storage.Append(event3)
	storage.Append(event4)

	// Read all events
	events, err := storage.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("Expected 4 events, got %d", len(events))
	}

	// Verify events are in global order (not grouped by stream)
	if events[0].GlobalSequence != 1 || events[1].GlobalSequence != 2 ||
		events[2].GlobalSequence != 3 || events[3].GlobalSequence != 4 {
		t.Error("Events not in correct global order")
	}

	// Verify stream IDs
	if events[0].StreamID != "stream-1" || events[1].StreamID != "stream-2" ||
		events[2].StreamID != "stream-1" || events[3].StreamID != "stream-3" {
		t.Error("Stream IDs don't match expected values")
	}
}

func TestDiskStorage_Close(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Close storage
	if err := storage.Close(); err != nil {
		t.Fatalf("Failed to close storage: %v", err)
	}

	// Closing again should not error
	if err := storage.Close(); err != nil {
		t.Errorf("Closing already-closed storage returned error: %v", err)
	}
}

func TestDiskStorage_FileFormat(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupDir(t, tempDir)

	storage, err := NewDiskStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create a simple event
	event, err := NewEvent("test-stream", "TestEvent", map[string]string{"key": "value"}, 1, 1)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Append event
	if err := storage.Append(event); err != nil {
		t.Fatalf("Failed to append event: %v", err)
	}
	storage.Close()

	// Manually read the file to verify format
	filePath := storage.FilePath()
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Should have at least 4 bytes for length prefix
	if len(fileData) < 4 {
		t.Fatalf("File too small: %d bytes", len(fileData))
	}

	// Read length prefix
	length := binary.BigEndian.Uint32(fileData[0:4])

	// Verify length matches actual data
	if int(length) != len(fileData)-4 {
		t.Errorf("Length prefix mismatch: prefix says %d, actual data is %d bytes", length, len(fileData)-4)
	}

	// Verify the JSON is valid
	jsonData := fileData[4:]
	var parsedEvent Event
	if err := json.Unmarshal(jsonData, &parsedEvent); err != nil {
		t.Errorf("Failed to parse JSON from file: %v", err)
	}
}
