package eventstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Event represents an immutable domain event in the event store.
// Events are the fundamental unit of data, representing something that happened in the system.
type Event struct {
	ID             string          `json:"id"`             // Unique identifier (UUID-like)
	StreamID       string          `json:"streamId"`       // Stream identifier (e.g., "account-123")
	EventType      string          `json:"eventType"`      // Type of event (e.g., "AccountOpened")
	Data           json.RawMessage `json:"data"`           // JSON payload of the event
	Timestamp      time.Time       `json:"timestamp"`      // When the event occurred
	Version        int             `json:"version"`        // Position in stream (1, 2, 3...)
	GlobalSequence int             `json:"globalSequence"` // Position in global log (1, 2, 3...)
}

// NewEvent creates a new Event with validation.
// Parameters:
//   - streamID: The stream this event belongs to (must be non-empty)
//   - eventType: The type of event (must be non-empty)
//   - data: The event payload (will be marshaled to JSON if not already)
//   - version: The position of this event in its stream
//   - globalSeq: The position of this event in the global event log
func NewEvent(streamID, eventType string, data interface{}, version, globalSeq int) (*Event, error) {
	// Validate required fields
	if streamID == "" {
		return nil, errors.New("streamID cannot be empty")
	}
	if eventType == "" {
		return nil, errors.New("eventType cannot be empty")
	}
	if version < 1 {
		return nil, fmt.Errorf("version must be >= 1, got %d", version)
	}
	if globalSeq < 1 {
		return nil, fmt.Errorf("globalSequence must be >= 1, got %d", globalSeq)
	}

	// Generate unique ID
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	// Marshal data to JSON
	var jsonData json.RawMessage
	switch v := data.(type) {
	case json.RawMessage:
		jsonData = v
	case []byte:
		// Validate it's valid JSON
		if !json.Valid(v) {
			return nil, errors.New("data is not valid JSON")
		}
		jsonData = json.RawMessage(v)
	case string:
		// Validate it's valid JSON
		if !json.Valid([]byte(v)) {
			return nil, errors.New("data string is not valid JSON")
		}
		jsonData = json.RawMessage(v)
	case nil:
		jsonData = json.RawMessage("{}")
	default:
		// Marshal the data structure to JSON
		marshaled, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
		}
		jsonData = json.RawMessage(marshaled)
	}

	return &Event{
		ID:             id,
		StreamID:       streamID,
		EventType:      eventType,
		Data:           jsonData,
		Timestamp:      time.Now().UTC(),
		Version:        version,
		GlobalSequence: globalSeq,
	}, nil
}

// generateID creates a unique identifier using crypto/rand.
// Format: 32 character hex string (similar to UUID without dashes)
func generateID() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 128 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// MarshalJSON implements json.Marshaler interface.
// This is automatically called by json.Marshal().
func (e *Event) MarshalJSON() ([]byte, error) {
	// Use a type alias to avoid infinite recursion
	type EventAlias Event
	return json.Marshal((*EventAlias)(e))
}

// UnmarshalJSON implements json.Unmarshaler interface.
// This is automatically called by json.Unmarshal().
func (e *Event) UnmarshalJSON(data []byte) error {
	// Use a type alias to avoid infinite recursion
	type EventAlias Event
	alias := (*EventAlias)(e)
	if err := json.Unmarshal(data, alias); err != nil {
		return err
	}
	return nil
}

// String returns a human-readable representation of the event.
func (e *Event) String() string {
	return fmt.Sprintf("Event{ID: %s, Stream: %s, Type: %s, Version: %d, GlobalSeq: %d}",
		e.ID, e.StreamID, e.EventType, e.Version, e.GlobalSequence)
}
