#!/bin/bash
# test_cli_parse_args.sh - 测试 cli_parse_args 函数

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 加载库文件
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/cli_parse_args.sh"

# 测试计数器
TESTS_PASSED=0
TESTS_FAILED=0

# 测试函数
run_test() {
    local test_name="$1"
    shift

    echo -n "测试: $test_name ... "

    if "$@" >/dev/null 2>&1; then
        echo -e "${GREEN}通过${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}失败${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# 测试用例

# 测试 1: 基本位置参数
test_basic_positional() {
    cli_parse_args "doing"
    [[ $(cli_get_positional_arg 0) == "doing" ]]
}

# 测试 2: 多个位置参数
test_multiple_positional() {
    cli_parse_args "doing" "config" "job_1"
    [[ $(cli_get_positional_arg 0) == "doing" &&
       $(cli_get_positional_arg 1) == "config" &&
       $(cli_get_positional_arg 2) == "job_1" ]]
}

# 测试 3: 长选项带值 (--option value)
test_long_option_value() {
    cli_parse_args "doing" "--module" "config"
    cli_has_option "--module" && [[ $(cli_get_option_value "--module") == "config" ]]
}

# 测试 4: 长选项=赋值 (--option=value)
test_long_option_equals() {
    cli_parse_args "doing" "--module=config" "--job=job_1"
    [[ $(cli_get_option_value "--module") == "config" &&
       $(cli_get_option_value "--job") == "job_1" ]]
}

# 测试 5: 长标志 (--flag)
test_long_flag() {
    cli_parse_args "doing" "--restart"
    cli_has_option "--restart"
}

# 测试 6: 短选项带值 (-a value)
test_short_option_value() {
    cli_parse_args "reset" "-l" "5"
    cli_has_option "-l" && [[ $(cli_get_option_value "-l") == "5" ]]
}

# 测试 7: 短标志 (-a)
test_short_flag() {
    cli_parse_args "stat" "-w"
    cli_has_option "-w"
}

# 测试 8: 多个短标志组合 (-abc)
test_multiple_short_flags() {
    cli_parse_args "cmd" "-abc"
    cli_has_option "-a" && cli_has_option "-b" && cli_has_option "-c"
}

# 测试 9: 混合参数
test_mixed_args() {
    cli_parse_args "doing" "--restart" "--module" "config" "--job=job_1"
    [[ $(cli_get_positional_arg 0) == "doing" ]] &&
    cli_has_option "--restart" &&
    [[ $(cli_get_option_value "--module") == "config" ]] &&
    [[ $(cli_get_option_value "--job") == "job_1" ]]
}

# 测试 10: 复杂场景 - morty doing --restart --module config
test_complex_doing_restart() {
    cli_parse_args "doing" "--restart" "--module" "config"
    [[ $(cli_get_positional_arg 0) == "doing" ]] &&
       cli_has_option "--restart" &&
       [[ $(cli_get_option_value "--module") == "config" ]]
}

# 测试 11: 默认值
test_default_value() {
    cli_parse_args "stat"
    [[ $(cli_get_option_value "-l" "10") == "10" ]]
}

# 测试 12: 位置参数数量
test_positional_count() {
    cli_parse_args "a" "b" "c"
    [[ $(cli_get_positional_count) -eq 3 ]]
}

# 测试 13: 获取所有位置参数
test_get_all_positional() {
    cli_parse_args "doing" "arg1" "arg2"
    local args=($(cli_get_positional_args))
    [[ ${#args[@]} -eq 3 && ${args[0]} == "doing" ]]
}

# 测试 14: 验证器场景 - morty doing --restart --module config
test_validator_doing_restart_module() {
    cli_parse_args "doing" "--restart" "--module" "config"
    [[ $(cli_get_positional_arg 0) == "doing" ]] &&
    cli_has_option "--restart" &&
    [[ $(cli_get_option_value "--module") == "config" ]]
}

# 测试 15: 验证器场景 - morty stat -w
test_validator_stat_watch() {
    cli_parse_args "stat" "-w"
    [[ $(cli_get_positional_arg 0) == "stat" ]] &&
    cli_has_option "-w"
}

# 测试 16: 验证器场景 - morty reset -l
test_validator_reset_list() {
    cli_parse_args "reset" "-l"
    [[ $(cli_get_positional_arg 0) == "reset" ]] &&
    cli_has_option "-l"
}

# 测试 17: 验证器场景 - morty reset -l 5
test_validator_reset_list_count() {
    cli_parse_args "reset" "-l" "5"
    [[ $(cli_get_positional_arg 0) == "reset" ]] &&
    [[ $(cli_get_option_value "-l") == "5" ]]
}

# 测试 18: 验证器场景 - morty reset -c abc123
test_validator_reset_commit() {
    cli_parse_args "reset" "-c" "abc123"
    [[ $(cli_get_positional_arg 0) == "reset" ]] &&
    [[ $(cli_get_option_value "-c") == "abc123" ]]
}

# 运行所有测试
echo "=================================="
echo "cli_parse_args 函数测试套件"
echo "=================================="
echo ""

run_test "基本位置参数" test_basic_positional
run_test "多个位置参数" test_multiple_positional
run_test "长选项带值 (--option value)" test_long_option_value
run_test "长选项=赋值 (--option=value)" test_long_option_equals
run_test "长标志 (--flag)" test_long_flag
run_test "短选项带值 (-a value)" test_short_option_value
run_test "短标志 (-a)" test_short_flag
run_test "多个短标志组合 (-abc)" test_multiple_short_flags
run_test "混合参数" test_mixed_args
run_test "复杂场景 - doing --restart --module config" test_complex_doing_restart
run_test "默认值" test_default_value
run_test "位置参数数量" test_positional_count
run_test "获取所有位置参数" test_get_all_positional
run_test "验证场景 - doing --restart --module config" test_validator_doing_restart_module
run_test "验证场景 - stat -w" test_validator_stat_watch
run_test "验证场景 - reset -l" test_validator_reset_list
run_test "验证场景 - reset -l 5" test_validator_reset_list_count
run_test "验证场景 - reset -c abc123" test_validator_reset_commit

echo ""
echo "=================================="
echo "测试结果: $TESTS_PASSED 通过, $TESTS_FAILED 失败"
echo "=================================="

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}所有测试通过!${NC}"
    exit 0
else
    echo -e "${RED}有测试失败!${NC}"
    exit 1
fi
