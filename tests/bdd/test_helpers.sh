#!/bin/bash
# Test helper functions for BDD testing

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test project path (will be set by create_test_project)
TEST_PROJECT_DIR=""

# Create a temporary test project with Git initialization
create_test_project() {
    local project_name="${1:-test_project}"

    # Create temporary directory
    TEST_PROJECT_DIR=$(mktemp -d "/tmp/${project_name}_XXXXXX")

    if [ ! -d "$TEST_PROJECT_DIR" ]; then
        echo -e "${RED}✗ FAILED${NC}: Could not create test project directory"
        return 1
    fi

    # Initialize Git repository
    cd "$TEST_PROJECT_DIR" || return 1
    git init -q
    git config user.name "BDD Test"
    git config user.email "bdd@test.local"

    # Create initial commit
    echo "# Test Project" > README.md
    git add README.md
    git commit -q -m "Initial commit"

    # Create .morty directory structure
    mkdir -p .morty/research .morty/plan

    # Copy prompts directory from Morty project
    # Find the morty project root (go up from tests/bdd)
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local morty_root="$(cd "$script_dir/../.." && pwd)"

    if [ -d "$morty_root/prompts" ]; then
        cp -r "$morty_root/prompts" "$TEST_PROJECT_DIR/"
        echo -e "${GREEN}✓${NC} Copied prompts directory"
    else
        echo -e "${YELLOW}⚠${NC} Prompts directory not found at $morty_root/prompts"
    fi

    # Create config file to set mock Claude CLI path
    if [ -n "$MOCK_CLAUDE_CLI" ] && [ -f "$MOCK_CLAUDE_CLI" ]; then
        cat > "$TEST_PROJECT_DIR/.morty/settings.json" <<EOF
{
  "version": "2.0",
  "ai_cli": {
    "command": "$MOCK_CLAUDE_CLI"
  }
}
EOF
        echo -e "${GREEN}✓${NC} Created config with mock CLI path"
    fi

    echo -e "${GREEN}✓${NC} Created test project: $TEST_PROJECT_DIR"
    return 0
}

# Cleanup test project
cleanup_test_project() {
    if [ -n "$TEST_PROJECT_DIR" ] && [ -d "$TEST_PROJECT_DIR" ]; then
        rm -rf "$TEST_PROJECT_DIR"
        echo -e "${GREEN}✓${NC} Cleaned up test project: $TEST_PROJECT_DIR"
    fi
    TEST_PROJECT_DIR=""
}

# Assert that a command succeeded (exit code 0)
assert_success() {
    local exit_code=$1
    local message="${2:-Command should succeed}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [ "$exit_code" -eq 0 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message (exit code: $exit_code)"
        return 1
    fi
}

# Assert that a command failed (exit code non-zero)
assert_failure() {
    local exit_code=$1
    local message="${2:-Command should fail}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [ "$exit_code" -ne 0 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message (expected failure but succeeded)"
        return 1
    fi
}

# Assert that a file exists
assert_file_exists() {
    local file_path="$1"
    local message="${2:-File should exist: $file_path}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [ -f "$file_path" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message"
        return 1
    fi
}

# Assert that a directory exists
assert_dir_exists() {
    local dir_path="$1"
    local message="${2:-Directory should exist: $dir_path}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [ -d "$dir_path" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message"
        return 1
    fi
}

# Assert that a file contains specific content
assert_file_contains() {
    local file_path="$1"
    local pattern="$2"
    local message="${3:-File should contain: $pattern}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [ ! -f "$file_path" ]; then
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message (file not found: $file_path)"
        return 1
    fi

    if grep -q "$pattern" "$file_path"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message"
        echo -e "  ${YELLOW}File content:${NC}"
        head -n 10 "$file_path" | sed 's/^/    /'
        return 1
    fi
}

# Assert that a Git commit exists with specific message pattern
assert_git_commit_exists() {
    local pattern="$1"
    local message="${2:-Git commit should exist with pattern: $pattern}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if git log --oneline | grep -q "$pattern"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message"
        echo -e "  ${YELLOW}Recent commits:${NC}"
        git log --oneline -5 | sed 's/^/    /'
        return 1
    fi
}

# Assert that output contains specific text
assert_output_contains() {
    local output="$1"
    local pattern="$2"
    local message="${3:-Output should contain: $pattern}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if echo "$output" | grep -q "$pattern"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC}: $message"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: $message"
        echo -e "  ${YELLOW}Actual output:${NC}"
        echo "$output" | head -n 10 | sed 's/^/    /'
        return 1
    fi
}

# Print test summary
print_test_summary() {
    local scenario_name="${1:-Test Scenario}"

    echo ""
    echo "========================================"
    echo "  $scenario_name - Summary"
    echo "========================================"
    echo -e "Total Tests:  $TESTS_TOTAL"

    if [ "$TESTS_PASSED" -gt 0 ]; then
        echo -e "${GREEN}Passed:       $TESTS_PASSED${NC}"
    else
        echo -e "Passed:       $TESTS_PASSED"
    fi

    if [ "$TESTS_FAILED" -gt 0 ]; then
        echo -e "${RED}Failed:       $TESTS_FAILED${NC}"
    else
        echo -e "Failed:       $TESTS_FAILED"
    fi

    echo "========================================"

    if [ "$TESTS_FAILED" -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        return 1
    fi
}

# Reset test counters (useful for running multiple test scenarios)
reset_test_counters() {
    TESTS_TOTAL=0
    TESTS_PASSED=0
    TESTS_FAILED=0
}

# Print section header
print_section() {
    local title="$1"
    echo ""
    echo -e "${BLUE}=== $title ===${NC}"
}

# Create status.json file for a module with jobs
create_status_json() {
    local module_name="$1"
    shift
    local job_names=("$@")

    local status_file=".morty/status.json"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Start JSON
    cat > "$status_file" <<EOF
{
  "version": "1.0",
  "global": {
    "status": "pending",
    "current_module": "",
    "current_job": "",
    "last_update": "$timestamp"
  },
  "modules": {
    "$module_name": {
      "status": "pending",
      "jobs": {
EOF

    # Add jobs
    local first=true
    for job_name in "${job_names[@]}"; do
        if [ "$first" = false ]; then
            echo "," >> "$status_file"
        fi
        first=false

        cat >> "$status_file" <<EOF
        "$job_name": {
          "status": "pending",
          "loop_count": 0,
          "retry_count": 0,
          "tasks_total": 0,
          "tasks_completed": 0,
          "started_at": "0001-01-01T00:00:00Z",
          "completed_at": "0001-01-01T00:00:00Z",
          "updated_at": "$timestamp"
        }
EOF
    done

    # Close JSON
    cat >> "$status_file" <<EOF

      },
      "updated_at": "$timestamp"
    }
  }
}
EOF

    echo -e "${GREEN}✓${NC} Created status.json for module '$module_name' with ${#job_names[@]} job(s)"
}

# Export functions and variables
export -f create_test_project
export -f cleanup_test_project
export -f create_status_json
export -f assert_success
export -f assert_failure
export -f assert_file_exists
export -f assert_dir_exists
export -f assert_file_contains
export -f assert_git_commit_exists
export -f assert_output_contains
export -f print_test_summary
export -f reset_test_counters
export -f print_section

export TESTS_TOTAL TESTS_PASSED TESTS_FAILED
export TEST_PROJECT_DIR
export RED GREEN YELLOW BLUE NC
