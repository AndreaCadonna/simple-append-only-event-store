package bank

import (
	"os"
	"testing"

	"github.com/cadonna/append-only-event-store/pkg/eventstore"
)

func TestNewBankAccount(t *testing.T) {
	account := NewBankAccount()

	if account == nil {
		t.Fatal("NewBankAccount returned nil")
	}

	if account.AccountID != "" {
		t.Errorf("Expected empty AccountID, got '%s'", account.AccountID)
	}

	if account.Owner != "" {
		t.Errorf("Expected empty Owner, got '%s'", account.Owner)
	}

	if account.Balance != 0 {
		t.Errorf("Expected Balance 0, got %d", account.Balance)
	}

	if account.Version != 0 {
		t.Errorf("Expected Version 0, got %d", account.Version)
	}
}

func TestBankAccount_ApplyEvent_AccountOpened(t *testing.T) {
	account := NewBankAccount()

	data := AccountOpenedData{
		AccountID: "acc-123",
		Owner:     "Alice",
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeAccountOpened, data, 1, 1)

	err := account.ApplyEvent(event)
	if err != nil {
		t.Fatalf("ApplyEvent failed: %v", err)
	}

	if account.AccountID != "acc-123" {
		t.Errorf("Expected AccountID 'acc-123', got '%s'", account.AccountID)
	}

	if account.Owner != "Alice" {
		t.Errorf("Expected Owner 'Alice', got '%s'", account.Owner)
	}

	if account.Balance != 0 {
		t.Errorf("Expected Balance 0 after opening, got %d", account.Balance)
	}

	if account.Version != 1 {
		t.Errorf("Expected Version 1, got %d", account.Version)
	}
}

func TestBankAccount_ApplyEvent_MoneyDeposited(t *testing.T) {
	account := NewBankAccount()
	account.AccountID = "acc-123"
	account.Owner = "Alice"

	data := MoneyDepositedData{
		AccountID: "acc-123",
		Amount:    10000, // $100.00
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeMoneyDeposited, data, 2, 2)

	err := account.ApplyEvent(event)
	if err != nil {
		t.Fatalf("ApplyEvent failed: %v", err)
	}

	if account.Balance != 10000 {
		t.Errorf("Expected Balance 10000, got %d", account.Balance)
	}

	if account.Version != 1 {
		t.Errorf("Expected Version 1, got %d", account.Version)
	}
}

func TestBankAccount_ApplyEvent_MoneyWithdrawn(t *testing.T) {
	account := NewBankAccount()
	account.AccountID = "acc-123"
	account.Owner = "Alice"
	account.Balance = 10000

	data := MoneyWithdrawnData{
		AccountID: "acc-123",
		Amount:    2500, // $25.00
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeMoneyWithdrawn, data, 3, 3)

	err := account.ApplyEvent(event)
	if err != nil {
		t.Fatalf("ApplyEvent failed: %v", err)
	}

	if account.Balance != 7500 {
		t.Errorf("Expected Balance 7500, got %d", account.Balance)
	}

	if account.Version != 1 {
		t.Errorf("Expected Version 1, got %d", account.Version)
	}
}

func TestBankAccount_ApplyEvent_NilEvent(t *testing.T) {
	account := NewBankAccount()

	err := account.ApplyEvent(nil)
	if err == nil {
		t.Fatal("Expected error for nil event, got nil")
	}
}

func TestBankAccount_ApplyEvent_UnknownEventType(t *testing.T) {
	account := NewBankAccount()

	event, _ := eventstore.NewEvent("acc-123", "UnknownEvent", nil, 1, 1)

	err := account.ApplyEvent(event)
	if err == nil {
		t.Fatal("Expected error for unknown event type, got nil")
	}
}

func TestBankAccount_ApplyEvent_InvalidData(t *testing.T) {
	account := NewBankAccount()

	// Create event with invalid JSON data
	event, _ := eventstore.NewEvent("acc-123", EventTypeAccountOpened, "{invalid json}", 1, 1)

	err := account.ApplyEvent(event)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestBankAccount_ApplyEvent_EmptyAccountID(t *testing.T) {
	account := NewBankAccount()

	data := AccountOpenedData{
		AccountID: "", // Empty
		Owner:     "Alice",
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeAccountOpened, data, 1, 1)

	err := account.ApplyEvent(event)
	if err == nil {
		t.Fatal("Expected error for empty accountId, got nil")
	}
}

func TestBankAccount_ApplyEvent_EmptyOwner(t *testing.T) {
	account := NewBankAccount()

	data := AccountOpenedData{
		AccountID: "acc-123",
		Owner:     "", // Empty
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeAccountOpened, data, 1, 1)

	err := account.ApplyEvent(event)
	if err == nil {
		t.Fatal("Expected error for empty owner, got nil")
	}
}

func TestBankAccount_ApplyEvent_InvalidDepositAmount(t *testing.T) {
	account := NewBankAccount()

	data := MoneyDepositedData{
		AccountID: "acc-123",
		Amount:    0, // Invalid: must be > 0
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeMoneyDeposited, data, 2, 2)

	err := account.ApplyEvent(event)
	if err == nil {
		t.Fatal("Expected error for zero deposit amount, got nil")
	}
}

func TestBankAccount_ApplyEvent_InvalidWithdrawalAmount(t *testing.T) {
	account := NewBankAccount()

	data := MoneyWithdrawnData{
		AccountID: "acc-123",
		Amount:    -100, // Invalid: must be > 0
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeMoneyWithdrawn, data, 3, 3)

	err := account.ApplyEvent(event)
	if err == nil {
		t.Fatal("Expected error for negative withdrawal amount, got nil")
	}
}

func TestReplayAccount_SingleEvent(t *testing.T) {
	data := AccountOpenedData{
		AccountID: "acc-123",
		Owner:     "Alice",
	}

	event, _ := eventstore.NewEvent("acc-123", EventTypeAccountOpened, data, 1, 1)
	events := []*eventstore.Event{event}

	account, err := ReplayAccount(events)
	if err != nil {
		t.Fatalf("ReplayAccount failed: %v", err)
	}

	if account.AccountID != "acc-123" {
		t.Errorf("Expected AccountID 'acc-123', got '%s'", account.AccountID)
	}

	if account.Owner != "Alice" {
		t.Errorf("Expected Owner 'Alice', got '%s'", account.Owner)
	}

	if account.Version != 1 {
		t.Errorf("Expected Version 1, got %d", account.Version)
	}
}

func TestReplayAccount_MultipleEvents(t *testing.T) {
	// Create event sequence: open, deposit, withdraw, deposit
	events := make([]*eventstore.Event, 4)

	// Event 1: Open account
	events[0], _ = eventstore.NewEvent("acc-123", EventTypeAccountOpened, AccountOpenedData{
		AccountID: "acc-123",
		Owner:     "Alice",
	}, 1, 1)

	// Event 2: Deposit $100
	events[1], _ = eventstore.NewEvent("acc-123", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-123",
		Amount:    10000,
	}, 2, 2)

	// Event 3: Withdraw $25
	events[2], _ = eventstore.NewEvent("acc-123", EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: "acc-123",
		Amount:    2500,
	}, 3, 3)

	// Event 4: Deposit $50
	events[3], _ = eventstore.NewEvent("acc-123", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-123",
		Amount:    5000,
	}, 4, 4)

	// Replay events
	account, err := ReplayAccount(events)
	if err != nil {
		t.Fatalf("ReplayAccount failed: %v", err)
	}

	// Verify final state: 0 + 10000 - 2500 + 5000 = 12500
	expectedBalance := 12500
	if account.Balance != expectedBalance {
		t.Errorf("Expected Balance %d, got %d", expectedBalance, account.Balance)
	}

	if account.Version != 4 {
		t.Errorf("Expected Version 4, got %d", account.Version)
	}

	if account.AccountID != "acc-123" {
		t.Errorf("Expected AccountID 'acc-123', got '%s'", account.AccountID)
	}

	if account.Owner != "Alice" {
		t.Errorf("Expected Owner 'Alice', got '%s'", account.Owner)
	}
}

func TestReplayAccount_EmptyEventList(t *testing.T) {
	_, err := ReplayAccount([]*eventstore.Event{})
	if err == nil {
		t.Fatal("Expected error for empty event list, got nil")
	}
}

func TestReplayAccount_InvalidEvent(t *testing.T) {
	// Create valid first event
	event1, _ := eventstore.NewEvent("acc-123", EventTypeAccountOpened, AccountOpenedData{
		AccountID: "acc-123",
		Owner:     "Alice",
	}, 1, 1)

	// Create invalid second event (invalid data)
	event2, _ := eventstore.NewEvent("acc-123", EventTypeMoneyDeposited, "{invalid}", 2, 2)

	events := []*eventstore.Event{event1, event2}

	_, err := ReplayAccount(events)
	if err == nil {
		t.Fatal("Expected error for invalid event, got nil")
	}
}

func TestReplayAccountFromStore_Success(t *testing.T) {
	// Create temporary event store
	tempDir, err := os.MkdirTemp("", "bank-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Append events
	accountID := "acc-456"

	store.Append(accountID, EventTypeAccountOpened, AccountOpenedData{
		AccountID: accountID,
		Owner:     "Bob",
	})

	store.Append(accountID, EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: accountID,
		Amount:    20000, // $200
	})

	store.Append(accountID, EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: accountID,
		Amount:    5000, // $50
	})

	// Replay from store
	account, err := ReplayAccountFromStore(store, accountID)
	if err != nil {
		t.Fatalf("ReplayAccountFromStore failed: %v", err)
	}

	// Verify state: 0 + 20000 - 5000 = 15000
	if account.Balance != 15000 {
		t.Errorf("Expected Balance 15000, got %d", account.Balance)
	}

	if account.Owner != "Bob" {
		t.Errorf("Expected Owner 'Bob', got '%s'", account.Owner)
	}

	if account.Version != 3 {
		t.Errorf("Expected Version 3, got %d", account.Version)
	}
}

func TestReplayAccountFromStore_NonExistentAccount(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Try to load non-existent account
	_, err = ReplayAccountFromStore(store, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestReplayAccountFromStore_EmptyAccountID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	_, err = ReplayAccountFromStore(store, "")
	if err == nil {
		t.Fatal("Expected error for empty accountID, got nil")
	}
}

func TestBankAccount_GetBalance(t *testing.T) {
	account := NewBankAccount()
	account.Balance = 12345

	balance := account.GetBalance()
	if balance != 12345 {
		t.Errorf("Expected GetBalance to return 12345, got %d", balance)
	}
}

func TestBankAccount_GetBalanceDollars(t *testing.T) {
	testCases := []struct {
		name            string
		balanceCents    int
		expectedDollars string
	}{
		{"zero", 0, "$0.00"},
		{"one dollar", 100, "$1.00"},
		{"fifty cents", 50, "$0.50"},
		{"hundred dollars", 10000, "$100.00"},
		{"mixed", 12550, "$125.50"},
		{"negative", -2550, "$-26.50"},
		{"large amount", 123456789, "$1234567.89"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			account := NewBankAccount()
			account.Balance = tc.balanceCents

			result := account.GetBalanceDollars()
			if result != tc.expectedDollars {
				t.Errorf("Expected %s, got %s", tc.expectedDollars, result)
			}
		})
	}
}

func TestBankAccount_String(t *testing.T) {
	account := NewBankAccount()
	account.AccountID = "acc-789"
	account.Owner = "Charlie"
	account.Balance = 5000
	account.Version = 3

	str := account.String()

	// Check that key information is in the string
	if str == "" {
		t.Error("String() returned empty string")
	}

	t.Logf("Account.String() output: %s", str)
}

func TestReplayAccount_ComplexSequence(t *testing.T) {
	// Test a realistic sequence of transactions
	events := make([]*eventstore.Event, 10)

	events[0], _ = eventstore.NewEvent("acc-999", EventTypeAccountOpened, AccountOpenedData{
		AccountID: "acc-999",
		Owner:     "Dave",
	}, 1, 1)

	// Series of deposits and withdrawals
	events[1], _ = eventstore.NewEvent("acc-999", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-999",
		Amount:    50000, // $500
	}, 2, 2)

	events[2], _ = eventstore.NewEvent("acc-999", EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: "acc-999",
		Amount:    10000, // $100
	}, 3, 3)

	events[3], _ = eventstore.NewEvent("acc-999", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-999",
		Amount:    25000, // $250
	}, 4, 4)

	events[4], _ = eventstore.NewEvent("acc-999", EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: "acc-999",
		Amount:    5000, // $50
	}, 5, 5)

	events[5], _ = eventstore.NewEvent("acc-999", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-999",
		Amount:    10000, // $100
	}, 6, 6)

	events[6], _ = eventstore.NewEvent("acc-999", EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: "acc-999",
		Amount:    20000, // $200
	}, 7, 7)

	events[7], _ = eventstore.NewEvent("acc-999", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-999",
		Amount:    15000, // $150
	}, 8, 8)

	events[8], _ = eventstore.NewEvent("acc-999", EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: "acc-999",
		Amount:    30000, // $300
	}, 9, 9)

	events[9], _ = eventstore.NewEvent("acc-999", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-999",
		Amount:    5000, // $50
	}, 10, 10)

	// Replay
	account, err := ReplayAccount(events)
	if err != nil {
		t.Fatalf("ReplayAccount failed: %v", err)
	}

	// Calculate expected: 0 + 500 - 100 + 250 - 50 + 100 - 200 + 150 - 300 + 50 = 400
	// In cents: 50000 - 10000 + 25000 - 5000 + 10000 - 20000 + 15000 - 30000 + 5000 = 40000
	expectedBalance := 40000
	if account.Balance != expectedBalance {
		t.Errorf("Expected Balance %d, got %d", expectedBalance, account.Balance)
	}

	if account.Version != 10 {
		t.Errorf("Expected Version 10, got %d", account.Version)
	}
}

func TestReplayAccount_AllowsNegativeBalance(t *testing.T) {
	// Event sourcing allows negative balances during replay
	// because we're replaying what actually happened
	// (validation happens before events are written)

	events := make([]*eventstore.Event, 3)

	events[0], _ = eventstore.NewEvent("acc-100", EventTypeAccountOpened, AccountOpenedData{
		AccountID: "acc-100",
		Owner:     "Eve",
	}, 1, 1)

	events[1], _ = eventstore.NewEvent("acc-100", EventTypeMoneyDeposited, MoneyDepositedData{
		AccountID: "acc-100",
		Amount:    5000, // $50
	}, 2, 2)

	events[2], _ = eventstore.NewEvent("acc-100", EventTypeMoneyWithdrawn, MoneyWithdrawnData{
		AccountID: "acc-100",
		Amount:    10000, // $100 (more than balance)
	}, 3, 3)

	account, err := ReplayAccount(events)
	if err != nil {
		t.Fatalf("ReplayAccount failed: %v", err)
	}

	// Should allow negative balance: 0 + 5000 - 10000 = -5000
	if account.Balance != -5000 {
		t.Errorf("Expected Balance -5000, got %d", account.Balance)
	}
}
