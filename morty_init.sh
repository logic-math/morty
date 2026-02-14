#!/bin/bash
# Morty Init - Create new project from scratch

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

show_help() {
    cat << 'EOF'
Morty Init - Create new project

Usage: morty init <project-name> [options]

Options:
    -h, --help          Show this help message
    --type TYPE         Project type (nodejs|python|rust|go|generic)

Examples:
    morty init my-app
    morty init my-api --type nodejs

EOF
}

# Parse arguments
PROJECT_NAME=""
PROJECT_TYPE="generic"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --type)
            PROJECT_TYPE="$2"
            shift 2
            ;;
        *)
            if [[ -z "$PROJECT_NAME" ]]; then
                PROJECT_NAME="$1"
            else
                log ERROR "Unknown argument: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$PROJECT_NAME" ]]; then
    log ERROR "Project name is required"
    show_help
    exit 1
fi

# Create project directory
if [[ -d "$PROJECT_NAME" ]]; then
    log ERROR "Directory '$PROJECT_NAME' already exists"
    exit 1
fi

log INFO "Creating project: $PROJECT_NAME"
mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

# Initialize git
git init > /dev/null 2>&1
log SUCCESS "Initialized git repository"

# Create .morty directory structure
mkdir -p .morty/{logs,specs}

# Create PROMPT.md
cat > .morty/PROMPT.md << 'EOF'
# Development Instructions

You are an AI assistant helping to develop this project. Follow these guidelines:

## Project Overview
[Describe the project purpose and goals]

## Development Principles
1. Write clean, maintainable code
2. Add comments for complex logic
3. Follow the project's coding standards
4. Write tests for new features
5. Update documentation when making changes

## Current Tasks
See `.morty/fix_plan.md` for prioritized tasks.

## Build and Test
See `.morty/AGENT.md` for build and test commands.

## Notes
- Always check existing code before making changes
- Prefer editing existing files over creating new ones
- Test changes before marking tasks complete

EOF

# Create fix_plan.md
cat > .morty/fix_plan.md << 'EOF'
# Task List

## High Priority
- [ ] Set up project structure
- [ ] Implement core functionality
- [ ] Add tests
- [ ] Write documentation

## Medium Priority
- [ ] Add error handling
- [ ] Optimize performance
- [ ] Add logging

## Low Priority
- [ ] Add examples
- [ ] Improve UI/UX

EOF

# Create AGENT.md
cat > .morty/AGENT.md << 'EOF'
# Build and Run Instructions

## Build
```bash
# Add build commands here
```

## Test
```bash
# Add test commands here
```

## Run
```bash
# Add run commands here
```

## Development
```bash
# Add development commands here
```

EOF

# Create basic source structure
mkdir -p src
cat > src/main.${PROJECT_TYPE} << 'EOF'
// Main entry point
// TODO: Implement main logic

EOF

# Create README
cat > README.md << EOF
# $PROJECT_NAME

Project created with Morty.

## Setup

\`\`\`bash
# Add setup instructions
\`\`\`

## Usage

\`\`\`bash
# Add usage instructions
\`\`\`

## Development

This project uses Morty for AI-assisted development.

\`\`\`bash
morty start    # Start development loop
morty monitor  # Start with monitoring
\`\`\`

EOF

# Create .gitignore
cat > .gitignore << 'EOF'
# Morty files
.morty/logs/
.morty/.loop_state
.morty/.session_id

# Common
node_modules/
__pycache__/
*.pyc
.env
.DS_Store
target/
dist/
build/

EOF

log SUCCESS "Project structure created"
log INFO ""
log INFO "Next steps:"
log INFO "  1. cd $PROJECT_NAME"
log INFO "  2. Edit .morty/PROMPT.md with your project goals"
log INFO "  3. Edit .morty/fix_plan.md with specific tasks"
log INFO "  4. Run 'morty start' to begin development"
log INFO ""
log SUCCESS "Project '$PROJECT_NAME' created successfully!"
