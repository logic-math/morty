#!/bin/bash
# Morty Fix Mode - 迭代式 PRD 改进与知识积累

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# 配置
CLAUDE_CMD="${CLAUDE_CODE_CLI:-ai_cli}"
FIX_SYSTEM_PROMPT="$SCRIPT_DIR/prompts/fix_mode_system.md"
WORK_DIR=".morty_fix_work"
show_help() {
    cat << 'EOF'
Morty Fix 模式 - 迭代式 PRD 改进

用法: morty fix <prd.md>

参数:
    prd.md          现有的 PRD/需求文档 (Markdown,只读)

描述:
    Fix 模式启动一个交互式 Claude Code 会话来:
    1. 阅读现有的 prd.md (不会修改)
    2. 在工作目录中进行对话和知识积累
    3. 根据工作目录内容生成/更新 .morty/ 目录
    4. 验证 .morty/ 目录结构正确

工作流程:
    - 首次运行: 创建 .morty/ 目录和所有文件
    - 再次运行: 合并修改到已有文件
      - fix_plan.md, AGENT.md, PROMPT.md: 直接重建
      - specs/*.md: 合并修改(不重建)

示例:
    morty fix prd.md              # 首次运行
    morty fix prd.md              # 再次运行,合并修改

特性:
    - prd.md 只读,用户权威信息来源
    - 工作目录隔离对话过程
    - 智能合并 specs/ 模块规范
    - 自动验证项目结构

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

# 检查运行模式
IS_PROJECT_VIEW_MODE=false

if [[ -z "$PRD_FILE" ]]; then
    # 无参数模式：检查 .morty 目录是否存在
    if [[ -d ".morty" ]]; then
        IS_PROJECT_VIEW_MODE=true
        log INFO "进入项目进展查看模式（无 PRD 文件）"
        log INFO "将基于现有 .morty/ 目录进行交互"
    else
        log ERROR "需要 PRD 文件，或确保 .morty/ 目录存在"
        show_help
        exit 1
    fi
else
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
fi

log INFO "╔════════════════════════════════════════════════════════════╗"
log INFO "║            MORTY FIX 模式 - PRD 迭代改进                  ║"
log INFO "╚════════════════════════════════════════════════════════════╝"
log INFO ""

if [[ "$IS_PROJECT_VIEW_MODE" == true ]]; then
    log INFO "模式: 项目进展查看"
    log INFO ""
else
    log INFO "PRD 文件(只读): $PRD_FILE"
    log INFO ""
fi

# 检查系统提示词是否存在
if [[ ! -f "$FIX_SYSTEM_PROMPT" ]]; then
    log ERROR "Fix 模式系统提示词未找到: $FIX_SYSTEM_PROMPT"
    log ERROR "请先运行安装: ./install.sh"
    exit 1
fi

# 检查是否首次运行
IS_FIRST_RUN=false
if [[ ! -d ".morty" ]]; then
    IS_FIRST_RUN=true
    log INFO "检测到首次运行 - 将创建 .morty/ 目录"
else
    log INFO "检测到已有 .morty/ 目录 - 将合并修改"
    log INFO "  - fix_plan.md, AGENT.md, PROMPT.md: 直接重建"
    log INFO "  - specs/*.md: 合并修改"
fi
log INFO ""

# 创建工作目录
if [[ -d "$WORK_DIR" ]]; then
    log WARN "工作目录已存在,清理中: $WORK_DIR"
    rm -rf "$WORK_DIR"
fi
mkdir -p "$WORK_DIR"

log INFO "工作目录: $WORK_DIR"
log INFO ""

# 如果是再次运行,复制现有 specs/ 到工作目录
if [[ "$IS_FIRST_RUN" == false ]] && [[ -d ".morty/specs" ]]; then
    log INFO "复制现有 specs/ 到工作目录..."
    cp -r .morty/specs "$WORK_DIR/"
    log INFO ""
fi

# 读取系统提示词
SYSTEM_PROMPT_CONTENT=$(cat "$FIX_SYSTEM_PROMPT")

# 构建交互式提示词
if [[ "$IS_PROJECT_VIEW_MODE" == true ]]; then
    # 项目进展查看模式：读取现有 .morty 目录内容
    CURRENT_PROMPT=$(cat ".morty/PROMPT.md" 2>/dev/null || echo "(暂无)")
    CURRENT_AGENT=$(cat ".morty/AGENT.md" 2>/dev/null || echo "(暂无)")
    CURRENT_FIX_PLAN=$(cat ".morty/fix_plan.md" 2>/dev/null || echo "(暂无)")
    CURRENT_SPECS=$(find .morty/specs -name "*.md" 2>/dev/null | head -5 | while read f; do echo "- $f"; done || echo "(暂无)")

    INTERACTIVE_PROMPT=$(cat << EOF
$SYSTEM_PROMPT_CONTENT

---

# 当前运行状态

**运行模式**: 项目进展查看模式（无 PRD 文件）
**工作目录**: \`$WORK_DIR/\`
**项目目录**: \`.morty/\`

---

# 当前项目状态

## PROMPT.md
\`\`\`markdown
$CURRENT_PROMPT
\`\`\`

## AGENT.md
\`\`\`markdown
$CURRENT_AGENT
\`\`\`

## fix_plan.md
\`\`\`markdown
$CURRENT_FIX_PLAN
\`\`\`

## specs/ 目录
$CURRENT_SPECS

---

# 工作目录说明

你的所有工作文件都应该在 \`$WORK_DIR/\` 中创建。

**重要**: 此模式下不要修改 .morty/ 目录中的文件，除非你明确知道需要做什么修改。

---

**指令**:
1. 分析当前项目状态和进展
2. 回答用户关于项目的问题
3. 如果需要，可以建议对 .morty/ 目录的修改（但不要直接修改）
4. 帮助用户理解当前任务进度和下一步行动

用户可以随时提问，例如：
- "当前项目进展如何？"
- "还有哪些任务待完成？"
- "请解释当前的设计决策"
- "建议下一步做什么？"

开始对话!
EOF
)
else
    # 正常模式：基于 PRD 文件
    CURRENT_PRD_CONTENT=$(cat "$PRD_FILE")

    INTERACTIVE_PROMPT=$(cat << EOF
$SYSTEM_PROMPT_CONTENT

---

# 当前运行状态

**运行模式**: $(if [[ "$IS_FIRST_RUN" == true ]]; then echo "首次运行"; else echo "再次运行(合并模式)"; fi)
**工作目录**: \`$WORK_DIR/\`
**PRD 文件**: \`$PRD_FILE\` (只读,不要修改)

---

# 当前 PRD 内容

PRD 文件名: **$PRD_FILENAME**

\`\`\`markdown
$CURRENT_PRD_CONTENT
\`\`\`

---

# 工作目录说明

你的所有工作文件都应该在 \`$WORK_DIR/\` 中创建:
- 对话中的临时文件
- 知识积累文件
- specs/ 模块规范(如果是再次运行,已有的 specs/ 已复制到工作目录)

**重要**: 不要修改用户的 prd.md 文件,它是只读的权威信息来源。

---

**指令**:
1. 分析 PRD 内容
2. 在工作目录中进行对话和知识积累
3. 最后生成 .morty/ 目录结构(从工作目录)
4. 运行项目结构验证

开始对话!
EOF
)
fi

log INFO "启动交互式 Fix 模式会话..."
log INFO ""
log INFO "使用说明:"
log INFO "  - Claude 会在工作目录 $WORK_DIR/ 中工作"
if [[ "$IS_PROJECT_VIEW_MODE" == true ]]; then
    log INFO "  - 当前模式: 项目进展查看（基于现有 .morty/ 目录）"
    log INFO "  - 可以询问项目进展、任务状态、设计决策等问题"
else
    log INFO "  - prd.md 是只读的,不会被修改"
    log INFO "  - 对话结束后会生成/更新 .morty/ 目录"
fi
log INFO "  - 会话在 Claude 输出时结束: <!-- FIX_MODE_COMPLETE -->"
log INFO ""

# 只在非 loop 监控模式下等待用户输入
if [[ -z "$MORTY_LOOP_MONITOR" ]]; then
    log INFO "按 Enter 开始交互式会话..."
    read -r
fi

# 将提示词保存到文件
PROMPT_FILE="$WORK_DIR/fix_prompt.md"
echo "$INTERACTIVE_PROMPT" > "$PROMPT_FILE"

log INFO "在 Fix 模式下启动 Claude Code..."
log INFO ""

# 构建 Claude 命令
CLAUDE_ARGS=(
    "$CLAUDE_CMD"
    "--dangerously-skip-permissions"
    "--allowedTools" "Read" "Write" "Glob" "Grep" "WebSearch" "WebFetch" "Edit"
)

# 交互式执行 Claude Code
cat "$PROMPT_FILE" | "${CLAUDE_ARGS[@]}"

CLAUDE_EXIT_CODE=$?

log INFO ""
log INFO "Fix 模式会话完成(退出码: $CLAUDE_EXIT_CODE)"
log INFO ""

# 检查 Claude 是否成功完成
if [[ $CLAUDE_EXIT_CODE -ne 0 ]]; then
    log ERROR "Claude Code 以错误码退出: $CLAUDE_EXIT_CODE"
    log ERROR "Fix 模式会话未成功完成"
    log INFO "工作目录保留在: $WORK_DIR"
    exit 1
fi

# 验证 .morty/ 目录是否已生成
if [[ ! -d ".morty" ]]; then
    log ERROR ".morty/ 目录未生成"
    log ERROR "Claude 应该在对话结束时创建此目录"
    log INFO "工作目录保留在: $WORK_DIR"
    exit 1
fi

log INFO "验证 .morty/ 目录结构..."
log INFO ""

# 运行项目结构检查
if morty_check_project_structure "."; then
    log INFO ""
    log SUCCESS "╔════════════════════════════════════════════════════════════╗"
    log SUCCESS "║              循环初始化成功!                               ║"
    log SUCCESS "╚════════════════════════════════════════════════════════════╝"
    log INFO ""
    log INFO "现在可以进入循环阶段:"
    log INFO "  运行 'morty loop' 开始开发循环"
    log INFO ""
    
    # 清理工作目录
    log INFO "清理工作目录..."
    rm -rf "$WORK_DIR"
    log INFO "工作目录已清理"
else
    log INFO ""
    log ERROR "╔════════════════════════════════════════════════════════════╗"
    log ERROR "║          项目结构验证失败!                                 ║"
    log ERROR "╚════════════════════════════════════════════════════════════╝"
    log INFO ""
    log ERROR ".morty/ 目录结构不符合要求"
    log INFO "请查看上述验证错误"
    log INFO ""
    log INFO "工作目录保留在: $WORK_DIR"
    log INFO "你可以手动检查和修复问题"
    exit 1
fi