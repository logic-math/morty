#!/bin/bash
# Morty Loop Monitor - tmux 集成监控
# 在 tmux 中启动循环并提供三面板监控

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Morty 系统目录(用于生成的 runner 脚本)
MORTY_LIB_DIR="${MORTY_LIB_DIR:-$HOME/.morty/lib}"

# 配置
MORTY_DIR=".morty"
LOG_DIR="$MORTY_DIR/logs"
STATUS_FILE="$MORTY_DIR/status.json"
FIX_PLAN_FILE="$MORTY_DIR/fix_plan.md"

# 默认参数
MAX_LOOPS="${MAX_LOOPS:-50}"
LOOP_DELAY="${LOOP_DELAY:-5}"

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --max-loops)
            MAX_LOOPS="$2"
            shift 2
            ;;
        --delay)
            LOOP_DELAY="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

# 获取 tmux base-index
get_tmux_base_index() {
    local base_index
    base_index=$(tmux show-options -gv base-index 2>/dev/null || echo "0")
    echo "${base_index:-0}"
}

# 创建 tmux 会话
SESSION_NAME="morty-loop-$(date +%s)"
PROJECT_DIR="$(pwd)"
BASE_WIN=$(get_tmux_base_index)

log INFO "创建 tmux 监控会话: $SESSION_NAME"
log INFO "项目目录: $PROJECT_DIR"
log INFO ""

# 创建新的 tmux 会话(分离模式)
tmux new-session -d -s "$SESSION_NAME" -c "$PROJECT_DIR"

# 水平分割窗口(左: 循环日志, 右: 监控和终端)
tmux split-window -h -t "$SESSION_NAME" -c "$PROJECT_DIR"

# 垂直分割右侧面板(上: Claude 监控 30%, 下: 交互终端 70%)
tmux split-window -v -t "$SESSION_NAME:${BASE_WIN}.1" -c "$PROJECT_DIR"

# 设置面板大小
# 左侧循环日志 50% 宽度
tmux resize-pane -t "$SESSION_NAME:${BASE_WIN}.0" -x 50%
# 右上 Claude 监控 30% 高度
tmux resize-pane -t "$SESSION_NAME:${BASE_WIN}.1" -y 30%

# 确保日志目录存在并清空历史日志
log INFO "清空历史日志..."
rm -rf "$LOG_DIR"/*
mkdir -p "$LOG_DIR"
log SUCCESS "✓ 日志目录已清空"

# 左侧面板(面板 0): 直接执行 morty_loop.sh --no-monitor
log INFO "配置左侧面板: 循环实时日志"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '║              循环实时日志 - 项目进度                       ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '启动循环...'" Enter

# 等待一下让其他面板初始化，然后启动 morty_loop.sh --no-monitor
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "sleep 2" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "$SCRIPT_DIR/../morty_loop.sh --no-monitor --max-loops $MAX_LOOPS --delay $LOOP_DELAY 2>&1 | tee .morty/logs/loop_runner.log" Enter

# 右上面板(面板 1): 交互式终端 (30% 高度)
log INFO "配置右上面板: 交互式终端"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "clear" Enter

# 交互式终端初始化脚本
TERMINAL_INIT_SCRIPT=$(cat << 'SCRIPT'
clear
echo "╔════════════════════════════════════════════════════════════╗"
echo "║              交互式终端 (30%)                              ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# 定义便捷命令
show_status() {
    echo ""
    echo "📊 当前状态:"
    if [[ -f ".morty/status.json" ]]; then
        cat ".morty/status.json" | jq -r '
            "  状态: \(.state)",
            "  循环: \(.loop_count) / \(.max_loops)",
            "  消息: \(.message)",
            "  时间: \(.timestamp)"
        ' 2>/dev/null || cat ".morty/status.json"
    else
        echo "  等待状态文件..."
    fi
    echo ""
}

show_progress() {
    echo ""
    echo "📝 任务进度:"
    if [[ -f ".morty/fix_plan.md" ]]; then
        total=$(grep -c "\- \[" ".morty/fix_plan.md" 2>/dev/null || echo "0")
        completed=$(grep -c "\- \[x\]" ".morty/fix_plan.md" 2>/dev/null || echo "0")
        pending=$(grep -c "\- \[ \]" ".morty/fix_plan.md" 2>/dev/null || echo "0")
        echo "  总任务: $total"
        echo "  已完成: $completed"
        echo "  待完成: $pending"

        if [[ $total -gt 0 ]]; then
            progress=$((completed * 100 / total))
            echo "  进度: $progress%"
        fi
    else
        echo "  无任务文件"
    fi
    echo ""
}

show_logs() {
    echo ""
    echo "📋 最新日志 (最后 30 行):"
    tail -30 .morty/logs/loop_*_output.log 2>/dev/null | tail -30
    echo ""
}

show_plan() {
    echo ""
    echo "📋 任务计划:"
    cat .morty/fix_plan.md 2>/dev/null || echo "  无任务文件"
    echo ""
}

show_help() {
    echo ""
    echo "═══════════════════════════════════════════════════════════"
    echo "便捷命令:"
    echo "  status       - 显示循环状态"
    echo "  progress     - 显示任务进度"
    echo "  logs         - 查看最新日志"
    echo "  plan         - 查看任务计划"
    echo "  help         - 显示此帮助"
    echo ""
    echo "快捷键:"
    echo "  Ctrl+B [     进入滚动模式(查看历史)"
    echo "  Ctrl+B 方向键 切换面板"
    echo "  Ctrl+B D     分离会话(后台运行)"
    echo "  Ctrl+B X     关闭当前面板"
    echo "═══════════════════════════════════════════════════════════"
    echo ""
}

# 注册别名
alias status='show_status'
alias progress='show_progress'
alias logs='show_logs'
alias plan='show_plan'
alias help='show_help'

# 显示欢迎信息
echo "💡 可用命令: status, progress, logs, plan, help"
echo "💡 或直接使用 bash 命令"
echo ""
SCRIPT
)

tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "$TERMINAL_INIT_SCRIPT" Enter

# 右下面板(面板 2): 直接进入 morty fix 模式 (70% 高度)
log INFO "配置右下面板: Fix 模式终端"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "clear" Enter

# Fix 模式终端初始化脚本
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '║              Fix 模式终端 (70%)                            ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo ''" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '💡 已进入 morty fix 模式，自动读取当前项目进展'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '💡 可随时对话，帮助理解项目进展并做出干预'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo ''" Enter

# 直接进入 morty fix 模式（不带参数，自动读取 .morty 目录）
# 设置环境变量告诉 morty_fix.sh 这是从 loop 监控调用的
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "MORTY_LOOP_MONITOR=1 $SCRIPT_DIR/../morty_fix.sh" Enter

# 设置面板标题
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.0" -T "循环日志"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.1" -T "交互终端(30%)"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2" -T "Fix模式(70%)"

# 聚焦到右下面板(Fix 模式终端)
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2"

log SUCCESS "tmux 会话已创建!"
log INFO ""
log INFO "会话名称: $SESSION_NAME"
log INFO ""
log INFO "面板布局:"
log INFO "  ┌──────────────────┬───────────────┐"
log INFO "  │                  │ 交互终端      │"
log INFO "  │  循环日志        │ (30%)         │"
log INFO "  │  (满屏)          ├───────────────┤"
log INFO "  │                  │ Fix模式(70%)  │"
log INFO "  └──────────────────┴───────────────┘"
log INFO ""
log INFO "左侧面板:"
log INFO "  • 循环日志(100%): Claude 实时输出日志"
log INFO ""
log INFO "右侧面板:"
log INFO "  • 右上(30%): 交互式终端 - 查看状态/进度/日志/计划 (status/progress/logs/plan)"
log INFO "  • 右下(70%): Fix 模式终端 - 运行 'morty fix' 进行交互式干预"
log INFO ""
log INFO "快捷键:"
log INFO "  Ctrl+B 然后 [        进入滚动模式(查看历史)"
log INFO "  Ctrl+B 然后 方向键    切换面板"
log INFO "  Ctrl+B 然后 D        分离会话(后台运行)"
log INFO "  Ctrl+B 然后 X        关闭当前面板"
log INFO ""
log INFO "重新连接会话:"
log INFO "  tmux attach -t $SESSION_NAME"
log INFO ""
log INFO "正在连接到会话..."
sleep 1

# 连接到会话(检测是否已在 tmux 中)
if [[ -n "$TMUX" ]]; then
    # 已在 tmux 会话中，切换到新会话
    tmux switch-client -t "$SESSION_NAME"
else
    # 不在 tmux 中，正常 attach
    tmux attach -t "$SESSION_NAME"
fi
