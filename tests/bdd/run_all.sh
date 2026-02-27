#!/bin/bash
# BDD Test Runner - Execute all test scenarios and generate report

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
MORTY_BIN="${MORTY_BIN:-$SCRIPT_DIR/../../bin/morty}"
MOCK_CLAUDE_CLI="$SCRIPT_DIR/mock_claude.sh"
SCENARIOS_DIR="$SCRIPT_DIR/scenarios"

# Test results
TOTAL_SCENARIOS=0
PASSED_SCENARIOS=0
FAILED_SCENARIOS=0
declare -a FAILED_SCENARIO_NAMES

# Print banner
print_banner() {
    echo ""
    echo -e "${CYAN}========================================"
    echo "  Morty BDD Test Suite"
    echo "========================================${NC}"
    echo ""
}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}=== Checking Prerequisites ===${NC}"
    echo ""

    local all_ok=true

    # Check Morty binary
    if [ -f "$MORTY_BIN" ]; then
        echo -e "${GREEN}✓${NC} Morty binary found: $MORTY_BIN"

        # Check if executable
        if [ -x "$MORTY_BIN" ]; then
            echo -e "${GREEN}✓${NC} Morty binary is executable"
        else
            echo -e "${RED}✗${NC} Morty binary is not executable"
            all_ok=false
        fi

        # Try to get version
        version_output=$("$MORTY_BIN" version 2>&1 || echo "")
        if [ -n "$version_output" ]; then
            echo -e "${GREEN}✓${NC} Morty version check passed"
            echo "  Version info: $(echo "$version_output" | head -n 1)"
        fi
    else
        echo -e "${RED}✗${NC} Morty binary not found: $MORTY_BIN"
        echo -e "  ${YELLOW}Please run: ./scripts/build.sh${NC}"
        all_ok=false
    fi

    # Check Mock Claude CLI
    if [ -f "$MOCK_CLAUDE_CLI" ]; then
        echo -e "${GREEN}✓${NC} Mock Claude CLI found: $MOCK_CLAUDE_CLI"

        if [ -x "$MOCK_CLAUDE_CLI" ]; then
            echo -e "${GREEN}✓${NC} Mock Claude CLI is executable"
        else
            echo -e "${YELLOW}⚠${NC} Mock Claude CLI is not executable, fixing..."
            chmod +x "$MOCK_CLAUDE_CLI"
        fi
    else
        echo -e "${RED}✗${NC} Mock Claude CLI not found: $MOCK_CLAUDE_CLI"
        all_ok=false
    fi

    # Check test helpers
    if [ -f "$SCRIPT_DIR/test_helpers.sh" ]; then
        echo -e "${GREEN}✓${NC} Test helpers found"
    else
        echo -e "${RED}✗${NC} Test helpers not found: $SCRIPT_DIR/test_helpers.sh"
        all_ok=false
    fi

    # Check scenarios directory
    if [ -d "$SCENARIOS_DIR" ]; then
        scenario_count=$(find "$SCENARIOS_DIR" -name "test_*.sh" -type f | wc -l)
        if [ "$scenario_count" -gt 0 ]; then
            echo -e "${GREEN}✓${NC} Found $scenario_count test scenario(s)"
        else
            echo -e "${RED}✗${NC} No test scenarios found in: $SCENARIOS_DIR"
            all_ok=false
        fi
    else
        echo -e "${RED}✗${NC} Scenarios directory not found: $SCENARIOS_DIR"
        all_ok=false
    fi

    # Check for Python (optional)
    if command -v python3 &> /dev/null; then
        echo -e "${GREEN}✓${NC} Python 3 available (for code execution tests)"
    else
        echo -e "${YELLOW}⚠${NC} Python 3 not found (some tests will be skipped)"
    fi

    echo ""

    if [ "$all_ok" = false ]; then
        echo -e "${RED}✗ Prerequisites check failed${NC}"
        echo ""
        exit 1
    else
        echo -e "${GREEN}✓ All prerequisites satisfied${NC}"
        echo ""
    fi
}

# Discover test scenarios
discover_scenarios() {
    find "$SCENARIOS_DIR" -name "test_*.sh" -type f | sort
}

# Run a single scenario
run_scenario() {
    local scenario_path="$1"
    local scenario_name=$(basename "$scenario_path" .sh)

    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}Running: $scenario_name${NC}"
    echo -e "${CYAN}========================================${NC}"

    TOTAL_SCENARIOS=$((TOTAL_SCENARIOS + 1))

    # Set environment variables
    export MORTY_BIN
    export CLAUDE_CODE_CLI="$MOCK_CLAUDE_CLI"
    export MOCK_LOG_FILE="/tmp/mock_claude_${scenario_name}.log"

    # Run the scenario
    local start_time=$(date +%s)

    if bash "$scenario_path"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))

        echo ""
        echo -e "${GREEN}✓ PASSED${NC}: $scenario_name (${duration}s)"
        PASSED_SCENARIOS=$((PASSED_SCENARIOS + 1))
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))

        echo ""
        echo -e "${RED}✗ FAILED${NC}: $scenario_name (${duration}s)"
        FAILED_SCENARIOS=$((FAILED_SCENARIOS + 1))
        FAILED_SCENARIO_NAMES+=("$scenario_name")
        return 1
    fi
}

# Print final report
print_report() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}  Test Report${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
    echo "Total Scenarios:  $TOTAL_SCENARIOS"

    if [ "$PASSED_SCENARIOS" -gt 0 ]; then
        echo -e "${GREEN}Passed:           $PASSED_SCENARIOS${NC}"
    else
        echo "Passed:           $PASSED_SCENARIOS"
    fi

    if [ "$FAILED_SCENARIOS" -gt 0 ]; then
        echo -e "${RED}Failed:           $FAILED_SCENARIOS${NC}"
    else
        echo "Failed:           $FAILED_SCENARIOS"
    fi

    echo ""

    if [ "$FAILED_SCENARIOS" -gt 0 ]; then
        echo -e "${RED}Failed Scenarios:${NC}"
        for scenario in "${FAILED_SCENARIO_NAMES[@]}"; do
            echo -e "  ${RED}✗${NC} $scenario"
        done
        echo ""
    fi

    echo -e "${CYAN}========================================${NC}"

    if [ "$FAILED_SCENARIOS" -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        echo ""
        return 1
    fi
}

# Main execution
main() {
    print_banner
    check_prerequisites

    echo -e "${BLUE}=== Discovering Test Scenarios ===${NC}"
    echo ""

    scenarios=$(discover_scenarios)
    scenario_count=$(echo "$scenarios" | wc -l)

    if [ -z "$scenarios" ]; then
        echo -e "${RED}✗ No test scenarios found${NC}"
        exit 1
    fi

    echo "Found $scenario_count scenario(s):"
    echo "$scenarios" | while read -r scenario; do
        echo "  - $(basename "$scenario" .sh)"
    done

    echo ""
    echo -e "${BLUE}=== Running Test Scenarios ===${NC}"

    # Run each scenario
    while IFS= read -r scenario; do
        run_scenario "$scenario"
    done <<< "$scenarios"

    # Print final report
    print_report
    report_exit=$?

    # Show mock logs location
    echo -e "${BLUE}Mock Claude CLI logs:${NC}"
    echo "  /tmp/mock_claude_*.log"
    echo ""

    exit $report_exit
}

# Run main function
main "$@"
