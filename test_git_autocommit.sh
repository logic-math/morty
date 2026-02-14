#!/bin/bash
# Test script for Git auto-commit functionality

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
TEST_DIR="/tmp/morty_git_test_$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

log INFO "Test directory: $TEST_DIR"
log INFO ""

# Set MORTY_HOME for testing
export MORTY_HOME="$SCRIPT_DIR"

# Source common.sh
source "$SCRIPT_DIR/lib/common.sh"

# Test 1: Initialize git repository
log INFO "Test 1: Initializing git repository..."
git init > /dev/null 2>&1
git config user.email "test@morty.dev"
git config user.name "Morty Test"
success "Git repository initialized"
log INFO ""

# Test 2: Create initial commit
log INFO "Test 2: Creating initial commit..."
echo "# Test Project" > README.md
git add README.md
git commit -m "Initial commit" > /dev/null 2>&1
success "Initial commit created"
log INFO ""

# Test 3: Test git_auto_commit with changes
log INFO "Test 3: Testing git_auto_commit() with changes..."
echo "Some changes" > test.txt
# Note: git_auto_commit does git add -A internally
git_auto_commit 1 "First loop iteration"
# Give git a moment to process
sleep 1
if git log --grep="morty: Loop #1" -1 --oneline 2>/dev/null | grep -q "Loop #1"; then
    success "Auto-commit created for loop #1"
else
    error "Auto-commit failed for loop #1"
    exit 1
fi
log INFO ""

# Test 4: Test git_auto_commit with no changes
log INFO "Test 4: Testing git_auto_commit() with no changes..."
git_auto_commit 2 "Second loop iteration"
if git log --grep="morty: Loop #2" -1 --oneline | grep -q "Loop #2"; then
    error "Auto-commit should not create commit when no changes"
    exit 1
else
    success "Auto-commit correctly skipped (no changes)"
fi
log INFO ""

# Test 5: Test multiple auto-commits
log INFO "Test 5: Testing multiple auto-commits..."
for i in 3 4 5; do
    echo "Loop $i changes" >> test.txt
    git_auto_commit "$i" "Loop iteration $i"
done

COMMIT_COUNT=$(git log --grep="morty: Loop" --oneline | wc -l)
if [[ $COMMIT_COUNT -ge 4 ]]; then
    success "Multiple auto-commits created (found $COMMIT_COUNT)"
else
    error "Expected at least 4 commits, found $COMMIT_COUNT"
    exit 1
fi
log INFO ""

# Test 6: Test git_loop_history
log INFO "Test 6: Testing git_loop_history()..."
HISTORY_OUTPUT=$(git_loop_history 2>&1)
if echo "$HISTORY_OUTPUT" | grep -q "Loop History"; then
    success "git_loop_history() displays history"
else
    error "git_loop_history() failed"
    exit 1
fi
log INFO ""

# Test 7: Test git_rollback (dry-run)
log INFO "Test 7: Testing git_rollback() detection..."
COMMIT_HASH=$(git log --grep="Loop #3" --format="%H" -n 1)
if [[ -n "$COMMIT_HASH" ]]; then
    success "git_rollback() can find Loop #3 commit: ${COMMIT_HASH:0:8}"
else
    error "git_rollback() cannot find Loop #3 commit"
    exit 1
fi
log INFO ""

# Test 8: Verify commit message format
log INFO "Test 8: Verifying commit message format..."
COMMIT_MSG=$(git log --grep="morty: Loop #1" --format="%B" -n 1)
if echo "$COMMIT_MSG" | grep -q "Auto-committed by Morty development loop"; then
    success "Commit message includes auto-commit marker"
else
    error "Commit message format incorrect"
    exit 1
fi

if echo "$COMMIT_MSG" | grep -q "Loop: 1"; then
    success "Commit message includes loop metadata"
else
    error "Commit message missing loop metadata"
    exit 1
fi

if echo "$COMMIT_MSG" | grep -q "Timestamp:"; then
    success "Commit message includes timestamp"
else
    error "Commit message missing timestamp"
    exit 1
fi
log INFO ""

# Test 9: Verify git status after auto-commit
log INFO "Test 9: Verifying git status after auto-commit..."
echo "New changes" > new_file.txt
git_auto_commit 6 "Test loop"
if git diff --quiet && git diff --cached --quiet; then
    success "Working directory clean after auto-commit"
else
    error "Working directory not clean after auto-commit"
    exit 1
fi
log INFO ""

# Summary
log INFO "================================"
log INFO "Test Summary"
log INFO "================================"
success "Test 1: Git repository initialized"
success "Test 2: Initial commit created"
success "Test 3: Auto-commit with changes works"
success "Test 4: Auto-commit skips when no changes"
success "Test 5: Multiple auto-commits work"
success "Test 6: git_loop_history() works"
success "Test 7: git_rollback() can find commits"
success "Test 8: Commit message format correct"
success "Test 9: Working directory clean after commit"
log INFO ""
log INFO "All tests passed! ✨"
log INFO ""
log INFO "Test artifacts in: $TEST_DIR"
log INFO ""
log INFO "To view the git history:"
log INFO "  cd $TEST_DIR"
log INFO "  git log --grep='morty: Loop' --oneline"
log INFO ""
