# Morty

Simplified AI Development Loop with Interactive PRD Refinement

## Overview

Morty is a streamlined AI development system that helps you:
1. **Refine PRDs** through interactive dialogue with Claude Code
2. **Generate projects** with comprehensive context
3. **Execute development loops** autonomously

## Key Features

### 🎯 Plan Mode - Interactive PRD Refinement
- Launch interactive Claude Code session
- Refine requirements through dialogue
- Generate comprehensive problem descriptions
- Auto-create project structure with context

### 🔄 Development Loop
- Autonomous AI development iterations
- Simple lifecycle: init → loop → error/done
- Exit hooks with context updates
- Real-time monitoring with tmux

### 📁 Project Management
- Enable Morty in existing projects
- Auto-detect project types
- Generate build/test commands
- Maintain context in `.morty/` directory

## Installation

```bash
cd morty
./install.sh
```

Ensure `~/.local/bin` is in your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Quick Start

### Step 1: Create Initial PRD

```bash
cat > requirements.md << 'EOF'
# Todo Application

## Overview
A simple command-line todo app for managing tasks.

## Features
- Add tasks
- List tasks
- Mark tasks complete
- Delete tasks

## Users
- Developers who prefer CLI tools
- People who want simple task management

## Requirements
- Fast and responsive
- Data persistence
- Easy to use
EOF
```

### Step 2: Launch Plan Mode

```bash
morty plan requirements.md
```

This launches an **interactive Claude Code session** where:
- Claude analyzes your initial PRD
- Asks clarifying questions
- Explores requirements deeply
- Refines through dialogue
- Generates comprehensive `problem_description.md`
- Auto-creates project structure

### Step 3: Start Development

```bash
cd todo-application
morty monitor
```

## Commands

### `morty plan <prd.md> [project-name]`
Interactive PRD refinement mode.

**What it does:**
1. Launches Claude Code with plan mode system prompt
2. Engages in dialogue to refine requirements
3. Generates `problem_description.md` (refined PRD)
4. Creates complete project structure:
   - `.morty/PROMPT.md` - Development instructions
   - `.morty/fix_plan.md` - Task breakdown
   - `.morty/AGENT.md` - Build/test commands
   - `.morty/specs/problem_description.md` - Refined PRD

**Example:**
```bash
morty plan requirements.md
morty plan docs/prd.md my-app
```

### `morty enable`
Enable Morty in existing project.

**Example:**
```bash
cd existing-project
morty enable
```

### `morty start`
Start development loop.

**Example:**
```bash
morty start
morty start --max-loops 100 --delay 10
```

### `morty monitor`
Start with tmux monitoring (recommended).

**Example:**
```bash
morty monitor
```

### `morty status`
Show current status.

**Example:**
```bash
morty status
```

### `morty rollback <loop-number>`
Rollback to a specific loop iteration.

**What it does:**
- Finds the git commit for the specified loop number
- Resets the working directory to that state
- Allows you to undo changes from problematic loops

**Example:**
```bash
morty rollback 5    # Rollback to loop #5
```

### `morty history`
Show loop history from git commits.

**What it does:**
- Displays the last 20 loop commits
- Shows loop numbers, timestamps, and summaries
- Helps identify which loop to rollback to

**Example:**
```bash
morty history
```

## Git Auto-Commit

Morty automatically commits changes after each successful loop iteration:

**Features:**
- **Auto-commit after each loop**: Creates a snapshot with loop metadata
- **Rollback capability**: Use `morty rollback <N>` to revert to any loop
- **Loop history**: Use `morty history` to view all loop commits
- **Commit metadata**: Each commit includes:
  - Loop number
  - Timestamp (ISO format)
  - Work summary
  - Auto-commit marker

**Example commit message:**
```
morty: Loop #5 - Loop iteration completed

Auto-committed by Morty development loop.

Loop: 5
Timestamp: 2024-01-15T10:30:45Z
Summary: Loop iteration completed

This commit represents the state after loop iteration 5.
You can rollback to this point using: git reset --hard HEAD~N
```

**Benefits:**
- **Safety**: Every loop creates a restore point
- **Debugging**: Easily identify when issues were introduced
- **Experimentation**: Try changes knowing you can rollback
- **Transparency**: Clear history of what Morty did in each loop

**Requirements:**
- Project must be a git repository
- Git must be installed and available

**Notes:**
- Only commits if there are changes (doesn't create empty commits)
- Commits are local (not pushed to remote)
- Uses `git add -A` to stage all changes

## Plan Mode Deep Dive

### How Plan Mode Works

Plan mode uses a sophisticated system prompt that enables Claude Code to:

1. **Deep Exploration**
   - Ask probing questions
   - Challenge assumptions
   - Explore edge cases
   - Identify dependencies

2. **Structured Thinking**
   - Break down complex problems
   - Identify patterns
   - Recognize gaps
   - Map relationships

3. **Technical Insight**
   - Assess feasibility
   - Suggest technologies
   - Identify challenges
   - Recommend architectures

4. **User-Centric Analysis**
   - Understand user personas
   - Identify core features
   - Prioritize by value
   - Consider accessibility

### Dialogue Phases

**Phase 1: Understanding**
- Claude summarizes initial PRD
- Identifies ambiguities
- Lists assumptions
- Asks critical questions

**Phase 2: Deep Dive**
- Explores functional requirements
- Discusses non-functional requirements
- Develops user stories
- Defines acceptance criteria

**Phase 3: Validation**
- Summarizes all requirements
- Confirms priorities
- Validates approach
- Checks for gaps

**Phase 4: Synthesis**
- Generates `problem_description.md`
- Creates project structure
- Outputs completion signal

### Claude Command Configuration

Plan mode launches Claude with these flags:

```bash
claude \
  -p "<interactive prompt>" \
  --continue \
  --dangerously-skip-permissions \
  --allowedTools Read Write Glob Grep WebSearch WebFetch
```

**Why these flags:**
- `--continue`: Maintains context across the dialogue
- `--dangerously-skip-permissions`: Full tool access for exploration
- `--allowedTools`: Enables research and file operations

### System Prompt Highlights

The plan mode system prompt (`prompts/plan_mode_system.md`) includes:

- **Dialogue Framework**: 4-phase refinement process
- **Question Patterns**: "What if...", "Why...", "How..."
- **Exploration Techniques**: 5 Whys, scenario mapping, constraint exploration
- **Output Template**: Comprehensive problem_description.md structure
- **Completion Signal**: `<!-- PLAN_MODE_COMPLETE -->` marker

## Project Structure

After running `morty plan`, you get:

```
my-project/
├── .morty/
│   ├── PROMPT.md              # Development instructions
│   ├── fix_plan.md            # Task breakdown
│   ├── AGENT.md               # Build/test commands
│   ├── specs/
│   │   └── problem_description.md  # Refined PRD
│   └── logs/                  # Execution logs
├── src/                       # Source code
├── README.md
└── .gitignore
```

### Key Files

**`.morty/PROMPT.md`**
- Development instructions for Claude
- References problem description
- Defines workflow and quality standards
- Includes RALPH_STATUS block format

**`.morty/fix_plan.md`**
- Prioritized task list
- Checkbox format: `- [ ] Task`
- Extracted from problem description

**`.morty/AGENT.md`**
- Build commands (auto-detected by project type)
- Test commands
- Development commands
- Supports: Python, Node.js, Rust, Go

**`.morty/specs/problem_description.md`**
- Comprehensive refined PRD
- Generated through plan mode dialogue
- Includes: goals, requirements, user stories, technical specs

## Development Loop Lifecycle

```
init → loop → [error | done]
       ↑  |
       └──┘
```

**States:**
- **init**: Project initialized
- **loop**: Execute development iterations
- **error**: Exit on error (updates PROMPT.md)
- **done**: Exit on completion (updates PROMPT.md)

**Exit Conditions:**
- All tasks in `fix_plan.md` completed
- Error detected in Claude output
- Completion signal detected
- Maximum loops reached

## Monitoring

Use tmux monitoring for best experience:

```bash
morty monitor
```

**3-pane layout:**
```
┌─────────────────┬─────────────────┐
│                 │  Live Logs      │
│  Morty Loop     ├─────────────────┤
│                 │  Status Monitor │
└─────────────────┴─────────────────┘
```

**tmux Controls:**
- `Ctrl+B` then `D` - Detach
- `Ctrl+B` then `←/→` - Switch panes
- `Ctrl+B` then `[` - Scroll mode (q to exit)

## Configuration

**Environment Variables:**
```bash
# Custom Claude Code CLI command (default: "claude")
export CLAUDE_CODE_CLI="ai_cli"     # Use your custom CLI wrapper

# Loop configuration
export MAX_LOOPS=100                # Maximum iterations (default: 50)
export LOOP_DELAY=10                # Seconds between loops (default: 5)
```

**Example: Using Custom CLI Wrapper**
```bash
# If you have a custom enterprise CLI wrapper
export CLAUDE_CODE_CLI="/path/to/ai_cli"

# Or with additional configuration
export CLAUDE_CODE_CLI="ai_cli --config enterprise"

# Then use Morty normally
morty plan requirements.md
morty monitor
```

**Project Files:**
- `.morty/PROMPT.md` - Customize development instructions
- `.morty/fix_plan.md` - Add/modify tasks
- `.morty/AGENT.md` - Update build/test commands

## Requirements

- Bash 4.0+
- Claude Code CLI (`claude` command)
- tmux (optional, for monitoring)
- jq (optional, for status display)
- Git

## Testing

```bash
# Run all tests
./tests/run_all_tests.sh

# Or run individual tests
./tests/test_plan_mode.sh          # Plan mode tests (10 tests)
./tests/test_git_autocommit.sh     # Git auto-commit tests (9 tests)
```

See [tests/README.md](tests/README.md) for detailed test documentation.

## Examples

### Example 1: Web API Project

```bash
# Create initial PRD
cat > api_prd.md << 'EOF'
# REST API for Blog

## Overview
A RESTful API for a blogging platform.

## Features
- User authentication
- Create/edit/delete posts
- Comments
- Tags and categories

## Technical Requirements
- Node.js + Express
- MongoDB database
- JWT authentication
- API documentation
EOF

# Refine through plan mode
morty plan api_prd.md blog-api

# Claude will ask questions like:
# - What's the expected load?
# - How should comments be moderated?
# - What's the permission model?
# - Should we support markdown?

# After dialogue, project is generated
cd blog-api
morty monitor
```

### Example 2: CLI Tool

```bash
cat > cli_prd.md << 'EOF'
# File Organizer CLI

## Overview
Organize files automatically based on rules.

## Features
- Scan directories
- Apply rules (by extension, date, size)
- Move/copy files
- Dry-run mode
EOF

morty plan cli_prd.md file-organizer
cd file-organizer
morty start
```

## Tips

1. **Start with a rough PRD** - Plan mode will help refine it
2. **Be specific in dialogue** - Answer Claude's questions thoughtfully
3. **Review generated files** - Customize `.morty/PROMPT.md` as needed
4. **Use monitoring** - `morty monitor` for real-time visibility
5. **Check logs** - `.morty/logs/` for detailed execution history

## Troubleshooting

### "Claude command not found"
Install Claude Code CLI:
```bash
npm install -g @anthropic-ai/claude-code
```

### Plan mode doesn't start
Ensure:
- PRD file exists and is Markdown (.md)
- Claude CLI is installed
- `prompts/plan_mode_system.md` exists

### Project not generated
Check if Claude created `problem_description.md` in the working directory during plan mode.

## Architecture

**Core Components:**
- `morty` - Main command router
- `morty_plan.sh` - Plan mode implementation
- `morty_enable.sh` - Project enablement
- `morty_loop.sh` - Development loop
- `morty_monitor.sh` - tmux monitoring
- `lib/common.sh` - Shared utilities
- `prompts/plan_mode_system.md` - Plan mode system prompt

**Design Principles:**
- Simplicity over complexity
- Interactive over automated
- Context-rich over minimal
- Dialogue-driven refinement

## License

MIT License

## Documentation

For detailed documentation, see the `docs/` directory:

- **[Configuration Guide](docs/CONFIGURATION.md)** - Environment variables and project configuration
- **[Plan Mode Guide](docs/PLAN_MODE_GUIDE.md)** - Comprehensive guide to interactive PRD refinement
- **[Git Auto-Commit Feature](docs/GIT_AUTOCOMMIT_FEATURE.md)** - Loop rollback and history management
- **[Changelog](docs/CHANGELOG.md)** - Version history and migration guides

## Acknowledgments

Inspired by [Ralph for Claude Code](https://github.com/frankbria/ralph-claude-code) by Frank Bria.

---

**Version**: 0.2.1 (Git Auto-Commit)
**Status**: Production Ready
