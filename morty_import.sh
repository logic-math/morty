#!/bin/bash
# Morty Import - Import PRD and create project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

show_help() {
    cat << 'EOF'
Morty Import - Import PRD/requirements document

Usage: morty import <prd.md> [project-name]

Arguments:
    prd.md          Path to PRD/requirements Markdown file
    project-name    Optional project name (defaults to filename)

Examples:
    morty import requirements.md
    morty import docs/prd.md my-project

EOF
}

# Parse arguments
PRD_FILE=""
PROJECT_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            if [[ -z "$PRD_FILE" ]]; then
                PRD_FILE="$1"
            elif [[ -z "$PROJECT_NAME" ]]; then
                PROJECT_NAME="$1"
            else
                log ERROR "Unknown argument: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$PRD_FILE" ]]; then
    log ERROR "PRD file is required"
    show_help
    exit 1
fi

# Check if PRD file exists
if [[ ! -f "$PRD_FILE" ]]; then
    log ERROR "PRD file not found: $PRD_FILE"
    exit 1
fi

# Check if it's a Markdown file
if [[ ! "$PRD_FILE" =~ \.md$ ]]; then
    log ERROR "Only Markdown (.md) files are supported"
    exit 1
fi

# Get absolute path
PRD_FILE=$(realpath "$PRD_FILE")

# Determine project name
if [[ -z "$PROJECT_NAME" ]]; then
    PROJECT_NAME=$(basename "$PRD_FILE" .md)
    PROJECT_NAME=$(echo "$PROJECT_NAME" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
fi

log INFO "Importing PRD from: $PRD_FILE"
log INFO "Creating project: $PROJECT_NAME"

# Create project directory
if [[ -d "$PROJECT_NAME" ]]; then
    log ERROR "Directory '$PROJECT_NAME' already exists"
    exit 1
fi

mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

# Initialize git
git init > /dev/null 2>&1

# Create .morty directory
mkdir -p .morty/{logs,specs}

# Copy PRD to specs
cp "$PRD_FILE" .morty/specs/requirements.md
log SUCCESS "Copied PRD to .morty/specs/requirements.md"

# Parse PRD and extract tasks
log INFO "Parsing PRD to extract tasks..."

# Extract tasks from PRD (look for checkbox lists and numbered lists)
extract_tasks() {
    local prd_file=$1
    local tasks=""

    # Extract checkbox items: - [ ] task or - [x] task
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*\[[[:space:]xX]?\][[:space:]]*(.+)$ ]]; then
            local task="${BASH_REMATCH[1]}"
            tasks+="- [ ] $task"$'\n'
        fi
    done < "$prd_file"

    # If no checkboxes found, extract numbered lists
    if [[ -z "$tasks" ]]; then
        while IFS= read -r line; do
            if [[ "$line" =~ ^[[:space:]]*[0-9]+\.[[:space:]]*(.+)$ ]]; then
                local task="${BASH_REMATCH[1]}"
                tasks+="- [ ] $task"$'\n'
            fi
        done < "$prd_file"
    fi

    # If still no tasks, create default tasks
    if [[ -z "$tasks" ]]; then
        tasks="- [ ] Review requirements in .morty/specs/requirements.md"$'\n'
        tasks+="- [ ] Set up project structure"$'\n'
        tasks+="- [ ] Implement core functionality"$'\n'
        tasks+="- [ ] Add tests"$'\n'
        tasks+="- [ ] Write documentation"$'\n'
    fi

    echo "$tasks"
}

EXTRACTED_TASKS=$(extract_tasks "$PRD_FILE")

# Create fix_plan.md with extracted tasks
cat > .morty/fix_plan.md << EOF
# Task List

Imported from: $(basename "$PRD_FILE")
Date: $(date '+%Y-%m-%d')

## Tasks

$EXTRACTED_TASKS

## Notes
- See .morty/specs/requirements.md for detailed requirements
- Mark tasks with [x] when completed

EOF

log SUCCESS "Created fix_plan.md with $(echo "$EXTRACTED_TASKS" | grep -c "^\- \[ \]") tasks"

# Extract project overview from PRD (first paragraph or section)
PROJECT_OVERVIEW=$(head -20 "$PRD_FILE" | grep -v "^#" | grep -v "^$" | head -5)

# Create PROMPT.md based on PRD
cat > .morty/PROMPT.md << EOF
# Development Instructions

You are developing a project based on the requirements in \`.morty/specs/requirements.md\`.

## Project Overview

$PROJECT_OVERVIEW

## Requirements

See \`.morty/specs/requirements.md\` for detailed requirements.

## Development Principles

1. Follow the requirements in the PRD
2. Write clean, maintainable code
3. Add appropriate error handling
4. Write tests for new features
5. Update documentation as you work

## Current Tasks

See \`.morty/fix_plan.md\` for prioritized tasks.

## Build and Test

See \`.morty/AGENT.md\` for build and test commands.

## Notes

- Always refer back to the requirements document
- Ask for clarification if requirements are unclear
- Mark tasks complete in fix_plan.md as you finish them

EOF

log SUCCESS "Created PROMPT.md with project context"

# Create AGENT.md with detected commands
BUILD_CMD=$(detect_build_command)
TEST_CMD=$(detect_test_command)

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
# Add run commands here
\`\`\`

## Development
\`\`\`bash
# Add development commands here
\`\`\`

EOF

# Create basic project structure
mkdir -p src

# Create README
cat > README.md << EOF
# $PROJECT_NAME

Project created from PRD with Morty.

## Requirements

See \`.morty/specs/requirements.md\` for detailed requirements.

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
log INFO "  2. Review .morty/PROMPT.md and .morty/fix_plan.md"
log INFO "  3. Run 'morty start' to begin development"
log INFO ""
log SUCCESS "Project '$PROJECT_NAME' imported successfully!"
