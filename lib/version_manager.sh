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
# 初始化
# ============================================

# 模块加载时记录日志（仅调试级别）
if type log_debug &>/dev/null; then
    log_debug "version_manager.sh module loaded"
fi
