package bank

import (
	"encoding/json"
	"fmt"

	"github.com/cadonna/append-only-event-store/pkg/eventstore"
)

// BankAccount represents the current state of a bank account.
// This is an aggregate in Domain-Driven Design terminology.
//
// The state is rebuilt by replaying events from the event store.
// This demonstrates the core principle of event sourcing:
// current state = f(all events in order)
type BankAccount struct {
	AccountID string // Unique identifier for the account
	Owner     string // Name of the account owner
	Balance   int    // Current balance in cents (to avoid float precision issues)
	Version   int    // Number of events that have been applied (for optimistic locking)
}

// NewBankAccount creates a new empty bank account.
// The account has no state until events are applied via ApplyEvent().
//
// Returns:
//   - *BankAccount: Empty account ready for event replay
func NewBankAccount() *BankAccount {
	return &BankAccount{
		Balance: 0,
		Version: 0,
	}
}

// ApplyEvent applies a single event to the account state.
// This mutates the account based on the event type and data.
//
// This is the core of event sourcing - each event represents a state change.
// By applying events in order, we rebuild the current state.
//
// Parameters:
//   - event: The event to apply
//
// Returns:
//   - error: If event data is invalid or event type is unknown
func (acc *BankAccount) ApplyEvent(event *eventstore.Event) error {
	if event == nil {
		return fmt.Errorf("cannot apply nil event")
	}

	// Switch on event type and apply the appropriate state change
	switch event.EventType {
	case EventTypeAccountOpened:
		return acc.applyAccountOpened(event)

	case EventTypeMoneyDeposited:
		return acc.applyMoneyDeposited(event)

	case EventTypeMoneyWithdrawn:
		return acc.applyMoneyWithdrawn(event)

	default:
		// Unknown event types are ignored (allows for forward compatibility)
		// In a production system, you might want to log this
		return fmt.Errorf("unknown event type: %s", event.EventType)
	}
}

// applyAccountOpened handles the AccountOpened event.
// This initializes the account with owner information.
func (acc *BankAccount) applyAccountOpened(event *eventstore.Event) error {
	var data AccountOpenedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal AccountOpened data: %w", err)
	}

	// Validate data
	if data.AccountID == "" {
		return fmt.Errorf("AccountOpened event has empty accountId")
	}
	if data.Owner == "" {
		return fmt.Errorf("AccountOpened event has empty owner")
	}

	// Apply state change
	acc.AccountID = data.AccountID
	acc.Owner = data.Owner
	acc.Balance = 0 // Accounts start with zero balance
	acc.Version++

	return nil
}

// applyMoneyDeposited handles the MoneyDeposited event.
// This adds money to the account balance.
func (acc *BankAccount) applyMoneyDeposited(event *eventstore.Event) error {
	var data MoneyDepositedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal MoneyDeposited data: %w", err)
	}

	// Validate data
	if data.Amount <= 0 {
		return fmt.Errorf("MoneyDeposited event has invalid amount: %d", data.Amount)
	}

	// Apply state change
	acc.Balance += data.Amount
	acc.Version++

	return nil
}

// applyMoneyWithdrawn handles the MoneyWithdrawn event.
// This removes money from the account balance.
//
// Note: We don't validate balance here because we're replaying history.
// Events represent what actually happened, not what should happen.
// Validation happens in commands (before events are written).
func (acc *BankAccount) applyMoneyWithdrawn(event *eventstore.Event) error {
	var data MoneyWithdrawnData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("failed to unmarshal MoneyWithdrawn data: %w", err)
	}

	// Validate data
	if data.Amount <= 0 {
		return fmt.Errorf("MoneyWithdrawn event has invalid amount: %d", data.Amount)
	}

	// Apply state change
	acc.Balance -= data.Amount
	acc.Version++

	return nil
}

// ReplayAccount rebuilds account state from a stream of events.
// This is the fundamental operation in event sourcing.
//
// The function creates a new empty account and applies each event in order,
// resulting in the current state of the account.
//
// Parameters:
//   - events: Slice of events to replay (should be in chronological order)
//
// Returns:
//   - *BankAccount: Account with state rebuilt from events
//   - error: If any event fails to apply
func ReplayAccount(events []*eventstore.Event) (*BankAccount, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("cannot replay account from empty event list")
	}

	// Start with empty account
	account := NewBankAccount()

	// Apply each event in order
	for i, event := range events {
		if event == nil {
			return nil, fmt.Errorf("event %d is nil", i)
		}
		if err := account.ApplyEvent(event); err != nil {
			return nil, fmt.Errorf("failed to apply event %d (type: %s): %w", i, event.EventType, err)
		}
	}

	// Verify we got the expected number of events
	if account.Version != len(events) {
		return nil, fmt.Errorf("version mismatch: applied %d events but version is %d", len(events), account.Version)
	}

	return account, nil
}

// ReplayAccountFromStore is a convenience function that loads events from
// the event store and replays them to build the current account state.
//
// This is the typical way to load an account in an event-sourced system.
//
// Parameters:
//   - store: The event store to load events from
//   - accountID: The account to load
//
// Returns:
//   - *BankAccount: Account with current state
//   - error: If loading events or replay fails
func ReplayAccountFromStore(store eventstore.EventStore, accountID string) (*BankAccount, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID cannot be empty")
	}

	// Load all events for this account
	events, err := store.GetStream(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load events for account %s: %w", accountID, err)
	}

	// Check if account exists
	if len(events) == 0 {
		return nil, fmt.Errorf("account %s does not exist (no events found)", accountID)
	}

	// Replay events to rebuild state
	account, err := ReplayAccount(events)
	if err != nil {
		return nil, fmt.Errorf("failed to replay account %s: %w", accountID, err)
	}

	return account, nil
}

// GetBalance returns the current balance in cents.
// This is a convenience method.
func (acc *BankAccount) GetBalance() int {
	return acc.Balance
}

// GetBalanceDollars returns the current balance formatted as dollars.
// For example, 12550 cents returns "$125.50"
func (acc *BankAccount) GetBalanceDollars() string {
	dollars := acc.Balance / 100
	cents := acc.Balance % 100
	if acc.Balance < 0 {
		// Handle negative balances correctly
		dollars = acc.Balance / 100
		cents = (-acc.Balance) % 100
		if cents != 0 {
			dollars-- // Adjust for the negative remainder
		}
	}
	return fmt.Sprintf("$%d.%02d", dollars, cents)
}

// String returns a human-readable representation of the account.
func (acc *BankAccount) String() string {
	return fmt.Sprintf("Account{ID: %s, Owner: %s, Balance: %s, Version: %d}",
		acc.AccountID, acc.Owner, acc.GetBalanceDollars(), acc.Version)
}
