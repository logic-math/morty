#!/bin/bash
# Morty Monitor - tmux monitoring dashboard

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# Configuration
MORTY_DIR=".morty"
STATUS_FILE="$MORTY_DIR/status.json"
LOG_DIR="$MORTY_DIR/logs"

# Check if tmux is available
if ! command -v tmux &> /dev/null; then
    log ERROR "tmux is not installed"
    exit 1
fi

# Get tmux base-index
get_tmux_base_index() {
    local base_index
    base_index=$(tmux show-options -gv base-index 2>/dev/null || echo "0")
    echo "${base_index:-0}"
}

# Setup tmux session
setup_tmux_session() {
    local session_name="morty-$(date +%s)"
    local project_dir="$(pwd)"
    local base_win=$(get_tmux_base_index)

    log INFO "Setting up tmux session: $session_name"

    # Create new tmux session (detached)
    tmux new-session -d -s "$session_name" -c "$project_dir"

    # Split window horizontally (left: loop, right: monitor)
    tmux split-window -h -t "$session_name" -c "$project_dir"

    # Split right pane vertically (top: logs, bottom: status)
    tmux split-window -v -t "$session_name:${base_win}.1" -c "$project_dir"

    # Left pane (pane 0): Morty loop
    tmux send-keys -t "$session_name:${base_win}.0" "morty start" Enter

    # Right-top pane (pane 1): Live logs
    tmux send-keys -t "$session_name:${base_win}.1" "tail -f '$project_dir/$LOG_DIR/morty.log' 2>/dev/null || echo 'Waiting for logs...'" Enter

    # Right-bottom pane (pane 2): Status monitor
    tmux send-keys -t "$session_name:${base_win}.2" "watch -n 2 'cat $project_dir/$STATUS_FILE 2>/dev/null | jq . 2>/dev/null || echo \"Waiting for status...\"'" Enter

    # Set pane titles
    tmux select-pane -t "$session_name:${base_win}.0" -T "Morty Loop"
    tmux select-pane -t "$session_name:${base_win}.1" -T "Logs"
    tmux select-pane -t "$session_name:${base_win}.2" -T "Status"

    # Set window title
    tmux rename-window -t "$session_name:${base_win}" "Morty: Loop | Logs | Status"

    # Focus on left pane
    tmux select-pane -t "$session_name:${base_win}.0"

    log SUCCESS "Tmux session created with 3 panes:"
    log INFO "  Left:         Morty loop"
    log INFO "  Right-top:    Live logs"
    log INFO "  Right-bottom: Status monitor"
    log INFO ""
    log INFO "Controls:"
    log INFO "  Ctrl+B then D     - Detach from session"
    log INFO "  Ctrl+B then ←/→   - Switch panes"
    log INFO "  tmux attach -t $session_name - Reattach"
    log INFO ""

    # Attach to session
    tmux attach-session -t "$session_name"
}

# Run setup
setup_tmux_session
