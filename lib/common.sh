#!/bin/bash
# Common utilities for Morty

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Logging function
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local color=""

    case $level in
        INFO)  color=$BLUE ;;
        WARN)  color=$YELLOW ;;
        ERROR) color=$RED ;;
        SUCCESS) color=$GREEN ;;
        LOOP) color=$PURPLE ;;
    esac

    echo -e "${color}[$timestamp] [$level] $message${NC}" >&2

    # Also log to file if LOG_FILE is set
    if [[ -n "${LOG_FILE:-}" ]]; then
        echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    fi
}

# Check if directory is a Morty project
is_morty_project() {
    [[ -f ".morty/PROMPT.md" ]]
}

# Get ISO timestamp
get_iso_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Detect project type
detect_project_type() {
    if [[ -f "package.json" ]]; then
        echo "nodejs"
    elif [[ -f "requirements.txt" ]] || [[ -f "pyproject.toml" ]]; then
        echo "python"
    elif [[ -f "Cargo.toml" ]]; then
        echo "rust"
    elif [[ -f "go.mod" ]]; then
        echo "go"
    else
        echo "generic"
    fi
}

# Detect build command
detect_build_command() {
    local project_type=$(detect_project_type)

    case $project_type in
        nodejs)
            if grep -q '"build"' package.json 2>/dev/null; then
                echo "npm run build"
            else
                echo "npm install"
            fi
            ;;
        python)
            if [[ -f "pyproject.toml" ]]; then
                echo "poetry install"
            else
                echo "pip install -r requirements.txt"
            fi
            ;;
        rust)
            echo "cargo build"
            ;;
        go)
            echo "go build"
            ;;
        *)
            echo "# Add build commands here"
            ;;
    esac
}

# Detect test command
detect_test_command() {
    local project_type=$(detect_project_type)

    case $project_type in
        nodejs)
            if grep -q '"test"' package.json 2>/dev/null; then
                echo "npm test"
            else
                echo "# Add test commands here"
            fi
            ;;
        python)
            if [[ -f "pytest.ini" ]] || grep -q "pytest" requirements.txt 2>/dev/null; then
                echo "pytest"
            else
                echo "python -m unittest"
            fi
            ;;
        rust)
            echo "cargo test"
            ;;
        go)
            echo "go test ./..."
            ;;
        *)
            echo "# Add test commands here"
            ;;
    esac
}

# Update PROMPT.md with context (hook for exit)
update_prompt_context() {
    local reason=$1
    local context=$2
    local prompt_file=".morty/PROMPT.md"

    if [[ ! -f "$prompt_file" ]]; then
        return 0
    fi

    local timestamp=$(get_iso_timestamp)
    local update_marker="<!-- MORTY_LAST_UPDATE -->"

    # Create update section
    local update_section="
$update_marker
**Last Update**: $timestamp
**Reason**: $reason
**Context**: $context
"

    # Check if marker exists
    if grep -q "$update_marker" "$prompt_file"; then
        # Replace existing update section
        sed -i "/$update_marker/,\$d" "$prompt_file"
    fi

    # Append new update section
    echo "$update_section" >> "$prompt_file"

    log INFO "Updated PROMPT.md with exit context"
}
