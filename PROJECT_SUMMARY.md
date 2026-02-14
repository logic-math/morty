# Morty Project Summary

## Overview

Morty is a simplified AI development loop system inspired by Ralph for Claude Code. It provides a streamlined interface for autonomous AI-assisted development with intelligent lifecycle management.

## Key Features Implemented

### 1. **Unified Command Interface**
- Single `morty` command with subcommands
- Simple, intuitive CLI design
- Consistent user experience

### 2. **Project Initialization** (3 Methods)

#### Method 1: Import from PRD (Markdown)
```bash
morty import requirements.md [project-name]
```
- Parses Markdown files for tasks (checkbox format)
- Extracts numbered lists as fallback
- Copies PRD to `.morty/specs/requirements.md`
- Generates `fix_plan.md` with extracted tasks
- Creates project structure with `.morty/` directory

#### Method 2: Enable in Existing Project
```bash
cd existing-project
morty enable [--force]
```
- Auto-detects project type (nodejs, python, rust, go, generic)
- Detects build and test commands
- Creates `.morty/` structure
- Updates `.gitignore`
- Preserves existing code

#### Method 3: Create New Project
```bash
morty init project-name [--type TYPE]
```
- Creates fresh project structure
- Initializes git repository
- Generates template files
- Ready for immediate development

### 3. **Development Loop Lifecycle**

Simple state machine:
```
init → loop → [error | done]
       ↑  |
       └──┘
```

**States**:
- **init**: Project initialized
- **loop**: Execute Claude Code iterations
- **error**: Exit on error (updates PROMPT.md)
- **done**: Exit on completion (updates PROMPT.md)

**Exit Conditions**:
- All tasks in `fix_plan.md` completed (all checkboxes marked `[x]`)
- Error detected in Claude output
- Completion signal detected ("done", "complete", "finished")
- Maximum loops reached (configurable, default: 50)

### 4. **Exit Hooks**

Automatic context updates to `PROMPT.md` on exit:
```markdown
<!-- MORTY_LAST_UPDATE -->
**Last Update**: 2026-02-14T09:30:00Z
**Reason**: error
**Context**: Error detected in Claude output
```

This helps with:
- Debugging failures
- Resuming work
- Understanding loop history

### 5. **tmux Monitoring**

3-pane layout for real-time visibility:
```
┌─────────────────┬─────────────────┐
│                 │  Live Logs      │
│  Morty Loop     │  (tail -f)      │
│  (execution)    ├─────────────────┤
│                 │  Status Monitor │
│                 │  (watch + jq)   │
└─────────────────┴─────────────────┘
```

**Features**:
- Left pane: Main loop execution
- Right-top: Live log streaming
- Right-bottom: JSON status updates
- Detachable sessions (Ctrl+B then D)
- Cross-platform tmux support (handles base-index)

### 6. **Project Structure**

All Morty files in `.morty/` directory:
```
project/
├── .morty/
│   ├── PROMPT.md          # Development instructions
│   ├── fix_plan.md        # Task list (checkboxes)
│   ├── AGENT.md           # Build/test commands
│   ├── specs/             # Specifications
│   │   └── requirements.md
│   ├── logs/              # Execution logs
│   │   ├── morty.log
│   │   └── loop_*.log
│   ├── status.json        # Current status
│   ├── .loop_state        # Loop counter
│   └── .session_id        # Session ID
├── src/                   # Source code (user's)
├── README.md
└── .gitignore
```

## Technical Implementation

### Core Components

1. **morty** (main command)
   - Command routing
   - Version management
   - Help system

2. **morty_init.sh**
   - Project creation
   - Template generation
   - Git initialization

3. **morty_import.sh**
   - PRD parsing (Markdown)
   - Task extraction (regex-based)
   - Project generation from requirements

4. **morty_enable.sh**
   - Project type detection
   - Build/test command detection
   - Existing project integration

5. **morty_loop.sh**
   - Main development loop
   - Claude Code execution
   - Exit condition checking
   - Status management
   - Exit hooks

6. **morty_monitor.sh**
   - tmux session setup
   - 3-pane layout
   - Live monitoring

7. **lib/common.sh**
   - Logging utilities
   - Project detection
   - Command detection
   - Exit hook implementation

### Key Design Decisions

1. **Simplicity**: Removed complex features from Ralph (rate limiting, circuit breaker, session continuity)
2. **Modularity**: Separate scripts for each function
3. **Extensibility**: Easy to add new features
4. **Testability**: Comprehensive test suite
5. **User-friendly**: Clear error messages and help text

## Testing

Comprehensive test suite (`test_morty.sh`):

**Test Coverage**:
1. ✅ PRD import and project creation
2. ✅ Project structure verification
3. ✅ Task extraction from Markdown
4. ✅ Existing project enablement
5. ✅ Project type detection
6. ✅ Build/test command detection
7. ✅ New project initialization
8. ✅ Status command

**Test Results**: 5/5 tests passing

## Installation

```bash
cd morty
./install.sh
```

Installs to:
- `~/.morty/` - Scripts and libraries
- `~/.local/bin/morty` - Main command

## Usage Examples

### Quick Start
```bash
# Import from PRD
morty import requirements.md my-app
cd my-app
morty monitor

# Enable in existing project
cd existing-project
morty enable
morty start

# Create new project
morty init new-app
cd new-app
morty start
```

### Configuration
```bash
# Environment variables
export MAX_LOOPS=100
export LOOP_DELAY=10

# Command options
morty start --max-loops 100 --delay 10
```

## Differences from Ralph

| Feature | Ralph | Morty |
|---------|-------|-------|
| Rate limiting | ✅ 100 calls/hour | ❌ Removed |
| Circuit breaker | ✅ Advanced | ❌ Removed |
| Session continuity | ✅ 24h expiry | ❌ Removed |
| Response analyzer | ✅ JSON/text parsing | ⚠️ Simplified |
| Exit detection | ✅ Dual-condition | ⚠️ Simplified |
| PRD import | ✅ Claude-powered | ⚠️ Regex-based |
| Project enable | ✅ Wizard | ⚠️ Simplified |
| tmux monitoring | ✅ 3-pane | ✅ 3-pane |
| Exit hooks | ❌ None | ✅ PROMPT.md updates |
| Installation | ✅ Global | ✅ Global |
| Project structure | ✅ `.ralph/` | ✅ `.morty/` |

**Simplifications**:
- No API rate limiting (assumes unlimited usage)
- No circuit breaker (assumes reliable execution)
- No session management (fresh context each loop)
- Simpler exit detection (task completion + error/done signals)
- Regex-based PRD parsing (no Claude Code dependency)

**Additions**:
- Exit hooks (automatic PROMPT.md updates)
- Simpler command interface
- Clearer lifecycle model

## File Statistics

```
Total files: 13
Total lines: ~2,042
Languages: Bash, Markdown

Core scripts: 7
Library modules: 1
Documentation: 4
Tests: 1
```

## Requirements

- Bash 4.0+
- Claude Code CLI (`claude` command)
- tmux (optional, for monitoring)
- jq (optional, for status display)
- Git (for project initialization)

## Future Enhancements

Potential additions (not implemented):
1. Rate limiting (if needed)
2. Session continuity (for long projects)
3. Claude-powered PRD parsing (better extraction)
4. Interactive task selection
5. Progress tracking
6. Metrics and analytics
7. Log rotation
8. Dry-run mode

## License

MIT License - see LICENSE file

## Acknowledgments

Inspired by [Ralph for Claude Code](https://github.com/frankbria/ralph-claude-code) by Frank Bria.

---

**Version**: 0.1.0
**Status**: Production Ready
**Tests**: 5/5 Passing
**Last Updated**: 2026-02-14
