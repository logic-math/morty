#!/bin/bash
# Morty Plan Mode - Interactive PRD refinement with Claude Code

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# Configuration
CLAUDE_CMD="${CLAUDE_CODE_CLI:-claude}"
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

# Read system prompt
SYSTEM_PROMPT_CONTENT=$(cat "$PLAN_SYSTEM_PROMPT")

# Build the interactive prompt (combining system prompt + PRD)
INTERACTIVE_PROMPT=$(cat << EOF
$SYSTEM_PROMPT_CONTENT

---

# Initial PRD to Refine

Project Name: **$PROJECT_NAME**

## Initial PRD Content

\`\`\`markdown
$INITIAL_PRD_CONTENT
\`\`\`

---

**Instructions**: Follow the dialogue framework in the system prompt above. Start with Phase 1 (Understanding) by analyzing this PRD and asking your first round of clarifying questions.

When the PRD is comprehensive and ready, output the completion signal with the refined problem_description.md content as specified in the system prompt.
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

# Save prompt to file for Claude (avoids command-line argument length issues)
PROMPT_FILE="$PLAN_WORK_DIR/plan_prompt.md"
echo "$INTERACTIVE_PROMPT" > "$PROMPT_FILE"

log INFO "Prompt saved to: $PROMPT_FILE"
log INFO "CLAUDE_CMD: $CLAUDE_CMD"

# Build Claude command with proper flags for plan mode
# Use stdin to pass the prompt (more reliable than -p with long content)
CLAUDE_ARGS=(
    "$CLAUDE_CMD"
    "--continue"
    "--allowedTools" "Read" "Write" "Glob" "Grep" "WebSearch" "WebFetch"
)

# Execute Claude Code interactively with prompt from stdin
cat "$PROMPT_FILE" | "${CLAUDE_ARGS[@]}"

CLAUDE_EXIT_CODE=$?

log INFO ""
log INFO "Plan Mode session completed (exit code: $CLAUDE_EXIT_CODE)"
log INFO ""

# Check if Claude completed successfully
if [[ $CLAUDE_EXIT_CODE -ne 0 ]]; then
    log ERROR "Claude Code exited with error code: $CLAUDE_EXIT_CODE"
    log INFO "Plan mode session did not complete successfully"
    exit 1
fi

# Check if project directory was created by Claude
if [[ ! -d "$PROJECT_NAME" ]]; then
    log ERROR "Project directory was not created: $PROJECT_NAME"
    log INFO "Claude should have created the project structure during plan mode"
    log INFO "Please review the session output above for errors"
    exit 1
fi

log SUCCESS "Project directory created: $PROJECT_NAME"
log INFO ""

# Validate project structure using check library
log INFO "Validating project structure..."
log INFO ""

cd "$PROJECT_NAME" || exit 1

# Run validation check
if morty_check_project_structure "."; then
    log INFO ""
    log SUCCESS "╔════════════════════════════════════════════════════════════╗"
    log SUCCESS "║          PROJECT GENERATED SUCCESSFULLY!                   ║"
    log SUCCESS "╚════════════════════════════════════════════════════════════╝"
    log INFO ""
    log INFO "Project: $PROJECT_NAME"
    log INFO "Location: $(pwd)"
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
else
    log INFO ""
    log ERROR "╔════════════════════════════════════════════════════════════╗"
    log ERROR "║          PROJECT VALIDATION FAILED!                        ║"
    log ERROR "╚════════════════════════════════════════════════════════════╝"
    log INFO ""
    log ERROR "The project structure does not meet requirements"
    log INFO "Please review the validation errors above"
    log INFO ""
    log INFO "Common issues:"
    log INFO "  - Missing required files"
    log INFO "  - Empty or incomplete files"
    log INFO "  - Missing required sections in problem_description.md"
    log INFO "  - No checkbox tasks in fix_plan.md"
    log INFO ""
    log INFO "You can manually fix the issues and run validation again:"
    log INFO "  morty_check_project_structure ."
    exit 1
fi

# Clean up working directory
cd ..
rm -rf "$PLAN_WORK_DIR"
