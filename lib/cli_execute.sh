#!/bin/bash
# cli_execute.sh - 命令执行库
#
# 使用方法:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/cli_execute.sh"
#   cli_execute <command> [args...]

# 确保不重复加载
[[ -n "${_CLI_EXECUTE_LOADED_:-}" ]] && return 0
_CLI_EXECUTE_LOADED_=1

# 加载依赖
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/cli_register_command.sh"

# ============================================================================
# 执行函数
# ============================================================================

# 执行命令
# Usage: cli_execute <command> [args...]
#
# 参数:
#   command: 命令名称
#   args:    传递给命令的参数
#
# 行为:
#   - 获取命令的处理函数
#   - 如果处理函数是脚本文件路径，执行该脚本
#   - 如果处理函数是函数名，调用该函数
#   - 如果找不到处理函数，返回错误
#
# 示例:
#   cli_execute "doing" "--restart" "--module" "config"
#   cli_execute "stat"
#   cli_execute "version"
#
cli_execute() {
    local command="${1:-}"
    shift 2>/dev/null || true

    if [[ -z "$command" ]]; then
        echo "错误: cli_execute: 命令名称不能为空" >&2
        return 1
    fi

    local handler
    handler=$(cli_get_handler "$command")

    if [[ -z "$handler" ]]; then
        echo "错误: 命令 '$command' 没有注册的处理函数" >&2
        return 1
    fi

    # 调试输出
    if [[ "${CLI_DEBUG:-0}" == "1" ]]; then
        echo "[debug] 执行命令: $command -> $handler" >&2
    fi

    # 检查处理函数类型
    if [[ -f "$handler" ]]; then
        # 处理函数是脚本文件路径
        if [[ -x "$handler" ]]; then
            if [[ "${CLI_VERBOSE:-0}" == "1" ]]; then
                echo "[verbose] 执行脚本: $handler $*" >&2
            fi
            exec "$handler" "$@"
        else
            echo "错误: 处理脚本不可执行: $handler" >&2
            return 1
        fi
    elif type "$handler" &>/dev/null; then
        # 处理函数是当前环境中的函数
        if [[ "${CLI_VERBOSE:-0}" == "1" ]]; then
            echo "[verbose] 调用函数: $handler $*" >&2
        fi
        "$handler" "$@"
        return $?
    else
        # 尝试作为外部命令执行
        if [[ "${CLI_VERBOSE:-0}" == "1" ]]; then
            echo "[verbose] 尝试执行命令: $handler $*" >&2
        fi
        "$handler" "$@"
        return $?
    fi
}

# 执行命令并捕获输出
# Usage: cli_execute_capture <command> [args...]
# Returns: 命令输出存储在 CLI_CAPTURE_OUTPUT 变量
cli_execute_capture() {
    local command="${1:-}"
    shift 2>/dev/null || true

    CLI_CAPTURE_OUTPUT=""
    CLI_CAPTURE_EXIT_CODE=0

    local handler
    handler=$(cli_get_handler "$command")

    if [[ -z "$handler" ]]; then
        echo "错误: 命令 '$command' 没有注册的处理函数" >&2
        CLI_CAPTURE_EXIT_CODE=1
        return 1
    fi

    if [[ -f "$handler" && -x "$handler" ]]; then
        CLI_CAPTURE_OUTPUT=$("$handler" "$@" 2>&1)
        CLI_CAPTURE_EXIT_CODE=$?
    elif type "$handler" &>/dev/null; then
        CLI_CAPTURE_OUTPUT=$("$handler" "$@" 2>&1)
        CLI_CAPTURE_EXIT_CODE=$?
    else
        echo "错误: 找不到处理函数: $handler" >&2
        CLI_CAPTURE_EXIT_CODE=1
        return 1
    fi

    return $CLI_CAPTURE_EXIT_CODE
}

# 以子进程方式执行命令（不替换当前进程）
# Usage: cli_execute_fork <command> [args...]
cli_execute_fork() {
    local command="${1:-}"
    shift 2>/dev/null || true

    local handler
    handler=$(cli_get_handler "$command")

    if [[ -z "$handler" ]]; then
        echo "错误: 命令 '$command' 没有注册的处理函数" >&2
        return 1
    fi

    if [[ "${CLI_DEBUG:-0}" == "1" ]]; then
        echo "[debug] 执行命令(fork): $command -> $handler" >&2
    fi

    if [[ -f "$handler" && -x "$handler" ]]; then
        "$handler" "$@"
        return $?
    elif type "$handler" &>/dev/null; then
        "$handler" "$@"
        return $?
    else
        "$handler" "$@"
        return $?
    fi
}

# 执行命令并计时
# Usage: cli_execute_timed <command> [args...]
# 结果存储在 CLI_EXEC_TIME 变量中（毫秒）
cli_execute_timed() {
    local command="${1:-}"
    shift 2>/dev/null || true

    local start_time end_time

    if [[ "$(uname)" == "Darwin" ]]; then
        # macOS
        start_time=$(perl -MTime::HiRes=time -e 'printf "%.0f\n", time * 1000')
    else
        # Linux
        start_time=$(date +%s%3N)
    fi

    cli_execute "$command" "$@"
    local exit_code=$?

    if [[ "$(uname)" == "Darwin" ]]; then
        end_time=$(perl -MTime::HiRes=time -e 'printf "%.0f\n", time * 1000')
    else
        end_time=$(date +%s%3N)
    fi

    CLI_EXEC_TIME=$((end_time - start_time))

    if [[ "${CLI_VERBOSE:-0}" == "1" ]]; then
        echo "[verbose] 执行耗时: ${CLI_EXEC_TIME}ms" >&2
    fi

    return $exit_code
}
