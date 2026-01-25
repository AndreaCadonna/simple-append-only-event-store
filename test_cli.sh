#!/bin/bash

# CLI Test Script for Event Store
# Tests all CLI commands end-to-end

# Note: We don't use 'set -e' because we want to continue testing even if individual tests fail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# CLI binary path
CLI="./eventstore"
TEST_DATA_DIR="./test-data"

# Print functions
print_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "[INFO] $1"
}

# Setup function
setup() {
    print_info "Setting up test environment..."

    # Build CLI if not exists
    if [ ! -f "$CLI" ]; then
        print_info "Building CLI..."
        go build -o eventstore ./cmd/eventstore
    fi

    # Clean test data directory
    rm -rf "$TEST_DATA_DIR"
    mkdir -p "$TEST_DATA_DIR"

    print_info "Setup complete"
    echo ""
}

# Cleanup function
cleanup() {
    print_info "Cleaning up..."
    rm -rf "$TEST_DATA_DIR"
}

# Test helper functions
run_command() {
    $CLI --data-dir="$TEST_DATA_DIR" "$@"
}

expect_success() {
    ((TESTS_RUN++))
    local test_name=$1
    shift
    print_test "$test_name"

    output=$(run_command "$@" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        print_pass "$test_name"
        return 0
    else
        print_fail "$test_name"
        echo "Output: $output"
        echo "Exit code: $exit_code"
        return 1
    fi
}

expect_failure() {
    ((TESTS_RUN++))
    local test_name=$1
    shift
    print_test "$test_name"

    output=$(run_command "$@" 2>&1)
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
        print_pass "$test_name"
        return 0
    else
        print_fail "$test_name - Expected failure but command succeeded"
        echo "Output: $output"
        return 1
    fi
}

expect_output_contains() {
    ((TESTS_RUN++))
    local test_name=$1
    local expected=$2
    shift 2
    print_test "$test_name"

    output=$(run_command "$@" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        if echo "$output" | grep -q "$expected"; then
            print_pass "$test_name"
            return 0
        else
            print_fail "$test_name - Expected output to contain '$expected'"
            echo "Output: $output"
            return 1
        fi
    else
        print_fail "$test_name - Command failed"
        echo "Output: $output"
        echo "Exit code: $exit_code"
        return 1
    fi
}

# Test suites

test_bank_open() {
    echo "=== Testing: Bank Open ==="

    expect_success "Open account acc-alice" bank open acc-alice "Alice Smith"
    expect_output_contains "Verify account was opened" "acc-alice" bank balance acc-alice
    expect_output_contains "Verify account has zero balance" "0 cents" bank balance acc-alice
    expect_failure "Prevent duplicate account" bank open acc-alice "Another Alice"

    echo ""
}

test_bank_deposit() {
    echo "=== Testing: Bank Deposit ==="

    expect_success "Open account acc-bob" bank open acc-bob "Bob Jones"
    expect_success "Deposit 10000 cents" bank deposit acc-bob 10000
    expect_output_contains "Verify balance is 10000" "10000 cents" bank balance acc-bob
    expect_success "Deposit another 5000 cents" bank deposit acc-bob 5000
    expect_output_contains "Verify balance is 15000" "15000 cents" bank balance acc-bob
    expect_failure "Prevent deposit to non-existent account" bank deposit acc-nonexistent 1000

    echo ""
}

test_bank_withdraw() {
    echo "=== Testing: Bank Withdraw ==="

    expect_success "Open account acc-charlie" bank open acc-charlie "Charlie Brown"
    expect_success "Deposit 20000 cents" bank deposit acc-charlie 20000
    expect_success "Withdraw 7500 cents" bank withdraw acc-charlie 7500
    expect_output_contains "Verify balance is 12500" "12500 cents" bank balance acc-charlie
    expect_failure "Prevent overdraw" bank withdraw acc-charlie 15000
    expect_output_contains "Balance unchanged after failed withdrawal" "12500 cents" bank balance acc-charlie

    echo ""
}

test_bank_balance() {
    echo "=== Testing: Bank Balance ==="

    expect_success "Open account acc-diana" bank open acc-diana "Diana Prince"
    expect_output_contains "New account has zero balance" "0 cents" bank balance acc-diana
    expect_success "Deposit 10000 cents" bank deposit acc-diana 10000
    expect_success "Deposit 5000 cents" bank deposit acc-diana 5000
    expect_success "Withdraw 3000 cents" bank withdraw acc-diana 3000
    expect_output_contains "Balance is 12000 cents" "12000 cents" bank balance acc-diana
    expect_output_contains "Balance shows dollars" "\$120.00" bank balance acc-diana
    expect_failure "Cannot get balance of non-existent account" bank balance acc-nonexistent

    echo ""
}

test_append() {
    echo "=== Testing: Generic Append ==="

    expect_success "Append event to stream-1" append stream-1 TestEvent '{"message":"Hello","value":42}'
    expect_success "Append another event to stream-1" append stream-1 AnotherEvent '{"data":"test"}'
    expect_success "Append event to stream-2" append stream-2 DifferentEvent '{"foo":"bar"}'
    expect_failure "Reject invalid JSON" append stream-3 BadEvent 'not-valid-json'

    echo ""
}

test_get_stream() {
    echo "=== Testing: Get Stream ==="

    expect_success "Open account acc-eve" bank open acc-eve "Eve Adams"
    expect_success "Deposit to acc-eve" bank deposit acc-eve 5000
    expect_success "Withdraw from acc-eve" bank withdraw acc-eve 2000
    expect_output_contains "Get stream shows all events" "AccountOpened" get-stream acc-eve
    expect_output_contains "Get stream shows deposits" "MoneyDeposited" get-stream acc-eve
    expect_output_contains "Get stream shows withdrawals" "MoneyWithdrawn" get-stream acc-eve
    expect_output_contains "Get stream shows event count" "Found 3 event" get-stream acc-eve
    expect_output_contains "Non-existent stream returns no events" "No events found" get-stream acc-nonexistent

    echo ""
}

test_get_all() {
    echo "=== Testing: Get All Events ==="

    expect_output_contains "Get all shows events from all streams" "acc-alice" get-all
    expect_output_contains "Get all shows multiple streams" "acc-bob" get-all
    expect_output_contains "Get all shows global sequence" "GlobalSeq=" get-all

    echo ""
}

test_persistence() {
    echo "=== Testing: Persistence Across Restarts ==="

    # Record balance before "restart"
    local balance_before=$(run_command bank balance acc-bob 2>&1 | grep "Balance:" | awk '{print $2}')
    print_info "Balance before restart: $balance_before cents"

    # Simulate restart by just running another command
    # (EventStore is opened/closed on each command)
    expect_output_contains "Balance persists after restart" "$balance_before cents" bank balance acc-bob

    # Add more transactions and verify persistence
    expect_success "Deposit after restart" bank deposit acc-bob 1000
    expect_output_contains "New balance includes post-restart transaction" "16000 cents" bank balance acc-bob

    echo ""
}

test_multiple_accounts() {
    echo "=== Testing: Multiple Accounts ==="

    expect_success "Open account acc-frank" bank open acc-frank "Frank Miller"
    expect_success "Open account acc-grace" bank open acc-grace "Grace Hopper"
    expect_success "Deposit to acc-frank" bank deposit acc-frank 10000
    expect_success "Deposit to acc-grace" bank deposit acc-grace 20000
    expect_success "Withdraw from acc-frank" bank withdraw acc-frank 3000
    expect_success "Withdraw from acc-grace" bank withdraw acc-grace 5000
    expect_output_contains "acc-frank has correct balance" "7000 cents" bank balance acc-frank
    expect_output_contains "acc-grace has correct balance" "15000 cents" bank balance acc-grace

    echo ""
}

test_edge_cases() {
    echo "=== Testing: Edge Cases ==="

    expect_success "Open account acc-edge" bank open acc-edge "Edge Case"
    expect_failure "Cannot deposit 0" bank deposit acc-edge 0
    expect_failure "Cannot withdraw 0" bank withdraw acc-edge 0
    expect_failure "Cannot deposit negative" bank deposit acc-edge -1000
    expect_failure "Cannot withdraw negative" bank withdraw acc-edge -1000
    expect_success "Can deposit 1 cent" bank deposit acc-edge 1
    expect_success "Can withdraw 1 cent" bank withdraw acc-edge 1
    expect_output_contains "Balance is 0 after depositing and withdrawing 1" "0 cents" bank balance acc-edge

    echo ""
}

test_event_ordering() {
    echo "=== Testing: Event Ordering ==="

    expect_success "Open account acc-order" bank open acc-order "Order Test"
    expect_success "Multiple deposits" bank deposit acc-order 1000
    expect_success "Multiple deposits" bank deposit acc-order 2000
    expect_success "Multiple deposits" bank deposit acc-order 3000

    # Check that versions are sequential
    local stream_output=$(run_command get-stream acc-order 2>&1)
    print_info "Checking event versions are sequential..."
    if echo "$stream_output" | grep -q '"version": 1' && \
       echo "$stream_output" | grep -q '"version": 2' && \
       echo "$stream_output" | grep -q '"version": 3' && \
       echo "$stream_output" | grep -q '"version": 4'; then
        ((TESTS_RUN++))
        print_pass "Event versions are sequential"
        ((TESTS_PASSED++))
    else
        ((TESTS_RUN++))
        print_fail "Event versions are not sequential"
        ((TESTS_FAILED++))
    fi

    echo ""
}

# Main test execution
main() {
    echo "========================================="
    echo "Event Store CLI Integration Tests"
    echo "========================================="
    echo ""

    setup

    # Run all test suites
    test_bank_open
    test_bank_deposit
    test_bank_withdraw
    test_bank_balance
    test_append
    test_get_stream
    test_get_all
    test_persistence
    test_multiple_accounts
    test_edge_cases
    test_event_ordering

    cleanup

    # Print summary
    echo "========================================="
    echo "Test Summary"
    echo "========================================="
    echo "Total Tests: $TESTS_RUN"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        exit 1
    fi
}

# Run main
main
