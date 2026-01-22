package eventstore

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	// EventLogFileName is the name of the file where events are stored
	EventLogFileName = "events.log"
)

// DiskStorage provides durable, append-only persistence of events to disk.
// Events are stored in a single log file with a length-prefixed format:
// [4 bytes: length][N bytes: JSON event]
//
// This format enables sequential reading without parsing JSON first.
type DiskStorage struct {
	filePath string
	file     *os.File
	mu       sync.Mutex // Protects file writes
}

// NewDiskStorage creates a new DiskStorage instance.
// It creates the data directory if it doesn't exist and opens/creates the log file.
//
// Parameters:
//   - dataDir: Directory where the event log file will be stored
//
// Returns:
//   - *DiskStorage: Initialized storage instance
//   - error: If directory creation or file opening fails
func NewDiskStorage(dataDir string) (*DiskStorage, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open or create the log file (append mode, create if not exists)
	filePath := filepath.Join(dataDir, EventLogFileName)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open event log file: %w", err)
	}

	return &DiskStorage{
		filePath: filePath,
		file:     file,
	}, nil
}

// Append writes an event to the end of the log file and syncs to disk.
// Format: [4-byte length (big-endian)][JSON event bytes]
//
// This method is thread-safe and guarantees durability through file.Sync().
//
// Parameters:
//   - event: The event to append
//
// Returns:
//   - error: If marshaling, writing, or syncing fails
func (ds *DiskStorage) Append(event *Event) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Marshal event to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Prepare length prefix (4 bytes, big-endian)
	length := uint32(len(jsonData))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)

	// Write length prefix
	if _, err := ds.file.Write(lengthBytes); err != nil {
		return fmt.Errorf("failed to write length prefix: %w", err)
	}

	// Write JSON data
	if _, err := ds.file.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write event data: %w", err)
	}

	// Sync to disk for durability (ensures data is persisted)
	if err := ds.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync to disk: %w", err)
	}

	return nil
}

// ReadAll reads all events from the log file.
// This method is called on startup to rebuild the in-memory index.
//
// Returns:
//   - []*Event: Slice of all events in the log (in order)
//   - error: If file seeking, reading, or unmarshaling fails
func (ds *DiskStorage) ReadAll() ([]*Event, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Seek to beginning of file
	if _, err := ds.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start of file: %w", err)
	}

	// Initialize with empty slice (not nil) so empty files return []
	events := make([]*Event, 0)
	lengthBytes := make([]byte, 4)

	for {
		// Read length prefix
		n, err := io.ReadFull(ds.file, lengthBytes)
		if err == io.EOF {
			// End of file - normal exit
			break
		}
		if err == io.ErrUnexpectedEOF {
			// Partial length prefix - corrupted file
			return nil, fmt.Errorf("corrupted log file: incomplete length prefix (read %d bytes)", n)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read length prefix: %w", err)
		}

		// Parse length
		length := binary.BigEndian.Uint32(lengthBytes)

		// Sanity check: prevent reading unreasonably large events (>10MB)
		if length > 10*1024*1024 {
			return nil, fmt.Errorf("corrupted log file: event length too large (%d bytes)", length)
		}

		// Read JSON data
		jsonData := make([]byte, length)
		if _, err := io.ReadFull(ds.file, jsonData); err != nil {
			return nil, fmt.Errorf("failed to read event data (expected %d bytes): %w", length, err)
		}

		// Unmarshal event
		var event Event
		if err := json.Unmarshal(jsonData, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %w", err)
		}

		events = append(events, &event)
	}

	return events, nil
}

// Close closes the underlying file handle.
// Should be called when the storage is no longer needed.
//
// Returns:
//   - error: If closing the file fails
func (ds *DiskStorage) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.file == nil {
		return nil
	}

	if err := ds.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	ds.file = nil
	return nil
}

// FilePath returns the path to the event log file.
// Useful for testing and debugging.
func (ds *DiskStorage) FilePath() string {
	return ds.filePath
}
