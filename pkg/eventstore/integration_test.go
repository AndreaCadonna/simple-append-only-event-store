package eventstore

import (
	"os"
	"testing"
)

// Integration tests verify the entire EventStore system working together:
// Event creation -> DiskStorage persistence -> Index building -> Queries

func TestIntegration_BasicWorkflow(t *testing.T) {
	// Setup
	tempDir, err := os.MkdirTemp("", "eventstore-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create event store
	store, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Append events
	event1, err := store.Append("account-123", "AccountOpened", map[string]interface{}{
		"accountId": "account-123",
		"owner":     "Alice",
	})
	if err != nil {
		t.Fatalf("Failed to append event1: %v", err)
	}

	event2, err := store.Append("account-123", "MoneyDeposited", map[string]interface{}{
		"accountId": "account-123",
		"amount":    10000,
	})
	if err != nil {
		t.Fatalf("Failed to append event2: %v", err)
	}

	event3, err := store.Append("account-456", "AccountOpened", map[string]interface{}{
		"accountId": "account-456",
		"owner":     "Bob",
	})
	if err != nil {
		t.Fatalf("Failed to append event3: %v", err)
	}

	// Query by stream
	aliceEvents, err := store.GetStream("account-123")
	if err != nil {
		t.Fatalf("Failed to get stream: %v", err)
	}

	if len(aliceEvents) != 2 {
		t.Fatalf("Expected 2 events for account-123, got %d", len(aliceEvents))
	}

	if aliceEvents[0].ID != event1.ID || aliceEvents[1].ID != event2.ID {
		t.Error("Events not in correct order or IDs don't match")
	}

	// Query all events
	allEvents, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("Failed to get all events: %v", err)
	}

	if len(allEvents) != 3 {
		t.Fatalf("Expected 3 total events, got %d", len(allEvents))
	}

	// Verify global ordering
	if allEvents[0].GlobalSequence != 1 || allEvents[1].GlobalSequence != 2 || allEvents[2].GlobalSequence != 3 {
		t.Error("Global sequences not correct")
	}

	if allEvents[2].ID != event3.ID {
		t.Error("Third event ID doesn't match")
	}
}

func TestIntegration_PersistenceAndRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "eventstore-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Phase 1: Create store and append events
	store1, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	event1, _ := store1.Append("stream-1", "Event1", map[string]interface{}{"phase": "1"})
	event2, _ := store1.Append("stream-2", "Event2", map[string]interface{}{"phase": "1"})
	store1.Close()

	// Phase 2: Reopen and verify recovery
	store2, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	// Verify events were recovered
	allEvents, _ := store2.GetAllEvents()
	if len(allEvents) != 2 {
		t.Fatalf("Expected 2 events after recovery, got %d", len(allEvents))
	}

	// Append more events
	event3, _ := store2.Append("stream-1", "Event3", map[string]interface{}{"phase": "2"})
	store2.Close()

	// Phase 3: Reopen again and verify everything
	store3, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store3: %v", err)
	}
	defer store3.Close()

	// Verify all events present
	allEvents, _ = store3.GetAllEvents()
	if len(allEvents) != 3 {
		t.Fatalf("Expected 3 events after second recovery, got %d", len(allEvents))
	}

	// Verify stream-1 has correct events
	stream1Events, _ := store3.GetStream("stream-1")
	if len(stream1Events) != 2 {
		t.Fatalf("Expected 2 events in stream-1, got %d", len(stream1Events))
	}

	if stream1Events[0].ID != event1.ID || stream1Events[1].ID != event3.ID {
		t.Error("stream-1 events don't match expected IDs")
	}

	// Verify versioning is correct
	if stream1Events[0].Version != 1 || stream1Events[1].Version != 2 {
		t.Error("stream-1 versions not correct after recovery")
	}

	// Verify stream-2
	stream2Events, _ := store3.GetStream("stream-2")
	if len(stream2Events) != 1 {
		t.Fatalf("Expected 1 event in stream-2, got %d", len(stream2Events))
	}

	if stream2Events[0].ID != event2.ID {
		t.Error("stream-2 event doesn't match expected ID")
	}
}

func TestIntegration_EventSourcingPattern(t *testing.T) {
	// This test demonstrates the event sourcing pattern:
	// Building current state by replaying events

	tempDir, err := os.MkdirTemp("", "eventstore-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Simulate bank account events
	accountID := "account-123"

	// Account opened
	store.Append(accountID, "AccountOpened", map[string]interface{}{
		"accountId": accountID,
		"owner":     "Alice",
	})

	// Money deposited
	store.Append(accountID, "MoneyDeposited", map[string]interface{}{
		"accountId": accountID,
		"amount":    10000, // $100.00
	})

	// Money withdrawn
	store.Append(accountID, "MoneyWithdrawn", map[string]interface{}{
		"accountId": accountID,
		"amount":    2500, // $25.00
	})

	// Money deposited again
	store.Append(accountID, "MoneyDeposited", map[string]interface{}{
		"accountId": accountID,
		"amount":    5000, // $50.00
	})

	// Replay events to calculate current balance
	events, err := store.GetStream(accountID)
	if err != nil {
		t.Fatalf("Failed to get stream: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("Expected 4 events, got %d", len(events))
	}

	// Verify event order and types
	expectedTypes := []string{"AccountOpened", "MoneyDeposited", "MoneyWithdrawn", "MoneyDeposited"}
	for i, event := range events {
		if event.EventType != expectedTypes[i] {
			t.Errorf("Event %d: expected type %s, got %s", i, expectedTypes[i], event.EventType)
		}
		if event.Version != i+1 {
			t.Errorf("Event %d: expected version %d, got %d", i, i+1, event.Version)
		}
	}

	// Expected balance: 0 + 10000 - 2500 + 5000 = 12500 ($125.00)
	// This would be calculated by replaying events in the domain layer
}

func TestIntegration_MultipleStreamsInterleaved(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "eventstore-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Append events to multiple streams in interleaved fashion
	events := make([]*Event, 15)
	events[0], _ = store.Append("order-1", "OrderCreated", nil)
	events[1], _ = store.Append("order-2", "OrderCreated", nil)
	events[2], _ = store.Append("order-1", "ItemAdded", nil)
	events[3], _ = store.Append("order-3", "OrderCreated", nil)
	events[4], _ = store.Append("order-2", "ItemAdded", nil)
	events[5], _ = store.Append("order-1", "ItemAdded", nil)
	events[6], _ = store.Append("order-2", "OrderSubmitted", nil)
	events[7], _ = store.Append("order-1", "OrderSubmitted", nil)
	events[8], _ = store.Append("order-3", "ItemAdded", nil)
	events[9], _ = store.Append("order-1", "OrderShipped", nil)
	events[10], _ = store.Append("order-3", "ItemAdded", nil)
	events[11], _ = store.Append("order-2", "OrderShipped", nil)
	events[12], _ = store.Append("order-3", "OrderSubmitted", nil)
	events[13], _ = store.Append("order-1", "OrderDelivered", nil)
	events[14], _ = store.Append("order-3", "OrderShipped", nil)

	// Verify global ordering
	allEvents, _ := store.GetAllEvents()
	if len(allEvents) != 15 {
		t.Fatalf("Expected 15 total events, got %d", len(allEvents))
	}

	for i, event := range allEvents {
		if event.GlobalSequence != i+1 {
			t.Errorf("Event %d: expected GlobalSequence %d, got %d", i, i+1, event.GlobalSequence)
		}
	}

	// Verify each stream has correct events
	order1Events, _ := store.GetStream("order-1")
	if len(order1Events) != 6 {
		t.Fatalf("Expected 6 events for order-1, got %d", len(order1Events))
	}

	order2Events, _ := store.GetStream("order-2")
	if len(order2Events) != 4 {
		t.Fatalf("Expected 4 events for order-2, got %d", len(order2Events))
	}

	order3Events, _ := store.GetStream("order-3")
	if len(order3Events) != 5 {
		t.Fatalf("Expected 5 events for order-3, got %d", len(order3Events))
	}

	// Verify stream versioning
	for i, event := range order1Events {
		if event.Version != i+1 {
			t.Errorf("order-1 event %d: expected version %d, got %d", i, i+1, event.Version)
		}
	}

	// Verify specific events
	if order1Events[0].EventType != "OrderCreated" {
		t.Error("order-1 first event should be OrderCreated")
	}
	if order1Events[5].EventType != "OrderDelivered" {
		t.Error("order-1 last event should be OrderDelivered")
	}
}

func TestIntegration_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "eventstore-stress-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Append 10,000 events across 100 streams
	streamCount := 100
	eventsPerStream := 100
	totalEvents := streamCount * eventsPerStream

	for s := 1; s <= streamCount; s++ {
		streamID := string(rune('a' + (s-1)%26)) + string(rune('a' + (s-1)/26))
		for e := 1; e <= eventsPerStream; e++ {
			_, err := store.Append(streamID, "Event", map[string]int{
				"stream": s,
				"event":  e,
			})
			if err != nil {
				t.Fatalf("Failed to append event %d to stream %s: %v", e, streamID, err)
			}
		}
	}

	// Verify total count
	allEvents, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("Failed to get all events: %v", err)
	}

	if len(allEvents) != totalEvents {
		t.Fatalf("Expected %d total events, got %d", totalEvents, len(allEvents))
	}

	// Verify a few random streams
	testStreams := []string{"aa", "ba", "ea", "ja"}
	for _, streamID := range testStreams {
		events, err := store.GetStream(streamID)
		if err != nil {
			t.Fatalf("Failed to get stream %s: %v", streamID, err)
		}

		if len(events) != eventsPerStream {
			t.Errorf("Stream %s: expected %d events, got %d", streamID, eventsPerStream, len(events))
		}
	}

	// Close and reopen to test persistence with large dataset
	store.Close()

	store2, err := NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen event store: %v", err)
	}
	defer store2.Close()

	// Verify all events still present
	allEvents2, _ := store2.GetAllEvents()
	if len(allEvents2) != totalEvents {
		t.Errorf("Expected %d events after reopen, got %d", totalEvents, len(allEvents2))
	}

	// Append more events to verify store is functional
	newEvent, err := store2.Append("test", "Event", nil)
	if err != nil {
		t.Fatalf("Failed to append after reopen: %v", err)
	}

	if newEvent.GlobalSequence != totalEvents+1 {
		t.Errorf("Expected GlobalSequence %d after reopen, got %d", totalEvents+1, newEvent.GlobalSequence)
	}
}
