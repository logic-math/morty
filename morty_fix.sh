#!/bin/bash
# Morty Fix Mode - 迭代式 PRD 改进与知识捕获

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# 配置
CLAUDE_CMD="${CLAUDE_CODE_CLI:-claude}"
FIX_SYSTEM_PROMPT="$SCRIPT_DIR/prompts/fix_mode_system.md"

show_help() {
    cat << 'EOF'
Morty Fix 模式 - 迭代式 PRD 改进

用法: morty fix <prd.md>

参数:
    prd.md          现有的 PRD/需求文档 (Markdown)

描述:
    Fix 模式启动一个交互式 Claude Code 会话来:
    1. 阅读现有的 PRD
    2. 通过对话引导需求改进(问题诊断/功能迭代/架构优化)
    3. 生成改进版的 prd.md
    4. 更新或创建 specs/ 目录下的模块规范文档
    5. 可选:生成/更新项目结构

示例:
    morty fix prd.md
    morty fix docs/requirements.md

特性:
    - 与 Claude Code 的交互式对话
    - 三种改进方向:问题修复、功能增强、架构重构
    - 模块化知识管理(specs/ 目录)
    - 迭代式知识积累
    - 可选的项目结构生成

工作流程:
    1. 阅读现有 prd.md
    2. 对话式需求改进
    3. 生成改进版 prd.md
    4. 更新 specs/*.md 模块规范
    5. 询问是否生成项目结构
    6. 手动运行 'morty start' 开始开发循环

EOF
}

# 解析参数
PRD_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            if [[ -z "$PRD_FILE" ]]; then
                PRD_FILE="$1"
            else
                log ERROR "未知参数: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$PRD_FILE" ]]; then
    log ERROR "需要 PRD 文件"
    show_help
    exit 1
fi

# 检查 PRD 文件是否存在
if [[ ! -f "$PRD_FILE" ]]; then
    log ERROR "PRD 文件未找到: $PRD_FILE"
    exit 1
fi

# 检查是否为 Markdown 文件
if [[ ! "$PRD_FILE" =~ \.md$ ]]; then
    log ERROR "仅支持 Markdown (.md) 文件"
    exit 1
fi

# 获取绝对路径
PRD_FILE=$(realpath "$PRD_FILE")
PRD_FILENAME=$(basename "$PRD_FILE")

log INFO "╔════════════════════════════════════════════════════════════╗"
log INFO "║            MORTY FIX 模式 - PRD 迭代改进                  ║"
log INFO "╚════════════════════════════════════════════════════════════╝"
log INFO ""
log INFO "PRD 文件: $PRD_FILE"
log INFO ""

# 检查系统提示词是否存在
if [[ ! -f "$FIX_SYSTEM_PROMPT" ]]; then
    log ERROR "Fix 模式系统提示词未找到: $FIX_SYSTEM_PROMPT"
    log INFO "创建系统提示词..."
    mkdir -p "$(dirname "$FIX_SYSTEM_PROMPT")"
    log ERROR "请先运行安装以创建系统提示词"
    exit 1
fi

# 创建工作目录
FIX_WORK_DIR=".morty_fix_$$"
mkdir -p "$FIX_WORK_DIR"

log INFO "工作目录: $FIX_WORK_DIR"
log INFO ""

# 复制 PRD 到工作目录
cp "$PRD_FILE" "$FIX_WORK_DIR/current_prd.md"

# 读取当前 PRD 内容
CURRENT_PRD_CONTENT=$(cat "$PRD_FILE")

# 读取系统提示词
SYSTEM_PROMPT_CONTENT=$(cat "$FIX_SYSTEM_PROMPT")

# 构建交互式提示词(结合系统提示词 + PRD)
INTERACTIVE_PROMPT=$(cat << EOF
$SYSTEM_PROMPT_CONTENT

---

# 当前 PRD 内容

PRD 文件名: **$PRD_FILENAME**

## PRD 内容

\`\`\`markdown
$CURRENT_PRD_CONTENT
\`\`\`

---

**指令**: 遵循上述系统提示词中的对话框架。从阶段 1(上下文收集)开始,分析这个 PRD 并提出你的第一轮澄清问题。

当 PRD 改进完成,模块规范已更新,并且(可选)项目结构已生成后,输出系统提示词中指定的完成信号。
EOF
)

log INFO "启动交互式 Fix 模式会话..."
log INFO ""
log INFO "使用说明:"
log INFO "  - Claude 会提问以改进 PRD"
log INFO "  - 深思熟虑地回答以帮助澄清需求"
log INFO "  - 自然地输入你的回应"
log INFO "  - Claude 会迭代直到 PRD 完善"
log INFO "  - 会话在 Claude 输出时结束: <!-- FIX_MODE_COMPLETE -->"
log INFO ""
log INFO "按 Enter 开始交互式会话..."
read -r

# 在交互模式下启动 Claude Code 与 fix 系统提示词
# 关键标志:
#   --continue: 启用会话连续性以保持上下文
#   --allowedTools: 允许完整工具访问
log INFO "在 Fix 模式下启动 Claude Code..."
log INFO ""

# 将提示词保存到文件供 Claude 使用(避免命令行参数长度问题)
PROMPT_FILE="$FIX_WORK_DIR/fix_prompt.md"
echo "$INTERACTIVE_PROMPT" > "$PROMPT_FILE"

log INFO "提示词已保存到: $PROMPT_FILE"
log INFO "CLAUDE_CMD: $CLAUDE_CMD"

# 为 fix 模式构建带有适当标志的 Claude 命令
# 使用 stdin 传递提示词(对于长内容更可靠)
CLAUDE_ARGS=(
    "$CLAUDE_CMD"
    "--continue"
    "--allowedTools" "Read" "Write" "Glob" "Grep" "WebSearch" "WebFetch"
)

# 从 stdin 交互式执行 Claude Code 与提示词
cat "$PROMPT_FILE" | "${CLAUDE_ARGS[@]}"

CLAUDE_EXIT_CODE=$?

log INFO ""
log INFO "Fix 模式会话完成(退出码: $CLAUDE_EXIT_CODE)"
log INFO ""

# 检查 Claude 是否成功完成
if [[ $CLAUDE_EXIT_CODE -ne 0 ]]; then
    log ERROR "Claude Code 以错误码退出: $CLAUDE_EXIT_CODE"
    log INFO "Fix 模式会话未成功完成"
    exit 1
fi

log INFO ""
log SUCCESS "╔════════════════════════════════════════════════════════════╗"
log SUCCESS "║              FIX 会话完成!                                 ║"
log SUCCESS "╚════════════════════════════════════════════════════════════╝"
log INFO ""
log INFO "生成的文件:"
log INFO "  ✓ prd.md (改进版)            产品需求文档"
log INFO "  ✓ specs/*.md                 模块规范文档"
log INFO "  ✓ .morty/PROMPT.md (可选)    开发指令"
log INFO "  ✓ .morty/fix_plan.md (可选)  任务分解"
log INFO "  ✓ .morty/AGENT.md (可选)     构建/测试命令"
log INFO ""
log INFO "下一步:"
log INFO "  1. 查看改进的 prd.md"
log INFO "  2. 查看 specs/ 目录中的模块规范"
log INFO "  3. 如果生成了项目结构,查看 .morty/fix_plan.md"
log INFO "  4. 运行 'morty start' 开始开发循环(手动步骤)"
log INFO ""
log SUCCESS "持续改进! 🔧"

# 清理工作目录
rm -rf "$FIX_WORK_DIR"
