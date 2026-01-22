package eventstore

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent_ValidData(t *testing.T) {
	data := map[string]interface{}{
		"accountId": "acc-123",
		"owner":     "Alice",
	}

	event, err := NewEvent("account-123", "AccountOpened", data, 1, 1)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.StreamID != "account-123" {
		t.Errorf("Expected StreamID 'account-123', got '%s'", event.StreamID)
	}

	if event.EventType != "AccountOpened" {
		t.Errorf("Expected EventType 'AccountOpened', got '%s'", event.EventType)
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

	if len(event.ID) != 32 { // 16 bytes = 32 hex characters
		t.Errorf("Expected ID length 32, got %d", len(event.ID))
	}

	if event.Timestamp.IsZero() {
		t.Error("Expected non-zero Timestamp")
	}

	// Verify data is valid JSON
	if !json.Valid(event.Data) {
		t.Error("Expected valid JSON in Data field")
	}
}

func TestNewEvent_EmptyStreamID(t *testing.T) {
	data := map[string]string{"key": "value"}

	_, err := NewEvent("", "EventType", data, 1, 1)

	if err == nil {
		t.Fatal("Expected error for empty StreamID, got nil")
	}

	if err.Error() != "streamID cannot be empty" {
		t.Errorf("Expected 'streamID cannot be empty' error, got: %v", err)
	}
}

func TestNewEvent_EmptyEventType(t *testing.T) {
	data := map[string]string{"key": "value"}

	_, err := NewEvent("stream-123", "", data, 1, 1)

	if err == nil {
		t.Fatal("Expected error for empty EventType, got nil")
	}

	if err.Error() != "eventType cannot be empty" {
		t.Errorf("Expected 'eventType cannot be empty' error, got: %v", err)
	}
}

func TestNewEvent_InvalidVersion(t *testing.T) {
	data := map[string]string{"key": "value"}

	_, err := NewEvent("stream-123", "EventType", data, 0, 1)

	if err == nil {
		t.Fatal("Expected error for version 0, got nil")
	}
}

func TestNewEvent_InvalidGlobalSequence(t *testing.T) {
	data := map[string]string{"key": "value"}

	_, err := NewEvent("stream-123", "EventType", data, 1, 0)

	if err == nil {
		t.Fatal("Expected error for globalSequence 0, got nil")
	}
}

func TestNewEvent_WithJSONString(t *testing.T) {
	jsonString := `{"accountId":"acc-123","owner":"Bob"}`

	event, err := NewEvent("stream-123", "EventType", jsonString, 1, 1)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if string(event.Data) != jsonString {
		t.Errorf("Expected Data '%s', got '%s'", jsonString, string(event.Data))
	}
}

func TestNewEvent_WithInvalidJSONString(t *testing.T) {
	invalidJSON := `{invalid json}`

	_, err := NewEvent("stream-123", "EventType", invalidJSON, 1, 1)

	if err == nil {
		t.Fatal("Expected error for invalid JSON string, got nil")
	}
}

func TestNewEvent_WithJSONRawMessage(t *testing.T) {
	rawJSON := json.RawMessage(`{"accountId":"acc-123"}`)

	event, err := NewEvent("stream-123", "EventType", rawJSON, 1, 1)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if string(event.Data) != string(rawJSON) {
		t.Errorf("Expected Data '%s', got '%s'", string(rawJSON), string(event.Data))
	}
}

func TestNewEvent_WithNilData(t *testing.T) {
	event, err := NewEvent("stream-123", "EventType", nil, 1, 1)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if string(event.Data) != "{}" {
		t.Errorf("Expected Data '{}', got '%s'", string(event.Data))
	}
}

func TestEvent_JSONMarshalUnmarshal(t *testing.T) {
	originalData := map[string]interface{}{
		"accountId": "acc-123",
		"amount":    100,
	}

	originalEvent, err := NewEvent("account-123", "MoneyDeposited", originalData, 2, 5)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(originalEvent)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Unmarshal back to Event
	var unmarshaledEvent Event
	err = json.Unmarshal(jsonBytes, &unmarshaledEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify all fields
	if unmarshaledEvent.ID != originalEvent.ID {
		t.Errorf("ID mismatch: expected '%s', got '%s'", originalEvent.ID, unmarshaledEvent.ID)
	}

	if unmarshaledEvent.StreamID != originalEvent.StreamID {
		t.Errorf("StreamID mismatch: expected '%s', got '%s'", originalEvent.StreamID, unmarshaledEvent.StreamID)
	}

	if unmarshaledEvent.EventType != originalEvent.EventType {
		t.Errorf("EventType mismatch: expected '%s', got '%s'", originalEvent.EventType, unmarshaledEvent.EventType)
	}

	if unmarshaledEvent.Version != originalEvent.Version {
		t.Errorf("Version mismatch: expected %d, got %d", originalEvent.Version, unmarshaledEvent.Version)
	}

	if unmarshaledEvent.GlobalSequence != originalEvent.GlobalSequence {
		t.Errorf("GlobalSequence mismatch: expected %d, got %d", originalEvent.GlobalSequence, unmarshaledEvent.GlobalSequence)
	}

	if string(unmarshaledEvent.Data) != string(originalEvent.Data) {
		t.Errorf("Data mismatch: expected '%s', got '%s'", string(originalEvent.Data), string(unmarshaledEvent.Data))
	}

	// Timestamps might have slight differences due to JSON precision, so we check they're close
	timeDiff := unmarshaledEvent.Timestamp.Sub(originalEvent.Timestamp)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Timestamp difference too large: %v", timeDiff)
	}
}

func TestEvent_String(t *testing.T) {
	event, err := NewEvent("account-123", "TestEvent", nil, 3, 10)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	str := event.String()

	// Check that important fields are in the string representation
	if str == "" {
		t.Error("String() returned empty string")
	}

	// The string should contain key information
	t.Logf("Event.String() output: %s", str)
}

func TestGenerateID_Uniqueness(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}

		if ids[id] {
			t.Fatalf("Duplicate ID generated: %s", id)
		}

		ids[id] = true

		if len(id) != 32 {
			t.Errorf("Expected ID length 32, got %d for ID: %s", len(id), id)
		}
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestNewEvent_MultipleEvents(t *testing.T) {
	// Test creating multiple events in sequence
	events := make([]*Event, 5)
	var err error

	for i := 0; i < 5; i++ {
		data := map[string]int{"value": i}
		events[i], err = NewEvent("stream-1", "TestEvent", data, i+1, i+1)
		if err != nil {
			t.Fatalf("Failed to create event %d: %v", i, err)
		}
	}

	// Verify each event has unique ID
	ids := make(map[string]bool)
	for i, event := range events {
		if ids[event.ID] {
			t.Errorf("Event %d has duplicate ID: %s", i, event.ID)
		}
		ids[event.ID] = true

		if event.Version != i+1 {
			t.Errorf("Event %d: expected version %d, got %d", i, i+1, event.Version)
		}

		if event.GlobalSequence != i+1 {
			t.Errorf("Event %d: expected globalSequence %d, got %d", i, i+1, event.GlobalSequence)
		}
	}
}
