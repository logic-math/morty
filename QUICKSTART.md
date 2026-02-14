# Morty Quick Start Guide

## Installation

```bash
cd morty
./install.sh
```

Ensure `~/.local/bin` is in your PATH:
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Usage Examples

### Example 1: Import from PRD

```bash
# Create a sample PRD
cat > requirements.md << 'EOF'
# Calculator App

## Features
- [ ] Add two numbers
- [ ] Subtract two numbers
- [ ] Multiply two numbers
- [ ] Divide two numbers
EOF

# Import and create project
morty import requirements.md calculator

# Start development
cd calculator
morty monitor  # or: morty start
```

### Example 2: Enable in Existing Project

```bash
cd my-existing-project
morty enable
morty start
```

### Example 3: Create New Project

```bash
morty init my-new-app
cd my-new-app
# Edit .morty/PROMPT.md and .morty/fix_plan.md
morty start
```

## Understanding the Lifecycle

Morty follows a simple state machine:

```
init → loop → [error | done]
       ↑  |
       └──┘
```

- **init**: Project initialized (from PRD, enable, or init command)
- **loop**: Execute development iterations
- **error**: Exit on error and update PROMPT.md with context
- **done**: Exit on completion and update PROMPT.md with context

## Monitoring

The `morty monitor` command creates a 3-pane tmux layout:

```
┌─────────────────┬─────────────────┐
│                 │  Live Logs      │
│  Morty Loop     ├─────────────────┤
│                 │  Status Monitor │
└─────────────────┴─────────────────┘
```

**tmux Tips**:
- `Ctrl+B` then `D` - Detach (keeps running)
- `Ctrl+B` then `←/→` - Switch panes
- `Ctrl+B` then `[` - Scroll mode (q to exit)

## Configuration

### Environment Variables

```bash
export MAX_LOOPS=100        # Maximum iterations (default: 50)
export LOOP_DELAY=10        # Seconds between loops (default: 5)
```

### Project Files

- `.morty/PROMPT.md` - Development instructions (customize this!)
- `.morty/fix_plan.md` - Task list with checkboxes
- `.morty/AGENT.md` - Build/test commands
- `.morty/specs/` - Detailed specifications

## Tips

1. **Start with clear tasks**: Edit `.morty/fix_plan.md` with specific, actionable items
2. **Use checkboxes**: Format tasks as `- [ ] Task description`
3. **Monitor progress**: Use `morty monitor` for real-time visibility
4. **Review logs**: Check `.morty/logs/` for detailed execution history
5. **Exit context**: Check PROMPT.md after exit for debugging info

## Troubleshooting

### "Not a Morty project"
Run `morty enable` in the project directory.

### Loop exits immediately
Check if all tasks in `.morty/fix_plan.md` are marked complete `[x]`.

### Claude command not found
Install Claude Code CLI: `npm install -g @anthropic-ai/claude-code`

### tmux not found
Install tmux: `sudo apt-get install tmux` (Ubuntu) or `brew install tmux` (macOS)

## Next Steps

- Customize `.morty/PROMPT.md` with project-specific guidelines
- Add detailed requirements to `.morty/specs/`
- Run tests: `./test_morty.sh` (for development)
