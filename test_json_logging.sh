#!/usr/bin/env bash
#
# test_json_logging.sh - 测试 JSON 格式日志功能
#

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 引入日志模块
source "${SCRIPT_DIR}/lib/logging.sh"

# 测试配置
export LOG_DIR="${SCRIPT_DIR}/.morty/logs/test_json_$$"
export LOG_MAIN_FILE="${LOG_DIR}/test.log"
export LOG_LEVEL=0  # DEBUG

# 清理函数
cleanup() {
    rm -rf "${LOG_DIR}"
}
trap cleanup EXIT

# 测试计数器
TESTS_PASSED=0
TESTS_FAILED=0

# 测试辅助函数
assert_contains() {
    local haystack="$1"
    local needle="$2"
    local msg="$3"

    if [[ "${haystack}" == *"${needle}"* ]]; then
        echo "✓ PASS: ${msg}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "✗ FAIL: ${msg}"
        echo "  Expected to contain: ${needle}"
        echo "  Got: ${haystack}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

assert_valid_json() {
    local json="$1"
    local msg="$2"

    if echo "${json}" | jq empty 2>/dev/null; then
        echo "✓ PASS: ${msg}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "✗ FAIL: ${msg}"
        echo "  Invalid JSON: ${json}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo "========================================"
echo "JSON 日志功能测试"
echo "========================================"

# 测试 1: JSON 格式基本输出
echo ""
echo "测试 1: JSON 格式基本输出"
log_set_format "json"
log_info "测试 JSON 格式消息"
log_content=$(cat "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "JSON 格式应为有效 JSON"
assert_contains "${log_content}" '"level":"INFO"' "应包含 INFO 级别"
assert_contains "${log_content}" '"message":"测试 JSON 格式消息"' "应包含正确消息"
assert_contains "${log_content}" '"timestamp":' "应包含时间戳"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 2: 特殊字符转义
echo ""
echo "测试 2: 特殊字符转义"
log_info '包含"引号"的消息'
log_info "包含\\反斜杠的消息"
log_info "包含\t制表符的消息"
log_info "包含\n换行的消息"

log_content=$(cat "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
# 验证每条都是有效 JSON
line_num=0
while IFS= read -r line; do
    line_num=$((line_num + 1))
    if [[ -n "${line}" ]]; then
        if echo "${line}" | jq empty 2>/dev/null; then
            echo "✓ PASS: 第 ${line_num} 行是有效 JSON"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo "✗ FAIL: 第 ${line_num} 行不是有效 JSON: ${line}"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    fi
done <<< "${log_content}"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 3: 上下文序列化 - JSON 对象
echo ""
echo "测试 3: 上下文序列化 - JSON 对象"
log_info "带 JSON 上下文的消息" '{"user":"admin","action":"login"}'
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "带 JSON 上下文应为有效 JSON"
assert_contains "${log_content}" '"user":"admin"' "应包含 user 字段"
assert_contains "${log_content}" '"action":"login"' "应包含 action 字段"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 4: 上下文序列化 - key=value 格式
echo ""
echo "测试 4: 上下文序列化 - key=value 格式"
log_info "带 key=value 上下文的消息" "user=admin,action=logout,duration=30"
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "key=value 格式应为有效 JSON"
assert_contains "${log_content}" '"user":"admin"' "应解析 user 字段"
assert_contains "${log_content}" '"action":"logout"' "应解析 action 字段"
assert_contains "${log_content}" '"duration":"30"' "应解析 duration 字段"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 5: log_structured 函数
echo ""
echo "测试 5: log_structured 函数"
log_structured "INFO" '{"event":"user_login","user_id":"12345","ip":"192.168.1.1"}'
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "结构化日志应为有效 JSON"
assert_contains "${log_content}" '"event":"user_login"' "应包含 event 字段"
assert_contains "${log_content}" '"user_id":"12345"' "应包含 user_id 字段"
assert_contains "${log_content}" '"ip":"192.168.1.1"' "应包含 ip 字段"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 6: log_structured 带 key=value 数据
echo ""
echo "测试 6: log_structured 带 key=value 数据"
log_structured "WARN" "component=database,severity=high,query_time=1500"
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "key=value 结构化日志应为有效 JSON"
assert_contains "${log_content}" '"component":"database"' "应包含 component 字段"
assert_contains "${log_content}" '"severity":"high"' "应包含 severity 字段"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 7: 日志格式切换
echo ""
echo "测试 7: 日志格式切换"
log_set_format "text"
log_info "这是一条文本格式消息"
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
if [[ "${log_content}" == \[*\]* ]]; then
    echo "✓ PASS: 文本格式正确"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "✗ FAIL: 文本格式不正确: ${log_content}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

log_set_format "json"
log_info "这是一条 JSON 格式消息"
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "切换回 JSON 格式后应为有效 JSON"
assert_contains "${log_content}" "这是一条 JSON 格式消息" "应包含正确消息内容"

# 清空日志文件
rm -f "${LOG_MAIN_FILE}"

# 测试 8: Job 上下文在 JSON 格式下
echo ""
echo "测试 8: Job 上下文在 JSON 格式下"
log_job_start "test_module" "test_job"
log_info "Job 上下文测试消息"
log_job_end

# 检查主日志
log_content=$(grep "Job 上下文测试" "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
if [[ -n "${log_content}" ]]; then
    assert_valid_json "${log_content}" "Job 上下文消息应为有效 JSON"
    assert_contains "${log_content}" '"module":"test_module"' "应包含 module 字段"
    assert_contains "${log_content}" '"job":"test_job"' "应包含 job 字段"
else
    echo "✗ FAIL: 未找到 Job 上下文测试消息"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# 清空日志
rm -f "${LOG_MAIN_FILE}"
unset _LOG_JOB_MODULE _LOG_JOB_NAME _LOG_JOB_FILE _LOG_JOB_START_TIME

# 测试 9: jq 兼容性测试
echo ""
echo "测试 9: jq 兼容性测试"
log_info "jq 兼容性测试消息" '{"data":{"nested":{"value":123},"array":[1,2,3]}}'
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")

# 使用 jq 提取字段
if command -v jq >/dev/null 2>&1; then
    extracted_msg=$(echo "${log_content}" | jq -r '.message' 2>/dev/null)
    if [[ "${extracted_msg}" == "jq 兼容性测试消息" ]]; then
        echo "✓ PASS: jq 可正确提取 message 字段"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "✗ FAIL: jq 提取 message 字段失败: ${extracted_msg}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    extracted_level=$(echo "${log_content}" | jq -r '.level' 2>/dev/null)
    if [[ "${extracted_level}" == "INFO" ]]; then
        echo "✓ PASS: jq 可正确提取 level 字段"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "✗ FAIL: jq 提取 level 字段失败: ${extracted_level}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # 测试嵌套对象
    nested_value=$(echo "${log_content}" | jq -r '.context.data.nested.value' 2>/dev/null)
    if [[ "${nested_value}" == "123" ]]; then
        echo "✓ PASS: jq 可正确提取嵌套对象值"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "✗ FAIL: jq 提取嵌套对象值失败: ${nested_value}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
else
    echo "⚠ SKIP: jq 未安装，跳过 jq 兼容性测试"
fi

# 测试 10: 边界情况 - 空消息
echo ""
echo "测试 10: 边界情况测试"
rm -f "${LOG_MAIN_FILE}"
log_info ""
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "空消息应为有效 JSON"

# 测试 11: 长消息
echo ""
echo "测试 11: 长消息测试"
rm -f "${LOG_MAIN_FILE}"
long_msg=$(head -c 1000 /dev/zero | tr '\0' 'A')
log_info "${long_msg}"
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "长消息应为有效 JSON"

# 测试 12: 包含 Unicode 的消息
echo ""
echo "测试 12: Unicode 消息测试"
rm -f "${LOG_MAIN_FILE}"
log_info "Unicode 测试: 你好世界"
log_content=$(tail -1 "${LOG_MAIN_FILE}" 2>/dev/null || echo "")
assert_valid_json "${log_content}" "Unicode 消息应为有效 JSON"

# 恢复文本格式
log_set_format "text"

# 测试结果汇总
echo ""
echo "========================================"
echo "测试结果汇总"
echo "========================================"
echo "通过: ${TESTS_PASSED}"
echo "失败: ${TESTS_FAILED}"
echo ""

if [[ ${TESTS_FAILED} -eq 0 ]]; then
    echo "✓ 所有测试通过！"
    exit 0
else
    echo "✗ 有测试失败"
    exit 1
fi
