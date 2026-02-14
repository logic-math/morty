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

    log INFO "验证 Morty 项目结构: $project_dir"
    echo ""

    # Check if project directory exists
    if [[ ! -d "$project_dir" ]]; then
        log ERROR "项目目录不存在: $project_dir"
        return 1
    fi

    cd "$project_dir" || return 1

    # Required files
    local required_files=(
        ".morty/PROMPT.md"
        ".morty/fix_plan.md"
        ".morty/AGENT.md"
    )

    # Check required files exist
    log INFO "检查必需文件..."
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log ERROR "缺少必需文件: $file"
            ((errors++))
        else
            # Check if file is not empty
            if [[ ! -s "$file" ]]; then
                log ERROR "文件为空: $file"
                ((errors++))
            else
                log SUCCESS "✓ $file"
            fi
        fi
    done
    echo ""

    # Check required directories
    log INFO "检查必需目录..."
    local required_dirs=(
        ".morty"
        ".morty/specs"
        ".morty/logs"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log ERROR "缺少必需目录: $dir"
            ((errors++))
        else
            log SUCCESS "✓ $dir/"
        fi
    done
    echo ""

    # Check specs/ directory has at least one module file
    log INFO "检查 specs/ 目录内容..."
    local specs_count=$(find ".morty/specs" -name "*.md" -type f 2>/dev/null | wc -l)
    if [[ $specs_count -eq 0 ]]; then
        log ERROR "specs/ 目录中没有模块规范文件(.md)"
        ((errors++))
    else
        log SUCCESS "✓ 找到 $specs_count 个模块规范文件"
        # List the module files
        find ".morty/specs" -name "*.md" -type f -exec basename {} \; | while read -r spec_file; do
            log INFO "  - $spec_file"
        done
    fi
    echo ""

    # Validate module specs content (sample check on first spec file)
    local first_spec=$(find ".morty/specs" -name "*.md" -type f 2>/dev/null | head -1)
    if [[ -n "$first_spec" ]]; then
        log INFO "验证模块规范内容(示例: $(basename "$first_spec"))..."

        # Check for key sections in module spec
        local has_purpose=$(grep -qi "目的\|Purpose" "$first_spec" && echo "yes" || echo "no")
        local has_scope=$(grep -qi "范围\|Scope" "$first_spec" && echo "yes" || echo "no")
        local has_tech=$(grep -qi "技术\|Technical" "$first_spec" && echo "yes" || echo "no")

        if [[ "$has_purpose" == "yes" ]]; then
            log SUCCESS "✓ 包含模块目的说明"
        else
            log WARN "建议添加模块目的说明"
        fi

        if [[ "$has_scope" == "yes" ]]; then
            log SUCCESS "✓ 包含范围定义"
        else
            log WARN "建议添加范围定义"
        fi

        if [[ "$has_tech" == "yes" ]]; then
            log SUCCESS "✓ 包含技术规范"
        else
            log WARN "建议添加技术规范"
        fi
        echo ""
    fi

    # Validate fix_plan.md has checkboxes
    if [[ -f ".morty/fix_plan.md" ]]; then
        log INFO "验证 fix_plan.md 内容..."
        local fix_plan=".morty/fix_plan.md"

        local checkbox_count=$(grep -c "\- \[ \]" "$fix_plan" 2>/dev/null || echo "0")
        if [[ $checkbox_count -gt 0 ]]; then
            log SUCCESS "✓ 包含 $checkbox_count 个未完成任务"
        else
            log ERROR "fix_plan.md 中没有找到复选框任务"
            ((errors++))
        fi
        echo ""
    fi

    # Validate PROMPT.md has RALPH_STATUS format
    if [[ -f ".morty/PROMPT.md" ]]; then
        log INFO "验证 PROMPT.md 内容..."
        local prompt=".morty/PROMPT.md"

        if grep -q "RALPH_STATUS" "$prompt"; then
            log SUCCESS "✓ 包含 RALPH_STATUS 格式"
        else
            log WARN "缺少 RALPH_STATUS 格式"
        fi
        echo ""
    fi

    # Validate AGENT.md has build/test commands
    if [[ -f ".morty/AGENT.md" ]]; then
        log INFO "验证 AGENT.md 内容..."
        local agent=".morty/AGENT.md"

        if grep -qi "build\|setup\|install\|构建\|安装" "$agent"; then
            log SUCCESS "✓ 包含构建/安装命令"
        else
            log WARN "缺少构建/安装命令"
        fi

        if grep -qi "test\|测试" "$agent"; then
            log SUCCESS "✓ 包含测试命令"
        else
            log WARN "缺少测试命令"
        fi
        echo ""
    fi

    # Summary
    log INFO "================================"
    if [[ $errors -eq 0 ]]; then
        log SUCCESS "项目结构验证: 通过 ✓"
        log INFO "所有必需文件和结构都存在"
        return 0
    else
        log ERROR "项目结构验证: 失败 ✗"
        log ERROR "发现 $errors 个关键错误"
        return 1
    fi
}
