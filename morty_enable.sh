#!/bin/bash
# Morty Enable - Enable Morty in existing project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

show_help() {
    cat << 'EOF'
Morty Enable - Enable Morty in existing project

Usage: morty enable [options]

Options:
    -h, --help          Show this help message
    --force             Overwrite existing .morty directory

Examples:
    cd existing-project
    morty enable

EOF
}

# Parse arguments
FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --force)
            FORCE=true
            shift
            ;;
        *)
            log ERROR "Unknown argument: $1"
            exit 1
            ;;
    esac
done

# Check if already a Morty project
if [[ -d ".morty" ]] && [[ "$FORCE" != "true" ]]; then
    log ERROR "This is already a Morty project (.morty directory exists)"
    log INFO "Use --force to overwrite"
    exit 1
fi

log INFO "Enabling Morty in current project..."

# Detect project context
PROJECT_TYPE=$(detect_project_type)
log INFO "Detected project type: $PROJECT_TYPE"

# Get project name from directory
PROJECT_NAME=$(basename "$(pwd)")

# Create or recreate .morty directory
if [[ -d ".morty" ]] && [[ "$FORCE" == "true" ]]; then
    log WARN "Removing existing .morty directory"
    rm -rf .morty
fi

mkdir -p .morty/{logs,specs}

# Detect build and test commands
BUILD_CMD=$(detect_build_command)
TEST_CMD=$(detect_test_command)

log INFO "Detected build command: $BUILD_CMD"
log INFO "Detected test command: $TEST_CMD"

# Create PROMPT.md
cat > .morty/PROMPT.md << EOF
# Development Instructions

You are helping to develop the $PROJECT_NAME project.

## Project Type
$PROJECT_TYPE

## Development Principles

1. Understand the existing codebase before making changes
2. Write clean, maintainable code
3. Follow the project's existing patterns and conventions
4. Add tests for new features
5. Update documentation when making changes

## Current Tasks

See \`.morty/fix_plan.md\` for prioritized tasks.

## Build and Test

See \`.morty/AGENT.md\` for build and test commands.

## Notes

- Always check existing code before implementing new features
- Prefer editing existing files over creating new ones
- Test changes before marking tasks complete
- Maintain backward compatibility unless explicitly asked to break it

EOF

# Create fix_plan.md with default tasks
cat > .morty/fix_plan.md << 'EOF'
# Task List

## High Priority
- [ ] Review existing codebase
- [ ] Identify areas for improvement
- [ ] Fix critical bugs

## Medium Priority
- [ ] Add missing tests
- [ ] Improve error handling
- [ ] Update documentation

## Low Priority
- [ ] Refactor code for better maintainability
- [ ] Optimize performance
- [ ] Add examples

## Notes
- Add specific tasks as needed
- Mark tasks with [x] when completed

EOF

# Create AGENT.md with detected commands
cat > .morty/AGENT.md << EOF
# Build and Run Instructions

## Build
\`\`\`bash
$BUILD_CMD
\`\`\`

## Test
\`\`\`bash
$TEST_CMD
\`\`\`

## Run
\`\`\`bash
# Add run commands here based on project type
EOF

case $PROJECT_TYPE in
    nodejs)
        echo "npm start" >> .morty/AGENT.md
        ;;
    python)
        echo "python src/main.py" >> .morty/AGENT.md
        ;;
    rust)
        echo "cargo run" >> .morty/AGENT.md
        ;;
    go)
        echo "go run ." >> .morty/AGENT.md
        ;;
    *)
        echo "# Add run command here" >> .morty/AGENT.md
        ;;
esac

cat >> .morty/AGENT.md << 'EOF'
```

## Development
```bash
# Add development commands here
```

EOF

# Update .gitignore if it exists, or create it
if [[ -f ".gitignore" ]]; then
    if ! grep -q ".morty/logs/" .gitignore; then
        log INFO "Adding Morty entries to .gitignore"
        cat >> .gitignore << 'EOF'

# Morty files
.morty/logs/
.morty/.loop_state
.morty/.session_id

EOF
    fi
else
    log INFO "Creating .gitignore"
    cat > .gitignore << 'EOF'
# Morty files
.morty/logs/
.morty/.loop_state
.morty/.session_id

EOF
fi

log SUCCESS "Morty enabled in project"
log INFO ""
log INFO "Next steps:"
log INFO "  1. Review and customize .morty/PROMPT.md"
log INFO "  2. Add specific tasks to .morty/fix_plan.md"
log INFO "  3. Update .morty/AGENT.md with correct commands"
log INFO "  4. Run 'morty start' to begin development"
log INFO ""
log SUCCESS "Project is now Morty-enabled!"
