#!/bin/bash
# Morty Plan Mode - Interactive PRD refinement with Claude Code

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# Configuration
CLAUDE_CMD="claude"
PLAN_SYSTEM_PROMPT="$SCRIPT_DIR/prompts/plan_mode_system.md"

show_help() {
    cat << 'EOF'
Morty Plan Mode - Interactive PRD Refinement

Usage: morty plan <prd.md> [project-name]

Arguments:
    prd.md          Initial PRD/requirements document (Markdown)
    project-name    Optional project name (defaults to filename)

Description:
    Plan mode launches an interactive Claude Code session to:
    1. Analyze the initial PRD
    2. Ask clarifying questions through dialogue
    3. Refine and expand requirements
    4. Generate comprehensive problem_description.md (ai_prd.md)
    5. Auto-generate project structure with:
       - .morty/PROMPT.md (development instructions)
       - .morty/fix_plan.md (actionable task breakdown)
       - .morty/AGENT.md (build/test commands)
       - .morty/specs/ (detailed specifications)

Examples:
    morty plan requirements.md
    morty plan docs/prd.md my-app

Features:
    - Interactive dialogue with Claude Code
    - Exploratory mode for deep understanding
    - Iterative refinement through conversation
    - Automatic project scaffolding
    - Context-aware file generation

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

log INFO "╔════════════════════════════════════════════════════════════╗"
log INFO "║              MORTY PLAN MODE - PRD REFINEMENT              ║"
log INFO "╚════════════════════════════════════════════════════════════╝"
log INFO ""
log INFO "Initial PRD: $PRD_FILE"
log INFO "Project name: $PROJECT_NAME"
log INFO ""

# Check if system prompt exists
if [[ ! -f "$PLAN_SYSTEM_PROMPT" ]]; then
    log ERROR "Plan mode system prompt not found: $PLAN_SYSTEM_PROMPT"
    log INFO "Creating system prompt..."
    mkdir -p "$(dirname "$PLAN_SYSTEM_PROMPT")"

    # Create default system prompt (will be created separately)
    log ERROR "Please run installation first to create system prompts"
    exit 1
fi

# Create working directory for plan mode
PLAN_WORK_DIR=".morty_plan_$$"
mkdir -p "$PLAN_WORK_DIR"

log INFO "Working directory: $PLAN_WORK_DIR"
log INFO ""

# Copy initial PRD to working directory
cp "$PRD_FILE" "$PLAN_WORK_DIR/initial_prd.md"

# Read initial PRD content
INITIAL_PRD_CONTENT=$(cat "$PRD_FILE")

# Build the interactive prompt
INTERACTIVE_PROMPT=$(cat << EOF
# Morty Plan Mode - PRD Refinement Session

You are now in **Plan Mode**. Your mission is to help refine and expand this initial PRD through interactive dialogue.

## Initial PRD Content

\`\`\`markdown
$INITIAL_PRD_CONTENT
\`\`\`

## Your Task

1. **Analyze** the initial PRD thoroughly
2. **Ask clarifying questions** to understand:
   - Project goals and objectives
   - Target users and use cases
   - Technical constraints and requirements
   - Success criteria and metrics
   - Timeline and priorities

3. **Engage in dialogue** with the user to:
   - Identify gaps and ambiguities
   - Explore edge cases
   - Understand dependencies
   - Clarify technical decisions

4. **Refine iteratively** through conversation until you have:
   - Clear problem statement
   - Comprehensive requirements
   - Detailed user stories
   - Technical specifications
   - Acceptance criteria

5. **Generate artifacts** when refinement is complete:
   - problem_description.md (refined PRD)
   - Project structure recommendations
   - Task breakdown
   - Development approach

## Dialogue Guidelines

- Ask **open-ended questions** to explore deeply
- **Challenge assumptions** constructively
- **Propose alternatives** when appropriate
- **Summarize understanding** regularly
- **Confirm decisions** before moving forward
- Use **"What if..."** scenarios to explore edge cases
- Ask **"Why..."** to understand motivations

## Output Format

During dialogue, use this format:

**Understanding**: [Your current understanding]
**Questions**: [Numbered list of questions]
**Observations**: [Insights or concerns]
**Next Steps**: [What should we explore next]

## Completion Signal

When the PRD is comprehensive and clear, output:

\`\`\`markdown
<!-- PLAN_MODE_COMPLETE -->

# Refined Problem Description

[Complete refined PRD content here]

## Project Metadata
- Project Name: $PROJECT_NAME
- Type: [detected or specified type]
- Complexity: [low/medium/high]
- Timeline: [estimated]

## Recommended Structure
[Suggested project organization]

## Task Breakdown
[High-level tasks with priorities]

## Technical Approach
[Recommended technologies and patterns]
\`\`\`

## Ready?

Let's begin! I'll start by analyzing the initial PRD and asking my first round of questions.

What would you like to know first, or should I begin with my analysis?
EOF
)

log INFO "Starting interactive Plan Mode session..."
log INFO ""
log INFO "Instructions:"
log INFO "  - Claude will ask questions to refine the PRD"
log INFO "  - Answer thoughtfully to help clarify requirements"
log INFO "  - Type your responses naturally"
log INFO "  - Claude will iterate until the PRD is complete"
log INFO "  - Session ends when Claude outputs: <!-- PLAN_MODE_COMPLETE -->"
log INFO ""
log INFO "Press Enter to start the interactive session..."
read -r

# Launch Claude Code in interactive mode with plan system prompt
# Key flags:
#   --continue: Enable session continuity for context preservation
#   --dangerously-skip-permissions: Allow full tool access in plan mode
#   -p: Initial prompt with PRD content
log INFO "Launching Claude Code in Plan Mode..."
log INFO ""

# Save prompt to file for Claude
PROMPT_FILE="$PLAN_WORK_DIR/plan_prompt.md"
echo "$INTERACTIVE_PROMPT" > "$PROMPT_FILE"

# Build Claude command with proper flags for plan mode
CLAUDE_ARGS=(
    "$CLAUDE_CMD"
    "-p" "$INTERACTIVE_PROMPT"
    "--continue"
    "--dangerously-skip-permissions"
    "--allowedTools" "Read" "Write" "Glob" "Grep" "WebSearch" "WebFetch"
)

# Execute Claude Code interactively
"${CLAUDE_ARGS[@]}"

CLAUDE_EXIT_CODE=$?

log INFO ""
log INFO "Plan Mode session completed (exit code: $CLAUDE_EXIT_CODE)"
log INFO ""

# Check if refined PRD was generated
REFINED_PRD="$PLAN_WORK_DIR/problem_description.md"

if [[ ! -f "$REFINED_PRD" ]]; then
    log WARN "No problem_description.md found in working directory"
    log INFO "Looking for output in current directory..."

    # Check if Claude created it in current directory
    if [[ -f "problem_description.md" ]]; then
        mv "problem_description.md" "$REFINED_PRD"
        log SUCCESS "Found problem_description.md"
    else
        log ERROR "Plan mode did not generate problem_description.md"
        log INFO "Please ensure Claude completed the refinement and created the file"
        exit 1
    fi
fi

log SUCCESS "Refined PRD generated: $REFINED_PRD"
log INFO ""

# Now generate the project structure
log INFO "Generating project structure from refined PRD..."
log INFO ""

# Create project directory
if [[ -d "$PROJECT_NAME" ]]; then
    log ERROR "Project directory already exists: $PROJECT_NAME"
    log INFO "Please remove it or choose a different name"
    exit 1
fi

mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

# Initialize git
git init > /dev/null 2>&1
log SUCCESS "Initialized git repository"

# Create .morty directory structure
mkdir -p .morty/{logs,specs}

# Copy refined PRD to specs
cp "$PLAN_WORK_DIR/problem_description.md" .morty/specs/problem_description.md
log SUCCESS "Copied refined PRD to .morty/specs/"

# Read refined PRD for context
REFINED_PRD_CONTENT=$(cat "$REFINED_PRD")

# Generate PROMPT.md based on refined PRD
log INFO "Generating PROMPT.md..."

cat > .morty/PROMPT.md << 'PROMPT_EOF'
# Development Instructions

You are developing this project based on the refined problem description in `.morty/specs/problem_description.md`.

## Problem Understanding

Read the problem description carefully. It contains:
- Clear problem statement
- Comprehensive requirements
- User stories and use cases
- Technical specifications
- Acceptance criteria

## Development Principles

1. **Requirement-Driven**: Always refer back to the problem description
2. **Incremental Progress**: Tackle tasks in priority order from fix_plan.md
3. **Quality First**: Write clean, tested, documented code
4. **User-Centric**: Keep the end user's needs in focus
5. **Iterative Refinement**: Improve as you learn

## Workflow

1. Check `.morty/fix_plan.md` for current task
2. Review relevant sections in problem_description.md
3. Implement the task following specifications
4. Test thoroughly
5. Update documentation
6. Mark task complete in fix_plan.md
7. Move to next task

## Current Context

- **Problem Description**: `.morty/specs/problem_description.md`
- **Task List**: `.morty/fix_plan.md`
- **Build Commands**: `.morty/AGENT.md`

## Communication

When you complete a task or need clarification:
- Explain what you did and why
- Reference specific requirements
- Note any decisions or trade-offs
- Ask questions if requirements are unclear

## Quality Standards

- All code must have clear purpose
- Edge cases must be handled
- Error messages must be helpful
- Documentation must be current
- Tests must be comprehensive

## RALPH_STATUS Block

At the end of each loop iteration, output:

```
RALPH_STATUS:
STATUS: [IN_PROGRESS|COMPLETE|BLOCKED]
EXIT_SIGNAL: [true|false]
WORK_TYPE: [implementation|testing|documentation|refactoring]
FILES_MODIFIED: [number]
SUMMARY: [Brief description of what was done]
NEXT_STEPS: [What should happen next]
```

Use EXIT_SIGNAL: true only when ALL tasks are complete and project is ready.

PROMPT_EOF

log SUCCESS "Created PROMPT.md"

# Generate fix_plan.md with task extraction
log INFO "Generating fix_plan.md..."

# Extract tasks from refined PRD (look for task sections, checkboxes, numbered items)
TASKS=$(cat "$REFINED_PRD" | grep -E "^[[:space:]]*[-*] \[[ x]\]|^[[:space:]]*[0-9]+\." | head -20)

if [[ -z "$TASKS" ]]; then
    # No explicit tasks found, create default structure
    cat > .morty/fix_plan.md << 'FIXPLAN_EOF'
# Task List

Generated from refined problem description.

## Phase 1: Project Setup
- [ ] Review problem_description.md thoroughly
- [ ] Set up development environment
- [ ] Create project structure
- [ ] Initialize testing framework

## Phase 2: Core Implementation
- [ ] Implement main functionality
- [ ] Add error handling
- [ ] Write unit tests
- [ ] Integration testing

## Phase 3: Documentation & Polish
- [ ] Write user documentation
- [ ] Add code comments
- [ ] Create examples
- [ ] Final testing

## Notes
- Refer to `.morty/specs/problem_description.md` for detailed requirements
- Mark tasks with [x] when completed
- Add new tasks as needed during development

FIXPLAN_EOF
else
    # Use extracted tasks
    cat > .morty/fix_plan.md << EOF
# Task List

Generated from refined problem description.

## Tasks

$TASKS

## Notes
- Refer to \`.morty/specs/problem_description.md\` for detailed requirements
- Mark tasks with [x] when completed
- Add new tasks as needed during development

EOF
fi

log SUCCESS "Created fix_plan.md"

# Generate AGENT.md with detected commands
log INFO "Generating AGENT.md..."

# Detect project type from refined PRD
PROJECT_TYPE="generic"
if grep -qi "python\|django\|flask\|fastapi" "$REFINED_PRD"; then
    PROJECT_TYPE="python"
elif grep -qi "javascript\|typescript\|node\|react\|vue" "$REFINED_PRD"; then
    PROJECT_TYPE="nodejs"
elif grep -qi "rust\|cargo" "$REFINED_PRD"; then
    PROJECT_TYPE="rust"
elif grep -qi "go\|golang" "$REFINED_PRD"; then
    PROJECT_TYPE="go"
fi

log INFO "Detected project type: $PROJECT_TYPE"

# Generate AGENT.md based on project type
case $PROJECT_TYPE in
    python)
        cat > .morty/AGENT.md << 'AGENT_EOF'
# Build and Run Instructions

## Setup
```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

## Development
```bash
# Run in development mode
python src/main.py

# Or if using a framework:
# flask run
# uvicorn main:app --reload
```

## Testing
```bash
# Run tests
pytest

# With coverage
pytest --cov=src tests/

# Specific test file
pytest tests/test_module.py
```

## Build
```bash
# Install in editable mode
pip install -e .

# Build distribution
python setup.py sdist bdist_wheel
```

## Linting
```bash
# Format code
black src/
isort src/

# Check style
flake8 src/
pylint src/

# Type checking
mypy src/
```

AGENT_EOF
        ;;
    nodejs)
        cat > .morty/AGENT.md << 'AGENT_EOF'
# Build and Run Instructions

## Setup
```bash
# Install dependencies
npm install
# or
yarn install
```

## Development
```bash
# Run in development mode
npm run dev
# or
yarn dev

# Start server
npm start
```

## Testing
```bash
# Run tests
npm test

# Watch mode
npm test -- --watch

# Coverage
npm test -- --coverage
```

## Build
```bash
# Production build
npm run build

# Type checking (if TypeScript)
npm run type-check
```

## Linting
```bash
# Lint code
npm run lint

# Format code
npm run format

# Fix issues
npm run lint:fix
```

AGENT_EOF
        ;;
    rust)
        cat > .morty/AGENT.md << 'AGENT_EOF'
# Build and Run Instructions

## Development
```bash
# Run in development mode
cargo run

# With arguments
cargo run -- arg1 arg2
```

## Testing
```bash
# Run tests
cargo test

# Specific test
cargo test test_name

# With output
cargo test -- --nocapture
```

## Build
```bash
# Debug build
cargo build

# Release build
cargo build --release

# Check without building
cargo check
```

## Linting
```bash
# Format code
cargo fmt

# Lint with Clippy
cargo clippy

# Check for errors
cargo clippy -- -D warnings
```

AGENT_EOF
        ;;
    go)
        cat > .morty/AGENT.md << 'AGENT_EOF'
# Build and Run Instructions

## Development
```bash
# Run
go run .

# Or specific file
go run main.go
```

## Testing
```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...

# Specific package
go test ./pkg/module
```

## Build
```bash
# Build binary
go build

# Build with output name
go build -o app

# Build for production
go build -ldflags="-s -w" -o app
```

## Linting
```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run golangci-lint
golangci-lint run
```

AGENT_EOF
        ;;
    *)
        cat > .morty/AGENT.md << 'AGENT_EOF'
# Build and Run Instructions

## Setup
```bash
# Add setup commands here based on project type
```

## Development
```bash
# Add development commands here
```

## Testing
```bash
# Add testing commands here
```

## Build
```bash
# Add build commands here
```

## Notes
Update these commands based on your specific project setup.

AGENT_EOF
        ;;
esac

log SUCCESS "Created AGENT.md (type: $PROJECT_TYPE)"

# Create basic source structure
mkdir -p src
log SUCCESS "Created src/ directory"

# Create README
cat > README.md << EOF
# $PROJECT_NAME

Project generated with Morty Plan Mode.

## Problem Description

See \`.morty/specs/problem_description.md\` for the refined problem description and requirements.

## Development

This project uses Morty for AI-assisted development.

\`\`\`bash
# Start development loop
morty start

# Or with monitoring
morty monitor
\`\`\`

## Project Structure

\`\`\`
$PROJECT_NAME/
├── .morty/
│   ├── PROMPT.md              # Development instructions
│   ├── fix_plan.md            # Task list
│   ├── AGENT.md               # Build/test commands
│   └── specs/
│       └── problem_description.md  # Refined PRD
├── src/                       # Source code
└── README.md
\`\`\`

## Getting Started

1. Review \`.morty/specs/problem_description.md\`
2. Check \`.morty/fix_plan.md\` for tasks
3. Run \`morty start\` to begin development

EOF

log SUCCESS "Created README.md"

# Create .gitignore
cat > .gitignore << 'GITIGNORE_EOF'
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
venv/
*.log

GITIGNORE_EOF

log SUCCESS "Created .gitignore"

# Clean up working directory
cd ..
rm -rf "$PLAN_WORK_DIR"

log INFO ""
log SUCCESS "╔════════════════════════════════════════════════════════════╗"
log SUCCESS "║          PROJECT GENERATED SUCCESSFULLY!                   ║"
log SUCCESS "╚════════════════════════════════════════════════════════════╝"
log INFO ""
log INFO "Project: $PROJECT_NAME"
log INFO "Location: $(pwd)/$PROJECT_NAME"
log INFO ""
log INFO "Generated files:"
log INFO "  ✓ .morty/PROMPT.md              Development instructions"
log INFO "  ✓ .morty/fix_plan.md            Task breakdown"
log INFO "  ✓ .morty/AGENT.md               Build/test commands"
log INFO "  ✓ .morty/specs/problem_description.md  Refined PRD"
log INFO "  ✓ src/                          Source directory"
log INFO "  ✓ README.md                     Project documentation"
log INFO ""
log INFO "Next steps:"
log INFO "  1. cd $PROJECT_NAME"
log INFO "  2. Review .morty/specs/problem_description.md"
log INFO "  3. Check .morty/fix_plan.md for tasks"
log INFO "  4. Run 'morty start' to begin development"
log INFO ""
log SUCCESS "Happy coding! 🚀"
