package bank

import (
	"os"
	"testing"

	"github.com/cadonna/append-only-event-store/pkg/eventstore"
)

// Helper to create a test event store
func createTestStore(t *testing.T) (eventstore.EventStore, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "bank-commands-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := eventstore.NewEventStore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create event store: %v", err)
	}

	return store, tempDir
}

// Helper to cleanup test event store
func cleanupTestStore(t *testing.T, store eventstore.EventStore, dir string) {
	t.Helper()
	if store != nil {
		store.Close()
	}
	if dir != "" {
		os.RemoveAll(dir)
	}
}

func TestOpenAccount_Success(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	event, err := OpenAccount(store, "acc-123", "Alice")
	if err != nil {
		t.Fatalf("OpenAccount failed: %v", err)
	}

	if event == nil {
		t.Fatal("Expected event, got nil")
	}

	if event.EventType != EventTypeAccountOpened {
		t.Errorf("Expected EventType %s, got %s", EventTypeAccountOpened, event.EventType)
	}

	if event.StreamID != "acc-123" {
		t.Errorf("Expected StreamID 'acc-123', got '%s'", event.StreamID)
	}

	if event.Version != 1 {
		t.Errorf("Expected Version 1, got %d", event.Version)
	}

	// Verify account was created by loading it
	account, err := GetAccount(store, "acc-123")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if account.Owner != "Alice" {
		t.Errorf("Expected Owner 'Alice', got '%s'", account.Owner)
	}

	if account.Balance != 0 {
		t.Errorf("Expected Balance 0, got %d", account.Balance)
	}
}

func TestOpenAccount_EmptyAccountID(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := OpenAccount(store, "", "Alice")
	if err == nil {
		t.Fatal("Expected error for empty accountID, got nil")
	}
}

func TestOpenAccount_EmptyOwner(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := OpenAccount(store, "acc-123", "")
	if err == nil {
		t.Fatal("Expected error for empty owner, got nil")
	}
}

func TestOpenAccount_DuplicateAccount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	// Open account first time
	_, err := OpenAccount(store, "acc-123", "Alice")
	if err != nil {
		t.Fatalf("First OpenAccount failed: %v", err)
	}

	// Try to open again
	_, err = OpenAccount(store, "acc-123", "Bob")
	if err == nil {
		t.Fatal("Expected error for duplicate account, got nil")
	}
}

func TestDeposit_Success(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	// Open account first
	OpenAccount(store, "acc-123", "Alice")

	// Deposit money
	event, err := Deposit(store, "acc-123", 10000) // $100
	if err != nil {
		t.Fatalf("Deposit failed: %v", err)
	}

	if event.EventType != EventTypeMoneyDeposited {
		t.Errorf("Expected EventType %s, got %s", EventTypeMoneyDeposited, event.EventType)
	}

	if event.Version != 2 {
		t.Errorf("Expected Version 2, got %d", event.Version)
	}

	// Verify balance
	balance, err := GetBalance(store, "acc-123")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance != 10000 {
		t.Errorf("Expected Balance 10000, got %d", balance)
	}
}

func TestDeposit_EmptyAccountID(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := Deposit(store, "", 10000)
	if err == nil {
		t.Fatal("Expected error for empty accountID, got nil")
	}
}

func TestDeposit_ZeroAmount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")

	_, err := Deposit(store, "acc-123", 0)
	if err == nil {
		t.Fatal("Expected error for zero amount, got nil")
	}
}

func TestDeposit_NegativeAmount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")

	_, err := Deposit(store, "acc-123", -100)
	if err == nil {
		t.Fatal("Expected error for negative amount, got nil")
	}
}

func TestDeposit_NonExistentAccount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := Deposit(store, "non-existent", 10000)
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestWithdraw_Success(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	// Setup: Open account and deposit
	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 10000) // $100

	// Withdraw money
	event, err := Withdraw(store, "acc-123", 2500) // $25
	if err != nil {
		t.Fatalf("Withdraw failed: %v", err)
	}

	if event.EventType != EventTypeMoneyWithdrawn {
		t.Errorf("Expected EventType %s, got %s", EventTypeMoneyWithdrawn, event.EventType)
	}

	if event.Version != 3 {
		t.Errorf("Expected Version 3, got %d", event.Version)
	}

	// Verify balance: 10000 - 2500 = 7500
	balance, err := GetBalance(store, "acc-123")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance != 7500 {
		t.Errorf("Expected Balance 7500, got %d", balance)
	}
}

func TestWithdraw_EmptyAccountID(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := Withdraw(store, "", 1000)
	if err == nil {
		t.Fatal("Expected error for empty accountID, got nil")
	}
}

func TestWithdraw_ZeroAmount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 10000)

	_, err := Withdraw(store, "acc-123", 0)
	if err == nil {
		t.Fatal("Expected error for zero amount, got nil")
	}
}

func TestWithdraw_NegativeAmount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 10000)

	_, err := Withdraw(store, "acc-123", -100)
	if err == nil {
		t.Fatal("Expected error for negative amount, got nil")
	}
}

func TestWithdraw_NonExistentAccount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := Withdraw(store, "non-existent", 1000)
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 5000) // $50

	// Try to withdraw more than balance
	_, err := Withdraw(store, "acc-123", 10000) // $100
	if err == nil {
		t.Fatal("Expected error for insufficient balance, got nil")
	}

	// Verify balance unchanged
	balance, _ := GetBalance(store, "acc-123")
	if balance != 5000 {
		t.Errorf("Expected Balance unchanged at 5000, got %d", balance)
	}
}

func TestGetBalance_Success(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 12345)

	balance, err := GetBalance(store, "acc-123")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance != 12345 {
		t.Errorf("Expected Balance 12345, got %d", balance)
	}
}

func TestGetBalance_EmptyAccountID(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := GetBalance(store, "")
	if err == nil {
		t.Fatal("Expected error for empty accountID, got nil")
	}
}

func TestGetBalance_NonExistentAccount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := GetBalance(store, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestGetAccount_Success(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 5000)

	account, err := GetAccount(store, "acc-123")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if account.AccountID != "acc-123" {
		t.Errorf("Expected AccountID 'acc-123', got '%s'", account.AccountID)
	}

	if account.Owner != "Alice" {
		t.Errorf("Expected Owner 'Alice', got '%s'", account.Owner)
	}

	if account.Balance != 5000 {
		t.Errorf("Expected Balance 5000, got %d", account.Balance)
	}

	if account.Version != 2 {
		t.Errorf("Expected Version 2, got %d", account.Version)
	}
}

func TestGetAccount_EmptyAccountID(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := GetAccount(store, "")
	if err == nil {
		t.Fatal("Expected error for empty accountID, got nil")
	}
}

func TestGetAccount_NonExistentAccount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	_, err := GetAccount(store, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent account, got nil")
	}
}

func TestCommands_CompleteWorkflow(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	// Open account
	_, err := OpenAccount(store, "acc-456", "Bob")
	if err != nil {
		t.Fatalf("OpenAccount failed: %v", err)
	}

	// Deposit $200
	_, err = Deposit(store, "acc-456", 20000)
	if err != nil {
		t.Fatalf("First Deposit failed: %v", err)
	}

	// Withdraw $50
	_, err = Withdraw(store, "acc-456", 5000)
	if err != nil {
		t.Fatalf("First Withdraw failed: %v", err)
	}

	// Deposit $100
	_, err = Deposit(store, "acc-456", 10000)
	if err != nil {
		t.Fatalf("Second Deposit failed: %v", err)
	}

	// Withdraw $75
	_, err = Withdraw(store, "acc-456", 7500)
	if err != nil {
		t.Fatalf("Second Withdraw failed: %v", err)
	}

	// Check final balance: 0 + 200 - 50 + 100 - 75 = 175
	// In cents: 20000 - 5000 + 10000 - 7500 = 17500
	balance, err := GetBalance(store, "acc-456")
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	expectedBalance := 17500
	if balance != expectedBalance {
		t.Errorf("Expected final balance %d, got %d", expectedBalance, balance)
	}

	// Verify account state
	account, err := GetAccount(store, "acc-456")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if account.Version != 5 {
		t.Errorf("Expected Version 5 (1 open + 2 deposits + 2 withdrawals), got %d", account.Version)
	}
}

func TestCommands_MultipleAccounts(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	// Create multiple accounts
	OpenAccount(store, "acc-1", "Alice")
	OpenAccount(store, "acc-2", "Bob")
	OpenAccount(store, "acc-3", "Charlie")

	// Perform operations on each
	Deposit(store, "acc-1", 10000)
	Deposit(store, "acc-2", 20000)
	Deposit(store, "acc-3", 30000)

	Withdraw(store, "acc-1", 2000)
	Withdraw(store, "acc-2", 5000)

	// Verify balances are independent
	balance1, _ := GetBalance(store, "acc-1")
	balance2, _ := GetBalance(store, "acc-2")
	balance3, _ := GetBalance(store, "acc-3")

	if balance1 != 8000 {
		t.Errorf("acc-1: expected balance 8000, got %d", balance1)
	}

	if balance2 != 15000 {
		t.Errorf("acc-2: expected balance 15000, got %d", balance2)
	}

	if balance3 != 30000 {
		t.Errorf("acc-3: expected balance 30000, got %d", balance3)
	}
}

func TestCommands_CannotWithdrawFromEmptyAccount(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")

	// Try to withdraw from account with zero balance
	_, err := Withdraw(store, "acc-123", 1000)
	if err == nil {
		t.Fatal("Expected error when withdrawing from empty account, got nil")
	}

	// Verify balance is still zero
	balance, _ := GetBalance(store, "acc-123")
	if balance != 0 {
		t.Errorf("Expected balance 0, got %d", balance)
	}
}

func TestCommands_ExactWithdrawal(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")
	Deposit(store, "acc-123", 5000)

	// Withdraw exact balance
	_, err := Withdraw(store, "acc-123", 5000)
	if err != nil {
		t.Fatalf("Exact withdrawal failed: %v", err)
	}

	// Verify balance is now zero
	balance, _ := GetBalance(store, "acc-123")
	if balance != 0 {
		t.Errorf("Expected balance 0 after exact withdrawal, got %d", balance)
	}
}

func TestCommands_MultipleDeposits(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")

	// Make multiple deposits
	deposits := []int{1000, 2000, 3000, 4000, 5000}
	expectedTotal := 0

	for _, amount := range deposits {
		_, err := Deposit(store, "acc-123", amount)
		if err != nil {
			t.Fatalf("Deposit of %d failed: %v", amount, err)
		}
		expectedTotal += amount
	}

	// Verify total balance
	balance, _ := GetBalance(store, "acc-123")
	if balance != expectedTotal {
		t.Errorf("Expected total balance %d, got %d", expectedTotal, balance)
	}
}

func TestCommands_AlternatingDepositsAndWithdrawals(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")

	// Pattern: deposit, withdraw, deposit, withdraw, etc.
	Deposit(store, "acc-123", 10000)   // Balance: 10000
	Withdraw(store, "acc-123", 3000)   // Balance: 7000
	Deposit(store, "acc-123", 5000)    // Balance: 12000
	Withdraw(store, "acc-123", 2000)   // Balance: 10000
	Deposit(store, "acc-123", 8000)    // Balance: 18000
	Withdraw(store, "acc-123", 6000)   // Balance: 12000

	balance, _ := GetBalance(store, "acc-123")
	if balance != 12000 {
		t.Errorf("Expected final balance 12000, got %d", balance)
	}
}

func TestCommands_LargeAmounts(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	OpenAccount(store, "acc-123", "Alice")

	// Deposit large amount (1 million dollars = 100,000,000 cents)
	largeAmount := 100000000
	_, err := Deposit(store, "acc-123", largeAmount)
	if err != nil {
		t.Fatalf("Large deposit failed: %v", err)
	}

	balance, _ := GetBalance(store, "acc-123")
	if balance != largeAmount {
		t.Errorf("Expected balance %d, got %d", largeAmount, balance)
	}

	// Withdraw half
	_, err = Withdraw(store, "acc-123", largeAmount/2)
	if err != nil {
		t.Fatalf("Large withdrawal failed: %v", err)
	}

	balance, _ = GetBalance(store, "acc-123")
	if balance != largeAmount/2 {
		t.Errorf("Expected balance %d, got %d", largeAmount/2, balance)
	}
}

func TestCommands_EventVersioning(t *testing.T) {
	store, dir := createTestStore(t)
	defer cleanupTestStore(t, store, dir)

	// Each operation should increment the version
	e1, _ := OpenAccount(store, "acc-123", "Alice")
	if e1.Version != 1 {
		t.Errorf("OpenAccount: expected version 1, got %d", e1.Version)
	}

	e2, _ := Deposit(store, "acc-123", 1000)
	if e2.Version != 2 {
		t.Errorf("First Deposit: expected version 2, got %d", e2.Version)
	}

	e3, _ := Deposit(store, "acc-123", 2000)
	if e3.Version != 3 {
		t.Errorf("Second Deposit: expected version 3, got %d", e3.Version)
	}

	e4, _ := Withdraw(store, "acc-123", 500)
	if e4.Version != 4 {
		t.Errorf("Withdraw: expected version 4, got %d", e4.Version)
	}

	// Verify account version matches event count
	account, _ := GetAccount(store, "acc-123")
	if account.Version != 4 {
		t.Errorf("Account version: expected 4, got %d", account.Version)
	}
}
