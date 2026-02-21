#!/usr/bin/env bash
#
# test_json_logging.sh - 测试 JSON 格式日志功能
#

# 获取脚本目录
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "${TEST_DIR}")"
LIB_DIR="${PROJECT_DIR}/lib"

# 引入日志模块
source "${LIB_DIR}/logging.sh"

# 测试目录
TEST_LOG_DIR="${TEST_DIR}/test_logs"
mkdir -p "${TEST_LOG_DIR}"

# 覆盖日志配置
LOG_DIR="${TEST_LOG_DIR}"
LOG_MAIN_FILE="${TEST_LOG_DIR}/test.log"
LOG_LEVEL=0  # DEBUG

# 测试结果
TESTS_PASSED=0
TESTS_FAILED=0

# 测试辅助函数
assert_contains() {
    local expected="$1"
    local actual="$2"
    local test_name="$3"

    if [[ "${actual}" == *"${expected}"* ]]; then
        echo "✅ PASS: ${test_name}"
        ((TESTS_PASSED++))
        return 0
    else
        echo "❌ FAIL: ${test_name}"
        echo "   Expected to contain: ${expected}"
        echo "   Actual: ${actual}"
        ((TESTS_FAILED++))
        return 1
    fi
}

assert_valid_json() {
    local json="$1"
    local test_name="$2"

    if echo "${json}" | jq -e . >/dev/null 2>&1; then
        echo "✅ PASS: ${test_name}"
        ((TESTS_PASSED++))
        return 0
    else
        echo "❌ FAIL: ${test_name}"
        echo "   Invalid JSON: ${json}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# 测试 1: JSON 格式基本输出
test_json_format_basic() {
    echo ""
    echo "=== Test 1: JSON 格式基本输出 ==="

    # 切换到 JSON 格式
    log_set_format json

    # 清空日志文件
    > "${LOG_MAIN_FILE}"

    # 写入日志
    log_info "测试消息"

    # 读取日志内容
    local log_content
    log_content=$(cat "${LOG_MAIN_FILE}")

    # 验证是有效的 JSON
    assert_valid_json "${log_content}" "JSON 格式有效性"

    # 验证包含必要字段
    assert_contains '"timestamp"' "${log_content}" "包含 timestamp 字段"
    assert_contains '"level":"INFO"' "${log_content}" "包含 level 字段"
    assert_contains '"message":"测试消息"' "${log_content}" "包含 message 字段"
}

# 测试 2: 特殊字符转义
test_json_escape() {
    echo ""
    echo "=== Test 2: 特殊字符转义 ==="

    log_set_format json
    > "${LOG_MAIN_FILE}"

    # 测试各种特殊字符
    log_info '包含"引号的"消息'
    log_info "包含\\反斜杠的消息"
    log_info "包含\n换行\t制表符的消息"

    local log_content
    log_content=$(cat "${LOG_MAIN_FILE}")

    # 验证是有效的 JSON（应该能解析）
    while IFS= read -r line; do
        assert_valid_json "${line}" "特殊字符行是有效 JSON"
    done <<< "${log_content}"
}

# 测试 3: 上下文数据序列化
test_context_serialization() {
    echo ""
    echo "=== Test 3: 上下文数据序列化 ==="

    log_set_format json
    > "${LOG_MAIN_FILE}"

    # 测试 key=value 格式的上下文
    log_info "用户登录" "user=admin,action=login"

    local log_content
    log_content=$(cat "${LOG_MAIN_FILE}")

    assert_valid_json "${log_content}" "上下文序列化为有效 JSON"
    assert_contains '"user":"admin"' "${log_content}" "上下文包含 user 字段"
    assert_contains '"action":"login"' "${log_content}" "上下文包含 action 字段"
}

# 测试 4: log_structured 函数
test_log_structured() {
    echo ""
    echo "=== Test 4: log_structured 函数 ==="

    > "${LOG_MAIN_FILE}"

    # 测试 JSON 对象输入
    log_structured INFO '{"event":"user_login","user_id":"12345"}'

    local log_content
    log_content=$(cat "${LOG_MAIN_FILE}")

    assert_valid_json "${log_content}" "结构化日志是有效 JSON"
    assert_contains '"event":"user_login"' "${log_content}" "包含 event 字段"
    assert_contains '"user_id":"12345"' "${log_content}" "包含 user_id 字段"
    assert_contains '"timestamp"' "${log_content}" "包含 timestamp 字段"
    assert_contains '"level":"INFO"' "${log_content}" "包含 level 字段"
}

# 测试 5: 格式切换
test_format_switching() {
    echo ""
    echo "=== Test 5: 格式切换 ==="

    > "${LOG_MAIN_FILE}"

    # 切换到文本格式
    log_set_format text
    log_info "文本格式消息"

    local text_content
    text_content=$(cat "${LOG_MAIN_FILE}")

    # 验证文本格式
    if [[ "${text_content}" == \[*INFO\]*文本格式消息* ]]; then
        echo "✅ PASS: 文本格式正确"
        ((TESTS_PASSED++))
    else
        echo "❌ FAIL: 文本格式不正确: ${text_content}"
        ((TESTS_FAILED++))
    fi

    # 切换到 JSON 格式
    log_set_format json
    log_info "JSON格式消息"

    local json_content
    json_content=$(tail -n 1 "${LOG_MAIN_FILE}")

    assert_valid_json "${json_content}" "切换到 JSON 格式有效"
    assert_contains '"message":"JSON格式消息"' "${json_content}" "JSON 消息内容正确"
}

# 测试 6: 关联数组上下文
test_associative_array_context() {
    echo ""
    echo "=== Test 6: 关联数组上下文 ==="

    > "${LOG_MAIN_FILE}"

    # 创建关联数组
    declare -A user_data
    user_data["username"]="testuser"
    user_data["role"]="admin"
    user_data["department"]="engineering"

    # 使用关联数组作为上下文
    log_structured INFO "user_data"

    local log_content
    log_content=$(cat "${LOG_MAIN_FILE}")

    assert_valid_json "${log_content}" "关联数组上下文是有效 JSON"
    assert_contains '"username":"testuser"' "${log_content}" "包含 username"
    assert_contains '"role":"admin"' "${log_content}" "包含 role"
    assert_contains '"department":"engineering"' "${log_content}" "包含 department"
}

# 测试 7: 模块和 Job 上下文
test_module_job_context() {
    echo ""
    echo "=== Test 7: 模块和 Job 上下文 ==="

    > "${LOG_MAIN_FILE}"

    # 开始 Job 上下文
    log_job_start "test_module" "test_job"

    # 写入日志
    log_info "Job 内消息"

    # 结束 Job
    log_job_end

    local log_content
    log_content=$(cat "${LOG_MAIN_FILE}")

    # 验证包含模块和 Job 信息
    assert_contains '"module":"test_module"' "${log_content}" "包含 module 字段"
    assert_contains '"job":"test_job"' "${log_content}" "包含 job 字段"
}

# 测试 8: jq 可解析性
test_jq_parsable() {
    echo ""
    echo "=== Test 8: jq 可解析性 ==="

    log_set_format json
    > "${LOG_MAIN_FILE}"

    log_info "可解析测试消息" "key1=value1,key2=value2"
    log_warn "警告消息"
    log_error "错误消息"

    # 使用 jq 提取所有消息
    local messages
    messages=$(jq -r '.message' "${LOG_MAIN_FILE}" 2>/dev/null)

    if [[ "${messages}" == *"可解析测试消息"* && \
          "${messages}" == *"警告消息"* && \
          "${messages}" == *"错误消息"* ]]; then
        echo "✅ PASS: jq 可以正确解析所有日志"
        ((TESTS_PASSED++))
    else
        echo "❌ FAIL: jq 解析失败"
        echo "   Messages: ${messages}"
        ((TESTS_FAILED++))
    fi

    # 使用 jq 提取特定级别的日志
    local error_msgs
    error_msgs=$(jq -r 'select(.level == "ERROR") | .message' "${LOG_MAIN_FILE}" 2>/dev/null)

    if [[ "${error_msgs}" == "错误消息" ]]; then
        echo "✅ PASS: jq 可以筛选 ERROR 级别日志"
        ((TESTS_PASSED++))
    else
        echo "❌ FAIL: jq 筛选失败"
        ((TESTS_FAILED++))
    fi
}

# 主测试流程
main() {
    echo "================================"
    echo "JSON 日志功能测试"
    echo "================================"

    # 检查 jq 是否可用
    if ! command -v jq >/dev/null 2>&1; then
        echo "警告: jq 未安装，部分测试将跳过"
    fi

    # 执行所有测试
    test_json_format_basic
    test_json_escape
    test_context_serialization
    test_log_structured
    test_format_switching
    test_associative_array_context
    test_module_job_context
    test_jq_parsable

    # 清理
    rm -rf "${TEST_LOG_DIR}"

    # 恢复默认格式
    log_set_format text

    # 输出结果
    echo ""
    echo "================================"
    echo "测试结果: ${TESTS_PASSED} 通过, ${TESTS_FAILED} 失败"
    echo "================================"

    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        echo "🎉 所有测试通过！"
        exit 0
    else
        echo "💥 有测试失败"
        exit 1
    fi
}

main "$@"
