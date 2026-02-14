# Morty Demo Guide

This guide demonstrates Morty's key features with practical examples.

## Demo 1: Import PRD and Create Project

```bash
# Step 1: Create a sample PRD
cat > todo_app.md << 'EOF'
# Todo Application PRD

## Overview
A simple command-line todo application.

## Features
- [ ] Add new tasks
- [ ] List all tasks
- [ ] Mark tasks as complete
- [ ] Delete tasks
- [ ] Save tasks to file
- [ ] Load tasks from file

## Technical Requirements
- Language: Python
- Storage: JSON file
- CLI: argparse

## Tasks
1. Implement Task class
2. Implement TodoList class
3. Add command-line interface
4. Add file persistence
5. Write unit tests
EOF

# Step 2: Import PRD and create project
morty import todo_app.md

# Step 3: Review generated files
cd todo-app
ls -la .morty/

# Step 4: Check extracted tasks
cat .morty/fix_plan.md

# Step 5: Review development instructions
cat .morty/PROMPT.md

# Step 6: Start development (requires Claude CLI)
# morty start --max-loops 10
```

## Demo 2: Enable Morty in Existing Project

```bash
# Step 1: Create a sample existing project
mkdir my-api
cd my-api

# Initialize as Node.js project
npm init -y
cat > package.json << 'EOF'
{
  "name": "my-api",
  "version": "1.0.0",
  "scripts": {
    "test": "jest",
    "build": "tsc",
    "start": "node dist/index.js"
  }
}
EOF

# Add some source code
mkdir src
cat > src/index.js << 'EOF'
// API entry point
console.log('API starting...');
EOF

git init

# Step 2: Enable Morty
morty enable

# Step 3: Verify Morty structure
ls -la .morty/

# Step 4: Check detected commands
cat .morty/AGENT.md

# Step 5: Add tasks
cat > .morty/fix_plan.md << 'EOF'
# Task List

## High Priority
- [ ] Add Express.js framework
- [ ] Create REST API endpoints
- [ ] Add input validation
- [ ] Add error handling

## Medium Priority
- [ ] Add authentication
- [ ] Add database integration
- [ ] Write API documentation

## Low Priority
- [ ] Add rate limiting
- [ ] Add logging
EOF

# Step 6: Start development
# morty monitor
```

## Demo 3: Create New Project from Scratch

```bash
# Step 1: Create new project
morty init calculator-lib --type python

# Step 2: Navigate to project
cd calculator-lib

# Step 3: Customize PROMPT.md
cat > .morty/PROMPT.md << 'EOF'
# Calculator Library Development

You are developing a Python calculator library.

## Requirements
- Pure Python (no external dependencies)
- Support basic operations: +, -, *, /
- Handle edge cases (division by zero, etc.)
- Type hints for all functions
- Comprehensive docstrings

## Development Principles
1. Test-driven development
2. PEP 8 compliance
3. 100% test coverage
4. Clear error messages

## Current Tasks
See `.morty/fix_plan.md` for specific tasks.
EOF

# Step 4: Add specific tasks
cat > .morty/fix_plan.md << 'EOF'
# Task List

## Phase 1: Core Functions
- [ ] Implement add(a, b)
- [ ] Implement subtract(a, b)
- [ ] Implement multiply(a, b)
- [ ] Implement divide(a, b) with zero check

## Phase 2: Testing
- [ ] Write tests for add
- [ ] Write tests for subtract
- [ ] Write tests for multiply
- [ ] Write tests for divide (including edge cases)

## Phase 3: Documentation
- [ ] Add module docstring
- [ ] Add function docstrings
- [ ] Create README with examples
EOF

# Step 5: Start development
# morty start --max-loops 20
```

## Demo 4: Monitor Development with tmux

```bash
# Navigate to any Morty project
cd todo-app

# Start with monitoring (creates 3-pane layout)
morty monitor

# In the tmux session:
# - Left pane: Shows loop execution
# - Right-top: Shows live logs (tail -f)
# - Right-bottom: Shows status (JSON)

# tmux controls:
# Ctrl+B then D       - Detach (keeps running)
# Ctrl+B then ←/→     - Switch panes
# Ctrl+B then [       - Scroll mode (q to exit)

# Reattach to session
tmux list-sessions
tmux attach -t morty-<timestamp>
```

## Demo 5: Check Status and Logs

```bash
cd todo-app

# Check current status
morty status

# View main log
tail -f .morty/logs/morty.log

# View specific loop log
ls -lt .morty/logs/
cat .morty/logs/loop_1_*.log

# Check exit context (after loop finishes)
tail .morty/PROMPT.md
```

## Demo 6: Simulate Loop Lifecycle

```bash
# Create a test project
morty init lifecycle-demo
cd lifecycle-demo

# Add tasks
cat > .morty/fix_plan.md << 'EOF'
# Task List
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
EOF

# Start with limited loops to see lifecycle
morty start --max-loops 3 --delay 2

# Observe:
# 1. init state (project initialized)
# 2. loop state (iterations 1, 2, 3)
# 3. max_loops state (exit after 3 loops)
# 4. PROMPT.md updated with exit context

# Check exit context
cat .morty/PROMPT.md | tail -10
```

## Demo 7: Test Error Handling

```bash
# Create project with intentional error
morty init error-demo
cd error-demo

# Add PROMPT that will cause error
cat > .morty/PROMPT.md << 'EOF'
# Error Demo

Please execute this invalid command: `invalid_command_xyz`

This will cause an error and trigger the error exit path.
EOF

# Start loop (will exit on error)
morty start --max-loops 5

# Check exit context
cat .morty/PROMPT.md | tail -10

# Should show:
# <!-- MORTY_LAST_UPDATE -->
# **Last Update**: <timestamp>
# **Reason**: error
# **Context**: Error detected in Claude output
```

## Demo 8: Test Completion Detection

```bash
# Create project with all tasks complete
morty init complete-demo
cd complete-demo

# Mark all tasks as complete
cat > .morty/fix_plan.md << 'EOF'
# Task List
- [x] Task 1 (already done)
- [x] Task 2 (already done)
- [x] Task 3 (already done)
EOF

# Start loop (should exit immediately)
morty start

# Check exit context
cat .morty/PROMPT.md | tail -10

# Should show:
# **Reason**: done
# **Context**: All tasks in fix_plan.md completed
```

## Demo 9: Custom Configuration

```bash
# Set custom environment variables
export MAX_LOOPS=100
export LOOP_DELAY=10

# Or use command-line options
morty start --max-loops 100 --delay 10

# Check effective configuration
morty start --help
```

## Demo 10: Run Full Test Suite

```bash
# Navigate to Morty source
cd /path/to/morty

# Run comprehensive tests
./test_morty.sh

# Should show:
# ✓ Test 1: Sample PRD created
# ✓ Test 2: PRD import and project structure verified
# ✓ Test 3: Morty enable in existing project verified
# ✓ Test 4: Morty init verified
# ✓ Test 5: Status command verified
# All tests passed! ✨
```

## Tips for Effective Demos

1. **Start simple**: Use Demo 1 (PRD import) for first-time users
2. **Show monitoring**: Demo 4 (tmux) demonstrates real-time visibility
3. **Explain lifecycle**: Demo 6 shows the state machine clearly
4. **Test edge cases**: Demos 7-8 show error handling and completion
5. **Customize**: Encourage users to modify PROMPT.md and fix_plan.md

## Common Demo Pitfalls

1. **Claude CLI not installed**: Ensure `claude` command is available
2. **tmux not installed**: Install for monitoring demos
3. **Long execution**: Use `--max-loops 3` for quick demos
4. **Unclear tasks**: Write specific, actionable tasks in fix_plan.md
5. **Missing context**: Always show PROMPT.md and fix_plan.md first

## Demo Script Template

```bash
#!/bin/bash
# Quick Morty demo

echo "=== Morty Demo ==="
echo ""

echo "1. Creating sample PRD..."
cat > demo_prd.md << 'EOF'
# Demo App
- [ ] Feature 1
- [ ] Feature 2
- [ ] Feature 3
EOF

echo "2. Importing PRD..."
morty import demo_prd.md

echo "3. Reviewing project structure..."
cd demo-app
tree .morty/

echo "4. Checking tasks..."
cat .morty/fix_plan.md

echo "5. Ready to start development!"
echo "   Run: morty start"
echo ""
echo "Demo complete!"
```

---

**Note**: Most demos require Claude Code CLI (`claude` command) to be installed and configured. The test suite (Demo 10) can run without Claude CLI.
