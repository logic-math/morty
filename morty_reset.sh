#!/bin/bash
# Morty Reset - 版本回滚功能

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MORTY_HOME="$(dirname "$SCRIPT_DIR")"
source "$MORTY_HOME/lib/common.sh"
source "$MORTY_HOME/lib/git_manager.sh"

# 配置
MORTY_DIR=".morty"

show_help() {
    cat << 'EOF'
Morty Reset - 版本回滚

用法: morty reset [options]

选项:
    -h, --help          显示帮助信息
    -c, --commit <id>   回滚到指定 commit
    -l, --list [N]      显示最近 N 次循环提交(默认: 20)
    -s, --status        显示当前状态

描述:
    reset 命令用于回滚到之前的循环状态。

    当执行 reset 时:
    1. 关闭当前运行的 tmux 会话(如果有)
    2. 使用 git reset --hard 回滚到指定 commit
    3. 保留所有日志文件(.morty/logs/)
    4. 下次运行 loop 时从回滚后的状态继续

    支持人工干预:
    - reset 后可以手动修改代码
    - 再次运行 loop 时会检测到 .morty/ 目录
    - 从当前状态继续执行循环

示例:
    morty reset -l              # 查看循环提交历史
    morty reset -c abc123       # 回滚到 commit abc123
    morty reset -s              # 查看当前状态

EOF
}

# 关闭 tmux 会话
stop_tmux_sessions() {
    # 查找所有 morty-loop 开头的会话
    local sessions=$(tmux list-sessions 2>/dev/null | grep "^morty-loop" | cut -d: -f1 || echo "")

    if [[ -z "$sessions" ]]; then
        log INFO "没有运行中的 tmux 会话"
        return 0
    fi

    log INFO "关闭 tmux 会话..."
    while IFS= read -r session; do
        if [[ -n "$session" ]]; then
            log INFO "  - 关闭会话: $session"
            tmux kill-session -t "$session" 2>/dev/null || true
        fi
    done <<< "$sessions"

    log SUCCESS "✓ Tmux 会话已关闭"
}

# 回滚到指定 commit
reset_to_commit() {
    local commit_id=$1

    if [[ -z "$commit_id" ]]; then
        log ERROR "请指定 commit ID"
        log INFO "使用 'morty reset -l' 查看提交历史"
        return 1
    fi

    # 检查 git 仓库
    if [[ ! -d ".git" ]]; then
        log ERROR "当前目录不是 git 仓库"
        return 1
    fi

    # 验证 commit 是否存在
    if ! git rev-parse --verify "$commit_id^{commit}" &>/dev/null; then
        log ERROR "无效的 commit ID: $commit_id"
        return 1
    fi

    log INFO "╔════════════════════════════════════════════════════════════╗"
    log INFO "║              MORTY RESET - 版本回滚                        ║"
    log INFO "╚════════════════════════════════════════════════════════════╝"
    log INFO ""

    # 显示目标 commit 信息
    log INFO "目标 commit:"
    git log -1 --oneline --format="  %C(yellow)%h%C(reset) - %s" "$commit_id"
    log INFO ""

    # 显示将要丢弃的 commit
    local commits_to_discard=$(git rev-list --count HEAD...$commit_id 2>/dev/null || git rev-list --count $commit_id...HEAD)
    if [[ $commits_to_discard -gt 0 ]]; then
        log WARN "将丢弃 $commits_to_discard 个提交:"
        git log --oneline --format="  %C(red)%h%C(reset) - %s" $commit_id..HEAD | head -10
        if [[ $commits_to_discard -gt 10 ]]; then
            log INFO "  ... 还有 $((commits_to_discard - 10)) 个提交"
        fi
        log INFO ""
    fi

    # 显示未提交的变更
    if has_uncommitted_changes; then
        log WARN "检测到未提交的变更,这些变更将被丢弃:"
        show_uncommitted_changes
    fi

    # 确认操作
    log WARN "⚠️  警告: 此操作将丢弃所有指定 commit 之后的变更!"
    log INFO ""
    read -p "确认回滚? (yes/no): " confirm

    if [[ "$confirm" != "yes" ]]; then
        log INFO "操作已取消"
        return 0
    fi

    log INFO ""
    log INFO "开始回滚..."
    log INFO ""

    # 1. 关闭 tmux 会话
    stop_tmux_sessions
    log INFO ""

    # 2. 执行 git reset
    log INFO "执行 git reset --hard $commit_id..."
    if git reset --hard "$commit_id"; then
        log SUCCESS "✓ 代码已回滚到 commit $commit_id"
    else
        log ERROR "Git reset 失败"
        return 1
    fi

    log INFO ""

    # 3. 清理 git 历史中未跟踪的文件(但保留 .morty/logs/)
    log INFO "清理未跟踪的文件..."
    git clean -fd -e ".morty/logs/*"
    log SUCCESS "✓ 清理完成(日志已保留)"

    log INFO ""
    log INFO "════════════════════════════════════════════════════════════"
    log SUCCESS "回滚完成!"
    log INFO ""
    log INFO "当前状态:"
    git log -1 --oneline --format="  %C(yellow)%h%C(reset) - %s"
    log INFO ""
    log INFO "下一步:"
    log INFO "  1. 检查代码状态: git status"
    log INFO "  2. 可选: 手动修改代码进行干预"
    log INFO "  3. 继续循环: morty loop"
    log INFO ""
}

# 显示当前状态
show_status() {
    log INFO "╔════════════════════════════════════════════════════════════╗"
    log INFO "║              当前状态                                      ║"
    log INFO "╚════════════════════════════════════════════════════════════╝"
    log INFO ""

    # Git 状态
    if [[ ! -d ".git" ]]; then
        log WARN "当前目录不是 git 仓库"
        log INFO ""
        return 0
    fi

    # 当前 commit
    log INFO "当前 commit:"
    git log -1 --oneline --format="  %C(yellow)%h%C(reset) - %C(cyan)%cd%C(reset) - %s" --date=format:"%Y-%m-%d %H:%M"
    log INFO ""

    # 最近的循环提交
    local last_loop=$(get_current_loop_number)
    log INFO "最近的循环: #$last_loop"
    log INFO ""

    # 未提交的变更
    if has_uncommitted_changes; then
        log WARN "有未提交的变更:"
        show_uncommitted_changes
    else
        log SUCCESS "工作区干净,没有未提交的变更"
        log INFO ""
    fi

    # Tmux 会话
    local sessions=$(tmux list-sessions 2>/dev/null | grep "^morty-loop" | cut -d: -f1 || echo "")
    if [[ -n "$sessions" ]]; then
        log INFO "运行中的 tmux 会话:"
        while IFS= read -r session; do
            if [[ -n "$session" ]]; then
                log INFO "  - $session"
            fi
        done <<< "$sessions"
        log INFO ""
    else
        log INFO "没有运行中的 tmux 会话"
        log INFO ""
    fi

    # .morty 目录状态
    if [[ -d "$MORTY_DIR" ]]; then
        log INFO ".morty/ 目录存在"
        log INFO "  - PROMPT.md: $(if [[ -f "$MORTY_DIR/PROMPT.md" ]]; then echo "✓"; else echo "✗"; fi)"
        log INFO "  - fix_plan.md: $(if [[ -f "$MORTY_DIR/fix_plan.md" ]]; then echo "✓"; else echo "✗"; fi)"
        log INFO "  - AGENT.md: $(if [[ -f "$MORTY_DIR/AGENT.md" ]]; then echo "✓"; else echo "✗"; fi)"
        log INFO "  - specs/: $(if [[ -d "$MORTY_DIR/specs" ]]; then echo "✓ ($(find "$MORTY_DIR/specs" -name "*.md" -type f 2>/dev/null | wc -l) 个文件)"; else echo "✗"; fi)"
        log INFO ""
    else
        log WARN ".morty/ 目录不存在"
        log INFO "请先运行 'morty fix prd.md' 初始化项目"
        log INFO ""
    fi
}

# 解析参数
COMMIT_ID=""
SHOW_LIST=false
LIST_LIMIT=20
SHOW_STATUS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -c|--commit)
            COMMIT_ID="$2"
            shift 2
            ;;
        -l|--list)
            SHOW_LIST=true
            if [[ -n "$2" ]] && [[ "$2" =~ ^[0-9]+$ ]]; then
                LIST_LIMIT="$2"
                shift 2
            else
                shift
            fi
            ;;
        -s|--status)
            SHOW_STATUS=true
            shift
            ;;
        *)
            log ERROR "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 执行操作
if [[ "$SHOW_STATUS" == true ]]; then
    show_status
elif [[ "$SHOW_LIST" == true ]]; then
    git_show_loop_history "$LIST_LIMIT"
elif [[ -n "$COMMIT_ID" ]]; then
    reset_to_commit "$COMMIT_ID"
else
    log ERROR "请指定操作"
    log INFO ""
    show_help
    exit 1
fi
