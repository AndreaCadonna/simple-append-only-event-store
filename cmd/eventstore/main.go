package main

import (
	"flag"
	"fmt"
	"os"
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

// handleAppend handles the 'append' command
func handleAppend(dataDir string, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: append command requires 3 arguments: <streamId> <eventType> <jsonData>")
		os.Exit(1)
	}

	streamID := args[0]
	eventType := args[1]
	jsonData := args[2]

	fmt.Printf("Not implemented: append\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
	fmt.Printf("  StreamID: %s\n", streamID)
	fmt.Printf("  EventType: %s\n", eventType)
	fmt.Printf("  Data: %s\n", jsonData)
}

// handleGetStream handles the 'get-stream' command
func handleGetStream(dataDir string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: get-stream command requires 1 argument: <streamId>")
		os.Exit(1)
	}

	streamID := args[0]

	fmt.Printf("Not implemented: get-stream\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
	fmt.Printf("  StreamID: %s\n", streamID)
}

// handleGetAll handles the 'get-all' command
func handleGetAll(dataDir string, args []string) {
	fmt.Printf("Not implemented: get-all\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
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

	fmt.Printf("Not implemented: bank open\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
	fmt.Printf("  AccountID: %s\n", accountID)
	fmt.Printf("  Owner: %s\n", owner)
}

// handleBankDeposit handles the 'bank deposit' subcommand
func handleBankDeposit(dataDir string, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: bank deposit requires 2 arguments: <accountId> <amount>")
		os.Exit(1)
	}

	accountID := args[0]
	amount := args[1]

	fmt.Printf("Not implemented: bank deposit\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
	fmt.Printf("  AccountID: %s\n", accountID)
	fmt.Printf("  Amount: %s\n", amount)
}

// handleBankWithdraw handles the 'bank withdraw' subcommand
func handleBankWithdraw(dataDir string, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: bank withdraw requires 2 arguments: <accountId> <amount>")
		os.Exit(1)
	}

	accountID := args[0]
	amount := args[1]

	fmt.Printf("Not implemented: bank withdraw\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
	fmt.Printf("  AccountID: %s\n", accountID)
	fmt.Printf("  Amount: %s\n", amount)
}

// handleBankBalance handles the 'bank balance' subcommand
func handleBankBalance(dataDir string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: bank balance requires 1 argument: <accountId>")
		os.Exit(1)
	}

	accountID := args[0]

	fmt.Printf("Not implemented: bank balance\n")
	fmt.Printf("  Data Dir: %s\n", dataDir)
	fmt.Printf("  AccountID: %s\n", accountID)
}
