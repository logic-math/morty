#!/bin/bash
# 测试 JSON 格式解析功能的测试脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 使用绝对路径确保正确找到文件
MORTY_DIR="/opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/morty"

# 基本的日志函数
log() {
    local level="$1"
    shift
    echo "[$level] $*"
}

# 测试计数
TESTS_PASSED=0
TESTS_FAILED=0

# 测试辅助函数
assert_eq() {
    local expected="$1"
    local actual="$2"
    local test_name="$3"

    if [[ "$expected" == "$actual" ]]; then
        log SUCCESS "✓ $test_name"
        ((TESTS_PASSED++)) || true
    else
        log ERROR "✗ $test_name"
        log INFO "  期望: $expected"
        log INFO "  实际: $actual"
        ((TESTS_FAILED++)) || true
    fi
}

# 手动定义被测试的函数
doing_parse_execution_result() {
    local output_file="$1"

    # 检查输出文件是否存在
    if [[ ! -f "$output_file" ]]; then
        echo "status=FAILED;tasks_completed=0;tasks_total=0;summary=输出文件不存在"
        return 1
    fi

    # 检查 jq 是否可用
    if command -v jq &> /dev/null; then
        # 尝试解析 JSON 格式的输出
        local json_valid=false
        if jq empty "$output_file" 2>/dev/null; then
            json_valid=true
        fi

        if [[ "$json_valid" == true ]]; then
            # 尝试从 JSON 中提取 RALPH_STATUS 字段
            local ralph_json=$(jq -r '.ralph_status // empty' "$output_file" 2>/dev/null)

            if [[ -n "$ralph_json" && "$ralph_json" != "null" ]]; then
                # 从嵌套的 ralph_status 对象中提取字段
                local result_status=$(echo "$ralph_json" | jq -r '.status // "UNKNOWN"')
                local tasks_completed=$(echo "$ralph_json" | jq -r '.tasks_completed // 0')
                local tasks_total=$(echo "$ralph_json" | jq -r '.tasks_total // 0')
                local summary=$(echo "$ralph_json" | jq -r '.summary // "未知"')
                local module=$(echo "$ralph_json" | jq -r '.module // ""')
                local job=$(echo "$ralph_json" | jq -r '.job // ""')

                echo "status=${result_status};tasks_completed=${tasks_completed};tasks_total=${tasks_total};summary=${summary};module=${module};job=${job}"
                return 0
            fi

            # 尝试直接从顶层字段提取（旧格式兼容）
            local result_status=$(jq -r '.status // "UNKNOWN"' "$output_file" 2>/dev/null)
            local tasks_completed=$(jq -r '.tasks_completed // 0' "$output_file" 2>/dev/null)
            local tasks_total=$(jq -r '.tasks_total // 0' "$output_file" 2>/dev/null)
            local summary=$(jq -r '.summary // "未知"' "$output_file" 2>/dev/null)
            local module=$(jq -r '.module // ""' "$output_file" 2>/dev/null)
            local job=$(jq -r '.job // ""' "$output_file" 2>/dev/null)

            if [[ "$result_status" != "UNKNOWN" && -n "$result_status" ]]; then
                echo "status=${result_status};tasks_completed=${tasks_completed};tasks_total=${tasks_total};summary=${summary};module=${module};job=${job}"
                return 0
            fi
        fi
    fi

    # JSON 解析失败或 jq 不可用，回退到文本解析
    _doing_parse_execution_result_text "$output_file"
}

_doing_parse_execution_result_text() {
    local output_file="$1"

    local result_status=""
    local tasks_completed=0
    local tasks_total=0
    local summary=""
    local module=""
    local job=""

    # 提取 RALPH_STATUS 块
    local ralph_status=$(grep -A10 "RALPH_STATUS" "$output_file" 2>/dev/null | grep -v "END_RALPH_STATUS" | head -12)

    if [[ -z "$ralph_status" ]]; then
        echo "status=UNKNOWN;tasks_completed=0;tasks_total=0;summary=无RALPH_STATUS"
        return 0
    fi

    # 解析 JSON 字段
    result_status=$(echo "$ralph_status" | grep -o '"status": *"[^"]*"' | cut -d'"' -f4)
    tasks_completed=$(echo "$ralph_status" | grep -o '"tasks_completed": *[0-9]*' | grep -o '[0-9]*')
    tasks_total=$(echo "$ralph_status" | grep -o '"tasks_total": *[0-9]*' | grep -o '[0-9]*')
    summary=$(echo "$ralph_status" | grep -o '"summary": *"[^"]*"' | cut -d'"' -f4)
    module=$(echo "$ralph_status" | grep -o '"module": *"[^"]*"' | cut -d'"' -f4)
    job=$(echo "$ralph_status" | grep -o '"job": *"[^"]*"' | cut -d'"' -f4)

    # 输出解析结果
    echo "status=${result_status:-UNKNOWN};tasks_completed=${tasks_completed:-0};tasks_total=${tasks_total:-0};summary=${summary:-未知};module=${module:-};job=${job:-}"
}

# 创建测试用的模拟输出文件
create_mock_json_output() {
    local output_file="$1"
    cat > "$output_file" << 'EOF'
{
  "ralph_status": {
    "module": "doing",
    "job": "job_6",
    "status": "COMPLETED",
    "tasks_completed": 5,
    "tasks_total": 5,
    "loop_count": 1,
    "debug_issues": 0,
    "summary": "Job completed successfully"
  }
}
EOF
}

create_mock_flat_json_output() {
    local output_file="$1"
    cat > "$output_file" << 'EOF'
{
  "module": "doing",
  "job": "job_6",
  "status": "COMPLETED",
  "tasks_completed": 5,
  "tasks_total": 5,
  "loop_count": 1,
  "debug_issues": 0,
  "summary": "Job completed successfully"
}
EOF
}

create_mock_text_output() {
    local output_file="$1"
    cat > "$output_file" << 'EOF'
Some log output here

<!-- RALPH_STATUS -->
{
  "module": "doing",
  "job": "job_6",
  "status": "COMPLETED",
  "tasks_completed": 5,
  "tasks_total": 5,
  "loop_count": 1,
  "debug_issues": 0,
  "summary": "Job completed successfully"
}
<!-- END_RALPH_STATUS -->
EOF
}

log INFO "================================"
log INFO "测试: JSON 格式解析功能"
log INFO "================================"

# 测试 1: 嵌套 ralph_status JSON 格式
log INFO ""
log INFO "测试 1: 嵌套 ralph_status JSON 格式"
TMP_FILE=$(mktemp)
create_mock_json_output "$TMP_FILE"
RESULT=$(doing_parse_execution_result "$TMP_FILE")
rm -f "$TMP_FILE"

STATUS=$(echo "$RESULT" | grep -o 'status=[^;]*' | cut -d'=' -f2)
TASKS_COMPLETED=$(echo "$RESULT" | grep -o 'tasks_completed=[^;]*' | cut -d'=' -f2)
assert_eq "COMPLETED" "$STATUS" "嵌套 JSON 格式 - 状态解析"
assert_eq "5" "$TASKS_COMPLETED" "嵌套 JSON 格式 - 任务完成数"

# 测试 2: 扁平 JSON 格式
log INFO ""
log INFO "测试 2: 扁平 JSON 格式"
TMP_FILE=$(mktemp)
create_mock_flat_json_output "$TMP_FILE"
RESULT=$(doing_parse_execution_result "$TMP_FILE")
rm -f "$TMP_FILE"

STATUS=$(echo "$RESULT" | grep -o 'status=[^;]*' | cut -d'=' -f2)
MODULE=$(echo "$RESULT" | grep -o 'module=[^;]*' | cut -d'=' -f2)
assert_eq "COMPLETED" "$STATUS" "扁平 JSON 格式 - 状态解析"
assert_eq "doing" "$MODULE" "扁平 JSON 格式 - 模块名解析"

# 测试 3: 文本格式回退
log INFO ""
log INFO "测试 3: 文本格式回退解析"
TMP_FILE=$(mktemp)
create_mock_text_output "$TMP_FILE"
RESULT=$(doing_parse_execution_result "$TMP_FILE")
rm -f "$TMP_FILE"

STATUS=$(echo "$RESULT" | grep -o 'status=[^;]*' | cut -d'=' -f2)
SUMMARY=$(echo "$RESULT" | grep -o 'summary=[^;]*' | cut -d'=' -f2)
assert_eq "COMPLETED" "$STATUS" "文本格式回退 - 状态解析"
assert_eq "Job completed successfully" "$SUMMARY" "文本格式回退 - 摘要解析"

# 测试 4: 验证 ai_cli 参数
log INFO ""
log INFO "测试 4: 验证 ai_cli 参数包含 --output-format json"
if grep -q '\-\-output-format' "$MORTY_DIR/morty_doing.sh"; then
    log SUCCESS "✓ ai_cli 调用包含 --output-format json 参数"
    ((TESTS_PASSED++)) || true
else
    log ERROR "✗ ai_cli 调用缺少 --output-format json 参数"
    ((TESTS_FAILED++)) || true
fi

# 测试 5: 验证 jq 解析逻辑存在
log INFO ""
log INFO "测试 5: 验证 jq 解析逻辑存在"
if grep -q 'jq.*ralph_status' "$MORTY_DIR/morty_doing.sh"; then
    log SUCCESS "✓ jq ralph_status 解析逻辑存在"
    ((TESTS_PASSED++)) || true
else
    log ERROR "✗ jq ralph_status 解析逻辑不存在"
    ((TESTS_FAILED++)) || true
fi

# 测试 6: 验证回退逻辑存在
log INFO ""
log INFO "测试 6: 验证文本解析回退逻辑存在"
if grep -q '_doing_parse_execution_result_text' "$MORTY_DIR/morty_doing.sh"; then
    log SUCCESS "✓ 文本解析回退函数存在"
    ((TESTS_PASSED++)) || true
else
    log ERROR "✗ 文本解析回退函数不存在"
    ((TESTS_FAILED++)) || true
fi

# 测试 7: 验证 prompts/doing.md 更新
log INFO ""
log INFO "测试 7: 验证 prompts/doing.md 包含 JSON 格式说明"
if grep -q 'output-format json' "$MORTY_DIR/prompts/doing.md"; then
    log SUCCESS "✓ prompts/doing.md 包含 --output-format json 说明"
    ((TESTS_PASSED++)) || true
else
    log ERROR "✗ prompts/doing.md 缺少 --output-format json 说明"
    ((TESTS_FAILED++)) || true
fi

# 测试 8: 验证 JSON Schema 字段说明
log INFO ""
log INFO "测试 8: 验证 JSON Schema 字段说明"
if grep -q 'status.*tasks_completed.*tasks_total.*summary' "$MORTY_DIR/prompts/doing.md" || \
   (grep -q '"status"' "$MORTY_DIR/prompts/doing.md" && \
    grep -q '"tasks_completed"' "$MORTY_DIR/prompts/doing.md" && \
    grep -q '"summary"' "$MORTY_DIR/prompts/doing.md"); then
    log SUCCESS "✓ prompts/doing.md 包含 JSON Schema 字段说明"
    ((TESTS_PASSED++)) || true
else
    log ERROR "✗ prompts/doing.md 缺少 JSON Schema 字段说明"
    ((TESTS_FAILED++)) || true
fi

# 输出测试结果
log INFO ""
log INFO "================================"
log INFO "测试结果: $TESTS_PASSED 通过, $TESTS_FAILED 失败"
log INFO "================================"

if [[ $TESTS_FAILED -eq 0 ]]; then
    log SUCCESS "所有测试通过!"
    exit 0
else
    log ERROR "有测试失败"
    exit 1
fi
