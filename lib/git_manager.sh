#!/bin/bash
# Git Manager - Git 版本管理模块
# 提供仓库管理、循环提交、历史查询和版本回滚功能

# 依赖: logging 模块 (log 函数)

# ============================================
# 仓库管理
# ============================================

# 检查并初始化 Git 仓库
# 如果在非 Git 目录调用，会初始化新的仓库
# 如果在已有 Git 仓库调用，正常返回
# Returns: 0 成功, 1 失败
git_init_if_needed() {
    # 检查当前目录是否在 Git 仓库内
    if git rev-parse --git-dir > /dev/null 2>&1; then
        # 已经在 Git 仓库内
        return 0
    fi

    # 未初始化，创建新仓库
    if [[ -n "${LOGGING_LOADED:-}" ]]; then
        log INFO "初始化 Git 仓库..."
    fi

    if ! git init > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "Git 初始化失败"
        fi
        return 1
    fi

    # 配置默认用户信息（如果未设置）
    if ! git config user.email > /dev/null 2>&1; then
        git config user.email "morty@localhost"
    fi
    if ! git config user.name > /dev/null 2>&1; then
        git config user.name "Morty"
    fi

    if [[ -n "${LOGGING_LOADED:-}" ]]; then
        log SUCCESS "✓ Git 仓库初始化完成"
    fi
    return 0
}

# 检查是否有未提交的变更
# Returns: 0 有未提交变更, 1 没有未提交变更
git_has_uncommitted_changes() {
    # 首先检查是否在 Git 仓库内
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        return 1
    fi

    # 检查工作区和暂存区的变更
    if ! git diff --quiet || ! git diff --cached --quiet; then
        return 0
    fi

    # 检查未跟踪的文件（排除 .morty/logs/）
    local untracked
    untracked=$(git ls-files --others --exclude-standard | grep -v "^\.morty/logs/" || true)
    if [[ -n "$untracked" ]]; then
        return 0
    fi

    return 1
}

# 获取仓库根目录绝对路径
# 如果在仓库内，输出根目录路径并返回 0
# 如果不在仓库内，返回 1
git_get_repo_root() {
    local root
    if ! root=$(git rev-parse --show-toplevel 2>/dev/null); then
        return 1
    fi
    echo "$root"
    return 0
}

# 检查路径是否被 gitignore 忽略
# Args: $1 - 要检查的路径
# Returns: 0 被忽略, 1 未被忽略, 2 错误（不在仓库内等）
git_is_ignored() {
    local path="${1:-}"

    if [[ -z "$path" ]]; then
        return 2
    fi

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        return 2
    fi

    # 使用 git check-ignore 检查
    if git check-ignore -q "$path" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# ============================================
# 循环提交
# ============================================

# 创建循环提交
# Args:
#   $1 - 循环编号
#   $2 - 状态 (completed/failed)
#   $3 - 额外消息（可选）
# Returns: 0 成功, 1 失败
git_create_loop_commit() {
    local loop_number="${1:-}"
    local status="${2:-completed}"
    local extra_msg="${3:-}"

    if [[ -z "$loop_number" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少循环编号"
        fi
        return 1
    fi

    # 检查是否有变更
    if ! git_has_uncommitted_changes; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log INFO "循环 #$loop_number: 无代码变更，跳过提交"
        fi
        return 0
    fi

    # 暂存所有变更
    git add -A

    # 获取变更统计
    local files_changed insertions deletions
    files_changed=$(git diff --cached --numstat | wc -l)
    insertions=$(git diff --cached --numstat | awk '{sum+=$1} END {print sum+0}')
    deletions=$(git diff --cached --numstat | awk '{sum+=$2} END {print sum+0}')

    # 获取变更的文件列表
    local changed_files
    changed_files=$(git diff --cached --name-only | head -10)

    # 构建提交信息
    local timestamp short_hash commit_msg
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    short_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "initial")

    commit_msg="morty: Loop #$loop_number - $status

自动提交由 Morty 开发循环创建。"

    if [[ -n "$extra_msg" ]]; then
        commit_msg="$commit_msg

$extra_msg"
    fi

    commit_msg="$commit_msg

循环信息:
- 循环编号: #$loop_number
- 状态: $status
- 时间戳: $timestamp
- 父提交: $short_hash

变更统计:
- 文件数: $files_changed
- 新增行: +$insertions
- 删除行: -$deletions

变更文件:"

    # 添加文件列表
    while IFS= read -r file; do
        if [[ -n "$file" ]]; then
            commit_msg="$commit_msg
  - $file"
        fi
    done <<< "$changed_files"

    local total_files
    total_files=$(git diff --cached --name-only | wc -l)
    if [[ $total_files -gt 10 ]]; then
        commit_msg="$commit_msg
  ... 还有 $((total_files - 10)) 个文件"
    fi

    commit_msg="$commit_msg

---
此提交代表循环 #$loop_number 的完整状态。
使用 'morty reset -c <commit-id>' 可以回滚到此状态。"

    # 创建提交
    if git commit -m "$commit_msg" > /dev/null 2>&1; then
        local commit_hash
        commit_hash=$(git rev-parse --short HEAD)
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 循环 #$loop_number 已提交: $commit_hash"
            log INFO "  变更: $files_changed 文件, +$insertions/-$deletions 行"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "循环 #$loop_number 提交失败"
        fi
        return 1
    fi
}

# 获取当前循环编号
# 从最近的提交历史中解析
# 输出: 当前循环编号（如果没有则为 0）
git_get_current_loop_number() {
    local last_loop
    last_loop=$(git log --oneline --grep="^morty: Loop" -n 1 --format="%s" 2>/dev/null | grep -oP 'Loop #\K[0-9]+' || echo "0")
    echo "$last_loop"
}

# 获取指定循环的提交 hash
# Args: $1 - 循环编号
# 输出: 提交 hash（如果未找到则为空）
git_get_last_loop_commit() {
    local loop_number="${1:-}"
    if [[ -z "$loop_number" ]]; then
        return 1
    fi

    local commit_hash
    commit_hash=$(git log --oneline --grep="morty: Loop #$loop_number " --format="%H" -n 1 2>/dev/null)
    echo "$commit_hash"
}

# ============================================
# 历史查询
# ============================================

# 显示循环提交历史
# Args: $1 - 显示数量（默认 10）
git_show_loop_history() {
    local n="${1:-10}"

    # 检查是否在 Git 仓库内
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "当前目录不是 Git 仓库"
        fi
        return 1
    fi

    if [[ -n "${LOGGING_LOADED:-}" ]]; then
        log INFO "════════════════════════════════════════════════════════════"
        log INFO "              循环提交历史"
        log INFO "════════════════════════════════════════════════════════════"
        log INFO ""
    fi

    # 显示 morty loop 相关的提交
    git log --oneline --grep="^morty: Loop" -n "$n" --format="%C(yellow)%h%C(reset) - %C(cyan)%cd%C(reset) - %s" --date=format:"%Y-%m-%d %H:%M"

    if [[ -n "${LOGGING_LOADED:-}" ]]; then
        log INFO ""
        log INFO "使用 'morty reset -c <commit-hash>' 回滚到指定循环"
    fi
}

# 获取指定循环的提交信息
# Args: $1 - 循环编号
# 输出: 提交信息的 JSON 格式（或空）
git_get_loop_by_number() {
    local loop_number="${1:-}"
    if [[ -z "$loop_number" ]]; then
        return 1
    fi

    local commit_hash
    commit_hash=$(git log --oneline --grep="morty: Loop #$loop_number " --format="%H" -n 1 2>/dev/null)

    if [[ -z "$commit_hash" ]]; then
        return 0
    fi

    git_parse_loop_commit "$commit_hash"
}

# 解析循环提交信息
# Args: $1 - 提交 hash
# 输出: JSON 格式的循环信息
git_parse_loop_commit() {
    local commit_hash="${1:-}"
    if [[ -z "$commit_hash" ]]; then
        return 1
    fi

    local message timestamp author
    message=$(git log -1 --format="%B" "$commit_hash" 2>/dev/null)
    timestamp=$(git log -1 --format="%aI" "$commit_hash" 2>/dev/null)
    author=$(git log -1 --format="%an" "$commit_hash" 2>/dev/null)

    # 解析循环编号和状态
    local loop_number status
    loop_number=$(echo "$message" | grep -oP '循环编号: #\K[0-9]+' | head -1)
    status=$(echo "$message" | grep -oP '状态: \K\w+' | head -1)

    # 解析统计信息
    local files_changed insertions deletions
    files_changed=$(echo "$message" | grep -oP '文件数: \K[0-9]+' | head -1)
    insertions=$(echo "$message" | grep -oP '新增行: \+\K[0-9]+' | head -1)
    deletions=$(echo "$message" | grep -oP '删除行: -\K[0-9]+' | head -1)

    # 输出 JSON
    cat << EOF
{
  "commit": "$commit_hash",
  "number": ${loop_number:-0},
  "status": "${status:-unknown}",
  "timestamp": "$timestamp",
  "author": "$author",
  "files_changed": ${files_changed:-0},
  "insertions": ${insertions:-0},
  "deletions": ${deletions:-0}
}
EOF
}

# ============================================
# 版本回滚
# ============================================

# 回滚到指定提交
# Args:
#   $1 - 提交 ID
#   $2 - 是否创建备份（默认 true）
# Returns: 0 成功, 1 失败
git_reset_to_commit() {
    local commit_id="${1:-}"
    local backup="${2:-true}"

    if [[ -z "$commit_id" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少提交 ID"
        fi
        return 1
    fi

    # 验证提交是否存在
    if ! git rev-parse --verify "$commit_id"^{commit} > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "无效的提交 ID: $commit_id"
        fi
        return 1
    fi

    # 创建备份
    if [[ "$backup" == "true" ]]; then
        if ! git_create_backup_branch; then
            if [[ -n "${LOGGING_LOADED:-}" ]]; then
                log WARN "备份分支创建失败，继续回滚"
            fi
        fi
    fi

    # 执行回滚
    if git reset --hard "$commit_id" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 已回滚到提交: ${commit_id:0:8}"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "回滚失败"
        fi
        return 1
    fi
}

# 回滚到指定循环
# Args:
#   $1 - 循环编号
#   $2 - 是否创建备份（默认 true）
# Returns: 0 成功, 1 失败
git_reset_to_loop() {
    local loop_number="${1:-}"
    local backup="${2:-true}"

    if [[ -z "$loop_number" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少循环编号"
        fi
        return 1
    fi

    local commit_hash
    commit_hash=$(git_get_last_loop_commit "$loop_number")

    if [[ -z "$commit_hash" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "未找到循环 #$loop_number 的提交"
        fi
        return 1
    fi

    git_reset_to_commit "$commit_hash" "$backup"
}

# 创建备份分支
# Returns: 0 成功, 1 失败
git_create_backup_branch() {
    local timestamp branch_name
    timestamp=$(date +"%Y%m%d-%H%M%S")
    branch_name="morty-backup-$timestamp"

    if git branch "$branch_name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log INFO "已创建备份分支: $branch_name"
        fi
        return 0
    else
        return 1
    fi
}

# 从备份分支恢复
# Args: $1 - 分支名称
git_restore_from_backup() {
    local branch_name="${1:-}"

    if [[ -z "$branch_name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少分支名称"
        fi
        return 1
    fi

    # 检查分支是否存在
    if ! git rev-parse --verify "$branch_name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "分支不存在: $branch_name"
        fi
        return 1
    fi

    # 重置到备份分支
    if git reset --hard "$branch_name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 已从备份 $branch_name 恢复"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "恢复失败"
        fi
        return 1
    fi
}

# ============================================
# 里程碑管理
# ============================================

# 创建里程碑标签
# Args:
#   $1 - 里程碑名称
#   $2 - 描述（可选）
git_create_milestone() {
    local name="${1:-}"
    local description="${2:-}"

    if [[ -z "$name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少里程碑名称"
        fi
        return 1
    fi

    local loop_number
    loop_number=$(git_get_current_loop_number)

    local tag_message
    tag_message="Milestone: $name

创建时间: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
关联循环: #$loop_number
描述: ${description:-无}"

    if git tag -a "$name" -m "$tag_message" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 创建里程碑: $name"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "创建里程碑失败: $name"
        fi
        return 1
    fi
}

# 列出所有里程碑
git_list_milestones() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        return 1
    fi

    git tag -l -n1 | grep -E "^Milestone:" || true
}

# 切换到里程碑状态
# Args: $1 - 里程碑名称
git_checkout_milestone() {
    local name="${1:-}"

    if [[ -z "$name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少里程碑名称"
        fi
        return 1
    fi

    # 检查标签是否存在
    if ! git rev-parse --verify "$name"^{tag} > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "里程碑不存在: $name"
        fi
        return 1
    fi

    # 创建备份后切换到标签
    git_create_backup_branch

    if git checkout "$name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 已切换到里程碑: $name"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "切换失败"
        fi
        return 1
    fi
}

# 删除里程碑
# Args: $1 - 里程碑名称
git_delete_milestone() {
    local name="${1:-}"

    if [[ -z "$name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少里程碑名称"
        fi
        return 1
    fi

    if git tag -d "$name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 删除里程碑: $name"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "删除里程碑失败: $name"
        fi
        return 1
    fi
}

# ============================================
# 实验分支
# ============================================

# 创建实验分支
# Args: $1 - 实验名称
git_create_experiment() {
    local name="${1:-}"

    if [[ -z "$name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少实验名称"
        fi
        return 1
    fi

    local branch_name="experiment/$name"

    if git checkout -b "$branch_name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 创建实验分支: $branch_name"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "创建实验分支失败: $branch_name"
        fi
        return 1
    fi
}

# 列出所有实验分支
git_list_experiments() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        return 1
    fi

    git branch -a | grep "experiment/" | sed 's/^[* ]*//' || true
}

# 合并实验分支到当前分支
# Args: $1 - 实验名称
git_merge_experiment() {
    local name="${1:-}"

    if [[ -z "$name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少实验名称"
        fi
        return 1
    fi

    local branch_name="experiment/$name"

    if git merge --no-ff "$branch_name" -m "合并实验分支: $name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 合并实验分支: $branch_name"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "合并实验分支失败，请解决冲突"
        fi
        return 1
    fi
}

# 删除实验分支
# Args: $1 - 实验名称
git_delete_experiment() {
    local name="${1:-}"

    if [[ -z "$name" ]]; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "缺少实验名称"
        fi
        return 1
    fi

    local branch_name="experiment/$name"

    if git branch -D "$branch_name" > /dev/null 2>&1; then
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log SUCCESS "✓ 删除实验分支: $branch_name"
        fi
        return 0
    else
        if [[ -n "${LOGGING_LOADED:-}" ]]; then
            log ERROR "删除实验分支失败: $branch_name"
        fi
        return 1
    fi
}
