# Morty Project Structure

Complete overview of the Morty project organization.

## Directory Tree

```
morty/
├── docs/                          # Documentation
│   ├── README.md                 # Documentation index
│   ├── CHANGELOG.md              # Version history
│   ├── PLAN_MODE_GUIDE.md        # Plan mode guide
│   └── GIT_AUTOCOMMIT_FEATURE.md # Git feature guide
│
├── tests/                         # Test suite
│   ├── README.md                 # Test documentation
│   ├── run_all_tests.sh          # Test runner
│   ├── test_plan_mode.sh         # Plan mode tests (10)
│   └── test_git_autocommit.sh    # Git tests (9)
│
├── lib/                           # Shared libraries
│   └── common.sh                 # Utility functions
│
├── prompts/                       # System prompts
│   └── plan_mode_system.md       # Plan mode prompt (~3000 lines)
│
├── morty                          # Main CLI command
├── morty_plan.sh                 # Plan mode implementation
├── morty_enable.sh               # Project enablement
├── morty_loop.sh                 # Development loop
├── morty_monitor.sh              # tmux monitoring
├── install.sh                    # Installation script
├── README.md                     # Main documentation
├── LICENSE                       # MIT License
└── .gitignore                    # Git ignore rules
```

## Core Files

### Main Entry Point
- **`morty`** - Main CLI command router
  - Routes to plan, enable, start, monitor, status, rollback, history
  - Version: 0.2.1

### Implementation Scripts
- **`morty_plan.sh`** - Interactive PRD refinement
  - Launches Claude Code with plan mode system prompt
  - Generates comprehensive problem descriptions
  - Auto-creates project structure

- **`morty_enable.sh`** - Enable Morty in existing projects
  - Auto-detects project type
  - Generates .morty/ configuration

- **`morty_loop.sh`** - Autonomous development loop
  - Executes Claude Code repeatedly
  - Manages lifecycle (init → loop → error/done)
  - Integrates git auto-commit

- **`morty_monitor.sh`** - tmux monitoring dashboard
  - 3-pane layout (loop, logs, status)
  - Real-time visibility

### Libraries
- **`lib/common.sh`** - Shared utility functions
  - Logging (log, success, error, warn)
  - Project detection (detect_project_type, detect_build_command, detect_test_command)
  - Git management (git_auto_commit, git_rollback, git_loop_history)
  - Context updates (update_prompt_context)
  - ISO timestamps (get_iso_timestamp)

### System Prompts
- **`prompts/plan_mode_system.md`** - Plan mode system prompt
  - ~3000 lines of comprehensive guidance
  - 4-phase dialogue framework
  - Exploration techniques (5 Whys, What-If)
  - Question patterns
  - Output template

## Documentation Structure

### `docs/` Directory
All user-facing documentation organized in one place:

1. **`docs/README.md`** - Documentation index
   - Quick reference
   - Documentation by use case
   - Workflow examples

2. **`docs/PLAN_MODE_GUIDE.md`** - Complete plan mode guide
   - What is plan mode
   - How it works
   - Dialogue techniques
   - Best practices

3. **`docs/GIT_AUTOCOMMIT_FEATURE.md`** - Git feature documentation
   - Overview and benefits
   - Core functions
   - Usage examples
   - Troubleshooting

4. **`docs/CHANGELOG.md`** - Version history
   - v0.2.1 - Git Auto-Commit
   - v0.2.0 - Plan Mode Edition
   - v0.1.0 - Initial Release
   - Migration guides

## Test Structure

### `tests/` Directory
All test scripts and test documentation:

1. **`tests/README.md`** - Test suite documentation
   - Test coverage
   - Running tests
   - Adding new tests

2. **`tests/run_all_tests.sh`** - Test runner
   - Runs all tests
   - Summary report
   - Exit codes

3. **`tests/test_plan_mode.sh`** - Plan mode tests
   - 10 tests covering:
     - Component verification
     - Command routing
     - Template generation
     - Project type detection

4. **`tests/test_git_autocommit.sh`** - Git feature tests
   - 9 tests covering:
     - Auto-commit functionality
     - Rollback detection
     - History display
     - Commit message format

## Generated Project Structure

When you run `morty plan <prd.md>`, it generates:

```
my-project/
├── .morty/                        # Morty configuration
│   ├── PROMPT.md                 # Development instructions
│   ├── fix_plan.md               # Task breakdown
│   ├── AGENT.md                  # Build/test commands
│   ├── specs/
│   │   └── problem_description.md  # Refined PRD
│   ├── logs/                     # Execution logs
│   └── .loop_state               # Loop state tracking
│
├── src/                          # Source code
├── README.md                     # Project readme
└── .gitignore                    # Git ignore
```

## Installation Structure

After running `./install.sh`, files are installed to:

```
~/.morty/                          # Installation directory
├── morty_plan.sh
├── morty_enable.sh
├── morty_loop.sh
├── morty_monitor.sh
├── lib/
│   └── common.sh
└── prompts/
    └── plan_mode_system.md

~/.local/bin/                      # User binaries
└── morty                          # Main command
```

## Key Design Principles

### 1. Separation of Concerns
- **docs/** - All documentation
- **tests/** - All test scripts
- **lib/** - Shared utilities
- **prompts/** - System prompts
- Root - Core implementation

### 2. Discoverability
- Clear naming conventions
- README files in each directory
- Documentation index
- Test documentation

### 3. Maintainability
- Modular architecture
- Shared utilities in lib/
- Comprehensive tests
- Version tracking

### 4. User Experience
- Single entry point (`morty`)
- Clear command structure
- Comprehensive documentation
- Easy installation

## File Sizes

| File | Lines | Purpose |
|------|-------|---------|
| `prompts/plan_mode_system.md` | ~3000 | Plan mode guidance |
| `morty_plan.sh` | ~600 | Plan mode implementation |
| `docs/PLAN_MODE_GUIDE.md` | ~420 | Plan mode user guide |
| `lib/common.sh` | ~260 | Utility functions |
| `docs/GIT_AUTOCOMMIT_FEATURE.md` | ~250 | Git feature guide |
| `morty_loop.sh` | ~305 | Development loop |
| `README.md` | ~530 | Main documentation |
| `docs/CHANGELOG.md` | ~145 | Version history |

## Dependencies

### Required
- Bash 4.0+
- Git
- Claude Code CLI (`claude` command)

### Optional
- tmux (for monitoring)
- jq (for status display)

## Version Information

- **Current Version**: 0.2.1 (Git Auto-Commit)
- **Previous Version**: 0.2.0 (Plan Mode Edition)
- **Initial Version**: 0.1.0
- **Status**: Production Ready

## Quick Navigation

### I want to...

**Understand plan mode:**
- Read: `docs/PLAN_MODE_GUIDE.md`
- Review: `prompts/plan_mode_system.md`

**Understand git features:**
- Read: `docs/GIT_AUTOCOMMIT_FEATURE.md`
- Review: `lib/common.sh` (lines 156-261)

**Run tests:**
- Run: `./tests/run_all_tests.sh`
- Read: `tests/README.md`

**See version history:**
- Read: `docs/CHANGELOG.md`

**Understand architecture:**
- Read: This file
- Review: `README.md` (Architecture section)

**Contribute:**
- Read: `tests/README.md` (Contributing section)
- Follow: Project structure conventions

## Statistics

- **Total Files**: 25
- **Total Directories**: 11
- **Lines of Code**: ~5000
- **Lines of Documentation**: ~5000
- **Test Coverage**: 19 tests (all passing)
- **Supported Project Types**: Python, Node.js, Rust, Go

## Maintenance

### Adding New Features
1. Implement in appropriate script (morty_*.sh)
2. Add functions to lib/common.sh if shared
3. Create tests in tests/test_<feature>.sh
4. Document in docs/<FEATURE>.md
5. Update docs/CHANGELOG.md
6. Update README.md if needed

### Adding Documentation
1. Create new file in docs/
2. Update docs/README.md index
3. Link from main README.md if appropriate

### Adding Tests
1. Create tests/test_<feature>.sh
2. Follow existing test structure
3. Update tests/README.md
4. Ensure run_all_tests.sh picks it up

---

**Last Updated**: 2026-02-14
**Structure Version**: 0.2.1
