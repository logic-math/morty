#!/bin/bash
# cli_parse_args.sh - 命令行参数解析库
#
# 使用方法:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/cli_parse_args.sh"
#   cli_parse_args "$@"
#
# 解析结果:
#   CLI_POSITIONAL_ARGS - 位置参数数组
#   CLI_OPTION_ARGS     - 带值的选项 (关联数组)
#   CLI_OPTION_FLAGS    - 无值标志数组
#   CLI_PARSE_ERROR     - 解析错误标志 (0=成功, 1=失败)

# 确保不重复加载
[[ -n "${_CLI_PARSE_ARGS_LOADED_:-}" ]] && return 0
_CLI_PARSE_ARGS_LOADED_=1

# ============================================================================
# 解析结果变量
# ============================================================================

# 位置参数数组
CLI_POSITIONAL_ARGS=()

# 带值的选项 (关联数组)
declare -A CLI_OPTION_ARGS

# 无值标志数组
CLI_OPTION_FLAGS=()

# 解析错误标志
CLI_PARSE_ERROR=0

# ============================================================================
# 核心解析函数
# ============================================================================

# 重置解析结果
# Usage: cli_parse_reset
cli_parse_reset() {
    CLI_POSITIONAL_ARGS=()
    CLI_OPTION_ARGS=()
    CLI_OPTION_FLAGS=()
    CLI_PARSE_ERROR=0
}

# 解析命令行参数
# Usage: cli_parse_args <args...>
# Results stored in: CLI_POSITIONAL_ARGS, CLI_OPTION_ARGS, CLI_OPTION_FLAGS
#
# 支持的参数格式:
#   --option value     # 长选项带值
#   --option=value     # 长选项=赋值
#   --flag             # 长标志
#   -a value           # 短选项带值
#   -abc               # 多个短标志组合
#   -a -b -c           # 多个独立短标志
#   positional         # 位置参数
#
# 示例:
#   cli_parse_args "doing" "--restart" "--module" "config" "--job=job_1"
#   cli_parse_args "stat" "-w"
#   cli_parse_args "reset" "-l" "5"
#
cli_parse_args() {
    cli_parse_reset

    local args=("$@")
    local i=0
    local len=${#args[@]}

    # 调试输出函数（如果可用）
    local debug_fn="${CLI_DEBUG_FN:-}"

    [[ -n "$debug_fn" ]] && $debug_fn "开始解析参数: ${args[*]}"

    while [[ $i -lt $len ]]; do
        local arg="${args[$i]}"

        # 检查是否为长选项 (--option)
        if [[ "$arg" == --* ]]; then
            local option="$arg"
            local value=""

            # 检查是否有 = 赋值 (--option=value)
            if [[ "$option" == *=* ]]; then
                value="${option#*=}"
                option="${option%%=*}"
            fi

            # 检查是否需要取下一个参数作为值
            if [[ -z "$value" && $((i + 1)) -lt $len ]]; then
                local next_arg="${args[$((i + 1))]}"
                # 下一个参数不是选项，则作为值
                if [[ ! "$next_arg" == -* ]]; then
                    value="$next_arg"
                    i=$((i + 1))
                fi
            fi

            if [[ -n "$value" ]]; then
                CLI_OPTION_ARGS["$option"]="$value"
                [[ -n "$debug_fn" ]] && $debug_fn "解析选项: $option = $value"
            else
                CLI_OPTION_FLAGS+=("$option")
                [[ -n "$debug_fn" ]] && $debug_fn "解析标志: $option"
            fi

        # 检查是否为短选项 (-a 或 -abc)
        elif [[ "$arg" == -* ]]; then
            local flags="${arg#-}"
            local j=0
            local flag_len=${#flags}

            while [[ $j -lt $flag_len ]]; do
                local flag="-${flags:$j:1}"

                # 如果是最后一个短标志，且下一个参数不是选项，则作为值
                if [[ $((j + 1)) -eq $flag_len && $((i + 1)) -lt $len ]]; then
                    local next_arg="${args[$((i + 1))]}"
                    if [[ ! "$next_arg" == -* ]]; then
                        CLI_OPTION_ARGS["$flag"]="$next_arg"
                        [[ -n "$debug_fn" ]] && $debug_fn "解析短选项: $flag = $next_arg"
                        i=$((i + 1))
                    else
                        CLI_OPTION_FLAGS+=("$flag")
                        [[ -n "$debug_fn" ]] && $debug_fn "解析短标志: $flag"
                    fi
                else
                    CLI_OPTION_FLAGS+=("$flag")
                    [[ -n "$debug_fn" ]] && $debug_fn "解析短标志: $flag"
                fi

                j=$((j + 1))
            done

        # 位置参数
        else
            CLI_POSITIONAL_ARGS+=("$arg")
            [[ -n "$debug_fn" ]] && $debug_fn "解析位置参数: $arg"
        fi

        i=$((i + 1))
    done

    return 0
}

# ============================================================================
# 辅助查询函数
# ============================================================================

# 检查是否设置了某个选项或标志
# Usage: cli_has_option <option>
# Returns: 0=设置, 1=未设置
# Examples:
#   cli_has_option "--restart"    # 检查长选项
#   cli_has_option "-w"           # 检查短选项
#   cli_has_option "--module"     # 检查带值选项
cli_has_option() {
    local option="$1"

    # 检查是否为带值选项
    [[ -n "${CLI_OPTION_ARGS[$option]:-}" ]] && return 0

    # 检查是否为标志
    for flag in "${CLI_OPTION_FLAGS[@]}"; do
        [[ "$flag" == "$option" ]] && return 0
    done

    return 1
}

# 获取选项的值
# Usage: cli_get_option_value <option> [default_value]
# Returns: 选项的值，如果未设置则返回默认值
# Examples:
#   cli_get_option_value "--module"          # 返回 module 的值或空
#   cli_get_option_value "--module" "config" # 返回 module 的值或 "config"
#   cli_get_option_value "-l" "10"           # 返回 -l 的值或 "10"
cli_get_option_value() {
    local option="$1"
    local default_value="${2:-}"
    echo "${CLI_OPTION_ARGS[$option]:-$default_value}"
}

# 获取所有位置参数
# Usage: cli_get_positional_args
# Returns: 每行一个位置参数
cli_get_positional_args() {
    printf '%s\n' "${CLI_POSITIONAL_ARGS[@]}"
}

# 获取第 N 个位置参数 (从0开始)
# Usage: cli_get_positional_arg <index>
# Returns: 位置参数值或空
# Examples:
#   cli_get_positional_arg 0   # 获取第一个位置参数（通常是命令名）
#   cli_get_positional_arg 1   # 获取第二个位置参数
cli_get_positional_arg() {
    local index="$1"
    if [[ $index -lt ${#CLI_POSITIONAL_ARGS[@]} ]]; then
        echo "${CLI_POSITIONAL_ARGS[$index]}"
    fi
}

# 获取位置参数数量
# Usage: cli_get_positional_count
# Returns: 位置参数的数量
cli_get_positional_count() {
    echo ${#CLI_POSITIONAL_ARGS[@]}
}

# 获取所有解析的选项键（用于调试）
# Usage: cli_get_option_keys
cli_get_option_keys() {
    printf '%s\n' "${!CLI_OPTION_ARGS[@]}"
}

# 获取所有解析的标志（用于调试）
# Usage: cli_get_option_flags
cli_get_option_flags() {
    printf '%s\n' "${CLI_OPTION_FLAGS[@]}"
}

# 检查是否有任何参数
# Usage: cli_has_args
# Returns: 0=有参数, 1=无参数
cli_has_args() {
    [[ ${#CLI_POSITIONAL_ARGS[@]} -gt 0 || ${#CLI_OPTION_ARGS[@]} -gt 0 || ${#CLI_OPTION_FLAGS[@]} -gt 0 ]]
}

# 以JSON格式输出解析结果（用于调试）
# Usage: cli_parse_to_json
cli_parse_to_json() {
    echo "{"
    echo '  "positional": ['
    local first=true
    for arg in "${CLI_POSITIONAL_ARGS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        printf '    "%s"' "$arg"
    done
    echo ""
    echo '  ],'
    echo '  "options": {'
    first=true
    for key in "${!CLI_OPTION_ARGS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        printf '    "%s": "%s"' "$key" "${CLI_OPTION_ARGS[$key]}"
    done
    echo ""
    echo '  },'
    echo '  "flags": ['
    first=true
    for flag in "${CLI_OPTION_FLAGS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        printf '    "%s"' "$flag"
    done
    echo ""
    echo '  ]'
    echo "}"
}
