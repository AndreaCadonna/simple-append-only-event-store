package bank

import (
	"fmt"

	"github.com/cadonna/append-only-event-store/pkg/eventstore"
)

// Commands in event sourcing represent user intentions.
// They validate business rules and, if valid, append events to the store.
//
// Key principle: Commands can be rejected, but events represent facts that already happened.

// OpenAccount creates a new bank account.
// This validates that the account doesn't already exist and that all required data is present.
//
// Parameters:
//   - store: The event store to write to
//   - accountID: Unique identifier for the new account (must be non-empty)
//   - owner: Name of the account owner (must be non-empty)
//
// Returns:
//   - *eventstore.Event: The AccountOpened event that was written
//   - error: If validation fails or account already exists
func OpenAccount(store eventstore.EventStore, accountID, owner string) (*eventstore.Event, error) {
	// Validate inputs
	if accountID == "" {
		return nil, fmt.Errorf("accountID cannot be empty")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}

	// Check if account already exists
	// An account exists if there are any events in its stream
	events, err := store.GetStream(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if account exists: %w", err)
	}

	if len(events) > 0 {
		return nil, fmt.Errorf("account %s already exists", accountID)
	}

	// Create event data
	data := AccountOpenedData{
		AccountID: accountID,
		Owner:     owner,
	}

	// Append event to store
	event, err := store.Append(accountID, EventTypeAccountOpened, data)
	if err != nil {
		return nil, fmt.Errorf("failed to open account: %w", err)
	}

	return event, nil
}

// Deposit adds money to an account.
// This validates that the amount is positive and the account exists.
//
// Parameters:
//   - store: The event store to write to
//   - accountID: Account to deposit to
//   - amount: Amount to deposit in cents (must be > 0)
//
// Returns:
//   - *eventstore.Event: The MoneyDeposited event that was written
//   - error: If validation fails or account doesn't exist
func Deposit(store eventstore.EventStore, accountID string, amount int) (*eventstore.Event, error) {
	// Validate inputs
	if accountID == "" {
		return nil, fmt.Errorf("accountID cannot be empty")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive, got %d", amount)
	}

	// Verify account exists by trying to load it
	// This also ensures we can't deposit to a non-existent account
	_, err := ReplayAccountFromStore(store, accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot deposit: %w", err)
	}

	// Create event data
	data := MoneyDepositedData{
		AccountID: accountID,
		Amount:    amount,
	}

	// Append event to store
	event, err := store.Append(accountID, EventTypeMoneyDeposited, data)
	if err != nil {
		return nil, fmt.Errorf("failed to deposit money: %w", err)
	}

	return event, nil
}

// Withdraw removes money from an account.
// This validates that:
//   - The amount is positive
//   - The account exists
//   - The account has sufficient balance
//
// This is where business rules are enforced!
//
// Parameters:
//   - store: The event store to write to
//   - accountID: Account to withdraw from
//   - amount: Amount to withdraw in cents (must be > 0)
//
// Returns:
//   - *eventstore.Event: The MoneyWithdrawn event that was written
//   - error: If validation fails, account doesn't exist, or insufficient balance
func Withdraw(store eventstore.EventStore, accountID string, amount int) (*eventstore.Event, error) {
	// Validate inputs
	if accountID == "" {
		return nil, fmt.Errorf("accountID cannot be empty")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("withdrawal amount must be positive, got %d", amount)
	}

	// Load current account state to check balance
	account, err := ReplayAccountFromStore(store, accountID)
	if err != nil {
		return nil, fmt.Errorf("cannot withdraw: %w", err)
	}

	// Business rule: Cannot overdraw account
	if account.Balance < amount {
		return nil, fmt.Errorf("insufficient balance: have %d cents, need %d cents", account.Balance, amount)
	}

	// Create event data
	data := MoneyWithdrawnData{
		AccountID: accountID,
		Amount:    amount,
	}

	// Append event to store
	event, err := store.Append(accountID, EventTypeMoneyWithdrawn, data)
	if err != nil {
		return nil, fmt.Errorf("failed to withdraw money: %w", err)
	}

	return event, nil
}

// GetBalance returns the current balance of an account.
// This is a query operation that doesn't modify state.
//
// Parameters:
//   - store: The event store to read from
//   - accountID: Account to query
//
// Returns:
//   - int: Current balance in cents
//   - error: If account doesn't exist or replay fails
func GetBalance(store eventstore.EventStore, accountID string) (int, error) {
	if accountID == "" {
		return 0, fmt.Errorf("accountID cannot be empty")
	}

	// Load current account state
	account, err := ReplayAccountFromStore(store, accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	return account.Balance, nil
}

// GetAccount returns the full account state.
// This is a query operation that doesn't modify state.
//
// This is useful for getting all account information (owner, balance, version, etc.)
//
// Parameters:
//   - store: The event store to read from
//   - accountID: Account to query
//
// Returns:
//   - *BankAccount: Current account state
//   - error: If account doesn't exist or replay fails
func GetAccount(store eventstore.EventStore, accountID string) (*BankAccount, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID cannot be empty")
	}

	// Load current account state
	account, err := ReplayAccountFromStore(store, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}
