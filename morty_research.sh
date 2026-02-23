#!/bin/bash
# Morty Research Mode - 交互式代码库/文档库研究

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/config.sh"

# 加载配置
config_load

# 从配置获取 AI CLI 命令
AI_CLI=$(config_get "cli.command" "claude")
RESEARCH_PROMPT="$SCRIPT_DIR/../prompts/research.md"

show_help() {
    cat << 'EOF'
Morty Research 模式 - 交互式代码库/文档库研究

用法: morty research [调查主题]

参数:
    调查主题          可选，研究的主题名称（用于生成报告文件名）

描述:
    Research 模式启动一个交互式 Claude Code 会话来:
    1. 探索当前工作空间的代码库或文档库
    2. 分析目录结构、核心功能、处理流程等
    3. 将研究结果记录到 .morty/research/ 目录

示例:
    morty research                    # 启动研究模式
    morty research "api架构"          # 指定研究主题
    morty research "数据库设计"        # 研究数据库相关代码

工作流程:
    1. 启动交互式 Claude Code 会话
    2. 在对话中探索工作空间
    3. 研究结果自动保存到 .morty/research/[主题].md
    4. 用户主动结束会话后退出

EOF
}

# 解析参数
RESEARCH_TOPIC="${1:-未指定}"

if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    show_help
    exit 0
fi

# 检查提示词文件
if [[ ! -f "$RESEARCH_PROMPT" ]]; then
    log ERROR "Research 模式提示词未找到: $RESEARCH_PROMPT"
    exit 1
fi

log INFO "启动 Research 模式..."
log INFO "调查主题: $RESEARCH_TOPIC"

# 读取系统提示词
SYSTEM_PROMPT=$(cat "$RESEARCH_PROMPT")

# 构建交互式提示词
INTERACTIVE_PROMPT="$SYSTEM_PROMPT

---

# 当前运行状态

**工作目录**: $(pwd)
**调查主题**: $RESEARCH_TOPIC

---

开始对话!"

log INFO "启动 Claude Code 交互式会话..."
log INFO ""

# 确保 .morty/research/ 目录存在
config_ensure_work_dir || {
    log ERROR "Failed to initialize work directory"
    exit 1
}

# 构建 Claude 命令 (使用 Plan 模式)
CLAUDE_CMD="$AI_CLI --permission-mode plan"

echo "$INTERACTIVE_PROMPT" | $CLAUDE_CMD

log INFO ""
log SUCCESS "Research 模式会话结束"
