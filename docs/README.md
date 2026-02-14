# Morty Documentation

Complete documentation for the Morty AI Development Loop system.

## 📚 Documentation Index

### Getting Started
- **[Main README](../README.md)** - Project overview, installation, and quick start guide

### Core Features

#### 1. Plan Mode - Interactive PRD Refinement
- **[Plan Mode Guide](PLAN_MODE_GUIDE.md)** - Comprehensive guide to plan mode
  - What is plan mode?
  - How it works
  - Dialogue techniques
  - Question patterns
  - Output format
  - Best practices
  - Troubleshooting

**Quick Summary:**
Plan mode launches an interactive Claude Code session that refines rough PRDs through dialogue, exploring requirements deeply using techniques like 5 Whys and What-If scenarios, then generates comprehensive problem descriptions and project structures.

#### 2. Git Auto-Commit - Loop Rollback
- **[Git Auto-Commit Feature](GIT_AUTOCOMMIT_FEATURE.md)** - Complete guide to git integration
  - Overview and benefits
  - Core functions
  - Usage examples
  - CLI commands
  - Testing
  - Troubleshooting

**Quick Summary:**
Morty automatically commits changes after each successful loop iteration, creating restore points with metadata that enable easy rollback to any previous state using `morty rollback <N>` and `morty history`.

#### 3. Development Loop
- **Main README** - Development loop documentation
  - Lifecycle (init → loop → error/done)
  - Exit hooks
  - Status tracking
  - tmux monitoring

### Version History
- **[Changelog](CHANGELOG.md)** - Complete version history
  - v0.2.1 - Git Auto-Commit
  - v0.2.0 - Plan Mode Edition
  - v0.1.0 - Initial Release
  - Migration guides

### Testing
- **[Test Suite Documentation](../tests/README.md)** - Complete testing guide
  - Plan mode tests (10 tests)
  - Git auto-commit tests (9 tests)
  - Test runner
  - CI/CD integration

## 🎯 Quick Reference

### Commands

```bash
# Plan Mode
morty plan <prd.md> [name]     # Interactive PRD refinement

# Project Management
morty enable                   # Enable in existing project

# Development Loop
morty start                    # Start loop
morty monitor                  # Start with tmux monitoring
morty status                   # Show current status

# Git Management
morty rollback <loop-number>   # Rollback to specific loop
morty history                  # Show loop commit history
```

### File Structure

```
morty/
├── docs/                      # Documentation (you are here)
│   ├── README.md             # This file
│   ├── PLAN_MODE_GUIDE.md    # Plan mode documentation
│   ├── GIT_AUTOCOMMIT_FEATURE.md  # Git feature documentation
│   └── CHANGELOG.md          # Version history
├── lib/                       # Shared libraries
│   └── common.sh             # Utility functions
├── prompts/                   # System prompts
│   └── plan_mode_system.md   # Plan mode prompt
├── morty                      # Main CLI command
├── morty_plan.sh             # Plan mode implementation
├── morty_enable.sh           # Project enablement
├── morty_loop.sh             # Development loop
├── morty_monitor.sh          # tmux monitoring
├── install.sh                # Installation script
└── README.md                 # Main documentation
```

### Project Structure (Generated)

```
my-project/
├── .morty/                    # Morty configuration
│   ├── PROMPT.md             # Development instructions
│   ├── fix_plan.md           # Task breakdown
│   ├── AGENT.md              # Build/test commands
│   ├── specs/
│   │   └── problem_description.md  # Refined PRD
│   └── logs/                 # Execution logs
└── src/                      # Source code
```

## 🔍 Documentation by Use Case

### I want to create a new project
1. Read: [Main README - Quick Start](../README.md#quick-start)
2. Read: [Plan Mode Guide](PLAN_MODE_GUIDE.md)
3. Run: `morty plan requirements.md`

### I want to enable Morty in an existing project
1. Read: [Main README - Commands](../README.md#commands)
2. Run: `cd existing-project && morty enable`

### I want to understand how plan mode works
1. Read: [Plan Mode Guide](PLAN_MODE_GUIDE.md)
2. Review: `prompts/plan_mode_system.md`

### I want to rollback a loop iteration
1. Read: [Git Auto-Commit Feature](GIT_AUTOCOMMIT_FEATURE.md)
2. Run: `morty history` to find the loop
3. Run: `morty rollback <loop-number>`

### I want to customize the development loop
1. Edit: `.morty/PROMPT.md` - Change instructions
2. Edit: `.morty/fix_plan.md` - Modify tasks
3. Edit: `.morty/AGENT.md` - Update build/test commands

### I want to monitor the loop in real-time
1. Read: [Main README - Monitoring](../README.md#monitoring)
2. Run: `morty monitor`
3. Use: `Ctrl+B` then `D` to detach, `tmux attach` to reattach

## 📖 Detailed Documentation

### Plan Mode System Prompt
- Location: `prompts/plan_mode_system.md`
- Size: ~3000 lines
- Contains:
  - Dialogue framework (4 phases)
  - Exploration techniques
  - Question patterns
  - Output template
  - Completion signals

### Test Suites
- `test_plan_mode.sh` - Plan mode tests (10 tests)
- `test_git_autocommit.sh` - Git feature tests (9 tests)

## 🚀 Workflow Examples

### Example 1: From Idea to Running Project

```bash
# 1. Write rough PRD
cat > idea.md << 'EOF'
# My App Idea
Brief description
EOF

# 2. Refine through plan mode
morty plan idea.md

# 3. Answer Claude's questions interactively
# (Dialogue happens here)

# 4. Project generated automatically
cd my-app

# 5. Review generated files
cat .morty/specs/problem_description.md
cat .morty/PROMPT.md
cat .morty/fix_plan.md

# 6. Start development with monitoring
morty monitor

# 7. Watch loops execute in real-time
# (Loop commits happen automatically)

# 8. View loop history
morty history

# 9. Rollback if needed
morty rollback 3
```

### Example 2: Enable in Existing Project

```bash
# 1. Navigate to project
cd existing-project

# 2. Enable Morty
morty enable

# 3. Review generated files
ls -la .morty/

# 4. Customize if needed
vim .morty/PROMPT.md

# 5. Start development
morty start
```

## 🔧 Configuration

### Environment Variables
```bash
export MAX_LOOPS=100        # Maximum iterations (default: 50)
export LOOP_DELAY=10        # Seconds between loops (default: 5)
```

### Project Configuration
- `.morty/PROMPT.md` - Customize development instructions
- `.morty/fix_plan.md` - Add/modify tasks
- `.morty/AGENT.md` - Update build/test commands

## 📝 Contributing

When adding new features:
1. Update relevant documentation in `docs/`
2. Add examples and usage instructions
3. Update CHANGELOG.md with version info
4. Add tests if applicable

## 🆘 Support

### Common Issues
- **Claude command not found**: Install Claude Code CLI
- **Plan mode doesn't start**: Check PRD file exists and is .md format
- **Git commands not working**: Ensure project is a git repository

### Getting Help
1. Check [Main README - Troubleshooting](../README.md#troubleshooting)
2. Check feature-specific docs in this directory
3. Review test scripts for usage examples

---

**Last Updated**: 2026-02-14
**Documentation Version**: 0.2.1
