# Morty BDD Test Suite

This directory contains Behavior-Driven Development (BDD) tests for Morty, validating complete user journeys through the Research → Plan → Doing workflow.

## Quick Start

### Prerequisites

- Go 1.21+ (for building Morty)
- Bash shell
- Git
- Python 3 (optional, for code execution tests)

### Running Tests

1. **Build Morty**:
   ```bash
   cd /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/morty
   ./scripts/build.sh
   ```

2. **Run all BDD tests**:
   ```bash
   cd tests/bdd
   ./run_all.sh
   ```

3. **Run a single scenario**:
   ```bash
   cd tests/bdd
   ./scenarios/test_hello_world.sh
   ```

### Expected Output

```
========================================
  Morty BDD Test Suite
========================================

=== Checking Prerequisites ===

✓ Morty binary found: ../../bin/morty
✓ Morty binary is executable
✓ Found 2 test scenario(s)
✓ All prerequisites satisfied

=== Running Test Scenarios ===

========================================
Running: test_hello_world
========================================

=== Setup ===
✓ Created test project: /tmp/hello_world_XXXXXX

=== Step 1: Research ===
✓ PASSED: morty research should succeed
✓ PASSED: Research file should be created

...

========================================
  Test Report
========================================

Total Scenarios:  2
Passed:           2
Failed:           0

✓ All tests passed!
```

## Architecture

### Test Flow

```
┌─────────────────┐
│  run_all.sh     │  Test Runner
└────────┬────────┘
         │
         ├─── Check Prerequisites (Morty binary, Mock CLI)
         │
         ├─── Discover Scenarios (scenarios/*.sh)
         │
         ├─── For each scenario:
         │    ┌──────────────────────────────────┐
         │    │  test_*.sh                       │
         │    ├──────────────────────────────────┤
         │    │  1. Create temp test project     │
         │    │  2. Run: morty research          │
         │    │  3. Run: morty plan              │
         │    │  4. Run: morty doing             │
         │    │     └─→ Uses Mock Claude CLI     │
         │    │  5. Verify generated files       │
         │    │  6. Verify Git commits           │
         │    │  7. Cleanup                      │
         │    └──────────────────────────────────┘
         │
         └─── Generate Test Report
```

### Mock Claude CLI

The BDD tests use a **Mock Claude CLI** to simulate AI responses without making actual API calls. This provides:

- **Fast execution**: Tests complete in seconds instead of minutes
- **Deterministic results**: Same input always produces same output
- **No API costs**: No Claude API usage
- **Offline testing**: No internet connection required

#### How It Works

1. **Environment Variable**: Tests set `CLAUDE_CODE_CLI` to point to `mock_claude.sh`
2. **Input Detection**: Mock CLI reads stdin and detects scenario type (calculator, hello world)
3. **Response Generation**: Returns pre-defined responses from `mock_responses.sh`
4. **Logging**: All interactions logged to `/tmp/mock_claude_*.log`

#### Mock Response Format

Mock responses are defined in `mock_responses.sh`:

- **Research responses**: Markdown documents with project analysis
- **Plan responses**: Structured plans with Modules and Jobs
- **Doing responses**: Actual Python code to be executed

#### Customizing Mock Responses

Edit `mock_responses.sh` to add new scenarios:

```bash
# Add new scenario detection
if echo "$input" | grep -qi "my_feature"; then
    scenario_type="my_feature"
fi

# Add new response function
get_my_feature_response() {
    cat <<'EOF'
# Your custom response here
EOF
}
```

## Test Scenarios

### Scenario 1: Hello World (Smoke Test)

**Purpose**: Validate the most basic Morty workflow

**Steps**:
1. Research "hello world"
2. Generate plan
3. Execute doing phase
4. Verify `hello.py` contains `print("Hello World")`
5. Verify Python code executes successfully

**Expected Duration**: < 10 seconds

**Use Case**: Quick smoke test to catch major regressions

### Scenario 2: Calculator Implementation

**Purpose**: Validate a realistic development task with multiple functions

**Steps**:
1. Research "implement calculator with addition"
2. Generate plan with multiple Jobs
3. Execute doing phase
4. Verify `calculator.py` contains `def add()` and other functions
5. Verify Python code executes and shows calculator operations
6. Verify Git commits with "morty:" prefix
7. Verify state management (status.json)

**Expected Duration**: < 30 seconds

**Use Case**: Comprehensive test of Research → Plan → Doing workflow

## Adding New Scenarios

### Step 1: Create Scenario Script

Create a new file `tests/bdd/scenarios/test_my_feature.sh`:

```bash
#!/bin/bash
set -e

# Load helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_helpers.sh"

# Configuration
MORTY_BIN="${MORTY_BIN:-$SCRIPT_DIR/../../../bin/morty}"
MOCK_CLAUDE_CLI="$SCRIPT_DIR/../mock_claude.sh"

print_section "My Feature Scenario"

# Create test project
create_test_project "my_feature" || exit 1
cd "$TEST_PROJECT_DIR"

# Set mock CLI
export CLAUDE_CODE_CLI="$MOCK_CLAUDE_CLI"

# Step 1: Research
output=$("$MORTY_BIN" research "my feature" 2>&1)
assert_success $? "morty research should succeed"

# Step 2: Plan
output=$("$MORTY_BIN" plan 2>&1)
assert_success $? "morty plan should succeed"

# Step 3: Doing
output=$("$MORTY_BIN" doing 2>&1)
assert_success $? "morty doing should succeed"

# Step 4: Verify
assert_file_exists "my_feature.py" "Feature file should be created"

# Cleanup
cd /tmp
cleanup_test_project

# Print summary
print_test_summary "My Feature Scenario"
exit $?
```

### Step 2: Add Mock Responses

Edit `mock_responses.sh` to add responses for your scenario:

```bash
# In get_mock_response() function, add detection:
if echo "$input" | grep -qi "my feature"; then
    scenario_type="my_feature"
fi

# Add response functions:
get_my_feature_research_response() {
    cat <<'EOF'
# Research: My Feature
...
EOF
}
```

### Step 3: Make Executable and Test

```bash
chmod +x tests/bdd/scenarios/test_my_feature.sh
./tests/bdd/scenarios/test_my_feature.sh
```

### Step 4: Run Full Suite

```bash
./tests/bdd/run_all.sh
```

## Test Helper Functions

The `test_helpers.sh` library provides:

### Project Management

- `create_test_project(name)` - Create isolated Git repository for testing
- `cleanup_test_project()` - Remove test project directory

### Assertions

- `assert_success(exit_code, message)` - Assert command succeeded
- `assert_failure(exit_code, message)` - Assert command failed
- `assert_file_exists(path, message)` - Assert file exists
- `assert_dir_exists(path, message)` - Assert directory exists
- `assert_file_contains(path, pattern, message)` - Assert file contains text
- `assert_git_commit_exists(pattern, message)` - Assert Git commit exists
- `assert_output_contains(output, pattern, message)` - Assert output contains text

### Reporting

- `print_test_summary(name)` - Print test results summary
- `print_section(title)` - Print section header
- `reset_test_counters()` - Reset test counters for new scenario

### Usage Example

```bash
# Create test environment
create_test_project "my_test"
cd "$TEST_PROJECT_DIR"

# Run command and assert
output=$(some_command 2>&1)
assert_success $? "Command should succeed"

# Verify files
assert_file_exists "output.txt" "Output file should be created"
assert_file_contains "output.txt" "expected text" "File should contain expected text"

# Cleanup
cd /tmp
cleanup_test_project

# Report results
print_test_summary "My Test"
```

## Troubleshooting

### Problem: "Morty binary not found"

**Solution**: Build Morty first:
```bash
cd /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/morty
./scripts/build.sh
```

### Problem: "Mock Claude CLI not executable"

**Solution**: Make it executable:
```bash
chmod +x tests/bdd/mock_claude.sh
```

Or run the test runner which does this automatically:
```bash
./tests/bdd/run_all.sh
```

### Problem: "Test timeout" or "Very slow execution"

**Possible Causes**:
- Mock CLI delay too high
- Real Claude CLI being used instead of mock

**Solution**: Check environment variable:
```bash
echo $CLAUDE_CODE_CLI
# Should be: /path/to/tests/bdd/mock_claude.sh
```

Adjust mock delay in `mock_claude.sh`:
```bash
MOCK_DELAY="${MOCK_DELAY:-0.1}"  # Reduce from 0.5 to 0.1
```

### Problem: "Git commit not found"

**Possible Causes**:
- Git not configured in test environment
- Morty auto-commit disabled

**Solution**: Check Git configuration in test:
```bash
git config user.name
git config user.email
```

The `create_test_project()` function should set these automatically.

### Problem: "Python code execution failed"

**Possible Causes**:
- Python 3 not installed
- Generated code has syntax errors
- Mock response doesn't match expected format

**Solution**:
1. Check Python availability: `python3 --version`
2. Manually inspect generated code: `cat calculator.py`
3. Check mock response in `mock_responses.sh`

### Problem: "No plan files found"

**Possible Causes**:
- Morty plan command failed
- Mock CLI didn't return valid plan format

**Solution**: Check mock log:
```bash
cat /tmp/mock_claude_*.log
```

Verify plan response format includes:
- `# Plan:` header
- `## Module:` section
- `### Job N:` definitions
- `**Tasks**:` lists

## CI/CD Integration

### GitHub Actions Example

Create `.github/workflows/bdd-tests.yml`:

```yaml
name: BDD Tests

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]

jobs:
  bdd-tests:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Set up Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.x'

    - name: Build Morty
      run: ./scripts/build.sh

    - name: Run BDD Tests
      run: |
        cd tests/bdd
        ./run_all.sh

    - name: Upload test logs
      if: failure()
      uses: actions/upload-artifact@v3
      with:
        name: bdd-test-logs
        path: /tmp/mock_claude_*.log
```

### Docker Example

Create `tests/bdd/Dockerfile`:

```dockerfile
FROM golang:1.21-alpine

RUN apk add --no-cache bash git python3

WORKDIR /app

COPY . .

RUN ./scripts/build.sh

CMD ["tests/bdd/run_all.sh"]
```

Run tests in Docker:
```bash
docker build -t morty-bdd-tests -f tests/bdd/Dockerfile .
docker run --rm morty-bdd-tests
```

## Performance Benchmarks

Target performance (using Mock CLI):

- **Hello World scenario**: < 10 seconds
- **Calculator scenario**: < 30 seconds
- **Full test suite**: < 1 minute

Actual performance depends on:
- Mock CLI delay setting (default: 0.5s)
- Disk I/O speed
- Git operations overhead

## Future Enhancements

### Phase 2: Additional Scenarios (Planned)

1. **Error Recovery** - Test failure retry and state recovery
2. **Daily Workflow** - Test module/job selection and incremental development
3. **Large Project** - Test performance with 10 modules × 20 jobs

See `.morty/research/morty-bdd-testing-strategy.md` for detailed plans.

### Phase 3: Integration Testing (Planned)

- Integration tests for module interactions (Go tests)
- Unit test improvements (increase coverage)
- CI/CD pipeline integration
- Test report generation (HTML, JUnit XML)

## Contributing

When adding new test scenarios:

1. Follow the naming convention: `test_<feature>.sh`
2. Use test helper functions for consistency
3. Include clear assertions with descriptive messages
4. Add cleanup to prevent test pollution
5. Update this README with scenario description
6. Ensure tests complete in < 30 seconds

## Support

For issues or questions:

- Check the Troubleshooting section above
- Review mock logs: `/tmp/mock_claude_*.log`
- Check test output for detailed error messages
- Review `.morty/plan/bdd.md` for implementation details

## License

Part of the Morty project. See main project LICENSE file.
