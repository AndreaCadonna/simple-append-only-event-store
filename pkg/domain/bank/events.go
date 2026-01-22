package bank

// Event type constants for bank domain events.
// These constants are used as the EventType field in Event structs.
const (
	// EventTypeAccountOpened is emitted when a new bank account is created
	EventTypeAccountOpened = "AccountOpened"

	// EventTypeMoneyDeposited is emitted when money is added to an account
	EventTypeMoneyDeposited = "MoneyDeposited"

	// EventTypeMoneyWithdrawn is emitted when money is removed from an account
	EventTypeMoneyWithdrawn = "MoneyWithdrawn"
)

// AccountOpenedData represents the data payload for an AccountOpened event.
// This event marks the creation of a new bank account.
type AccountOpenedData struct {
	AccountID string `json:"accountId"` // Unique identifier for the account
	Owner     string `json:"owner"`     // Name of the account owner
}

// MoneyDepositedData represents the data payload for a MoneyDeposited event.
// This event records money being added to an account.
type MoneyDepositedData struct {
	AccountID string `json:"accountId"` // Account receiving the deposit
	Amount    int    `json:"amount"`    // Amount in cents (to avoid float precision issues)
}

// MoneyWithdrawnData represents the data payload for a MoneyWithdrawn event.
// This event records money being removed from an account.
type MoneyWithdrawnData struct {
	AccountID string `json:"accountId"` // Account from which money is withdrawn
	Amount    int    `json:"amount"`    // Amount in cents (to avoid float precision issues)
}
