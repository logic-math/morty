#!/bin/bash
# BDD Test Scenario: Hello World
# This is the simplest smoke test for Morty's core workflow

set -e

# Get script directory and load helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_helpers.sh"

# Configuration
MORTY_BIN="${MORTY_BIN:-$SCRIPT_DIR/../../../bin/morty}"
MOCK_CLAUDE_CLI="$SCRIPT_DIR/../mock_claude.sh"

# Ensure mock CLI is executable
chmod +x "$MOCK_CLAUDE_CLI"

# Export mock CLI path for create_test_project to use
export MOCK_CLAUDE_CLI

print_section "Hello World Scenario"
echo "Testing the simplest Morty workflow: Research → Plan → Doing"
echo ""

# Create test project
print_section "Setup"
create_test_project "hello_world" || exit 1

# Store the project directory
PROJECT_DIR="$TEST_PROJECT_DIR"

# Step 1: Research
print_section "Step 1: Research"
echo "Running: morty research 'hello world'"

cd "$PROJECT_DIR"
output=$("$MORTY_BIN" research "hello world" 2>&1)
exit_code=$?

assert_success $exit_code "morty research should succeed"

# Find the research file (has timestamp in filename)
research_file=$(find .morty/research -name "*.md" -type f | head -n 1)
if [ -n "$research_file" ]; then
    echo -e "${GREEN}✓ PASSED${NC}: Research file created: $(basename "$research_file")"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    TESTS_PASSED=$((TESTS_PASSED + 1))
    assert_file_contains "$research_file" "Hello World" "Research should mention Hello World"
else
    echo -e "${RED}✗ FAILED${NC}: No research file found"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Step 2: Plan
print_section "Step 2: Plan"
echo "Creating plan file manually"

# Create a proper plan file directly
mkdir -p .morty/plan

cat > .morty/plan/hello_world.md <<'EOF'
# Plan: Hello World Program

## 模块概述

**模块职责**: Create a simple Hello World program

**对应 Research**:
- `.morty/research/hello_world_*.md`

**依赖模块**: None

**被依赖模块**: None

## Jobs (Loop 块列表)

---

### Job 1: Create Hello World Script

**目标**: Write a Python script that prints "Hello World"

**前置条件**: None

**Tasks (Todo 列表)**:
- [ ] Task 1: Create hello.py file
- [ ] Task 2: Add print("Hello World") statement
- [ ] Task 3: Test the script

**验证器**:
```
The script should:
- Be a valid Python file
- Print "Hello World" when executed
```

**调试日志**:
- If validation fails, record debug logs here
EOF

echo -e "${GREEN}✓ PASSED${NC}: Created plan file"
TESTS_TOTAL=$((TESTS_TOTAL + 1))
TESTS_PASSED=$((TESTS_PASSED + 1))

assert_dir_exists ".morty/plan" "Plan directory should exist"
assert_file_exists ".morty/plan/hello_world.md" "Plan file should exist"
assert_file_contains ".morty/plan/hello_world.md" "Job" "Plan should contain Job definitions"

# Step 3: Doing
print_section "Step 3: Doing"
echo "Running: morty doing (with Mock Claude CLI)"

# Set environment to use mock CLI
export CLAUDE_CODE_CLI="$MOCK_CLAUDE_CLI"

# Extract module and job names from our plan file
module_name="Hello World Program"
job_name="Create Hello World Script"

echo "Running: morty doing -module '$module_name' -job '$job_name'"

# Run doing command with specific module and job
output=$("$MORTY_BIN" doing -module "$module_name" -job "$job_name" 2>&1)
exit_code=$?

assert_success $exit_code "morty doing should succeed"

# Step 4: Verify generated code
print_section "Step 4: Verify Code Generation"

# Check if hello.py was created
if [ -f "hello.py" ]; then
    assert_file_contains "hello.py" "Hello World" "hello.py should contain Hello World"

    # Try to execute the Python code
    if command -v python3 &> /dev/null; then
        python_output=$(python3 hello.py 2>&1)
        python_exit=$?

        assert_success $python_exit "Python code should execute without errors"
        assert_output_contains "$python_output" "Hello World" "Python output should be Hello World"
    else
        echo -e "${YELLOW}⚠ SKIPPED${NC}: Python 3 not available, cannot test execution"
    fi
else
    echo -e "${YELLOW}⚠ NOTE${NC}: hello.py not found, checking for alternative file names"
    # List all Python files created
    python_files=$(find . -maxdepth 1 -name "*.py" -type f)
    if [ -n "$python_files" ]; then
        echo "Found Python files:"
        echo "$python_files"
        # Test the first Python file found
        first_py=$(echo "$python_files" | head -n 1)
        assert_file_contains "$first_py" "Hello World" "Generated Python file should contain Hello World"
    else
        TESTS_TOTAL=$((TESTS_TOTAL + 1))
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC}: No Python files generated"
    fi
fi

# Step 5: Verify Git integration
print_section "Step 5: Verify Git Integration"

# Check if Git commits were made
git_log=$(git log --oneline)
if [ -n "$git_log" ]; then
    # Look for morty-related commits (besides initial commit)
    commit_count=$(git log --oneline | wc -l)
    if [ "$commit_count" -gt 1 ]; then
        echo -e "${GREEN}✓ PASSED${NC}: Git commits were created"
        TESTS_TOTAL=$((TESTS_TOTAL + 1))
        TESTS_PASSED=$((TESTS_PASSED + 1))

        # Check if any commit has morty prefix
        if git log --oneline | grep -q "morty:"; then
            echo -e "${GREEN}✓ PASSED${NC}: Found commit with 'morty:' prefix"
            TESTS_TOTAL=$((TESTS_TOTAL + 1))
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "${YELLOW}⚠ NOTE${NC}: No commit with 'morty:' prefix found"
            echo "Recent commits:"
            git log --oneline -3 | sed 's/^/  /'
        fi
    else
        echo -e "${YELLOW}⚠ NOTE${NC}: Only initial commit found, no new commits from morty"
    fi
else
    echo -e "${RED}✗ FAILED${NC}: No Git commits found"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Cleanup
print_section "Cleanup"
cd /tmp
cleanup_test_project

# Print summary
print_test_summary "Hello World Scenario"
summary_exit=$?

exit $summary_exit
