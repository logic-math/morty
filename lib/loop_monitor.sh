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

# 水平分割窗口(左: Claude 监控, 右: 日志和终端)
tmux split-window -h -t "$SESSION_NAME" -c "$PROJECT_DIR"

# 垂直分割右侧面板(上: 日志 30%, 下: bash 70%)
tmux split-window -v -t "$SESSION_NAME:${BASE_WIN}.1" -c "$PROJECT_DIR"

# 设置面板大小
# 左侧 Claude 监控 50%
tmux resize-pane -t "$SESSION_NAME:${BASE_WIN}.0" -x 50%
# 右上日志 30%
tmux resize-pane -t "$SESSION_NAME:${BASE_WIN}.1" -y 30%

# 创建循环执行脚本(在后台运行,不受 tmux 影响)
LOOP_RUNNER="$LOG_DIR/loop_runner_$(date +%s).sh"
cat > "$LOOP_RUNNER" << 'LOOP_SCRIPT'
#!/bin/bash
# Loop Runner - 后台循环执行脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

source "$SCRIPT_DIR/.morty/lib/common.sh" 2>/dev/null || source "$(dirname "$0")/../lib/common.sh"
source "$SCRIPT_DIR/.morty/lib/git_manager.sh" 2>/dev/null || source "$(dirname "$0")/../lib/git_manager.sh"

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

# 初始化 git(如果需要)
echo "检查 git 仓库..."
init_git_if_needed
echo ""

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
        # 即使失败也创建提交(记录错误状态)
        echo ""
        echo "创建错误状态提交..."
        create_loop_commit "$LOOP_COUNT" "error"
        exit 1
    fi

    # 创建循环提交
    echo ""
    echo "创建循环提交..."
    if create_loop_commit "$LOOP_COUNT" "completed"; then
        echo "✓ 循环 #$LOOP_COUNT 已提交到 git"
    else
        echo "⚠ 循环 #$LOOP_COUNT 提交失败(继续执行)"
    fi
    echo ""

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

# 左侧面板(面板 0): 循环实时日志(项目进度)
log INFO "配置左侧面板: 循环实时日志"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "clear" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╔════════════════════════════════════════════════════════════╗'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '║              循环实时日志 - 项目进度                       ║'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '╚════════════════════════════════════════════════════════════╝'" Enter
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "echo '等待日志...'\" Enter

# 在后台启动循环,然后监控日志
LOOP_STARTER_SCRIPT=$(cat << 'SCRIPT'
# 等待一下让其他面板初始化
sleep 2

# 启动循环(后台)
nohup LOOP_RUNNER_PATH > /dev/null 2>&1 &

# 等待日志文件生成
while [[ ! -f .morty/logs/loop_*_output.log ]]; do
    sleep 1
done

# 尾随日志
tail -f .morty/logs/loop_*_output.log 2>/dev/null
SCRIPT
)

# 替换 LOOP_RUNNER_PATH
LOOP_STARTER_SCRIPT="${LOOP_STARTER_SCRIPT//LOOP_RUNNER_PATH/$LOOP_RUNNER}"

tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.0" "$LOOP_STARTER_SCRIPT" Enter

# 右上面板(面板 1): Claude Code 监控 (30% 高度)
log INFO "配置右上面板: Claude Code 监控"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "clear" Enter

# Claude 监控脚本
CLAUDE_MONITOR_SCRIPT=$(cat << 'SCRIPT'
monitor_claude() {
    while true; do
        clear
        echo "╔════════════════════════════════════════════════════════════╗"
        echo "║              CLAUDE CODE 监控 (30%)                        ║"
        echo "╚════════════════════════════════════════════════════════════╝"
        echo ""

        # 显示当前循环信息
        if [[ -f ".morty/status.json" ]]; then
            echo "📊 循环状态:"
            state=$(jq -r '.state // "unknown"' ".morty/status.json" 2>/dev/null)
            loop_count=$(jq -r '.loop_count // 0' ".morty/status.json" 2>/dev/null)
            max_loops=$(jq -r '.max_loops // 50' ".morty/status.json" 2>/dev/null)
            echo "  状态: $state | 循环: $loop_count/$max_loops"
        fi

        # 显示最新的循环日志文件信息
        latest_log=$(ls -t .morty/logs/loop_*_output.log 2>/dev/null | head -1)
        if [[ -n "$latest_log" ]]; then
            # 统计 token 使用(从日志中提取)
            echo ""
            echo "🔢 Token 使用:"

            # 查找包含 token 信息的行
            token_info=$(grep -i "token" "$latest_log" | tail -3)
            if [[ -n "$token_info" ]]; then
                echo "$token_info" | while read -r line; do
                    echo "  ${line:0:60}"
                done
            else
                echo "  等待 token 信息..."
            fi

            # 显示日志文件大小
            log_size=$(du -h "$latest_log" | cut -f1)
            echo ""
            echo "📦 日志: $log_size"

            # 显示最近的错误(如果有)
            echo ""
            echo "⚠️  错误:"
            recent_errors=$(grep -i "error\|failed\|exception" "$latest_log" 2>/dev/null | tail -2)
            if [[ -n "$recent_errors" ]]; then
                echo "$recent_errors" | while read -r line; do
                    echo "  ${line:0:55}..."
                done
            else
                echo "  无错误"
            fi
        else
            echo ""
            echo "⏳ 等待循环启动..."
        fi

        # 显示会话信息
        if [[ -f ".morty/.session_id" ]]; then
            session_id=$(cat ".morty/.session_id")
            echo ""
            echo "🔗 会话: ${session_id:0:40}..."
        fi

        # 显示系统资源
        echo ""
        echo "💻 资源: CPU $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)% | 内存 $(free -h | awk '/^Mem:/ {print $3 "/" $2}')"

        echo ""
        echo "刷新: 5s"

        sleep 5
    done
}

# 启动监控
monitor_claude
SCRIPT
)

tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.1" "$CLAUDE_MONITOR_SCRIPT" Enter

# 右下面板(面板 2): 交互式终端 (70% 高度)
log INFO "配置右下面板: 交互式终端"
tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "clear" Enter

# 交互式终端初始化脚本
TERMINAL_INIT_SCRIPT=$(cat << 'SCRIPT'
clear
echo "╔════════════════════════════════════════════════════════════╗"
echo "║              交互式终端 (70%)                              ║"
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

tmux send-keys -t "$SESSION_NAME:${BASE_WIN}.2" "$TERMINAL_INIT_SCRIPT" Enter

# 设置面板标题
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.0" -T "循环日志(项目进度)"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.1" -T "Claude监控(30%)"
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2" -T "交互终端(70%)"

# 聚焦到右下面板(交互式终端)
tmux select-pane -t "$SESSION_NAME:${BASE_WIN}.2"

log SUCCESS "tmux 会话已创建!"
log INFO ""
log INFO "会话名称: $SESSION_NAME"
log INFO ""
log INFO "面板布局:"
log INFO "  ┌──────────────────┬───────────────┐"
log INFO "  │                  │ Claude监控    │"
log INFO "  │  循环日志        │ (Token/30%)   │"
log INFO "  │  (项目进度)      ├───────────────┤"
log INFO "  │                  │ 交互终端(70%) │"
log INFO "  └──────────────────┴───────────────┘"
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
