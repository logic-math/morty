# Morty Test Suite

Automated tests for Morty features.

## Test Files

### 1. Plan Mode Tests (`test_plan_mode.sh`)

Tests the plan mode functionality and project structure generation.

**Test Coverage (10 tests):**
1. Sample PRD creation
2. Plan mode components verification
3. Command routing verification
4. Help text verification
5. System prompt content verification
6. Plan script structure verification
7. Claude command construction verification
8. Project type detection verification
9. AGENT.md templates verification
10. PROMPT.md template verification

**Run:**
```bash
./tests/test_plan_mode.sh
```

**Expected Output:**
```
✓ Test 1: Sample PRD created
✓ Test 2: Plan mode components verified
✓ Test 3: Command routing verified
✓ Test 4: Help text verified
✓ Test 5: System prompt content verified
✓ Test 6: Plan script structure verified
✓ Test 7: Claude command construction verified
✓ Test 8: Project type detection verified
✓ Test 9: AGENT.md templates verified
✓ Test 10: PROMPT.md template verified

All tests passed! ✨
```

### 2. Git Auto-Commit Tests (`test_git_autocommit.sh`)

Tests the git auto-commit functionality, rollback, and history features.

**Test Coverage (9 tests):**
1. Git repository initialization
2. Initial commit creation
3. Auto-commit with changes
4. Auto-commit skips when no changes
5. Multiple auto-commits
6. git_loop_history() functionality
7. git_rollback() commit detection
8. Commit message format validation
9. Working directory clean after commit

**Run:**
```bash
./tests/test_git_autocommit.sh
```

**Expected Output:**
```
✓ Test 1: Git repository initialized
✓ Test 2: Initial commit created
✓ Test 3: Auto-commit with changes works
✓ Test 4: Auto-commit skips when no changes
✓ Test 5: Multiple auto-commits work
✓ Test 6: git_loop_history() works
✓ Test 7: git_rollback() can find commits
✓ Test 8: Commit message format correct
✓ Test 9: Working directory clean after commit

All tests passed! ✨
```

## Running All Tests

```bash
cd morty

# Run all tests
for test in tests/test_*.sh; do
    echo "Running $test..."
    $test
    echo ""
done
```

Or create a test runner:

```bash
#!/bin/bash
# tests/run_all_tests.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FAILED=0

for test in "$SCRIPT_DIR"/test_*.sh; do
    echo "========================================="
    echo "Running $(basename "$test")..."
    echo "========================================="

    if "$test"; then
        echo "✓ $(basename "$test") passed"
    else
        echo "✗ $(basename "$test") failed"
        FAILED=$((FAILED + 1))
    fi
    echo ""
done

if [[ $FAILED -eq 0 ]]; then
    echo "All tests passed! ✨"
    exit 0
else
    echo "$FAILED test(s) failed"
    exit 1
fi
```

## Test Environment

Tests run in isolated temporary directories:
- Plan mode tests: `/tmp/morty_plan_test_<timestamp>`
- Git tests: `/tmp/morty_git_test_<timestamp>`

Test artifacts are preserved for inspection after test completion.

## Test Dependencies

### Required
- Bash 4.0+
- Git (for git auto-commit tests)
- Standard Unix tools (grep, sed, cat, etc.)

### Optional
- Claude Code CLI (for full integration testing)
- tmux (for monitoring tests)

## Adding New Tests

When adding new features, create corresponding test files:

1. **Create test file**: `tests/test_<feature>.sh`
2. **Follow naming convention**: `test_<feature_name>.sh`
3. **Use consistent structure**:
   ```bash
   #!/bin/bash
   set -e

   # Setup
   TEST_DIR="/tmp/morty_<feature>_test_$(date +%s)"
   mkdir -p "$TEST_DIR"
   cd "$TEST_DIR"

   # Tests
   log INFO "Test 1: Description..."
   # test code
   success "Test 1 passed"

   # Cleanup (optional)
   # Test artifacts preserved by default
   ```

4. **Make executable**: `chmod +x tests/test_<feature>.sh`
5. **Update this README** with test description

## Test Status

| Test Suite | Tests | Status | Last Run |
|------------|-------|--------|----------|
| Plan Mode | 10 | ✅ Passing | 2026-02-14 |
| Git Auto-Commit | 9 | ✅ Passing | 2026-02-14 |

**Total**: 19 tests, all passing

## Continuous Integration

Tests can be integrated into CI/CD pipelines:

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run tests
        run: |
          for test in tests/test_*.sh; do
            $test
          done
```

## Troubleshooting

### Tests fail with "command not found"
Ensure MORTY_HOME is set correctly in tests:
```bash
export MORTY_HOME="$SCRIPT_DIR"
```

### Git tests fail
Ensure git is installed and configured:
```bash
git config --global user.email "test@morty.dev"
git config --global user.name "Morty Test"
```

### Permission denied
Make test scripts executable:
```bash
chmod +x tests/*.sh
```

## Test Coverage

Tests cover:
- ✅ Plan mode functionality
- ✅ Git auto-commit features
- ✅ Command routing
- ✅ Project structure generation
- ✅ Template generation
- ✅ Error handling
- ✅ Commit message format
- ✅ Rollback detection

Not yet covered:
- ⏳ Development loop execution (requires Claude Code)
- ⏳ tmux monitoring integration
- ⏳ Project enablement workflow
- ⏳ End-to-end integration tests

## Contributing

When contributing tests:
1. Follow existing test structure and naming
2. Include clear test descriptions
3. Ensure tests are isolated and repeatable
4. Update this README with new tests
5. Verify all tests pass before committing

---

**Test Suite Version**: 0.2.1
**Last Updated**: 2026-02-14
