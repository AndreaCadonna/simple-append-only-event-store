package bank

import (
	"os"
	"sync"
	"testing"

	"github.com/cadonna/append-only-event-store/pkg/eventstore"
)

// Integration tests verify the bank domain working with a real EventStore:
// Commands -> Event appending -> Event replay -> Balance calculation

func TestIntegration_BankAccountLifecycle(t *testing.T) {
	// Setup
	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	accountID := "acc-alice"
	owner := "Alice Smith"

	// Test 1: Open account
	_, err = OpenAccount(store, accountID, owner)
	if err != nil {
		t.Fatalf("Failed to open account: %v", err)
	}

	// Verify account exists and has zero balance
	balance, err := GetBalance(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get balance after opening: %v", err)
	}
	if balance != 0 {
		t.Errorf("Expected balance 0 after opening account, got %d", balance)
	}

	// Test 2: Deposit money
	_, err = Deposit(store, accountID, 10000)
	if err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}

	balance, err = GetBalance(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get balance after deposit: %v", err)
	}
	if balance != 10000 {
		t.Errorf("Expected balance 10000 after deposit, got %d", balance)
	}

	// Test 3: Multiple deposits
	_, err = Deposit(store, accountID, 5000)
	if err != nil {
		t.Fatalf("Failed to deposit 2: %v", err)
	}

	_, err = Deposit(store, accountID, 2500)
	if err != nil {
		t.Fatalf("Failed to deposit 3: %v", err)
	}

	balance, err = GetBalance(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get balance after multiple deposits: %v", err)
	}
	if balance != 17500 {
		t.Errorf("Expected balance 17500 after multiple deposits, got %d", balance)
	}

	// Test 4: Withdraw money
	_, err = Withdraw(store, accountID, 7500)
	if err != nil {
		t.Fatalf("Failed to withdraw: %v", err)
	}

	balance, err = GetBalance(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get balance after withdrawal: %v", err)
	}
	if balance != 10000 {
		t.Errorf("Expected balance 10000 after withdrawal, got %d", balance)
	}

	// Test 5: Verify event history
	events, err := store.GetStream(accountID)
	if err != nil {
		t.Fatalf("Failed to get stream: %v", err)
	}

	expectedEventCount := 5 // 1 open + 3 deposits + 1 withdrawal
	if len(events) != expectedEventCount {
		t.Errorf("Expected %d events, got %d", expectedEventCount, len(events))
	}

	expectedTypes := []string{
		EventTypeAccountOpened,
		EventTypeMoneyDeposited,
		EventTypeMoneyDeposited,
		EventTypeMoneyDeposited,
		EventTypeMoneyWithdrawn,
	}

	for i, event := range events {
		if event.EventType != expectedTypes[i] {
			t.Errorf("Event %d: expected type %s, got %s", i, expectedTypes[i], event.EventType)
		}
		if event.Version != i+1 {
			t.Errorf("Event %d: expected version %d, got %d", i, i+1, event.Version)
		}
	}
}

func TestIntegration_BusinessRuleValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	accountID := "acc-test"

	// Test 1: Cannot deposit to non-existent account
	_, err = Deposit(store, accountID, 1000)
	if err == nil {
		t.Error("Expected error when depositing to non-existent account")
	}

	// Test 2: Cannot withdraw from non-existent account
	_, err = Withdraw(store, accountID, 1000)
	if err == nil {
		t.Error("Expected error when withdrawing from non-existent account")
	}

	// Test 3: Cannot open duplicate account
	_, err = OpenAccount(store, accountID, "Alice")
	if err != nil {
		t.Fatalf("Failed to open account first time: %v", err)
	}

	_, err = OpenAccount(store, accountID, "Bob")
	if err == nil {
		t.Error("Expected error when opening duplicate account")
	}

	// Test 4: Cannot deposit negative amount
	_, err = Deposit(store, accountID, -1000)
	if err == nil {
		t.Error("Expected error when depositing negative amount")
	}

	// Test 5: Cannot deposit zero
	_, err = Deposit(store, accountID, 0)
	if err == nil {
		t.Error("Expected error when depositing zero")
	}

	// Test 6: Cannot withdraw negative amount
	_, err = Withdraw(store, accountID, -500)
	if err == nil {
		t.Error("Expected error when withdrawing negative amount")
	}

	// Test 7: Cannot withdraw zero
	_, err = Withdraw(store, accountID, 0)
	if err == nil {
		t.Error("Expected error when withdrawing zero")
	}

	// Test 8: Cannot overdraw account
	Deposit(store, accountID, 1000)
	_, err = Withdraw(store, accountID, 2000)
	if err == nil {
		t.Error("Expected error when overdrawing account")
	}

	// Verify no invalid events were written
	events, _ := store.GetStream(accountID)
	expectedValidEvents := 2 // 1 open + 1 valid deposit
	if len(events) != expectedValidEvents {
		t.Errorf("Expected %d valid events, got %d (invalid events should not be written)", expectedValidEvents, len(events))
	}
}

func TestIntegration_PersistenceAcrossRestart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	accountID := "acc-alice"

	// Phase 1: Create store, open account, do transactions
	{
		store, err := eventstore.NewEventStore(tempDir)
		if err != nil {
			t.Fatalf("Failed to create event store: %v", err)
		}

		OpenAccount(store, accountID, "Alice Smith")
		Deposit(store, accountID, 10000)
		Deposit(store, accountID, 5000)
		Withdraw(store, accountID, 3000)

		balance, _ := GetBalance(store, accountID)
		if balance != 12000 {
			t.Errorf("Phase 1: Expected balance 12000, got %d", balance)
		}

		store.Close()
	}

	// Phase 2: Reopen store and verify balance is correct
	{
		store, err := eventstore.NewEventStore(tempDir)
		if err != nil {
			t.Fatalf("Failed to reopen event store: %v", err)
		}

		balance, err := GetBalance(store, accountID)
		if err != nil {
			t.Fatalf("Failed to get balance after reopen: %v", err)
		}
		if balance != 12000 {
			t.Errorf("Phase 2: Expected balance 12000 after restart, got %d", balance)
		}

		// Continue transactions
		Deposit(store, accountID, 8000)
		Withdraw(store, accountID, 5000)

		balance, _ = GetBalance(store, accountID)
		if balance != 15000 {
			t.Errorf("Phase 2: Expected balance 15000 after more transactions, got %d", balance)
		}

		store.Close()
	}

	// Phase 3: Final verification
	{
		store, err := eventstore.NewEventStore(tempDir)
		if err != nil {
			t.Fatalf("Failed to reopen event store second time: %v", err)
		}
		defer store.Close()

		balance, err := GetBalance(store, accountID)
		if err != nil {
			t.Fatalf("Failed to get balance on final reopen: %v", err)
		}
		if balance != 15000 {
			t.Errorf("Phase 3: Expected balance 15000 on final restart, got %d", balance)
		}

		// Verify event count
		events, _ := store.GetStream(accountID)
		expectedEvents := 6 // 1 open + 3 deposits + 2 withdrawals
		if len(events) != expectedEvents {
			t.Errorf("Expected %d total events, got %d", expectedEvents, len(events))
		}
	}
}

func TestIntegration_MultipleAccounts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	// Create multiple accounts
	accounts := []struct {
		id    string
		owner string
	}{
		{"acc-alice", "Alice Smith"},
		{"acc-bob", "Bob Jones"},
		{"acc-charlie", "Charlie Brown"},
	}

	for _, acc := range accounts {
		_, err := OpenAccount(store, acc.id, acc.owner)
		if err != nil {
			t.Fatalf("Failed to open account %s: %v", acc.id, err)
		}
	}

	// Do transactions on different accounts
	Deposit(store, "acc-alice", 10000)
	Deposit(store, "acc-bob", 5000)
	Deposit(store, "acc-charlie", 15000)
	Deposit(store, "acc-alice", 5000)
	Withdraw(store, "acc-bob", 2000)
	Withdraw(store, "acc-charlie", 5000)
	Deposit(store, "acc-bob", 10000)

	// Verify each account's balance
	expectedBalances := map[string]int{
		"acc-alice":   15000, // 10000 + 5000
		"acc-bob":     13000, // 5000 - 2000 + 10000
		"acc-charlie": 10000, // 15000 - 5000
	}

	for accID, expectedBalance := range expectedBalances {
		balance, err := GetBalance(store, accID)
		if err != nil {
			t.Fatalf("Failed to get balance for %s: %v", accID, err)
		}
		if balance != expectedBalance {
			t.Errorf("Account %s: expected balance %d, got %d", accID, expectedBalance, balance)
		}
	}

	// Verify event counts per account
	expectedEventCounts := map[string]int{
		"acc-alice":   3, // 1 open + 2 deposits
		"acc-bob":     4, // 1 open + 2 deposits + 1 withdrawal
		"acc-charlie": 3, // 1 open + 1 deposit + 1 withdrawal
	}

	for accID, expectedCount := range expectedEventCounts {
		events, err := store.GetStream(accID)
		if err != nil {
			t.Fatalf("Failed to get stream for %s: %v", accID, err)
		}
		if len(events) != expectedCount {
			t.Errorf("Account %s: expected %d events, got %d", accID, expectedCount, len(events))
		}
	}

	// Verify total events
	allEvents, _ := store.GetAllEvents()
	expectedTotalEvents := 10 // 3 opens + 5 deposits + 2 withdrawals
	if len(allEvents) != expectedTotalEvents {
		t.Errorf("Expected %d total events, got %d", expectedTotalEvents, len(allEvents))
	}
}

func TestIntegration_ConcurrentTransactions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	accountID := "acc-concurrent"
	OpenAccount(store, accountID, "Concurrent User")

	// Perform concurrent deposits
	var wg sync.WaitGroup
	depositCount := 50
	depositAmount := 100

	for i := 0; i < depositCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Deposit(store, accountID, depositAmount)
		}()
	}

	wg.Wait()

	// Verify final balance
	balance, err := GetBalance(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	expectedBalance := depositCount * depositAmount
	if balance != expectedBalance {
		t.Errorf("Expected balance %d after %d concurrent deposits, got %d", expectedBalance, depositCount, balance)
	}

	// Verify event count
	events, _ := store.GetStream(accountID)
	expectedEventCount := 1 + depositCount // 1 open + 50 deposits
	if len(events) != expectedEventCount {
		t.Errorf("Expected %d events, got %d", expectedEventCount, len(events))
	}

	// Verify all versions are sequential
	for i, event := range events {
		if event.Version != i+1 {
			t.Errorf("Event %d: expected version %d, got %d", i, i+1, event.Version)
		}
	}
}

func TestIntegration_GetAccountFullState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	accountID := "acc-full-state"
	owner := "Full State User"

	OpenAccount(store, accountID, owner)
	Deposit(store, accountID, 10000)
	Withdraw(store, accountID, 3000)

	// Get full account state
	account, err := GetAccount(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	// Verify all fields
	if account.AccountID != accountID {
		t.Errorf("Expected accountID %s, got %s", accountID, account.AccountID)
	}
	if account.Owner != owner {
		t.Errorf("Expected owner %s, got %s", owner, account.Owner)
	}
	if account.Balance != 7000 {
		t.Errorf("Expected balance 7000, got %d", account.Balance)
	}
	if account.Version != 3 {
		t.Errorf("Expected version 3, got %d", account.Version)
	}
}

func TestIntegration_LargeNumberOfTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large transaction test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "bank-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create event store: %v", err)
	}
	defer store.Close()

	accountID := "acc-heavy"
	OpenAccount(store, accountID, "Heavy User")

	// Perform many transactions
	transactionCount := 500
	for i := 1; i <= transactionCount; i++ {
		Deposit(store, accountID, 100)
	}

	// Verify balance
	balance, err := GetBalance(store, accountID)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	expectedBalance := transactionCount * 100
	if balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, balance)
	}

	// Verify event count
	events, _ := store.GetStream(accountID)
	expectedEvents := 1 + transactionCount
	if len(events) != expectedEvents {
		t.Errorf("Expected %d events, got %d", expectedEvents, len(events))
	}

	// Close and reopen to verify persistence
	store.Close()

	store2, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	balance2, _ := GetBalance(store2, accountID)
	if balance2 != expectedBalance {
		t.Errorf("Expected balance %d after restart, got %d", expectedBalance, balance2)
	}
}
