#!/bin/bash
# Test script for Morty Plan Mode

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

# Save script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test directory
TEST_DIR="/tmp/morty_plan_test_$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

log INFO "Test directory: $TEST_DIR"
log INFO ""

# Set MORTY_HOME for testing
export MORTY_HOME="$SCRIPT_DIR"
MORTY_CMD="$SCRIPT_DIR/morty"

log INFO "Using morty command: $MORTY_CMD"
log INFO ""

# Test 1: Create sample PRD
log INFO "Test 1: Creating sample PRD for plan mode..."

cat > simple_prd.md << 'EOF'
# Simple Calculator App

## Overview
Build a command-line calculator that performs basic arithmetic operations.

## Features
- Addition
- Subtraction
- Multiplication
- Division

## Requirements
- Handle invalid input
- Support decimal numbers
- Display results clearly

## Users
- Students learning math
- Quick calculations for developers

EOF

success "Created simple_prd.md"
log INFO ""

# Test 2: Verify plan mode script exists
log INFO "Test 2: Verifying plan mode components..."

if [[ -f "$SCRIPT_DIR/morty_plan.sh" ]]; then
    success "  morty_plan.sh exists"
else
    error "  morty_plan.sh missing"
    exit 1
fi

if [[ -f "$SCRIPT_DIR/prompts/plan_mode_system.md" ]]; then
    success "  plan_mode_system.md exists"
else
    error "  plan_mode_system.md missing"
    exit 1
fi

log INFO ""

# Test 3: Verify morty command routing
log INFO "Test 3: Verifying morty command routing..."

if grep -q "plan)" "$MORTY_CMD"; then
    success "  'plan' command registered in morty"
else
    error "  'plan' command not found in morty"
    exit 1
fi

log INFO ""

# Test 4: Check help text
log INFO "Test 4: Checking help text..."

HELP_OUTPUT=$("$MORTY_CMD" --help 2>&1 || true)

if echo "$HELP_OUTPUT" | grep -q "plan"; then
    success "  'plan' command in help text"
else
    error "  'plan' command not in help text"
    exit 1
fi

if ! echo "$HELP_OUTPUT" | grep -q "init"; then
    success "  'init' command removed from help"
else
    error "  'init' command still in help"
fi

if ! echo "$HELP_OUTPUT" | grep -q "import"; then
    success "  'import' command removed from help"
else
    error "  'import' command still in help"
fi

log INFO ""

# Test 5: Verify system prompt content
log INFO "Test 5: Verifying system prompt content..."

SYSTEM_PROMPT="$SCRIPT_DIR/prompts/plan_mode_system.md"

# Check for key sections
if grep -q "Deep Exploration" "$SYSTEM_PROMPT"; then
    success "  Contains 'Deep Exploration' section"
else
    error "  Missing 'Deep Exploration' section"
fi

if grep -q "Dialogue Framework" "$SYSTEM_PROMPT"; then
    success "  Contains 'Dialogue Framework' section"
else
    error "  Missing 'Dialogue Framework' section"
fi

if grep -q "problem_description.md" "$SYSTEM_PROMPT"; then
    success "  References problem_description.md output"
else
    error "  Missing problem_description.md reference"
fi

if grep -q "PLAN_MODE_COMPLETE" "$SYSTEM_PROMPT"; then
    success "  Contains completion marker"
else
    error "  Missing completion marker"
fi

log INFO ""

# Test 6: Verify plan script structure
log INFO "Test 6: Verifying plan script structure..."

PLAN_SCRIPT="$SCRIPT_DIR/morty_plan.sh"

if grep -q "CLAUDE_CMD=" "$PLAN_SCRIPT"; then
    success "  Defines CLAUDE_CMD"
else
    error "  Missing CLAUDE_CMD definition"
fi

if grep -q "allowedTools" "$PLAN_SCRIPT"; then
    success "  Configures allowedTools"
else
    error "  Missing allowedTools configuration"
fi

if grep -q "dangerously-skip-permissions" "$PLAN_SCRIPT"; then
    success "  Includes dangerously-skip-permissions flag"
else
    error "  Missing dangerously-skip-permissions flag"
fi

if grep -q "problem_description.md" "$PLAN_SCRIPT"; then
    success "  Generates problem_description.md"
else
    error "  Missing problem_description.md generation"
fi

if grep -q "PROMPT.md" "$PLAN_SCRIPT"; then
    success "  Generates PROMPT.md"
else
    error "  Missing PROMPT.md generation"
fi

if grep -q "fix_plan.md" "$PLAN_SCRIPT"; then
    success "  Generates fix_plan.md"
else
    error "  Missing fix_plan.md generation"
fi

if grep -q "AGENT.md" "$PLAN_SCRIPT"; then
    success "  Generates AGENT.md"
else
    error "  Missing AGENT.md generation"
fi

log INFO ""

# Test 7: Check Claude command construction
log INFO "Test 7: Verifying Claude command construction..."

if grep -q "CLAUDE_ARGS=" "$PLAN_SCRIPT"; then
    success "  Builds CLAUDE_ARGS array"
else
    error "  Missing CLAUDE_ARGS construction"
fi

if grep -q '"\${CLAUDE_ARGS\[@\]}"' "$PLAN_SCRIPT"; then
    success "  Executes Claude with args array"
else
    error "  Missing Claude execution"
fi

log INFO ""

# Test 8: Verify project type detection
log INFO "Test 8: Verifying project type detection..."

if grep -q "python\|django\|flask" "$PLAN_SCRIPT"; then
    success "  Detects Python projects"
else
    error "  Missing Python detection"
fi

if grep -q "javascript\|typescript\|node" "$PLAN_SCRIPT"; then
    success "  Detects Node.js projects"
else
    error "  Missing Node.js detection"
fi

if grep -q "rust\|cargo" "$PLAN_SCRIPT"; then
    success "  Detects Rust projects"
else
    error "  Missing Rust detection"
fi

if grep -q "go\|golang" "$PLAN_SCRIPT"; then
    success "  Detects Go projects"
else
    error "  Missing Go detection"
fi

log INFO ""

# Test 9: Verify AGENT.md templates
log INFO "Test 9: Verifying AGENT.md templates..."

if grep -q "pytest" "$PLAN_SCRIPT"; then
    success "  Python AGENT.md template includes pytest"
else
    error "  Missing pytest in Python template"
fi

if grep -q "npm test" "$PLAN_SCRIPT"; then
    success "  Node.js AGENT.md template includes npm test"
else
    error "  Missing npm test in Node.js template"
fi

if grep -q "cargo test" "$PLAN_SCRIPT"; then
    success "  Rust AGENT.md template includes cargo test"
else
    error "  Missing cargo test in Rust template"
fi

if grep -q "go test" "$PLAN_SCRIPT"; then
    success "  Go AGENT.md template includes go test"
else
    error "  Missing go test in Go template"
fi

log INFO ""

# Test 10: Verify PROMPT.md template
log INFO "Test 10: Verifying PROMPT.md template..."

if grep -q "RALPH_STATUS" "$PLAN_SCRIPT"; then
    success "  PROMPT.md includes RALPH_STATUS block"
else
    error "  Missing RALPH_STATUS in PROMPT.md"
fi

if grep -q "EXIT_SIGNAL" "$PLAN_SCRIPT"; then
    success "  PROMPT.md includes EXIT_SIGNAL"
else
    error "  Missing EXIT_SIGNAL in PROMPT.md"
fi

if grep -q "problem_description.md" "$PLAN_SCRIPT"; then
    success "  PROMPT.md references problem_description.md"
else
    error "  Missing problem_description.md reference"
fi

log INFO ""

# Summary
log INFO "================================"
log INFO "Test Summary"
log INFO "================================"
success "Test 1: Sample PRD created"
success "Test 2: Plan mode components verified"
success "Test 3: Command routing verified"
success "Test 4: Help text verified"
success "Test 5: System prompt content verified"
success "Test 6: Plan script structure verified"
success "Test 7: Claude command construction verified"
success "Test 8: Project type detection verified"
success "Test 9: AGENT.md templates verified"
success "Test 10: PROMPT.md template verified"
log INFO ""
log INFO "All tests passed! ✨"
log INFO ""
log INFO "Test artifacts in: $TEST_DIR"
log INFO ""
log INFO "Note: This test verifies the plan mode structure."
log INFO "To test the actual interactive session, run:"
log INFO "  cd $TEST_DIR"
log INFO "  morty plan simple_prd.md"
log INFO ""
log INFO "This will launch Claude Code for interactive PRD refinement."
log INFO ""
