#!/bin/bash
# Morty Obs - 可观察性监控面板

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# 配置
MORTY_DIR=".morty"
STATUS_FILE="$MORTY_DIR/status.json"
LOG_DIR="$MORTY_DIR/logs"

show_help() {
    cat << 'EOF'
Morty Obs - 可观察性监控

用法: morty obs

描述:
    使用 tmux 创建一个三面板监控界面:
    - 左侧: Loop 循环执行
    - 右上: 实时日志
    - 右下: 状态监控

要求:
    - 必须安装 tmux
    - 必须在项目目录中(有 .morty/ 目录)

快捷键:
    Ctrl+B 然后 [    进入滚动模式(查看历史)
    Ctrl+B 然后 方向键  切换面板
    Ctrl+B 然后 D     分离会话
    Ctrl+B 然后 X     关闭当前面板

EOF
}

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log ERROR "未知参数: $1"
            exit 1
            ;;
    esac
done

# 检查 tmux 是否安装
if ! command -v tmux &> /dev/null; then
    log ERROR "tmux 未安装"
    log INFO ""
    log INFO "安装 tmux:"
    log INFO "  Ubuntu/Debian: sudo apt-get install tmux"
    log INFO "  CentOS/RHEL: sudo yum install tmux"
    log INFO "  macOS: brew install tmux"
    exit 1
fi

# 检查是否在 Morty 项目中
if [[ ! -d "$MORTY_DIR" ]]; then
    log ERROR "不在 Morty 项目目录中(缺少 .morty/ 目录)"
    log INFO ""
    log INFO "请先运行 'morty fix prd.md' 初始化项目"
    exit 1
fi

log INFO "╔════════════════════════════════════════════════════════════╗"
log INFO "║              MORTY OBS - 可观察性监控                      ║"
log INFO "╚════════════════════════════════════════════════════════════╝"
log INFO ""

# 获取 tmux base-index
get_tmux_base_index() {
    local base_index
    base_index=$(tmux show-options -gv base-index 2>/dev/null || echo "0")
    echo "${base_index:-0}"
}

# 创建 tmux 会话
SESSION_NAME="morty-obs-$(date +%s)"
PROJECT_DIR="$(pwd)"
BASE_WIN=$(get_tmux_base_index)

log INFO "创建 tmux 会话: $SESSION_NAME"
log INFO "项目目录: $PROJECT_DIR"
log INFO ""

# 创建新的 tmux 会话(分离模式)
tmux new-session -d -s "$SESSION_NAME" -c "$PROJECT_DIR"

# 水平分割窗口(左: loop, 右: 监控)
tmux split-window -h -t "$SESSION_NAME" -c "$PROJECT_DIR"

# 垂直分割右侧面板(上: 日志, 下: 状态)
tmux split-window -v -t "$SESSION_NAME:${BASE_WIN}.1" -c "$PROJECT_DIR"

# 设置面板大小(左侧 60%, 右侧 40%)
tmux resize-pane -t "$SESSION_NAME:${BASE_WIN}.0" -x 60%

# 左侧面板(面板 0): Morty loop
log INFO "配置左侧面板: Loop 执行"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '║              MORTY LOOP - 开发循环                         ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo ''" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "morty loop" Enter

# 右上面板(面板 1): 实时日志
log INFO "配置右上面板: 实时日志"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '║              实时日志                                      ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '等待日志...'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "sleep 2 && tail -f $LOG_DIR/*.log 2>/dev/null || tail -f $LOG_DIR/loop_*.log" Enter

# 右下面板(面板 2): 状态监控
log INFO "配置右下面板: 状态监控"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '║              状态监控                                      ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "echo ''" Enter

# 状态监控循环脚本
STATUS_MONITOR_SCRIPT=$(cat << 'SCRIPT'
while true; do
    clear
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              状态监控                                      ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""

    if [[ -f ".morty/status.json" ]]; then
        echo "📊 当前状态:"
        cat ".morty/status.json" | jq -r '
            "  状态: \(.state)",
            "  循环: \(.loop_count) / \(.max_loops)",
            "  消息: \(.message)",
            "  时间: \(.timestamp)"
        ' 2>/dev/null || cat ".morty/status.json"
        echo ""
    else
        echo "⏳ 等待状态文件..."
        echo ""
    fi

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
    echo "🔄 刷新: 每 3 秒"
    echo ""
    echo "快捷键:"
    echo "  Ctrl+B [     进入滚动模式"
    echo "  Ctrl+B 方向键 切换面板"
    echo "  Ctrl+B D     分离会话"

    sleep 3
done
SCRIPT
)

tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "$STATUS_MONITOR_SCRIPT" Enter

# 设置面板标题
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.0" -T "Loop 执行"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.1" -T "实时日志"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2" -T "状态监控"

# 聚焦到左侧面板
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.0"

log SUCCESS "tmux 会话已创建!"
log INFO ""
log INFO "会话名称: $SESSION_NAME"
log INFO ""
log INFO "面板布局:"
log INFO "  ┌─────────────────┬──────────────┐"
log INFO "  │                 │  实时日志    │"
log INFO "  │  Loop 执行      ├──────────────┤"
log INFO "  │                 │  状态监控    │"
log INFO "  └─────────────────┴──────────────┘"
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

# 连接到会话
tmux attach -t "$SESSION_NAME"
