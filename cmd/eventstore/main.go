package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/cadonna/append-only-event-store/pkg/domain/bank"
	"github.com/cadonna/append-only-event-store/pkg/eventstore"
)

const (
	// Default data directory for event storage
	defaultDataDir = "./data"
)

func main() {
	// Global flags
	dataDir := flag.String("data-dir", defaultDataDir, "Directory for event store data")
	flag.Parse()

	// Get command and arguments
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	commandArgs := args[1:]

	// Route to appropriate handler
	switch command {
	case "append":
		handleAppend(*dataDir, commandArgs)
	case "get-stream":
		handleGetStream(*dataDir, commandArgs)
	case "get-all":
		handleGetAll(*dataDir, commandArgs)
	case "bank":
		handleBank(*dataDir, commandArgs)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Event Store - Append-only event sourcing system")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  eventstore [flags] <command> [args...]")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("  --data-dir <path>    Directory for event store data (default: ./data)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  append <streamId> <eventType> <jsonData>")
	fmt.Println("      Append an event to the event store")
	fmt.Println()
	fmt.Println("  get-stream <streamId>")
	fmt.Println("      Get all events for a specific stream")
	fmt.Println()
	fmt.Println("  get-all")
	fmt.Println("      Get all events in the event store")
	fmt.Println()
	fmt.Println("  bank <subcommand> [args...]")
	fmt.Println("      Bank domain commands:")
	fmt.Println("        open <accountId> <owner>")
	fmt.Println("            Open a new bank account")
	fmt.Println("        deposit <accountId> <amount>")
	fmt.Println("            Deposit money (amount in cents)")
	fmt.Println("        withdraw <accountId> <amount>")
	fmt.Println("            Withdraw money (amount in cents)")
	fmt.Println("        balance <accountId>")
	fmt.Println("            Get account balance")
	fmt.Println()
	fmt.Println("  help")
	fmt.Println("      Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  eventstore bank open acc-1 \"Alice\"")
	fmt.Println("  eventstore bank deposit acc-1 10000")
	fmt.Println("  eventstore bank withdraw acc-1 2500")
	fmt.Println("  eventstore bank balance acc-1")
	fmt.Println("  eventstore get-stream acc-1")
	fmt.Println("  eventstore get-all")
}

// initStore creates and initializes an EventStore instance.
// If initialization fails, it prints the error and exits the program.
func initStore(dataDir string) eventstore.EventStore {
	store, err := eventstore.NewEventStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize event store: %v\n", err)
		os.Exit(1)
	}
	return store
}

// formatEvent formats an event as pretty-printed JSON for display.
func formatEvent(event *eventstore.Event) string {
	jsonBytes, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting event: %v", err)
	}
	return string(jsonBytes)
}

// handleAppend handles the 'append' command
func handleAppend(dataDir string, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: append command requires 3 arguments: <streamId> <eventType> <jsonData>")
		os.Exit(1)
	}

	streamID := args[0]
	eventType := args[1]
	jsonData := args[2]

	// Validate JSON data
	if !json.Valid([]byte(jsonData)) {
		fmt.Fprintf(os.Stderr, "Error: Invalid JSON data: %s\n", jsonData)
		os.Exit(1)
	}

	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Append event
	event, err := store.Append(streamID, eventType, json.RawMessage(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to append event: %v\n", err)
		os.Exit(1)
	}

	// Print success
	fmt.Println("Event appended successfully:")
	fmt.Println(formatEvent(event))
}

// handleGetStream handles the 'get-stream' command
func handleGetStream(dataDir string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: get-stream command requires 1 argument: <streamId>")
		os.Exit(1)
	}

	streamID := args[0]

	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Get stream events
	events, err := store.GetStream(streamID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get stream: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if len(events) == 0 {
		fmt.Printf("No events found for stream: %s\n", streamID)
		return
	}

	fmt.Printf("Found %d event(s) for stream '%s':\n\n", len(events), streamID)
	for i, event := range events {
		fmt.Printf("Event %d:\n", i+1)
		fmt.Println(formatEvent(event))
		fmt.Println()
	}
}

// handleGetAll handles the 'get-all' command
func handleGetAll(dataDir string, args []string) {
	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Get all events
	events, err := store.GetAllEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get all events: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if len(events) == 0 {
		fmt.Println("No events found in the event store")
		return
	}

	fmt.Printf("Found %d event(s) in total:\n\n", len(events))
	for i, event := range events {
		fmt.Printf("Event %d (GlobalSeq=%d, Stream=%s):\n", i+1, event.GlobalSequence, event.StreamID)
		fmt.Println(formatEvent(event))
		fmt.Println()
	}
}

// handleBank handles the 'bank' command and its subcommands
func handleBank(dataDir string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: bank command requires a subcommand")
		fmt.Fprintln(os.Stderr, "Available subcommands: open, deposit, withdraw, balance")
		os.Exit(1)
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "open":
		handleBankOpen(dataDir, subArgs)
	case "deposit":
		handleBankDeposit(dataDir, subArgs)
	case "withdraw":
		handleBankWithdraw(dataDir, subArgs)
	case "balance":
		handleBankBalance(dataDir, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown bank subcommand: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, "Available subcommands: open, deposit, withdraw, balance")
		os.Exit(1)
	}
}

// handleBankOpen handles the 'bank open' subcommand
func handleBankOpen(dataDir string, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: bank open requires 2 arguments: <accountId> <owner>")
		os.Exit(1)
	}

	accountID := args[0]
	owner := args[1]

	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Open account
	event, err := bank.OpenAccount(store, accountID, owner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open account: %v\n", err)
		os.Exit(1)
	}

	// Print success
	fmt.Printf("Account '%s' opened successfully for owner '%s'\n", accountID, owner)
	fmt.Println("\nEvent created:")
	fmt.Println(formatEvent(event))
}

// handleBankDeposit handles the 'bank deposit' subcommand
func handleBankDeposit(dataDir string, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: bank deposit requires 2 arguments: <accountId> <amount>")
		os.Exit(1)
	}

	accountID := args[0]
	amountStr := args[1]

	// Parse amount
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid amount '%s': must be an integer (cents)\n", amountStr)
		os.Exit(1)
	}

	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Deposit money
	event, err := bank.Deposit(store, accountID, amount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to deposit: %v\n", err)
		os.Exit(1)
	}

	// Print success
	fmt.Printf("Deposited %d cents ($%.2f) to account '%s'\n", amount, float64(amount)/100.0, accountID)
	fmt.Println("\nEvent created:")
	fmt.Println(formatEvent(event))
}

// handleBankWithdraw handles the 'bank withdraw' subcommand
func handleBankWithdraw(dataDir string, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: bank withdraw requires 2 arguments: <accountId> <amount>")
		os.Exit(1)
	}

	accountID := args[0]
	amountStr := args[1]

	// Parse amount
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid amount '%s': must be an integer (cents)\n", amountStr)
		os.Exit(1)
	}

	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Withdraw money
	event, err := bank.Withdraw(store, accountID, amount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to withdraw: %v\n", err)
		os.Exit(1)
	}

	// Print success
	fmt.Printf("Withdrew %d cents ($%.2f) from account '%s'\n", amount, float64(amount)/100.0, accountID)
	fmt.Println("\nEvent created:")
	fmt.Println(formatEvent(event))
}

// handleBankBalance handles the 'bank balance' subcommand
func handleBankBalance(dataDir string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: bank balance requires 1 argument: <accountId>")
		os.Exit(1)
	}

	accountID := args[0]

	// Initialize event store
	store := initStore(dataDir)
	defer store.Close()

	// Get balance
	balance, err := bank.GetBalance(store, accountID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get balance: %v\n", err)
		os.Exit(1)
	}

	// Print balance
	fmt.Printf("Account: %s\n", accountID)
	fmt.Printf("Balance: %d cents ($%.2f)\n", balance, float64(balance)/100.0)
}
