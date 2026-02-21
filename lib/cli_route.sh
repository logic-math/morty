#!/bin/bash
# cli_route.sh - 命令路由库
#
# 使用方法:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/cli_route.sh"
#   cli_route <command> [args...]

# 确保不重复加载
[[ -n "${_CLI_ROUTE_LOADED_:-}" ]] && return 0
_CLI_ROUTE_LOADED_=1

# 加载依赖
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/cli_register_command.sh"

# ============================================================================
# 路由函数
# ============================================================================

# 路由命令到对应的处理函数
# Usage: cli_route <command> [args...]
#
# 参数:
#   command: 命令名称
#   args:    传递给命令的参数
#
# 行为:
#   - 如果没有指定命令，显示全局帮助
#   - 如果命令未注册，检查是否为 help/version 特殊命令
#   - 如果命令不存在，显示错误
#   - 否则执行命令
#
# 示例:
#   cli_route "doing" "--restart" "--module" "config"
#   cli_route "stat" "-w"
#   cli_route "help" "doing"
#
cli_route() {
    local command="${1:-}"
    shift 2>/dev/null || true

    # 如果没有指定命令，显示帮助
    if [[ -z "$command" ]]; then
        if type cli_show_global_help &>/dev/null; then
            cli_show_global_help
        else
            echo "用法: morty <command> [options]"
            echo "运行 'morty help' 查看可用命令。"
        fi
        return 0
    fi

    # 调试输出
    if [[ "${CLI_DEBUG:-0}" == "1" ]]; then
        echo "[debug] 路由命令: $command" >&2
    fi

    # 检查命令是否已注册
    if ! cli_is_command_registered "$command"; then
        # 检查是否为特殊命令
        case "$command" in
            help|--help|-h)
                if type cli_show_global_help &>/dev/null; then
                    cli_show_global_help
                else
                    echo "帮助系统未加载"
                fi
                return 0
                ;;
            version|--version|-v)
                if type cli_show_version &>/dev/null; then
                    cli_show_version
                else
                    echo "Morty 2.0.0"
                fi
                return 0
                ;;
            *)
                echo "错误: 未知命令 '$command'。运行 'morty help' 查看可用命令。" >&2
                return 2
                ;;
        esac
    fi

    # 执行命令
    if type cli_execute &>/dev/null; then
        cli_execute "$command" "$@"
    else
        # 直接执行处理函数
        local handler
        handler=$(cli_get_handler "$command")

        if [[ -f "$handler" && -x "$handler" ]]; then
            exec "$handler" "$@"
        elif type "$handler" &>/dev/null; then
            "$handler" "$@"
        else
            echo "错误: 找不到处理函数: $handler" >&2
            return 1
        fi
    fi
}

# 批量路由多个命令（用于测试）
# Usage: cli_route_batch <commands...>
cli_route_batch() {
    local results=()

    for cmd in "$@"; do
        echo "=== 路由: $cmd ==="
        cli_route $cmd
        results+=("$cmd: exit_code=$?")
        echo ""
    done

    echo "结果摘要:"
    for result in "${results[@]}"; do
        echo "  $result"
    done
}

# 获取命令路由信息（用于调试）
# Usage: cli_route_info <command>
cli_route_info() {
    local command="${1:-}"

    echo "路由信息:"
    echo "  命令: $command"

    if cli_is_command_registered "$command"; then
        echo "  状态: 已注册"
        echo "  处理器: $(cli_get_handler "$command")"
        echo "  描述: $(cli_get_description "$command")"
        echo "  选项: $(cli_get_options "$command")"
    else
        echo "  状态: 未注册"
        case "$command" in
            help|--help|-h)
                echo "  类型: 内置帮助命令"
                ;;
            version|--version|-v)
                echo "  类型: 内置版本命令"
                ;;
            *)
                echo "  类型: 未知命令"
                ;;
        esac
    fi
}
