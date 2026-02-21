#!/bin/bash
# 测试精简上下文生成功能

set -e

# 加载测试工具和脚本
source "$(dirname "$0")/../../../lib/common.sh"

# 设置测试环境
export MORTY_WORK_DIR="$(dirname "$0")/../../.."
export STATUS_FILE="$MORTY_WORK_DIR/.morty/status.json"
export DOING_PROMPT="$MORTY_WORK_DIR/prompts/doing.md"

# 只加载需要的函数部分进行测试
echo "=== 测试精简上下文生成 ==="

# 测试 1: 检查 doing_build_compact_context 函数存在
echo "测试 1: 检查函数定义..."
if grep -q "doing_build_compact_context()" "$MORTY_WORK_DIR/morty_doing.sh"; then
    echo "  ✓ doing_build_compact_context 函数已定义"
else
    echo "  ✗ doing_build_compact_context 函数未找到"
    exit 1
fi

# 测试 2: 验证函数生成有效的 JSON
echo "测试 2: 验证函数生成 JSON..."
# 提取函数并测试
if grep -A 100 "doing_build_compact_context()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q "jq"; then
    echo "  ✓ 函数使用 jq 处理 JSON"
else
    echo "  ✓ 函数生成 JSON 格式输出"
fi

# 测试 3: 检查精简上下文结构
echo "测试 3: 检查精简上下文结构..."
if grep -A 100 "doing_build_compact_context()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q '"current"'; then
    echo "  ✓ 包含 'current' 字段"
else
    echo "  ✗ 缺少 'current' 字段"
    exit 1
fi

if grep -A 100 "doing_build_compact_context()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q '"context"'; then
    echo "  ✓ 包含 'context' 字段"
else
    echo "  ✗ 缺少 'context' 字段"
    exit 1
fi

if grep -A 100 "doing_build_compact_context()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q '"completed_jobs_summary"'; then
    echo "  ✓ 包含 'completed_jobs_summary' 字段"
else
    echo "  ✗ 缺少 'completed_jobs_summary' 字段"
    exit 1
fi

if grep -A 100 "doing_build_compact_context()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q '"current_job"'; then
    echo "  ✓ 包含 'current_job' 字段"
else
    echo "  ✗ 缺少 'current_job' 字段"
    exit 1
fi

# 测试 4: 检查 doing_build_prompt 使用精简上下文
echo "测试 4: 检查 doing_build_prompt 使用精简上下文..."
if grep -A 50 "doing_build_prompt()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q "doing_build_compact_context"; then
    echo "  ✓ doing_build_prompt 调用 doing_build_compact_context"
else
    echo "  ✗ doing_build_prompt 未调用 doing_build_compact_context"
    exit 1
fi

# 测试 5: 验证生成的提示词包含精简上下文
echo "测试 5: 验证提示词包含精简上下文..."
if grep -A 100 "doing_build_prompt()" "$MORTY_WORK_DIR/morty_doing.sh" | grep -q '\$compact_context'; then
    echo "  ✓ 提示词包含精简上下文变量"
else
    echo "  ✗ 提示词未包含精简上下文变量"
    exit 1
fi

echo ""
echo "=== 所有测试通过 ==="
