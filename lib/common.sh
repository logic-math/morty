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

# Git auto-commit after each loop
# Creates a snapshot of changes for easy rollback
git_auto_commit() {
    local loop_count=$1
    local work_summary=${2:-"Loop iteration"}

    # Check if git is available
    if ! command -v git &> /dev/null; then
        return 0
    fi

    # Check if this is a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        return 0
    fi

    # Stage all changes first (including untracked files)
    git add -A

    # Check if there are any staged changes
    if git diff --cached --quiet; then
        log INFO "No changes to commit in loop #$loop_count"
        return 0
    fi

    # Create commit message
    local commit_msg="morty: Loop #$loop_count - $work_summary

Auto-committed by Morty development loop.

Loop: $loop_count
Timestamp: $(get_iso_timestamp)
Summary: $work_summary

This commit represents the state after loop iteration $loop_count.
You can rollback to this point using: git reset --hard HEAD~N"

    # Commit changes
    if git commit -m "$commit_msg" > /dev/null 2>&1; then
        local commit_hash=$(git rev-parse --short HEAD)
        log SUCCESS "Auto-committed changes: $commit_hash (Loop #$loop_count)"
        return 0
    else
        log WARN "Failed to auto-commit changes in loop #$loop_count"
        return 1
    fi
}

# Rollback to a specific loop
git_rollback() {
    local target_loop=$1

    if ! command -v git &> /dev/null; then
        log ERROR "Git is not installed"
        return 1
    fi

    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log ERROR "Not a git repository"
        return 1
    fi

    # Find the commit for the target loop
    local commit_hash=$(git log --grep="Loop #$target_loop" --format="%H" -n 1)

    if [[ -z "$commit_hash" ]]; then
        log ERROR "No commit found for Loop #$target_loop"
        return 1
    fi

    log INFO "Rolling back to Loop #$target_loop (commit: ${commit_hash:0:8})"

    # Confirm with user
    echo -n "This will reset your working directory. Continue? [y/N] "
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        git reset --hard "$commit_hash"
        log SUCCESS "Rolled back to Loop #$target_loop"
        return 0
    else
        log INFO "Rollback cancelled"
        return 1
    fi
}

# Show loop history from git commits
git_loop_history() {
    if ! command -v git &> /dev/null; then
        log ERROR "Git is not installed"
        return 1
    fi

    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log ERROR "Not a git repository"
        return 1
    fi

    log INFO "Loop History (from git commits):"
    echo ""

    git log --grep="morty: Loop" --format="%C(yellow)%h%C(reset) - %C(green)%ad%C(reset) - %s" --date=relative -20

    echo ""
    log INFO "To rollback: morty rollback <loop-number>"
}

# Check Morty project structure validity
# Returns 0 if valid, 1 if invalid
morty_check_project_structure() {
    local project_dir="${1:-.}"
    local errors=0

    log INFO "Validating Morty project structure: $project_dir"
    echo ""

    # Check if project directory exists
    if [[ ! -d "$project_dir" ]]; then
        log ERROR "Project directory does not exist: $project_dir"
        return 1
    fi

    cd "$project_dir" || return 1

    # Required files
    local required_files=(
        ".morty/specs/problem_description.md"
        ".morty/PROMPT.md"
        ".morty/fix_plan.md"
        ".morty/AGENT.md"
        "README.md"
        ".gitignore"
    )

    # Check required files exist
    log INFO "Checking required files..."
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log ERROR "Missing required file: $file"
            ((errors++))
        else
            # Check if file is not empty
            if [[ ! -s "$file" ]]; then
                log ERROR "File is empty: $file"
                ((errors++))
            else
                log SUCCESS "✓ $file"
            fi
        fi
    done
    echo ""

    # Check required directories
    log INFO "Checking required directories..."
    local required_dirs=(
        ".morty"
        ".morty/specs"
        ".morty/logs"
        "src"
        "tests"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log WARN "Missing directory: $dir (will be created)"
            mkdir -p "$dir"
        else
            log SUCCESS "✓ $dir/"
        fi
    done
    echo ""

    # Validate problem_description.md content
    if [[ -f ".morty/specs/problem_description.md" ]]; then
        log INFO "Validating problem_description.md content..."
        local prd=".morty/specs/problem_description.md"

        # Check for key sections
        local required_sections=(
            "Problem Description"
            "Executive Summary"
            "Goals and Objectives"
            "Functional Requirements"
            "Technical Specifications"
        )

        for section in "${required_sections[@]}"; do
            if grep -qi "$section" "$prd"; then
                log SUCCESS "✓ Contains section: $section"
            else
                log WARN "Missing section: $section"
            fi
        done
        echo ""
    fi

    # Validate fix_plan.md has checkboxes
    if [[ -f ".morty/fix_plan.md" ]]; then
        log INFO "Validating fix_plan.md content..."
        local fix_plan=".morty/fix_plan.md"

        local checkbox_count=$(grep -c "\- \[ \]" "$fix_plan" 2>/dev/null || echo "0")
        if [[ $checkbox_count -gt 0 ]]; then
            log SUCCESS "✓ Contains $checkbox_count unchecked tasks"
        else
            log ERROR "No checkbox tasks found in fix_plan.md"
            ((errors++))
        fi
        echo ""
    fi

    # Validate PROMPT.md has RALPH_STATUS format
    if [[ -f ".morty/PROMPT.md" ]]; then
        log INFO "Validating PROMPT.md content..."
        local prompt=".morty/PROMPT.md"

        if grep -q "RALPH_STATUS" "$prompt"; then
            log SUCCESS "✓ Contains RALPH_STATUS format"
        else
            log WARN "Missing RALPH_STATUS format"
        fi
        echo ""
    fi

    # Validate AGENT.md has build/test commands
    if [[ -f ".morty/AGENT.md" ]]; then
        log INFO "Validating AGENT.md content..."
        local agent=".morty/AGENT.md"

        if grep -qi "build\|setup\|install" "$agent"; then
            log SUCCESS "✓ Contains build/setup commands"
        else
            log WARN "Missing build/setup commands"
        fi

        if grep -qi "test" "$agent"; then
            log SUCCESS "✓ Contains test commands"
        else
            log WARN "Missing test commands"
        fi
        echo ""
    fi

    # Summary
    log INFO "================================"
    if [[ $errors -eq 0 ]]; then
        log SUCCESS "Project structure validation: PASSED ✓"
        log INFO "All required files and structure are present"
        return 0
    else
        log ERROR "Project structure validation: FAILED ✗"
        log ERROR "Found $errors critical error(s)"
        return 1
    fi
}
