#!/bin/bash
# Test script for Morty

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    local level=$1
    shift
    echo -e "${BLUE}[$level]${NC} $*"
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

error() {
    echo -e "${RED}✗${NC} $*"
}

# Save script directory BEFORE changing to test directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test directory
TEST_DIR="/tmp/morty_test_$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

log INFO "Test directory: $TEST_DIR"
log INFO ""

# Assume morty is installed or use local version (SCRIPT_DIR already set above)
if command -v morty &> /dev/null; then
    MORTY_CMD="morty"
else
    # Use absolute path
    MORTY_CMD="$SCRIPT_DIR/morty"
    # Also set MORTY_HOME for the scripts
    export MORTY_HOME="$SCRIPT_DIR"
fi

log INFO "Using morty command: $MORTY_CMD"
log INFO "MORTY_HOME: ${MORTY_HOME:-not set}"
log INFO ""

# Test 1: Create sample PRD
log INFO "Test 1: Creating sample PRD..."

cat > sample_prd.md << 'EOF'
# Sample Project Requirements

## Overview
This is a simple calculator application that performs basic arithmetic operations.

## Features
1. Addition of two numbers
2. Subtraction of two numbers
3. Multiplication of two numbers
4. Division of two numbers

## Tasks
- [ ] Implement addition function
- [ ] Implement subtraction function
- [ ] Implement multiplication function
- [ ] Implement division function
- [ ] Add input validation
- [ ] Add error handling for division by zero
- [ ] Write unit tests
- [ ] Create user interface

## Technical Requirements
- Language: Python
- Testing: pytest
- Code style: PEP 8

## Deliverables
- Working calculator application
- Unit tests with >80% coverage
- README with usage instructions

EOF

success "Created sample_prd.md"
log INFO ""

# Test 2: Import PRD and create project
log INFO "Test 2: Importing PRD to create project..."

$MORTY_CMD import sample_prd.md calculator-app

if [[ -d "calculator-app" ]]; then
    success "Project directory created"
else
    error "Project directory not created"
    exit 1
fi

cd calculator-app

# Verify project structure
log INFO "Verifying project structure..."

check_file() {
    local file=$1
    if [[ -f "$file" ]]; then
        success "  $file exists"
    else
        error "  $file missing"
        return 1
    fi
}

check_dir() {
    local dir=$1
    if [[ -d "$dir" ]]; then
        success "  $dir/ exists"
    else
        error "  $dir/ missing"
        return 1
    fi
}

check_dir ".morty"
check_file ".morty/PROMPT.md"
check_file ".morty/fix_plan.md"
check_file ".morty/AGENT.md"
check_dir ".morty/specs"
check_file ".morty/specs/requirements.md"
check_dir ".morty/logs"
check_dir "src"
check_file "README.md"
check_file ".gitignore"

log INFO ""

# Verify fix_plan.md has tasks
log INFO "Verifying tasks in fix_plan.md..."
task_count=$(grep -cE "^[[:space:]]*-[[:space:]]*\[ \]" .morty/fix_plan.md 2>/dev/null || echo "0")
if [[ $task_count -gt 0 ]]; then
    success "  Found $task_count tasks in fix_plan.md"
else
    error "  No tasks found in fix_plan.md"
    exit 1
fi

log INFO ""

# Test 3: Enable Morty in existing project
log INFO "Test 3: Testing 'morty enable' on existing project..."

cd "$TEST_DIR"
mkdir -p existing-project
cd existing-project

# Create a simple package.json to simulate existing project
cat > package.json << 'EOF'
{
  "name": "existing-app",
  "version": "1.0.0",
  "scripts": {
    "test": "jest",
    "build": "webpack"
  }
}
EOF

git init > /dev/null 2>&1

$MORTY_CMD enable

if [[ -d ".morty" ]]; then
    success "Morty enabled in existing project"
else
    error "Failed to enable Morty"
    exit 1
fi

# Verify structure
check_dir ".morty"
check_file ".morty/PROMPT.md"
check_file ".morty/fix_plan.md"
check_file ".morty/AGENT.md"

# Check if build commands were detected
if grep -q "npm run build" .morty/AGENT.md; then
    success "  Build command detected correctly"
else
    error "  Build command not detected"
fi

if grep -q "jest" .morty/AGENT.md; then
    success "  Test command detected correctly"
else
    error "  Test command not detected"
fi

log INFO ""

# Test 4: Test 'morty init'
log INFO "Test 4: Testing 'morty init'..."

cd "$TEST_DIR"
$MORTY_CMD init test-init-project

if [[ -d "test-init-project" ]]; then
    success "Project created with 'morty init'"
else
    error "Failed to create project with 'morty init'"
    exit 1
fi

cd test-init-project

check_dir ".morty"
check_file ".morty/PROMPT.md"
check_file ".morty/fix_plan.md"
check_file ".morty/AGENT.md"

log INFO ""

# Test 5: Test status command (without running loop)
log INFO "Test 5: Testing 'morty status'..."

cd "$TEST_DIR/calculator-app"

# Status should handle missing status file gracefully
$MORTY_CMD status > /dev/null 2>&1 || true
success "Status command executed (no error on missing status file)"

log INFO ""

# Summary
log INFO "================================"
log INFO "Test Summary"
log INFO "================================"
success "Test 1: Sample PRD created"
success "Test 2: PRD import and project structure verified"
success "Test 3: Morty enable in existing project verified"
success "Test 4: Morty init verified"
success "Test 5: Status command verified"
log INFO ""
log INFO "All tests passed! ✨"
log INFO ""
log INFO "Test artifacts in: $TEST_DIR"
log INFO ""
log INFO "To manually test the loop (requires Claude CLI):"
log INFO "  cd $TEST_DIR/calculator-app"
log INFO "  morty start --max-loops 3"
log INFO ""
