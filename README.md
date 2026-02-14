# Morty

Simplified AI Development Loop - A streamlined version inspired by Ralph for Claude Code.

## Features

- **Project Initialization**: Create new projects or import from PRD documents
- **Project Enablement**: Add Morty to existing projects
- **Development Loop**: Autonomous AI development with lifecycle management
- **tmux Monitoring**: Real-time monitoring with split-pane dashboard
- **Exit Hooks**: Automatic context updates to PROMPT.md on exit

## Installation

```bash
cd morty
./install.sh
```

This installs Morty to `~/.morty` and adds the `morty` command to `~/.local/bin/`.

Make sure `~/.local/bin` is in your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Quick Start

### Option 1: Create New Project
```bash
morty init my-project
cd my-project
# Edit .morty/PROMPT.md and .morty/fix_plan.md
morty start
```

### Option 2: Import from PRD
```bash
morty import requirements.md my-project
cd my-project
morty start
```

### Option 3: Enable in Existing Project
```bash
cd existing-project
morty enable
morty start
```

## Commands

- `morty init <project>` - Create new project from scratch
- `morty import <prd.md> [name]` - Import PRD and create project
- `morty enable` - Enable Morty in existing project
- `morty start` - Start development loop
- `morty monitor` - Start with tmux monitoring (recommended)
- `morty status` - Show current status
- `morty version` - Show version

## Project Structure

```
my-project/
├── .morty/
│   ├── PROMPT.md          # Development instructions
│   ├── fix_plan.md        # Task list with checkboxes
│   ├── AGENT.md           # Build/test commands
│   ├── specs/             # Specifications
│   └── logs/              # Execution logs
├── src/                   # Source code
└── README.md
```

## Loop Lifecycle

The development loop follows a simple state machine:

1. **init** - Initialize from new or existing project
2. **loop** - Execute development iterations
3. **error** - Exit on error (updates PROMPT.md)
4. **done** - Exit on completion (updates PROMPT.md)

Exit hooks automatically update PROMPT.md with context for debugging or resuming.

## Monitoring

Use tmux monitoring for the best experience:

```bash
morty monitor
```

This creates a 3-pane layout:
- **Left**: Morty loop execution
- **Right-top**: Live logs
- **Right-bottom**: Status monitor

**tmux Controls**:
- `Ctrl+B` then `D` - Detach from session
- `Ctrl+B` then `←/→` - Switch panes
- `tmux attach -t <session>` - Reattach

## Configuration

Edit `.morty/PROMPT.md` to customize development instructions.

Environment variables:
- `MAX_LOOPS` - Maximum loop iterations (default: 50)
- `LOOP_DELAY` - Delay between loops in seconds (default: 5)

## Requirements

- Bash 4.0+
- Claude Code CLI (`claude` command)
- tmux (for monitoring, optional)
- jq (for status display, optional)

## License

MIT License

## Acknowledgments

Inspired by [Ralph for Claude Code](https://github.com/frankbria/ralph-claude-code) by Frank Bria.
