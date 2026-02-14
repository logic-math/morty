#!/bin/bash
# Git 版本管理功能

# 检查并初始化 git 仓库
init_git_if_needed() {
    if [[ ! -d ".git" ]]; then
        log INFO "检测到项目未初始化 git 仓库"
        log INFO "正在初始化 git..."

        git init

        # 创建 .gitignore
        if [[ ! -f ".gitignore" ]]; then
            cat > .gitignore << 'EOF'
# Morty 临时文件
.morty/logs/*.log
.morty/.session_id
.morty/status.json

# 常见临时文件
*.pyc
__pycache__/
node_modules/
.DS_Store
*.swp
*.swo
*~
EOF
            log SUCCESS "✓ 创建 .gitignore"
        fi

        # 初始提交
        git add .
        git commit -m "chore: 初始化 Morty 项目

由 Morty 自动创建的初始提交。

项目结构:
- .morty/PROMPT.md - 开发指令
- .morty/fix_plan.md - 任务计划
- .morty/AGENT.md - 构建命令
- .morty/specs/ - 模块规范

Morty 版本: 0.3.0"

        log SUCCESS "✓ Git 仓库初始化完成"
        log INFO ""
    fi
}

# 创建循环提交
create_loop_commit() {
    local loop_count=$1
    local loop_status=${2:-"completed"}

    # 检查是否有变更
    if git diff --quiet && git diff --cached --quiet; then
        log INFO "循环 #$loop_count: 无代码变更,跳过提交"
        return 0
    fi

    # 暂存所有变更
    git add -A

    # 检查是否有已暂存的变更
    if git diff --cached --quiet; then
        log INFO "循环 #$loop_count: 无变更需要提交"
        return 0
    fi

    # 生成提交信息
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local short_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "initial")

    # 获取变更统计
    local files_changed=$(git diff --cached --numstat | wc -l)
    local insertions=$(git diff --cached --numstat | awk '{sum+=$1} END {print sum+0}')
    local deletions=$(git diff --cached --numstat | awk '{sum+=$2} END {print sum+0}')

    # 获取变更的文件列表
    local changed_files=$(git diff --cached --name-only | head -10)
    local file_count=$(echo "$changed_files" | wc -l)

    # 构建提交信息
    local commit_msg="morty: Loop #$loop_count - $loop_status

自动提交由 Morty 开发循环创建。

循环信息:
- 循环编号: #$loop_count
- 状态: $loop_status
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

    if [[ $file_count -gt 10 ]]; then
        commit_msg="$commit_msg
  ... 还有 $((file_count - 10)) 个文件"
    fi

    commit_msg="$commit_msg

---
此提交代表循环 #$loop_count 的完整状态。
使用 'morty reset -c <commit-id>' 可以回滚到此状态。

Co-Authored-By: Claude Code (Morty Loop)
Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"

    # 创建提交
    if git commit -m "$commit_msg"; then
        local commit_hash=$(git rev-parse --short HEAD)
        log SUCCESS "✓ 循环 #$loop_count 已提交: $commit_hash"
        log INFO "  变更: $files_changed 文件, +$insertions/-$deletions 行"
        return 0
    else
        log ERROR "循环 #$loop_count 提交失败"
        return 1
    fi
}

# 显示循环提交历史
show_loop_history() {
    local limit=${1:-20}

    log INFO "╔════════════════════════════════════════════════════════════╗"
    log INFO "║              循环提交历史                                  ║"
    log INFO "╚════════════════════════════════════════════════════════════╝"
    log INFO ""

    if [[ ! -d ".git" ]]; then
        log ERROR "当前目录不是 git 仓库"
        return 1
    fi

    # 显示 morty loop 相关的提交
    git log --oneline --grep="^morty: Loop" -n "$limit" --format="%C(yellow)%h%C(reset) - %C(cyan)%cd%C(reset) - %s" --date=format:"%Y-%m-%d %H:%M"

    log INFO ""
    log INFO "使用 'morty reset -c <commit-hash>' 回滚到指定循环"
    log INFO "使用 'git show <commit-hash>' 查看提交详情"
}

# 获取当前循环编号
get_current_loop_number() {
    # 从最近的提交中提取循环编号
    local last_loop=$(git log --oneline --grep="^morty: Loop" -n 1 --format="%s" 2>/dev/null | grep -oP 'Loop #\K\d+' || echo "0")
    echo "$last_loop"
}

# 检查是否有未提交的变更
has_uncommitted_changes() {
    if [[ ! -d ".git" ]]; then
        return 1
    fi

    # 检查工作区和暂存区
    if ! git diff --quiet || ! git diff --cached --quiet; then
        return 0  # 有未提交的变更
    fi

    # 检查未跟踪的文件(排除 .morty/logs/)
    local untracked=$(git ls-files --others --exclude-standard | grep -v "^\.morty/logs/")
    if [[ -n "$untracked" ]]; then
        return 0  # 有未跟踪的文件
    fi

    return 1  # 没有未提交的变更
}

# 显示未提交的变更
show_uncommitted_changes() {
    log WARN "检测到未提交的变更:"
    log INFO ""

    # 显示已修改的文件
    local modified=$(git diff --name-only)
    if [[ -n "$modified" ]]; then
        log INFO "已修改的文件:"
        echo "$modified" | while read -r file; do
            log INFO "  M $file"
        done
        log INFO ""
    fi

    # 显示已暂存的文件
    local staged=$(git diff --cached --name-only)
    if [[ -n "$staged" ]]; then
        log INFO "已暂存的文件:"
        echo "$staged" | while read -r file; do
            log INFO "  A $file"
        done
        log INFO ""
    fi

    # 显示未跟踪的文件(排除 .morty/logs/)
    local untracked=$(git ls-files --others --exclude-standard | grep -v "^\.morty/logs/")
    if [[ -n "$untracked" ]]; then
        log INFO "未跟踪的文件:"
        echo "$untracked" | while read -r file; do
            log INFO "  ? $file"
        done
        log INFO ""
    fi
}
