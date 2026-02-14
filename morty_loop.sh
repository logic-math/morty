#!/bin/bash
# Morty Loop - Main development loop

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# Configuration
MORTY_DIR=".morty"
PROMPT_FILE="$MORTY_DIR/PROMPT.md"
LOG_DIR="$MORTY_DIR/logs"
STATUS_FILE="$MORTY_DIR/status.json"
LOOP_STATE_FILE="$MORTY_DIR/.loop_state"
SESSION_FILE="$MORTY_DIR/.session_id"
LOG_FILE="$LOG_DIR/morty.log"

CLAUDE_CMD="${CLAUDE_CODE_CLI:-claude}"
MAX_LOOPS="${MAX_LOOPS:-50}"
LOOP_DELAY="${LOOP_DELAY:-5}"
USE_TMUX=false

# Initialize
mkdir -p "$LOG_DIR"

show_help() {
    cat << 'EOF'
Morty Loop - Start development loop

Usage: morty start [options]
       morty monitor [options]

Options:
    -h, --help          Show this help message
    -m, --monitor       Start with tmux monitoring
    -s, --status        Show current status and exit
    --max-loops N       Maximum number of loops (default: 50)
    --delay N           Delay between loops in seconds (default: 5)

Examples:
    morty start
    morty monitor
    morty start --max-loops 100

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -m|--monitor)
            USE_TMUX=true
            shift
            ;;
        -s|--status)
            if [[ -f "$STATUS_FILE" ]]; then
                cat "$STATUS_FILE"
            else
                echo "No status file found. Morty may not be running."
            fi
            exit 0
            ;;
        --max-loops)
            MAX_LOOPS="$2"
            shift 2
            ;;
        --delay)
            LOOP_DELAY="$2"
            shift 2
            ;;
        *)
            log ERROR "Unknown argument: $1"
            exit 1
            ;;
    esac
done

# Check if this is a Morty project
if [[ ! -f "$PROMPT_FILE" ]]; then
    log ERROR "Not a Morty project (missing $PROMPT_FILE)"
    log INFO ""
    log INFO "To fix this:"
    log INFO "  1. Create new project: morty init my-project"
    log INFO "  2. Import from PRD: morty import requirements.md"
    log INFO "  3. Enable in existing project: morty enable"
    exit 1
fi

# Update status
update_status() {
    local state=$1
    local loop_count=$2
    local message=${3:-""}

    cat > "$STATUS_FILE" << EOF
{
    "state": "$state",
    "loop_count": $loop_count,
    "timestamp": "$(get_iso_timestamp)",
    "message": "$message"
}
EOF
}

# Check if should exit
should_exit() {
    local fix_plan="$MORTY_DIR/fix_plan.md"

    if [[ ! -f "$fix_plan" ]]; then
        return 1  # Don't exit
    fi

    # Check if all tasks are complete
    local total_tasks=$(grep -cE "^[[:space:]]*-[[:space:]]*\[[ xX]\]" "$fix_plan" 2>/dev/null || echo "0")
    local completed_tasks=$(grep -cE "^[[:space:]]*-[[:space:]]*\[[xX]\]" "$fix_plan" 2>/dev/null || echo "0")

    if [[ $total_tasks -gt 0 ]] && [[ $completed_tasks -eq $total_tasks ]]; then
        return 0  # Exit
    fi

    return 1  # Don't exit
}

# Execute Claude Code
execute_claude() {
    local loop_count=$1
    local timestamp=$(date '+%Y-%m-%d_%H-%M-%S')
    local output_file="$LOG_DIR/loop_${loop_count}_${timestamp}.log"

    log LOOP "Executing Claude Code (Loop #$loop_count)"

    # Build command
    local prompt_content=$(cat "$PROMPT_FILE")

    # Add loop context
    local loop_context="Loop #$loop_count. "

    # Add task count
    if [[ -f "$MORTY_DIR/fix_plan.md" ]]; then
        local incomplete=$(grep -cE "^[[:space:]]*-[[:space:]]*\[ \]" "$MORTY_DIR/fix_plan.md" 2>/dev/null || echo "0")
        loop_context+="Remaining tasks: $incomplete. "
    fi

    # Execute Claude (with timeout)
    local exit_code=0
    if timeout 600s $CLAUDE_CMD -p "$loop_context$prompt_content" > "$output_file" 2>&1; then
        log SUCCESS "Claude Code execution completed"
        exit_code=0
    else
        exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            log ERROR "Claude Code execution timed out (10 minutes)"
        else
            log ERROR "Claude Code execution failed (exit code: $exit_code)"
        fi
    fi

    # Analyze output for errors
    if grep -qiE "(error|exception|failed)" "$output_file" 2>/dev/null; then
        log WARN "Errors detected in output"
        return 2  # Error state
    fi

    # Check for completion signals
    if grep -qiE "(done|complete|finished|all tasks complete)" "$output_file" 2>/dev/null; then
        log INFO "Completion signal detected"
        return 3  # Done state
    fi

    return $exit_code
}

# Cleanup and exit
cleanup() {
    local reason=$1
    local context=${2:-"Loop interrupted"}

    log INFO "Cleaning up and exiting..."

    # Update PROMPT.md with exit context (hook)
    update_prompt_context "$reason" "$context"

    # Update status
    update_status "$reason" "${loop_count:-0}" "$context"

    # Remove loop state
    rm -f "$LOOP_STATE_FILE"

    log SUCCESS "Cleanup complete"
}

# Signal handlers
trap 'cleanup "interrupted" "User interrupted (Ctrl+C)"; exit 130' SIGINT SIGTERM

# Setup tmux if requested
if [[ "$USE_TMUX" == "true" ]]; then
    if ! command -v tmux &> /dev/null; then
        log ERROR "tmux is not installed"
        log INFO "Install: sudo apt-get install tmux (Ubuntu) or brew install tmux (macOS)"
        exit 1
    fi

    log INFO "Starting tmux monitoring session..."
    exec "$SCRIPT_DIR/morty_monitor.sh"
fi

# Main loop
main() {
    log SUCCESS "Starting Morty development loop"
    log INFO "Project: $(basename "$(pwd)")"
    log INFO "Max loops: $MAX_LOOPS"
    log INFO "Delay between loops: ${LOOP_DELAY}s"
    log INFO ""

    local loop_count=0
    local state="init"

    # Save initial state
    update_status "init" 0 "Initializing"

    while [[ $loop_count -lt $MAX_LOOPS ]]; do
        loop_count=$((loop_count + 1))

        log LOOP "=== Loop #$loop_count ==="

        # Save loop state
        echo "$loop_count" > "$LOOP_STATE_FILE"

        # Update status
        update_status "running" "$loop_count" "Executing loop"

        # Check for exit conditions
        if should_exit; then
            state="done"
            log SUCCESS "All tasks completed!"
            cleanup "done" "All tasks in fix_plan.md completed"
            update_status "done" "$loop_count" "All tasks completed"
            break
        fi

        # Execute Claude
        execute_claude "$loop_count"
        local exec_result=$?

        case $exec_result in
            0)
                # Success
                state="running"
                log SUCCESS "Loop #$loop_count completed successfully"

                # Auto-commit changes after successful loop
                git_auto_commit "$loop_count" "Loop iteration completed"
                ;;
            2)
                # Error detected
                state="error"
                log ERROR "Error detected in loop #$loop_count"
                cleanup "error" "Error detected in Claude output"
                update_status "error" "$loop_count" "Error detected"
                break
                ;;
            3)
                # Done signal
                state="done"
                log SUCCESS "Completion signal received"

                # Auto-commit final state
                git_auto_commit "$loop_count" "Project completion"

                cleanup "done" "Claude indicated completion"
                update_status "done" "$loop_count" "Completion signal received"
                break
                ;;
            *)
                # Other failure
                state="error"
                log ERROR "Unexpected error in loop #$loop_count"
                cleanup "error" "Unexpected error (exit code: $exec_result)"
                update_status "error" "$loop_count" "Unexpected error"
                break
                ;;
        esac

        # Delay between loops
        if [[ $loop_count -lt $MAX_LOOPS ]]; then
            log INFO "Waiting ${LOOP_DELAY}s before next loop..."
            sleep "$LOOP_DELAY"
        fi

        log LOOP "=== End Loop #$loop_count ==="
        log INFO ""
    done

    # Check if max loops reached
    if [[ $loop_count -ge $MAX_LOOPS ]]; then
        log WARN "Maximum loops ($MAX_LOOPS) reached"
        cleanup "max_loops" "Reached maximum loop count"
        update_status "max_loops" "$loop_count" "Maximum loops reached"
    fi

    log SUCCESS "Morty loop finished"
    log INFO "Final state: $state"
    log INFO "Total loops: $loop_count"
}

# Run main loop
main
