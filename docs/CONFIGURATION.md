# Morty Configuration Guide

Complete guide to configuring Morty through environment variables and project files.

## Environment Variables

Morty can be customized through environment variables set in your shell or `.bashrc`/`.zshrc`.

### Claude Code CLI Command

#### `CLAUDE_CODE_CLI`

Specify a custom Claude Code CLI command or wrapper.

**Default**: `claude`

**Use Cases:**
- Enterprise CLI wrappers with authentication
- Custom scripts that wrap Claude Code
- Alternative Claude Code installations
- CLI tools with pre-configured settings

**Examples:**

```bash
# Use a custom enterprise wrapper
export CLAUDE_CODE_CLI="ai_cli"

# Use with full path
export CLAUDE_CODE_CLI="/opt/company/bin/ai_cli"

# Use with arguments
export CLAUDE_CODE_CLI="ai_cli --config enterprise --auth sso"

# Use a different Claude Code installation
export CLAUDE_CODE_CLI="/usr/local/bin/claude-enterprise"
```

**When to Use:**
- Your company has a custom CLI wrapper for authentication
- You need to pass specific flags to Claude Code
- You want to use a different Claude Code installation
- You have a script that sets up environment before calling Claude

**Example Enterprise Wrapper (`ai_cli`):**

```bash
#!/bin/bash
# ai_cli - Enterprise Claude Code wrapper

# Load enterprise credentials
source /opt/company/config/ai_credentials.sh

# Set up proxy
export HTTP_PROXY="http://proxy.company.com:8080"
export HTTPS_PROXY="http://proxy.company.com:8080"

# Set enterprise API endpoint
export CLAUDE_API_ENDPOINT="https://api.company.com/claude"

# Call actual Claude Code with enterprise settings
exec claude \
  --api-key "$ENTERPRISE_API_KEY" \
  --endpoint "$CLAUDE_API_ENDPOINT" \
  "$@"
```

Then use it with Morty:
```bash
export CLAUDE_CODE_CLI="ai_cli"
morty plan requirements.md
```

### Loop Configuration

#### `MAX_LOOPS`

Maximum number of loop iterations before stopping.

**Default**: `50`

**Range**: 1-1000

**Examples:**
```bash
export MAX_LOOPS=100    # Allow up to 100 iterations
export MAX_LOOPS=10     # Quick test with 10 iterations
export MAX_LOOPS=500    # Long-running project
```

**When to Adjust:**
- **Lower (10-20)**: Quick prototypes or testing
- **Default (50)**: Most projects
- **Higher (100-500)**: Large, complex projects

#### `LOOP_DELAY`

Seconds to wait between loop iterations.

**Default**: `5`

**Range**: 0-3600 (seconds)

**Examples:**
```bash
export LOOP_DELAY=10    # Wait 10 seconds between loops
export LOOP_DELAY=0     # No delay (maximum speed)
export LOOP_DELAY=60    # Wait 1 minute between loops
```

**When to Adjust:**
- **0**: Maximum speed for testing
- **5-10**: Normal development
- **30-60**: Rate limiting or resource constraints

### Complete Configuration Example

```bash
# ~/.bashrc or ~/.zshrc

# Custom Claude Code CLI
export CLAUDE_CODE_CLI="ai_cli"

# Loop settings
export MAX_LOOPS=100
export LOOP_DELAY=10

# Add morty to PATH
export PATH="$HOME/.local/bin:$PATH"
```

## Project Configuration Files

Each Morty project has configuration files in the `.morty/` directory.

### `.morty/PROMPT.md`

Main development instructions for Claude Code.

**Purpose**: Guides Claude's behavior during each loop iteration.

**Customize:**
```markdown
# Development Instructions

You are working on a [project type] project.

## Project Context
[Describe the project, goals, constraints]

## Development Guidelines
- Follow [coding standards]
- Use [specific libraries/frameworks]
- Test coverage: [requirements]

## Current Phase
[What should Claude focus on now]

## Quality Standards
- Code style: [standards]
- Testing: [requirements]
- Documentation: [requirements]

## Exit Conditions
Signal completion when:
- [ ] All tasks in fix_plan.md are complete
- [ ] All tests pass
- [ ] Documentation is updated
```

### `.morty/fix_plan.md`

Task breakdown and checklist.

**Purpose**: Prioritized list of tasks for Claude to complete.

**Format:**
```markdown
# Task List

## Phase 1: Foundation
- [x] Set up project structure
- [x] Configure build system
- [ ] Implement core data models

## Phase 2: Features
- [ ] Implement authentication
- [ ] Add API endpoints
- [ ] Create UI components

## Phase 3: Polish
- [ ] Write tests
- [ ] Add error handling
- [ ] Update documentation
```

**Tips:**
- Use checkboxes: `- [ ]` (incomplete) or `- [x]` (complete)
- Organize by phases or priorities
- Be specific and actionable
- Update as project evolves

### `.morty/AGENT.md`

Build and test commands.

**Purpose**: Tells Morty how to build and test the project.

**Format:**
```markdown
# Build Commands

```bash
npm install
npm run build
```

# Test Commands

```bash
npm test
npm run lint
```

# Run Commands

```bash
npm start
```
```

**Auto-Detection:**
Morty auto-generates this based on project type:
- **Python**: `pip install`, `pytest`
- **Node.js**: `npm install`, `npm test`
- **Rust**: `cargo build`, `cargo test`
- **Go**: `go build`, `go test`

**Customize:**
Add project-specific commands, environment setup, etc.

### `.morty/specs/problem_description.md`

Comprehensive problem description (generated by plan mode).

**Purpose**: Complete specification of what the project should do.

**Sections:**
- Executive Summary
- Problem Statement
- Goals and Objectives
- Target Users
- Functional Requirements
- Non-Functional Requirements
- Technical Specifications
- User Stories
- Edge Cases
- Development Approach

**Read-Only**: Generated by plan mode, typically not edited manually.

## Configuration Precedence

Configuration is applied in this order (later overrides earlier):

1. **Default values** (hardcoded in scripts)
2. **Environment variables** (set in shell)
3. **Project files** (`.morty/` directory)
4. **Command-line flags** (if applicable)

Example:
```bash
# Default
CLAUDE_CMD="claude"

# Overridden by environment variable
export CLAUDE_CODE_CLI="ai_cli"
# Now CLAUDE_CMD="ai_cli"

# Command-line flag (if supported)
morty start --max-loops 200
# Overrides MAX_LOOPS for this run only
```

## Advanced Configuration

### Per-Project Environment

Use `.envrc` (with direnv) for per-project configuration:

```bash
# my-project/.envrc
export CLAUDE_CODE_CLI="ai_cli --project my-project"
export MAX_LOOPS=200
export LOOP_DELAY=15
```

Then:
```bash
cd my-project
direnv allow  # Loads .envrc
morty monitor
```

### Shell Aliases

Create shortcuts for common configurations:

```bash
# ~/.bashrc

# Quick test mode
alias morty-test='MAX_LOOPS=10 LOOP_DELAY=0 morty start'

# Production mode
alias morty-prod='MAX_LOOPS=200 LOOP_DELAY=30 morty monitor'

# Enterprise mode
alias morty-enterprise='CLAUDE_CODE_CLI="ai_cli --auth sso" morty'
```

### Configuration Validation

Check your configuration:

```bash
# Show environment
env | grep -E "(CLAUDE_CODE_CLI|MAX_LOOPS|LOOP_DELAY)"

# Test Claude Code CLI
$CLAUDE_CODE_CLI --version

# Check project configuration
cat .morty/PROMPT.md
cat .morty/fix_plan.md
cat .morty/AGENT.md
```

## Troubleshooting

### "Claude command not found"

**Problem**: Default `claude` command not found.

**Solutions:**
1. Install Claude Code CLI: `npm install -g @anthropic-ai/claude-code`
2. Use custom CLI: `export CLAUDE_CODE_CLI="your-cli"`
3. Check PATH: `which claude`

### Custom CLI not working

**Problem**: `CLAUDE_CODE_CLI` set but Morty still uses `claude`.

**Solutions:**
1. Verify environment variable: `echo $CLAUDE_CODE_CLI`
2. Export in current shell: `export CLAUDE_CODE_CLI="ai_cli"`
3. Add to shell profile: `~/.bashrc` or `~/.zshrc`
4. Restart shell or `source ~/.bashrc`

### CLI wrapper fails

**Problem**: Custom CLI wrapper exits with error.

**Solutions:**
1. Test wrapper directly: `$CLAUDE_CODE_CLI --help`
2. Check wrapper permissions: `chmod +x /path/to/ai_cli`
3. Check wrapper dependencies (auth, network, etc.)
4. Add debug logging to wrapper script

### Loop runs too fast/slow

**Problem**: Loop iterations too fast or too slow.

**Solutions:**
1. Adjust `LOOP_DELAY`: `export LOOP_DELAY=10`
2. Check Claude Code response time
3. Monitor system resources
4. Consider rate limiting

## Best Practices

### 1. Document Your Configuration

Create a `CONFIG.md` in your project:
```markdown
# Project Configuration

## Required Environment
```bash
export CLAUDE_CODE_CLI="ai_cli"
export MAX_LOOPS=100
```

## Setup
1. Install dependencies
2. Configure authentication
3. Set environment variables
4. Run `morty monitor`
```

### 2. Use Version Control

Commit project configuration files:
```bash
git add .morty/PROMPT.md
git add .morty/fix_plan.md
git add .morty/AGENT.md
git commit -m "chore: Update Morty configuration"
```

**Don't commit:**
- `.morty/logs/` (temporary)
- `.morty/.loop_state` (temporary)
- `.morty/status.json` (temporary)

### 3. Share Configuration

Team configuration in README:
```markdown
## Development with Morty

### Setup
```bash
export CLAUDE_CODE_CLI="ai_cli --team our-team"
export MAX_LOOPS=100
```

### Start Development
```bash
morty monitor
```
```

### 4. Test Configuration

Before long runs:
```bash
# Quick test
MAX_LOOPS=3 LOOP_DELAY=0 morty start

# Verify CLI works
$CLAUDE_CODE_CLI --version

# Check project files
ls -la .morty/
```

## Examples

### Example 1: Enterprise Setup

```bash
# ~/.bashrc
export CLAUDE_CODE_CLI="/opt/company/bin/ai_cli"
export MAX_LOOPS=100
export LOOP_DELAY=10

# Project-specific
cd my-project
cat > .morty/PROMPT.md << 'EOF'
# Enterprise Project Development

Follow company coding standards:
- Style guide: https://company.com/style
- Security: https://company.com/security
- Review: All code must pass security scan

Use company libraries:
- Auth: @company/auth
- Logging: @company/logger
EOF

morty monitor
```

### Example 2: Multi-Environment

```bash
# Development
export CLAUDE_CODE_CLI="claude"
export MAX_LOOPS=50
export LOOP_DELAY=5

# Staging
export CLAUDE_CODE_CLI="ai_cli --env staging"
export MAX_LOOPS=100
export LOOP_DELAY=10

# Production
export CLAUDE_CODE_CLI="ai_cli --env production --auth strict"
export MAX_LOOPS=200
export LOOP_DELAY=30
```

### Example 3: Team Workflow

```bash
# team-config.sh (committed to repo)
#!/bin/bash
export CLAUDE_CODE_CLI="ai_cli --team our-team"
export MAX_LOOPS=100
export LOOP_DELAY=10

# Usage
source team-config.sh
morty monitor
```

---

**Last Updated**: 2026-02-14
**Configuration Version**: 0.2.1
