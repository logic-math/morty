#!/bin/bash
# Morty Loop - 主开发循环(集成可观察性监控)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/git_manager.sh"

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

# 初始化
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/loop_$(date +%Y%m%d_%H%M%S).log"

show_help() {
    cat << 'EOF'
Morty Loop - 开发循环(集成可观察性监控)

用法: morty loop [options]

选项:
    -h, --help          显示帮助信息
    --max-loops N       最大循环次数(默认: 50)
    --delay N           循环间延迟秒数(默认: 5)
    --no-monitor        不启动 tmux 监控,直接运行循环

描述:
    默认情况下,loop 会在 tmux 中启动并提供三面板监控:
    - 左侧(50%): 循环实时日志(项目进度、任务执行)
    - 右上(30%): Claude Code 监控(Token 使用、错误、资源)
    - 右下(70%): 交互式命令行(便捷命令: status, progress, logs, plan)

    使用 --no-monitor 可以直接运行循环而不启动 tmux。

示例:
    morty loop                    # 启动带监控的循环(推荐)
    morty loop --max-loops 100    # 自定义最大循环次数
    morty loop --no-monitor       # 不启动监控,直接运行

快捷键(tmux 模式):
    Ctrl+B 然后 [        进入滚动模式(查看历史)
    Ctrl+B 然后 方向键    切换面板
    Ctrl+B 然后 D        分离会话(后台运行)
    Ctrl+B 然后 X        关闭当前面板

EOF
}

# 解析参数
USE_MONITOR=true
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --max-loops)
            MAX_LOOPS="$2"
            shift 2
            ;;
        --delay)
            LOOP_DELAY="$2"
            shift 2
            ;;
        --no-monitor)
            USE_MONITOR=false
            shift
            ;;
        *)
            log ERROR "未知参数: $1"
            exit 1
            ;;
    esac
done

# 检查项目结构
check_project_structure() {
    # 检查 .morty/ 目录是否存在
    if [[ ! -d "$MORTY_DIR" ]]; then
        log ERROR ".morty/ 目录不存在"
        log INFO ""
        log INFO "请先运行 'morty fix prd.md' 初始化项目"
        return 1
    fi

    # 检查必需文件
    local missing_files=()

    if [[ ! -f "$PROMPT_FILE" ]]; then
        missing_files+=("$PROMPT_FILE")
    fi

    if [[ ! -f "$AGENT_FILE" ]]; then
        missing_files+=("$AGENT_FILE")
    fi

    if [[ ! -f "$FIX_PLAN_FILE" ]]; then
        missing_files+=("$FIX_PLAN_FILE")
    fi

    if [[ ! -d "$SPECS_DIR" ]]; then
        missing_files+=("$SPECS_DIR")
    fi

    if [[ ${#missing_files[@]} -gt 0 ]]; then
        log ERROR "缺少必需文件/目录:"
        for file in "${missing_files[@]}"; do
            log ERROR "  - $file"
        done
        log INFO ""
        log INFO "请先运行 'morty fix prd.md' 初始化项目"
        return 1
    fi

    return 0
}

# 如果使用监控模式,启动 tmux
if [[ "$USE_MONITOR" == true ]]; then
    # 检查 tmux 是否安装
    if ! command -v tmux &> /dev/null; then
        log ERROR "tmux 未安装"
        log INFO ""
        log INFO "安装 tmux:"
        log INFO "  Ubuntu/Debian: sudo apt-get install tmux"
        log INFO "  CentOS/RHEL: sudo yum install tmux"
        log INFO "  macOS: brew install tmux"
        log INFO ""
        log INFO "或者使用 --no-monitor 参数直接运行循环"
        exit 1
    fi

    # 检查项目结构
    if ! check_project_structure; then
        exit 1
    fi

    log INFO "╔════════════════════════════════════════════════════════════╗"
    log INFO "║              MORTY LOOP - 开发循环(监控模式)               ║"
    log INFO "╚════════════════════════════════════════════════════════════╝"
    log INFO ""

    # 启动 tmux 监控模式
    exec "$SCRIPT_DIR/lib/loop_monitor.sh" \
        --max-loops "$MAX_LOOPS" \
        --delay "$LOOP_DELAY"
fi

# 非监控模式:直接运行循环
log INFO "╔════════════════════════════════════════════════════════════╗"
log INFO "║              MORTY LOOP - 开发循环                         ║"
log INFO "╚════════════════════════════════════════════════════════════╝"
log INFO ""

# 检查项目结构
if ! check_project_structure; then
    exit 1
fi

log SUCCESS "✓ 项目结构完整"
log INFO ""

# 清空日志目录
log INFO "清空历史日志..."
rm -rf "$LOG_DIR"/*
mkdir -p "$LOG_DIR"
log SUCCESS "✓ 日志目录已清空"
log INFO ""

# 初始化 git 仓库(确保代码变更可被追溯)
init_git_if_needed
log INFO ""

# 读取必要文件
log INFO "读取项目文件..."

PROMPT_CONTENT=$(cat "$PROMPT_FILE")
log SUCCESS "✓ 读取 PROMPT.md"

AGENT_CONTENT=$(cat "$AGENT_FILE")
log SUCCESS "✓ 读取 AGENT.md"

FIX_PLAN_CONTENT=$(cat "$FIX_PLAN_FILE")
log SUCCESS "✓ 读取 fix_plan.md"

# 列出 specs 文件
SPEC_FILES=$(find "$SPECS_DIR" -name "*.md" -type f 2>/dev/null | sort)
SPEC_COUNT=$(echo "$SPEC_FILES" | wc -l)
log SUCCESS "✓ 找到 $SPEC_COUNT 个模块规范"

log INFO ""

# 显示配置
log INFO "循环配置:"
log INFO "  - 最大循环次数: $MAX_LOOPS"
log INFO "  - 循环间延迟: ${LOOP_DELAY}s"
log INFO "  - 日志文件: $LOG_FILE"
log INFO ""

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

    cat << EOF
$PROMPT_CONTENT

---

# 当前循环状态

**循环次数**: $loop_count / $MAX_LOOPS

## 当前任务列表

\`\`\`markdown
$FIX_PLAN_CONTENT
\`\`\`

## 可用的模块规范

以下模块规范可供参考(按需读取):

$(echo "$SPEC_FILES" | while read -r spec_file; do
    echo "- \`$spec_file\`"
done)

## 构建和测试命令

\`\`\`markdown
$AGENT_CONTENT
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
log INFO "开始开发循环..."
log INFO ""

LOOP_COUNT=0

while [[ $LOOP_COUNT -lt $MAX_LOOPS ]]; do
    LOOP_COUNT=$((LOOP_COUNT + 1))

    log INFO "════════════════════════════════════════════════════════════"
    log LOOP "循环 #$LOOP_COUNT"
    log INFO "════════════════════════════════════════════════════════════"
    log INFO ""

    # 更新状态
    update_status "running" "$LOOP_COUNT" "执行循环 $LOOP_COUNT"

    # 构建提示词
    LOOP_PROMPT=$(build_loop_prompt "$LOOP_COUNT")
    PROMPT_FILE_TEMP="$LOG_DIR/loop_${LOOP_COUNT}_prompt.md"
    echo "$LOOP_PROMPT" > "$PROMPT_FILE_TEMP"

    log INFO "提示词已保存: $PROMPT_FILE_TEMP"
    log INFO ""

    # 构建 Claude 命令
    CLAUDE_ARGS=(
        "$CLAUDE_CMD"
        "--verbose"
        "--debug"
        "--dangerously-skip-permissions"
    )

    # 如果有 session ID,使用它
    if [[ -f "$SESSION_FILE" ]]; then
        SESSION_ID=$(cat "$SESSION_FILE")
        CLAUDE_ARGS+=("--session-id" "$SESSION_ID")
        log INFO "使用会话 ID: $SESSION_ID"
    fi

    # 执行 Claude
    LOOP_LOG="$LOG_DIR/loop_${LOOP_COUNT}_output.log"
    log INFO "执行 Claude Code..."
    log INFO ""

    if cat "$PROMPT_FILE_TEMP" | "${CLAUDE_ARGS[@]}" 2>&1 | tee "$LOOP_LOG"; then
        CLAUDE_EXIT_CODE=0
    else
        CLAUDE_EXIT_CODE=$?
    fi

    log INFO ""
    log INFO "循环 #$LOOP_COUNT 完成(退出码: $CLAUDE_EXIT_CODE)"
    log INFO ""

    # 创建 git commit (用于历史追溯)
    if [[ $CLAUDE_EXIT_CODE -eq 0 ]]; then
        create_loop_commit "$LOOP_COUNT" "completed"
        log INFO ""
    fi

    # 检查退出码
    if [[ $CLAUDE_EXIT_CODE -ne 0 ]]; then
        log ERROR "Claude Code 执行失败"
        update_status "error" "$LOOP_COUNT" "Claude 执行失败"
        exit 1
    fi

    # 检查是否完成
    # 查找 EXIT_SIGNAL: true
    if grep -q "EXIT_SIGNAL: true" "$LOOP_LOG"; then
        log INFO ""
        log SUCCESS "检测到退出信号 - 项目完成!"
        update_status "completed" "$LOOP_COUNT" "项目完成"
        break
    fi

    # 检查是否所有任务完成
    UNCHECKED_TASKS=$(grep -c "\- \[ \]" "$FIX_PLAN_FILE" 2>/dev/null || echo "0")
    if [[ $UNCHECKED_TASKS -eq 0 ]]; then
        log INFO ""
        log SUCCESS "所有任务已完成!"
        update_status "completed" "$LOOP_COUNT" "所有任务完成"
        break
    fi

    log INFO "剩余任务: $UNCHECKED_TASKS"
    log INFO ""

    # 延迟
    if [[ $LOOP_COUNT -lt $MAX_LOOPS ]]; then
        log INFO "等待 ${LOOP_DELAY}s 后继续..."
        sleep "$LOOP_DELAY"
        log INFO ""
    fi
done

# 循环结束
log INFO ""
log INFO "════════════════════════════════════════════════════════════"

if [[ $LOOP_COUNT -ge $MAX_LOOPS ]]; then
    log WARN "达到最大循环次数: $MAX_LOOPS"
    update_status "max_loops_reached" "$LOOP_COUNT" "达到最大循环次数"
else
    log SUCCESS "开发循环正常结束"
fi

log INFO ""
log INFO "总循环次数: $LOOP_COUNT"
log INFO "日志文件: $LOG_FILE"
log INFO ""
log SUCCESS "循环完成! 🎉"
