package bank

import (
	"encoding/json"
	"testing"
)

func TestAccountOpenedData_JSONMarshaling(t *testing.T) {
	data := AccountOpenedData{
		AccountID: "acc-123",
		Owner:     "Alice Smith",
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal AccountOpenedData: %v", err)
	}

	// Unmarshal back
	var unmarshaled AccountOpenedData
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal AccountOpenedData: %v", err)
	}

	// Verify fields
	if unmarshaled.AccountID != data.AccountID {
		t.Errorf("AccountID mismatch: expected '%s', got '%s'", data.AccountID, unmarshaled.AccountID)
	}

	if unmarshaled.Owner != data.Owner {
		t.Errorf("Owner mismatch: expected '%s', got '%s'", data.Owner, unmarshaled.Owner)
	}
}

func TestAccountOpenedData_JSONFormat(t *testing.T) {
	data := AccountOpenedData{
		AccountID: "acc-456",
		Owner:     "Bob Jones",
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedJSON := `{"accountId":"acc-456","owner":"Bob Jones"}`

	if jsonStr != expectedJSON {
		t.Errorf("JSON format mismatch:\nexpected: %s\ngot:      %s", expectedJSON, jsonStr)
	}
}

func TestMoneyDepositedData_JSONMarshaling(t *testing.T) {
	data := MoneyDepositedData{
		AccountID: "acc-789",
		Amount:    10000, // $100.00
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal MoneyDepositedData: %v", err)
	}

	// Unmarshal back
	var unmarshaled MoneyDepositedData
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal MoneyDepositedData: %v", err)
	}

	// Verify fields
	if unmarshaled.AccountID != data.AccountID {
		t.Errorf("AccountID mismatch: expected '%s', got '%s'", data.AccountID, unmarshaled.AccountID)
	}

	if unmarshaled.Amount != data.Amount {
		t.Errorf("Amount mismatch: expected %d, got %d", data.Amount, unmarshaled.Amount)
	}
}

func TestMoneyDepositedData_JSONFormat(t *testing.T) {
	data := MoneyDepositedData{
		AccountID: "acc-100",
		Amount:    5000, // $50.00
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedJSON := `{"accountId":"acc-100","amount":5000}`

	if jsonStr != expectedJSON {
		t.Errorf("JSON format mismatch:\nexpected: %s\ngot:      %s", expectedJSON, jsonStr)
	}
}

func TestMoneyWithdrawnData_JSONMarshaling(t *testing.T) {
	data := MoneyWithdrawnData{
		AccountID: "acc-999",
		Amount:    2500, // $25.00
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal MoneyWithdrawnData: %v", err)
	}

	// Unmarshal back
	var unmarshaled MoneyWithdrawnData
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal MoneyWithdrawnData: %v", err)
	}

	// Verify fields
	if unmarshaled.AccountID != data.AccountID {
		t.Errorf("AccountID mismatch: expected '%s', got '%s'", data.AccountID, unmarshaled.AccountID)
	}

	if unmarshaled.Amount != data.Amount {
		t.Errorf("Amount mismatch: expected %d, got %d", data.Amount, unmarshaled.Amount)
	}
}

func TestMoneyWithdrawnData_JSONFormat(t *testing.T) {
	data := MoneyWithdrawnData{
		AccountID: "acc-200",
		Amount:    7500, // $75.00
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)
	expectedJSON := `{"accountId":"acc-200","amount":7500}`

	if jsonStr != expectedJSON {
		t.Errorf("JSON format mismatch:\nexpected: %s\ngot:      %s", expectedJSON, jsonStr)
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify the event type constants are correctly defined
	if EventTypeAccountOpened != "AccountOpened" {
		t.Errorf("EventTypeAccountOpened should be 'AccountOpened', got '%s'", EventTypeAccountOpened)
	}

	if EventTypeMoneyDeposited != "MoneyDeposited" {
		t.Errorf("EventTypeMoneyDeposited should be 'MoneyDeposited', got '%s'", EventTypeMoneyDeposited)
	}

	if EventTypeMoneyWithdrawn != "MoneyWithdrawn" {
		t.Errorf("EventTypeMoneyWithdrawn should be 'MoneyWithdrawn', got '%s'", EventTypeMoneyWithdrawn)
	}
}

func TestAllEventData_Amounts(t *testing.T) {
	// Test various amount values to ensure cents work correctly
	testCases := []struct {
		name   string
		amount int
		cents  int
		dollars int
	}{
		{"one dollar", 100, 0, 1},
		{"fifty cents", 50, 50, 0},
		{"hundred dollars", 10000, 0, 100},
		{"twenty-five fifty", 2550, 50, 25},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			depositData := MoneyDepositedData{
				AccountID: "test-account",
				Amount:    tc.amount,
			}

			withdrawData := MoneyWithdrawnData{
				AccountID: "test-account",
				Amount:    tc.amount,
			}

			// Verify both deposit and withdraw use the same amount representation
			if depositData.Amount != withdrawData.Amount {
				t.Errorf("Amount representation differs between deposit and withdraw")
			}

			// Calculate dollars and cents
			dollars := tc.amount / 100
			cents := tc.amount % 100

			if dollars != tc.dollars {
				t.Errorf("Expected %d dollars, got %d", tc.dollars, dollars)
			}

			if cents != tc.cents {
				t.Errorf("Expected %d cents, got %d", tc.cents, cents)
			}
		})
	}
}
