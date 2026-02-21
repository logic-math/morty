#!/usr/bin/env bash
#
# version_manager.sh - 版本控制管理模块
#
# 提供 Git 仓库操作和版本状态查询功能
# 用于 morty 系统的版本管理集成
#

# 防止重复加载
[[ -n "${_VERSION_MANAGER_SH_LOADED:-}" ]] && return 0
_VERSION_MANAGER_SH_LOADED=1

# 获取脚本所在目录
_VERSION_MANAGER_SH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 引入 logging.sh（如果存在）
if [[ -f "${_VERSION_MANAGER_SH_DIR}/logging.sh" ]]; then
    # shellcheck source=./logging.sh
    source "${_VERSION_MANAGER_SH_DIR}/logging.sh"
fi

# ============================================
# 错误码定义
# ============================================

readonly VERSION_ERR_NOT_GIT_REPO=1
readonly VERSION_ERR_GIT_NOT_INSTALLED=2
readonly VERSION_ERR_INIT_FAILED=3
readonly VERSION_ERR_INVALID_PATH=4

# ============================================
# 内部工具函数
# ============================================

# 检查 git 命令是否可用
# 返回: 0 可用, 2 不可用
_version_check_git() {
    if ! command -v git >/dev/null 2>&1; then
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi
    return 0
}

# 检查当前目录是否在 Git 仓库内
# 返回: 0 在仓库内, 1 不在仓库内
_version_is_in_git_repo() {
    _version_check_git || return $VERSION_ERR_GIT_NOT_INSTALLED

    if git rev-parse --git-dir >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

# ============================================
# 公共 API
# ============================================

# 初始化 Git 仓库（如果需要）
# Usage: version_init_if_needed [directory]
# 参数:
#   directory - 可选，要初始化的目录，默认为当前目录
# 返回:
#   0 - 成功（已有仓库或新初始化成功）
#   2 - git 命令不可用
#   3 - 初始化失败
version_init_if_needed() {
    local target_dir="${1:-.}"

    # 检查 git 是否可用
    if ! _version_check_git; then
        if type log_error &>/dev/null; then
            log_error "Git is not installed"
        fi
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 切换到目标目录
    if [[ ! -d "$target_dir" ]]; then
        if type log_error &>/dev/null; then
            log_error "Directory does not exist: $target_dir"
        fi
        return $VERSION_ERR_INVALID_PATH
    fi

    # 保存当前目录
    local original_dir="$PWD"
    cd "$target_dir" || return $VERSION_ERR_INVALID_PATH

    # 检查是否已在 Git 仓库内
    if git rev-parse --git-dir >/dev/null 2>&1; then
        # 已在 Git 仓库内，正常返回
        cd "$original_dir" || true
        return 0
    fi

    # 初始化新的 Git 仓库
    if git init >/dev/null 2>&1; then
        if type log_info &>/dev/null; then
            log_info "Initialized empty Git repository in $(pwd)/.git"
        fi
        cd "$original_dir" || true
        return 0
    else
        if type log_error &>/dev/null; then
            log_error "Failed to initialize Git repository in $(pwd)"
        fi
        cd "$original_dir" || true
        return $VERSION_ERR_INIT_FAILED
    fi
}

# 检查是否有未提交的更改
# Usage: version_has_uncommitted_changes [directory]
# 参数:
#   directory - 可选，要检查的目录，默认为当前目录
# 返回:
#   0 - 有未提交的更改
#   1 - 没有未提交的更改
#   2 - git 命令不可用
#   1 (VERSION_ERR_NOT_GIT_REPO) - 不在 Git 仓库内
version_has_uncommitted_changes() {
    local target_dir="${1:-.}"

    # 检查 git 是否可用
    if ! _version_check_git; then
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 检查是否有未提交的更改
    # 使用 --porcelain 获取机器可读的输出
    local status_output
    status_output=$(git status --porcelain 2>/dev/null)

    # 恢复原始目录
    cd "$original_dir" || true

    # 如果输出不为空，说明有未提交的更改
    if [[ -n "$status_output" ]]; then
        return 0  # true - 有未提交的更改
    else
        return 1  # false - 没有未提交的更改
    fi
}

# 获取 Git 仓库根目录的绝对路径
# Usage: version_get_repo_root [directory]
# 参数:
#   directory - 可选，起始目录，默认为当前目录
# 输出:
#   仓库根目录的绝对路径（stdout）
# 返回:
#   0 - 成功
#   1 (VERSION_ERR_NOT_GIT_REPO) - 不在 Git 仓库内
#   2 - git 命令不可用
version_get_repo_root() {
    local target_dir="${1:-.}"

    # 检查 git 是否可用
    if ! _version_check_git; then
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 获取仓库根目录
    local repo_root
    repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

    # 恢复原始目录
    cd "$original_dir" || true

    # 检查是否成功获取根目录
    if [[ -z "$repo_root" ]]; then
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 输出绝对路径
    echo "$repo_root"
    return 0
}

# 检查路径是否被 Git 忽略
# Usage: version_is_ignored <path>
# 参数:
#   path - 要检查的路径（相对路径或绝对路径）
# 返回:
#   0 - 路径被忽略
#   1 - 路径未被忽略
#   2 - git 命令不可用
#   1 (VERSION_ERR_NOT_GIT_REPO) - 不在 Git 仓库内
#   4 (VERSION_ERR_INVALID_PATH) - 路径参数无效
version_is_ignored() {
    local check_path="$1"

    # 检查参数
    if [[ -z "$check_path" ]]; then
        return $VERSION_ERR_INVALID_PATH
    fi

    # 检查 git 是否可用
    if ! _version_check_git; then
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 确定要检查的目录（path 所在的目录或 path 本身）
    local check_dir
    if [[ -d "$check_path" ]]; then
        check_dir="$check_path"
    else
        check_dir="$(dirname "$check_path")"
    fi

    # 如果目录不存在，尝试使用当前目录
    if [[ ! -d "$check_dir" ]]; then
        check_dir="."
    fi

    # 切换到目标目录（如果需要）
    if [[ "$check_dir" != "." ]]; then
        cd "$check_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 获取文件名部分
    local file_name
    file_name="$(basename "$check_path")"

    # 使用 git check-ignore 检查是否被忽略
    if git check-ignore -q "$file_name" 2>/dev/null; then
        cd "$original_dir" || true
        return 0  # 被忽略
    else
        cd "$original_dir" || true
        return 1  # 未被忽略
    fi
}

# ============================================
# 循环提交功能
# ============================================

# 获取当前循环编号
# 从最新的 morty 提交信息中解析循环编号
# Usage: version_get_current_loop_number [directory]
# 参数:
#   directory - 可选，Git 仓库目录，默认为当前目录
# 输出:
#   当前循环编号（stdout），如果没有 morty 提交则输出 0
# 返回:
#   0 - 成功
#   1 - 不在 Git 仓库内
#   2 - git 命令不可用
version_get_current_loop_number() {
    local target_dir="${1:-.}"

    # 检查 git 是否可用
    if ! _version_check_git; then
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 获取最新的 morty 提交信息
    local latest_commit_msg
    latest_commit_msg=$(git log --grep="^morty\\[" --pretty=format:"%s" -n 1 2>/dev/null)

    # 恢复原始目录
    cd "$original_dir" || true

    # 如果没有找到 morty 提交，返回 0
    if [[ -z "$latest_commit_msg" ]]; then
        echo "0"
        return 0
    fi

    # 解析循环编号，格式: morty[loop:N,status:X]
    local loop_number
    loop_number=$(echo "$latest_commit_msg" | grep -oE 'loop:[0-9]+' | cut -d: -f2)

    if [[ -n "$loop_number" ]]; then
        echo "$loop_number"
    else
        echo "0"
    fi

    return 0
}

# 生成循环提交信息
# 内部函数，用于构建格式化的提交信息
# Usage: _version_generate_commit_message <loop_number> <status> [<extra_info>]
# 参数:
#   loop_number - 循环编号
#   status - 循环状态（running/completed/failed）
#   extra_info - 可选，额外信息（变更统计等）
# 输出:
#   格式化的提交信息（stdout）
_version_generate_commit_message() {
    local loop_number="$1"
    local status="$2"
    local extra_info="${3:-}"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # 构建提交信息头
    local message="morty[loop:${loop_number},status:${status}]"

    # 添加元数据
    message="${message}

Loop Metadata:
- Loop Number: ${loop_number}
- Status: ${status}
- Timestamp: ${timestamp}"

    # 添加额外信息（如果有）
    if [[ -n "$extra_info" ]]; then
        message="${message}

${extra_info}"
    fi

    echo "$message"
}

# 获取变更统计信息
# 内部函数，用于统计变更文件
# Usage: _version_get_change_stats
# 输出:
#   变更统计信息（stdout）
_version_get_change_stats() {
    local stats=""

    # 获取暂存区的变更统计
    local staged_files
    staged_files=$(git diff --cached --name-only 2>/dev/null | wc -l)

    # 获取未暂存的变更统计
    local unstaged_files
    unstaged_files=$(git diff --name-only 2>/dev/null | wc -l)

    # 获取未跟踪的文件统计
    local untracked_files
    untracked_files=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l)

    # 获取详细的文件列表
    local file_list
    file_list=$(git status --porcelain 2>/dev/null)

    stats="Changes:
- Staged files: ${staged_files}
- Unstaged files: ${unstaged_files}
- Untracked files: ${untracked_files}

Files Changed:"

    if [[ -n "$file_list" ]]; then
        # 格式化文件列表，添加缩进
        local formatted_list
        formatted_list=$(echo "$file_list" | sed 's/^/  /')
        stats="${stats}
${formatted_list}"
    else
        stats="${stats}
  (no changes)"
    fi

    echo "$stats"
}

# 自动暂存变更（受 gitignore 限制）
# Usage: _version_stage_changes
# 返回:
#   0 - 成功
#   1 - 无变更可暂存
_version_stage_changes() {
    # 检查是否有变更需要暂存
    local has_changes=false

    # 检查已修改但未暂存的文件
    if git diff --name-only 2>/dev/null | grep -q .; then
        has_changes=true
    fi

    # 检查未跟踪的文件（排除被忽略的文件）
    if git ls-files --others --exclude-standard 2>/dev/null | grep -q .; then
        has_changes=true
    fi

    if [[ "$has_changes" != "true" ]]; then
        return 1
    fi

    # 暂存所有变更（git add 会自动遵守 .gitignore）
    git add -A 2>/dev/null
    return 0
}

# 创建循环提交
# 自动暂存变更、生成提交信息并创建 Git 提交
# Usage: version_create_loop_commit <loop_number> <status> [directory]
# 参数:
#   loop_number - 循环编号
#   status - 循环状态（running/completed/failed）
#   directory - 可选，Git 仓库目录，默认为当前目录
# 输出:
#   提交结果信息（stdout）
# 返回:
#   0 - 提交成功
#   1 - 无变更需要提交
#   2 - git 命令不可用
#   3 - 提交失败
#   4 - 无效参数
version_create_loop_commit() {
    local loop_number="$1"
    local status="$2"
    local target_dir="${3:-.}"

    # 验证参数
    if [[ -z "$loop_number" ]]; then
        if type log_error &>/dev/null; then
            log_error "Loop number is required"
        fi
        echo "Error: Loop number is required" >&2
        return 4
    fi

    if [[ -z "$status" ]]; then
        if type log_error &>/dev/null; then
            log_error "Status is required"
        fi
        echo "Error: Status is required" >&2
        return 4
    fi

    # 验证状态值
    case "$status" in
        running|completed|failed|pending)
            ;;
        *)
            if type log_error &>/dev/null; then
                log_error "Invalid status: $status (must be running, completed, failed, or pending)"
            fi
            echo "Error: Invalid status: $status" >&2
            return 4
            ;;
    esac

    # 检查 git 是否可用
    if ! _version_check_git; then
        if type log_error &>/dev/null; then
            log_error "Git is not installed"
        fi
        echo "Error: Git is not installed" >&2
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            if type log_error &>/dev/null; then
                log_error "Directory does not exist: $target_dir"
            fi
            echo "Error: Directory does not exist: $target_dir" >&2
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        if type log_error &>/dev/null; then
            log_error "Not a git repository: $target_dir"
        fi
        echo "Error: Not a git repository" >&2
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 检查是否有未提交的更改
    if ! version_has_uncommitted_changes; then
        if type log_info &>/dev/null; then
            log_info "No changes to commit"
        fi
        echo "No changes to commit"
        cd "$original_dir" || true
        return 1
    fi

    # 获取变更统计
    local change_stats
    change_stats=$(_version_get_change_stats)

    # 暂存所有变更
    if ! _version_stage_changes; then
        if type log_info &>/dev/null; then
            log_info "No changes to stage"
        fi
        echo "No changes to commit"
        cd "$original_dir" || true
        return 1
    fi

    # 重新检查暂存后是否仍有变更
    if ! git diff --cached --quiet 2>/dev/null; then
        # 有暂存的变更
        :
    else
        if type log_info &>/dev/null; then
            log_info "No changes to commit after staging"
        fi
        echo "No changes to commit"
        cd "$original_dir" || true
        return 1
    fi

    # 生成提交信息
    local commit_message
    commit_message=$(_version_generate_commit_message "$loop_number" "$status" "$change_stats")

    # 创建提交
    local commit_output
    commit_output=$(git commit -m "$commit_message" 2>&1)
    local commit_result=$?

    if [[ $commit_result -eq 0 ]]; then
        # 获取提交哈希
        local commit_hash
        commit_hash=$(git rev-parse --short HEAD 2>/dev/null)

        if type log_info &>/dev/null; then
            log_info "Created loop commit: ${commit_hash}"
        fi

        echo "Created commit: ${commit_hash}"
        echo "$commit_output"
        cd "$original_dir" || true
        return 0
    else
        if type log_error &>/dev/null; then
            log_error "Failed to create commit: $commit_output"
        fi
        echo "Error: Failed to create commit" >&2
        echo "$commit_output" >&2
        cd "$original_dir" || true
        return 3
    fi
}

# ============================================
# 版本回滚与历史查询功能
# ============================================

# 错误码定义（追加）
readonly VERSION_ERR_INVALID_COMMIT=5
readonly VERSION_ERR_RESET_FAILED=6

# 创建回滚前备份分支
# 在回滚前自动创建一个备份分支，以便需要时可以恢复
# Usage: _version_create_backup_branch [directory]
# 参数:
#   directory - 可选，Git 仓库目录，默认为当前目录
# 输出:
#   备份分支名称（stdout）
# 返回:
#   0 - 成功
#   1 - 不在 Git 仓库内
#   2 - git 命令不可用
_version_create_backup_branch() {
    local target_dir="${1:-.}"

    # 检查 git 是否可用
    if ! _version_check_git; then
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 生成备份分支名称：backup/YYYYMMDD-HHMMSS
    local timestamp
    timestamp=$(date +"%Y%m%d-%H%M%S")
    local backup_branch="backup/${timestamp}"

    # 创建备份分支（指向当前 HEAD）
    if git branch "$backup_branch" HEAD 2>/dev/null; then
        if type log_info &>/dev/null; then
            log_info "Created backup branch: ${backup_branch}"
        fi
        echo "$backup_branch"
        cd "$original_dir" || true
        return 0
    else
        if type log_error &>/dev/null; then
            log_error "Failed to create backup branch"
        fi
        cd "$original_dir" || true
        return 1
    fi
}

# 保护 .morty/logs/ 目录（暂存变更以便回滚后恢复）
# Usage: _version_protect_logs_dir [directory]
# 参数:
#   directory - 可选，Git 仓库目录，默认为当前目录
# 返回:
#   0 - 成功
#   1 - 无需保护（目录不存在或无变更）
_version_protect_logs_dir() {
    local target_dir="${1:-.}"

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查 .morty/logs/ 目录是否存在
    local logs_dir=".morty/logs"
    if [[ ! -d "$logs_dir" ]]; then
        cd "$original_dir" || true
        return 1
    fi

    # 检查 logs 目录是否有未提交的变更
    # 使用 git status 检查是否有变更
    local has_changes=false

    # 检查 logs 目录下的所有文件
    if git status --porcelain "$logs_dir" 2>/dev/null | grep -q .; then
        has_changes=true
    fi

    # 如果没有变更，也创建一个临时 stash 来保护当前状态
    # 这样 reset --hard 不会删除这些文件
    if [[ "$has_changes" == "true" ]]; then
        # 暂存 logs 目录的变更
        git add "$logs_dir" 2>/dev/null || true
    fi

    cd "$original_dir" || true
    return 0
}

# 重置到指定 commit
# Usage: version_reset_to_commit <commit_hash> [directory]
# 参数:
#   commit_hash - 目标 commit hash（可以是短 hash）
#   directory - 可选，Git 仓库目录，默认为当前目录
# 输出:
#   重置结果信息（stdout）
# 返回:
#   0 - 重置成功
#   1 - 不在 Git 仓库内
#   2 - git 命令不可用
#   4 - 无效参数
#   5 - 无效的 commit hash
#   6 - 重置失败
version_reset_to_commit() {
    local commit_hash="$1"
    local target_dir="${2:-.}"

    # 验证参数
    if [[ -z "$commit_hash" ]]; then
        if type log_error &>/dev/null; then
            log_error "Commit hash is required"
        fi
        echo "Error: Commit hash is required" >&2
        return 4
    fi

    # 检查 git 是否可用
    if ! _version_check_git; then
        if type log_error &>/dev/null; then
            log_error "Git is not installed"
        fi
        echo "Error: Git is not installed" >&2
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            if type log_error &>/dev/null; then
                log_error "Directory does not exist: $target_dir"
            fi
            echo "Error: Directory does not exist: $target_dir" >&2
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        if type log_error &>/dev/null; then
            log_error "Not a git repository: $target_dir"
        fi
        echo "Error: Not a git repository" >&2
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 验证 commit hash 是否有效
    local full_commit
    full_commit=$(git rev-parse --verify "$commit_hash" 2>/dev/null)
    if [[ -z "$full_commit" ]]; then
        if type log_error &>/dev/null; then
            log_error "Invalid commit hash: $commit_hash"
        fi
        echo "Error: Invalid commit hash: $commit_hash" >&2
        cd "$original_dir" || true
        return $VERSION_ERR_INVALID_COMMIT
    fi

    # 获取当前 commit 信息（用于日志）
    local current_commit
    current_commit=$(git rev-parse --short HEAD 2>/dev/null)

    # 步骤 1: 创建备份分支
    local backup_branch
    backup_branch=$(_version_create_backup_branch ".")
    if [[ $? -ne 0 ]]; then
        if type log_warn &>/dev/null; then
            log_warn "Failed to create backup branch"
        fi
    else
        echo "Created backup branch: $backup_branch"
    fi

    # 步骤 2: 保护 .morty/logs/ 目录
    # 将 logs 目录的内容保存到临时位置
    local logs_backup_needed=false
    local logs_backup_dir=""
    if [[ -d ".morty/logs" ]]; then
        logs_backup_needed=true
        logs_backup_dir=$(mktemp -d -t morty_logs_backup.XXXXXX)
        cp -r .morty/logs/* "$logs_backup_dir/" 2>/dev/null || true
        if type log_info &>/dev/null; then
            log_info "Backed up .morty/logs/ to temporary location"
        fi
    fi

    # 步骤 3: 执行 git reset --hard
    local reset_output
    reset_output=$(git reset --hard "$commit_hash" 2>&1)
    local reset_result=$?

    if [[ $reset_result -ne 0 ]]; then
        if type log_error &>/dev/null; then
            log_error "Failed to reset to commit: $reset_output"
        fi
        echo "Error: Failed to reset to commit $commit_hash" >&2
        echo "$reset_output" >&2
        cd "$original_dir" || true
        return $VERSION_ERR_RESET_FAILED
    fi

    # 步骤 4: 恢复 .morty/logs/ 目录
    if [[ "$logs_backup_needed" == "true" && -n "$logs_backup_dir" && -d "$logs_backup_dir" ]]; then
        # 确保 .morty 目录存在
        mkdir -p .morty/logs
        # 恢复日志文件
        cp -r "$logs_backup_dir/"* .morty/logs/ 2>/dev/null || true
        # 清理临时目录
        rm -rf "$logs_backup_dir"
        if type log_info &>/dev/null; then
            log_info "Restored .morty/logs/ directory"
        fi
        echo "Protected .morty/logs/ directory restored"
    fi

    # 获取新的 HEAD commit
    local new_commit
    new_commit=$(git rev-parse --short HEAD 2>/dev/null)

    if type log_info &>/dev/null; then
        log_info "Reset from ${current_commit} to ${new_commit}"
    fi

    echo "Successfully reset to commit: $new_commit"
    if [[ -n "$backup_branch" ]]; then
        echo "Backup branch created: $backup_branch"
    fi

    cd "$original_dir" || true
    return 0
}

# 显示最近 n 次循环历史
# Usage: version_show_loop_history [n] [directory]
# 参数:
#   n - 可选，显示最近的 n 次循环，默认为 10
#   directory - 可选，Git 仓库目录，默认为当前目录
# 输出:
#   格式化的循环历史（stdout）
# 返回:
#   0 - 成功
#   1 - 不在 Git 仓库内
#   2 - git 命令不可用
version_show_loop_history() {
    local n="${1:-10}"
    local target_dir="${2:-.}"

    # 确保 n 是数字
    if ! [[ "$n" =~ ^[0-9]+$ ]]; then
        n=10
    fi

    # 检查 git 是否可用
    if ! _version_check_git; then
        if type log_error &>/dev/null; then
            log_error "Git is not installed"
        fi
        echo "Error: Git is not installed" >&2
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            if type log_error &>/dev/null; then
                log_error "Directory does not exist: $target_dir"
            fi
            echo "Error: Directory does not exist: $target_dir" >&2
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        if type log_error &>/dev/null; then
            log_error "Not a git repository: $target_dir"
        fi
        echo "Error: Not a git repository" >&2
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 获取 morty 循环提交历史
    local history
    history=$(git log --grep="^morty\[" --pretty=format:"%h|%s|%ci" -n "$n" 2>/dev/null)

    if [[ -z "$history" ]]; then
        echo "No loop history found"
        cd "$original_dir" || true
        return 0
    fi

    # 格式化输出
    echo "=== Loop History (last $n) ==="
    echo ""

    local count=1
    while IFS='|' read -r hash subject date; do
        # 解析循环编号和状态
        local loop_num=$(echo "$subject" | grep -oE 'loop:[0-9]+' | cut -d: -f2)
        local status=$(echo "$subject" | grep -oE 'status:[a-z]+' | cut -d: -f2)

        printf "%2d. [%s] Loop %s - %s (%s)\n" "$count" "$hash" "$loop_num" "$status" "$date"
        count=$((count + 1))
    done <<< "$history"

    cd "$original_dir" || true
    return 0
}

# 获取指定编号的循环提交信息
# Usage: version_get_loop_by_number <n> [directory]
# 参数:
#   n - 循环编号
#   directory - 可选，Git 仓库目录，默认为当前目录
# 输出:
#   循环提交的 commit hash（stdout）
# 返回:
#   0 - 成功
#   1 - 不在 Git 仓库内
#   2 - git 命令不可用
#   4 - 无效参数
#   5 - 未找到指定编号的循环
version_get_loop_by_number() {
    local loop_number="$1"
    local target_dir="${2:-.}"

    # 验证参数
    if [[ -z "$loop_number" ]]; then
        if type log_error &>/dev/null; then
            log_error "Loop number is required"
        fi
        echo "Error: Loop number is required" >&2
        return 4
    fi

    # 确保 loop_number 是数字
    if ! [[ "$loop_number" =~ ^[0-9]+$ ]]; then
        if type log_error &>/dev/null; then
            log_error "Invalid loop number: $loop_number"
        fi
        echo "Error: Invalid loop number: $loop_number" >&2
        return 4
    fi

    # 检查 git 是否可用
    if ! _version_check_git; then
        if type log_error &>/dev/null; then
            log_error "Git is not installed"
        fi
        echo "Error: Git is not installed" >&2
        return $VERSION_ERR_GIT_NOT_INSTALLED
    fi

    # 保存当前目录
    local original_dir="$PWD"

    # 如果指定了目录，切换到该目录
    if [[ "$target_dir" != "." ]]; then
        if [[ ! -d "$target_dir" ]]; then
            if type log_error &>/dev/null; then
                log_error "Directory does not exist: $target_dir"
            fi
            echo "Error: Directory does not exist: $target_dir" >&2
            return $VERSION_ERR_INVALID_PATH
        fi
        cd "$target_dir" || return $VERSION_ERR_INVALID_PATH
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        if type log_error &>/dev/null; then
            log_error "Not a git repository: $target_dir"
        fi
        echo "Error: Not a git repository" >&2
        cd "$original_dir" || true
        return $VERSION_ERR_NOT_GIT_REPO
    fi

    # 搜索指定循环编号的提交
    local commit_hash
    commit_hash=$(git log --grep="morty\[loop:${loop_number}," --pretty=format:"%h" -n 1 2>/dev/null)

    if [[ -z "$commit_hash" ]]; then
        if type log_error &>/dev/null; then
            log_error "Loop $loop_number not found"
        fi
        echo "Error: Loop $loop_number not found" >&2
        cd "$original_dir" || true
        return 5
    fi

    echo "$commit_hash"

    cd "$original_dir" || true
    return 0
}

# ============================================
# 初始化
# ============================================

# 模块加载时记录日志（仅调试级别）
if type log_debug &>/dev/null; then
    log_debug "version_manager.sh module loaded"
fi
