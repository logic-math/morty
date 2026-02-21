#!/bin/bash
# cli_register_command.sh - 命令注册库
#
# 使用方法:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/cli_register_command.sh"
#   cli_register_command "doing" "$MORTY_HOME/morty_doing.sh" "执行开发计划" "--restart --module --job"

# 确保不重复加载
[[ -n "${_CLI_REGISTER_COMMAND_LOADED_:-}" ]] && return 0
_CLI_REGISTER_COMMAND_LOADED_=1

# ============================================================================
# 命令注册表
# ============================================================================

# 关联数组存储命令处理函数
declare -A _CLI_COMMAND_HANDLERS
declare -A _CLI_COMMAND_DESCRIPTIONS
declare -A _CLI_COMMAND_OPTIONS
declare -a _CLI_COMMAND_ORDER

# ============================================================================
# 命令注册函数
# ============================================================================

# 注册一个命令
# Usage: cli_register_command <name> <handler> <description> [options]
#
# 参数:
#   name:        命令名称 (如: doing, stat, reset)
#   handler:     处理函数名或脚本路径
#   description: 命令描述
#   options:     可选，命令支持的选项列表 (如: "--restart --module --job")
#
# 示例:
#   cli_register_command "doing" "morty_doing.sh" "执行开发计划" "--restart --module --job"
#   cli_register_command "stat" "handler_stat" "显示状态" "-t -w"
#   cli_register_command "version" "cli_show_version" "显示版本" ""
#
cli_register_command() {
    local name="${1:-}"
    local handler="${2:-}"
    local description="${3:-}"
    local options="${4:-}"

    # 参数验证
    if [[ -z "$name" ]]; then
        echo "错误: cli_register_command: 命令名称不能为空" >&2
        return 1
    fi

    if [[ -z "$handler" ]]; then
        echo "错误: cli_register_command: 处理函数不能为空 ($name)" >&2
        return 1
    fi

    if [[ -z "$description" ]]; then
        echo "错误: cli_register_command: 命令描述不能为空 ($name)" >&2
        return 1
    fi

    # 检查命令是否已注册
    if [[ -n "${_CLI_COMMAND_HANDLERS[$name]:-}" ]]; then
        echo "警告: 命令 '$name' 已存在，将被覆盖" >&2
    fi

    # 注册命令
    _CLI_COMMAND_HANDLERS["$name"]="$handler"
    _CLI_COMMAND_DESCRIPTIONS["$name"]="$description"
    _CLI_COMMAND_OPTIONS["$name"]="$options"

    # 维护命令顺序（如果还未添加）
    local found=false
    for cmd in "${_CLI_COMMAND_ORDER[@]}"; do
        if [[ "$cmd" == "$name" ]]; then
            found=true
            break
        fi
    done
    [[ "$found" == "false" ]] && _CLI_COMMAND_ORDER+=("$name")

    return 0
}

# 注销一个命令
# Usage: cli_unregister_command <name>
cli_unregister_command() {
    local name="${1:-}"

    if [[ -z "$name" ]]; then
        echo "错误: cli_unregister_command: 命令名称不能为空" >&2
        return 1
    fi

    unset "_CLI_COMMAND_HANDLERS[$name]"
    unset "_CLI_COMMAND_DESCRIPTIONS[$name]"
    unset "_CLI_COMMAND_OPTIONS[$name]"

    # 从顺序数组中移除
    local new_order=()
    for cmd in "${_CLI_COMMAND_ORDER[@]}"; do
        [[ "$cmd" != "$name" ]] && new_order+=("$cmd")
    done
    _CLI_COMMAND_ORDER=("${new_order[@]}")

    return 0
}

# 检查命令是否已注册
# Usage: cli_is_command_registered <name>
# Returns: 0=已注册, 1=未注册
cli_is_command_registered() {
    local name="${1:-}"
    [[ -n "${_CLI_COMMAND_HANDLERS[$name]:-}" ]]
}

# 获取命令的处理函数
# Usage: cli_get_handler <name>
# Returns: 处理函数名或脚本路径
cli_get_handler() {
    local name="${1:-}"
    echo "${_CLI_COMMAND_HANDLERS[$name]:-}"
}

# 获取命令的描述
# Usage: cli_get_description <name>
# Returns: 命令描述
cli_get_description() {
    local name="${1:-}"
    echo "${_CLI_COMMAND_DESCRIPTIONS[$name]:-}"
}

# 获取命令的选项列表
# Usage: cli_get_options <name>
# Returns: 命令支持的选项列表
cli_get_options() {
    local name="${1:-}"
    echo "${_CLI_COMMAND_OPTIONS[$name]:-}"
}

# 获取所有已注册的命令列表
# Usage: cli_get_commands
# Returns: 每行一个命令名称
cli_get_commands() {
    printf '%s\n' "${_CLI_COMMAND_ORDER[@]}"
}

# 获取已注册命令的数量
# Usage: cli_get_command_count
cli_get_command_count() {
    echo ${#_CLI_COMMAND_ORDER[@]}
}

# 清空所有命令注册
# Usage: cli_clear_commands
cli_clear_commands() {
    _CLI_COMMAND_HANDLERS=()
    _CLI_COMMAND_DESCRIPTIONS=()
    _CLI_COMMAND_OPTIONS=()
    _CLI_COMMAND_ORDER=()
}

# 获取命令信息（JSON格式，用于调试）
# Usage: cli_get_command_info <name>
cli_get_command_info() {
    local name="${1:-}"

    if ! cli_is_command_registered "$name"; then
        echo "{}"
        return 1
    fi

    echo "{"
    echo "  \"name\": \"$name\","
    echo "  \"handler\": \"$(cli_get_handler "$name")\","
    echo "  \"description\": \"$(cli_get_description "$name")\","
    echo "  \"options\": \"$(cli_get_options "$name")\""
    echo "}"
}

# 列出所有命令（JSON格式，用于调试）
# Usage: cli_list_commands_json
cli_list_commands_json() {
    echo "["
    local first=true
    for cmd in "${_CLI_COMMAND_ORDER[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        echo "  {"
        echo "    \"name\": \"$cmd\","
        echo "    \"handler\": \"${_CLI_COMMAND_HANDLERS[$cmd]}\","
        echo "    \"description\": \"${_CLI_COMMAND_DESCRIPTIONS[$cmd]}\","
        echo "    \"options\": \"${_CLI_COMMAND_OPTIONS[$cmd]}\""
        printf "  }"
    done
    echo ""
    echo "]"
}
