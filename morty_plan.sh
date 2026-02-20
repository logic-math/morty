#!/bin/bash
# Morty Plan Mode - 基于研究结果创建 TDD 开发计划

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# 配置
CLAUDE_CMD="${CLAUDE_CODE_CLI:-ai_cli}"
PLAN_PROMPT="$SCRIPT_DIR/prompts/plan.md"
MORTY_DIR=".morty"
PLAN_DIR="$MORTY_DIR/plan"
RESEARCH_DIR="$MORTY_DIR/research"

show_help() {
    cat << 'EOF'
Morty Plan 模式 - 基于研究结果创建 TDD 开发计划

用法: morty plan [选项]

选项:
    -h, --help          显示帮助信息

描述:
    Plan 模式基于 research 模式的研究结果，创建分层的 TDD 开发计划。

    工作流程:
    1. 检查 .morty/research/ 目录下的研究结果
    2. 启动交互式 Claude Code 会话
    3. 根据需要读取相关 research 文件作为事实性信息
    4. 设计系统架构，划分功能模块
    5. 为每个模块创建 [模块名].md 文件（含 Jobs + 验证器）
    6. 创建 [生产测试].md 端到端测试计划
    7. 生成 plan/README.md 索引文件

    输出结构:
    .morty/plan/
    ├── README.md           # Plan 索引
    ├── [模块A].md          # 功能模块 A 计划（含 Jobs + 验证器）
    ├── [模块B].md          # 功能模块 B 计划
    └── [生产测试].md       # 端到端测试计划

示例:
    morty plan              # 启动 Plan 模式

前置条件:
    - 必须先运行 morty research 生成研究结果
    - .morty/research/ 目录必须存在且非空

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
            show_help
            exit 1
            ;;
    esac
done

# 检查提示词文件
if [[ ! -f "$PLAN_PROMPT" ]]; then
    log ERROR "Plan 模式提示词未找到: $PLAN_PROMPT"
    log ERROR "请先运行安装: ./install.sh"
    exit 1
fi

# 检查前置条件：.morty 目录存在
if [[ ! -d "$MORTY_DIR" ]]; then
    log ERROR ".morty/ 目录不存在"
    log INFO ""
    log INFO "请先运行 morty research 进行项目研究"
    exit 1
fi

# 检查前置条件：research 目录存在且有内容
if [[ ! -d "$RESEARCH_DIR" ]]; then
    log ERROR ".morty/research/ 目录不存在"
    log INFO ""
    log INFO "请先运行 morty research 生成研究结果"
    exit 1
fi

# 检查 research 目录是否有 .md 文件
RESEARCH_FILES=$(find "$RESEARCH_DIR" -name "*.md" -type f 2>/dev/null || true)
if [[ -z "$RESEARCH_FILES" ]]; then
    log ERROR ".morty/research/ 目录中没有 .md 文件"
    log INFO ""
    log INFO "请先运行 morty research 生成研究结果"
    exit 1
fi

# 创建 plan 目录
mkdir -p "$PLAN_DIR"

log INFO "╔════════════════════════════════════════════════════════════╗"
log INFO "║              MORTY PLAN 模式 - TDD 开发计划                ║"
log INFO "╚════════════════════════════════════════════════════════════╝"
log INFO ""

# 统计 research 文件
RESEARCH_COUNT=$(echo "$RESEARCH_FILES" | wc -l)
log INFO "发现 $RESEARCH_COUNT 个研究文件:"
echo "$RESEARCH_FILES" | while read -r file; do
    log INFO "  - $(basename "$file")"
done
log INFO ""

# 读取系统提示词
SYSTEM_PROMPT=$(cat "$PLAN_PROMPT")

# 构建可用研究文件列表（仅文件名，不加载内容）
RESEARCH_FILE_LIST=""
while IFS= read -r file; do
    if [[ -f "$file" ]]; then
        RESEARCH_FILE_LIST="${RESEARCH_FILE_LIST}- \`$file\`\n"
    fi
done <<< "$RESEARCH_FILES"

# 构建交互式提示词
INTERACTIVE_PROMPT=$(cat << EOF
$SYSTEM_PROMPT

---

# 当前运行状态

**工作目录**: $(pwd)
**Plan 目录**: $PLAN_DIR/
**Research 目录**: $RESEARCH_DIR/
**研究文件数量**: $RESEARCH_COUNT

---

# 可用研究文件

以下研究文件包含事实性信息，请根据需要使用 Read 工具读取：

$RESEARCH_FILE_LIST

**说明**:
- 以上文件是 Research 模式生成的研究结果
- 请根据架构设计需要，选择性读取相关文件
- 这些文件包含目录结构、核心功能、处理流程等事实性信息

---

# Plan 目录结构

Plan 模式将在 $PLAN_DIR/ 目录下创建以下文件：
- README.md - Plan 索引
- [模块名].md - 各功能模块计划
- [生产测试].md - 端到端测试计划

---

开始对话！请基于 Research 文件中的事实性信息进行架构设计，划分功能模块，创建 Plan 文件。
EOF
)

log INFO "启动 Claude Code 交互式会话..."
log INFO ""
log INFO "Plan 模式将："
log INFO "  1. 根据需要读取 research 文件获取事实性信息"
log INFO "  2. 设计系统架构，划分功能模块"
log INFO "  3. 为每个模块创建 [模块名].md 文件"
log INFO "  4. 创建 [生产测试].md 端到端测试计划"
log INFO "  5. 生成 README.md 索引"
log INFO ""

# 构建 Claude 命令
CLAUDE_ARGS=(
    "$CLAUDE_CMD"
    "--dangerously-skip-permissions"
    "--allowedTools" "Read" "Write" "Glob" "Grep" "WebSearch" "WebFetch" "Edit" "Task"
)

# 将提示词通过管道传递给 Claude Code
echo "$INTERACTIVE_PROMPT" | "${CLAUDE_ARGS[@]}"

CLAUDE_EXIT_CODE=$?

log INFO ""
log INFO "Claude Code 退出码: $CLAUDE_EXIT_CODE"
log INFO ""

# 检查 plan 目录是否生成内容
if [[ -d "$PLAN_DIR" ]]; then
    PLAN_FILES=$(find "$PLAN_DIR" -name "*.md" -type f 2>/dev/null || true)
    if [[ -n "$PLAN_FILES" ]]; then
        PLAN_COUNT=$(echo "$PLAN_FILES" | wc -l)
        log SUCCESS "Plan 文件已生成: $PLAN_COUNT 个"
        echo "$PLAN_FILES" | while read -r file; do
            log INFO "  - $(basename "$file")"
        done
        log INFO ""
        log INFO "下一步:"
        log INFO "  运行 'morty doing' 开始分层 TDD 开发"
    else
        log WARN "Plan 目录为空，可能没有生成文件"
    fi
else
    log WARN "Plan 目录未创建"
fi

log INFO ""
log SUCCESS "Plan 模式会话结束"
