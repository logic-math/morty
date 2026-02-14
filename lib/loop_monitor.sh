#!/bin/bash
# Morty Loop Monitor - tmux 集成监控
# 在 tmux 中启动循环并提供三面板监控

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

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

# 水平分割窗口(左: loop, 右: 监控)
tmux split-window -h -t "$SESSION_NAME" -c "$PROJECT_DIR"

# 垂直分割右侧面板(上: 日志, 下: bash + 状态)
tmux split-window -v -t "$SESSION_NAME:${BASE_WIN}.1" -c "$PROJECT_DIR"

# 设置面板大小(左侧 60%, 右侧 40%)
tmux resize-pane -t "$SESSION_NAME:${BASE_WIN}.0" -x 60%

# 创建循环执行脚本(在后台运行,不受 tmux 影响)
LOOP_RUNNER="$LOG_DIR/loop_runner_$(date +%s).sh"
cat > "$LOOP_RUNNER" << 'LOOP_SCRIPT'
#!/bin/bash
# Loop Runner - 后台循环执行脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

source "$SCRIPT_DIR/.morty/lib/common.sh" 2>/dev/null || source "$(dirname "$0")/../lib/common.sh"

# 配置
MORTY_DIR=".morty"
PROMPT_FILE="$MORTY_DIR/PROMPT.md"
AGENT_FILE="$MORTY_DIR/AGENT.md"
FIX_PLAN_FILE="$MORTY_DIR/fix_plan.md"
SPECS_DIR="$MORTY_DIR/specs"
LOG_DIR="$MORTY_DIR/logs"
STATUS_FILE="$MORTY_DIR/status.json"
SESSION_FILE="$MORTY_DIR/.session_id"

CLAUDE_CMD="${CLAUDE_CODE_CLI:-claude}"
MAX_LOOPS="${MAX_LOOPS:-50}"
LOOP_DELAY="${LOOP_DELAY:-5}"

LOG_FILE="$LOG_DIR/loop_$(date +%Y%m%d_%H%M%S).log"

# 更新状态
update_status() {
    local state=$1
    local loop_count=$2
    local message=${3:-""}

    cat > "$STATUS_FILE" << EOF
{
  "state": "$state",
  "loop_count": $loop_count,
  "max_loops": $MAX_LOOPS,
  "message": "$message",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
}

# 构建循环提示词
build_loop_prompt() {
    local loop_count=$1
    local prompt_content=$(cat "$PROMPT_FILE")
    local fix_plan_content=$(cat "$FIX_PLAN_FILE")
    local agent_content=$(cat "$AGENT_FILE")
    local spec_files=$(find "$SPECS_DIR" -name "*.md" -type f 2>/dev/null | sort)

    cat << EOF
$prompt_content

---

# 当前循环状态

**循环次数**: $loop_count / $MAX_LOOPS

## 当前任务列表

\`\`\`markdown
$fix_plan_content
\`\`\`

## 可用的模块规范

以下模块规范可供参考(按需读取):

$(echo "$spec_files" | while read -r spec_file; do
    echo "- \`$spec_file\`"
done)

## 构建和测试命令

\`\`\`markdown
$agent_content
\`\`\`

---

**指令**:
1. 查看 fix_plan.md 中的任务列表
2. 选择下一个未完成的任务
3. 如需要,读取相关的模块规范文件
4. 实现任务
5. 测试代码
6. 更新文档
7. 在 fix_plan.md 中标记任务完成
8. 输出 RALPH_STATUS 块

开始工作!
EOF
}

# 主循环
echo "╔════════════════════════════════════════════════════════════╗"
echo "║              MORTY LOOP - 开发循环                         ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "开始开发循环..."
echo ""

LOOP_COUNT=0

while [[ $LOOP_COUNT -lt $MAX_LOOPS ]]; do
    LOOP_COUNT=$((LOOP_COUNT + 1))

    echo "════════════════════════════════════════════════════════════"
    echo "循环 #$LOOP_COUNT"
    echo "════════════════════════════════════════════════════════════"
    echo ""

    # 更新状态
    update_status "running" "$LOOP_COUNT" "执行循环 $LOOP_COUNT"

    # 构建提示词
    LOOP_PROMPT=$(build_loop_prompt "$LOOP_COUNT")
    PROMPT_FILE_TEMP="$LOG_DIR/loop_${LOOP_COUNT}_prompt.md"
    echo "$LOOP_PROMPT" > "$PROMPT_FILE_TEMP"

    echo "提示词已保存: $PROMPT_FILE_TEMP"
    echo ""

    # 构建 Claude 命令
    CLAUDE_ARGS=(
        "$CLAUDE_CMD"
        "--dangerously-skip-permissions"
    )

    # 如果有 session ID,使用它
    if [[ -f "$SESSION_FILE" ]]; then
        SESSION_ID=$(cat "$SESSION_FILE")
        CLAUDE_ARGS+=("--session-id" "$SESSION_ID")
        echo "使用会话 ID: $SESSION_ID"
    fi

    # 执行 Claude
    LOOP_LOG="$LOG_DIR/loop_${LOOP_COUNT}_output.log"
    echo "执行 Claude Code..."
    echo ""

    if cat "$PROMPT_FILE_TEMP" | "${CLAUDE_ARGS[@]}" 2>&1 | tee "$LOOP_LOG"; then
        CLAUDE_EXIT_CODE=0
    else
        CLAUDE_EXIT_CODE=$?
    fi

    echo ""
    echo "循环 #$LOOP_COUNT 完成(退出码: $CLAUDE_EXIT_CODE)"
    echo ""

    # 检查退出码
    if [[ $CLAUDE_EXIT_CODE -ne 0 ]]; then
        echo "ERROR: Claude Code 执行失败"
        update_status "error" "$LOOP_COUNT" "Claude 执行失败"
        exit 1
    fi

    # 检查是否完成
    # 查找 EXIT_SIGNAL: true
    if grep -q "EXIT_SIGNAL: true" "$LOOP_LOG"; then
        echo ""
        echo "检测到退出信号 - 项目完成!"
        update_status "completed" "$LOOP_COUNT" "项目完成"
        break
    fi

    # 检查是否所有任务完成
    UNCHECKED_TASKS=$(grep -c "\- \[ \]" "$FIX_PLAN_FILE" 2>/dev/null || echo "0")
    if [[ $UNCHECKED_TASKS -eq 0 ]]; then
        echo ""
        echo "所有任务已完成!"
        update_status "completed" "$LOOP_COUNT" "所有任务完成"
        break
    fi

    echo "剩余任务: $UNCHECKED_TASKS"
    echo ""

    # 延迟
    if [[ $LOOP_COUNT -lt $MAX_LOOPS ]]; then
        echo "等待 ${LOOP_DELAY}s 后继续..."
        sleep "$LOOP_DELAY"
        echo ""
    fi
done

# 循环结束
echo ""
echo "════════════════════════════════════════════════════════════"

if [[ $LOOP_COUNT -ge $MAX_LOOPS ]]; then
    echo "WARN: 达到最大循环次数: $MAX_LOOPS"
    update_status "max_loops_reached" "$LOOP_COUNT" "达到最大循环次数"
else
    echo "SUCCESS: 开发循环正常结束"
fi

echo ""
echo "总循环次数: $LOOP_COUNT"
echo "日志文件: $LOG_FILE"
echo ""
echo "循环完成! 🎉"
LOOP_SCRIPT

# 替换环境变量
sed -i "s/MAX_LOOPS:-50/MAX_LOOPS:-$MAX_LOOPS/" "$LOOP_RUNNER"
sed -i "s/LOOP_DELAY:-5/LOOP_DELAY:-$LOOP_DELAY/" "$LOOP_RUNNER"

chmod +x "$LOOP_RUNNER"

# 左侧面板(面板 0): Morty loop 执行
log INFO "配置左侧面板: Loop 执行"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '║              MORTY LOOP - 开发循环                         ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo ''" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "$LOOP_RUNNER" Enter

# 右上面板(面板 1): 实时日志
log INFO "配置右上面板: 实时日志"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '║              实时日志                                      ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "echo '等待日志...'\" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "sleep 2 && tail -f $LOG_DIR/*.log 2>/dev/null || tail -f $LOG_DIR/loop_*.log" Enter

# 右下面板(面板 2): 状态监控 + 交互式 bash
log INFO "配置右下面板: 状态监控 + 交互式终端"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "clear" Enter

# 状态监控脚本(显示后进入交互模式)
STATUS_DISPLAY_SCRIPT=$(cat << 'SCRIPT'
show_status() {
    clear
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║              状态监控 + 交互式终端                         ║"
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
    echo "═══════════════════════════════════════════════════════════"
    echo "命令:"
    echo "  status    - 刷新状态"
    echo "  logs      - 查看最新日志"
    echo "  plan      - 查看任务计划"
    echo ""
    echo "快捷键:"
    echo "  Ctrl+B [     进入滚动模式"
    echo "  Ctrl+B 方向键 切换面板"
    echo "  Ctrl+B D     分离会话"
    echo ""
}

# 定义便捷命令
alias status='show_status'
alias logs='tail -20 .morty/logs/loop_*.log | tail -50'
alias plan='cat .morty/fix_plan.md'

# 显示初始状态
show_status

# 提示用户
echo "💡 输入命令或使用 bash (输入 'status' 刷新状态)"
SCRIPT
)

tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "$STATUS_DISPLAY_SCRIPT" Enter

# 设置面板标题
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.0" -T "Loop 执行"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.1" -T "实时日志"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2" -T "状态 + Bash"

# 聚焦到右下面板(交互式终端)
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2"

log SUCCESS "tmux 会话已创建!"
log INFO ""
log INFO "会话名称: $SESSION_NAME"
log INFO ""
log INFO "面板布局:"
log INFO "  ┌─────────────────┬──────────────┐"
log INFO "  │                 │  实时日志    │"
log INFO "  │  Loop 执行      ├──────────────┤"
log INFO "  │                 │  状态 + Bash │"
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
