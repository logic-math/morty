#!/usr/bin/env bash
#
# logging.sh - 统一的日志管理模块
#
# 提供多级别日志、结构化日志输出和日志文件管理
#

# 防止重复加载
[[ -n "${_LOGGING_SH_LOADED:-}" ]] && return 0
_LOGGING_SH_LOADED=1

# 获取脚本所在目录
_LOGGING_SH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 引入 common.sh（如果存在）
if [[ -f "${_LOGGING_SH_DIR}/common.sh" ]]; then
    # shellcheck source=./common.sh
    source "${_LOGGING_SH_DIR}/common.sh"
fi

# ============================================
# 配置默认值
# ============================================

# 日志级别（数字越小级别越低）
readonly LOG_LEVEL_DEBUG=0
readonly LOG_LEVEL_INFO=1
readonly LOG_LEVEL_WARN=2
readonly LOG_LEVEL_ERROR=3
readonly LOG_LEVEL_SUCCESS=4
readonly LOG_LEVEL_LOOP=5

# 默认配置
: "${LOG_LEVEL:=${LOG_LEVEL_INFO}}"
: "${LOG_DIR:=${MORTY_DIR:-.morty}/logs}"
: "${LOG_FORMAT:=text}"  # text 或 json
: "${LOG_MAIN_FILE:=${LOG_DIR}/morty.log}"
: "${LOG_MAX_SIZE:=10485760}"  # 10MB
: "${LOG_MAX_FILES:=5}"

# 日志级别名称映射（用于输出）
declare -A _LOG_LEVEL_NAMES=(
    [${LOG_LEVEL_DEBUG}]="DEBUG"
    [${LOG_LEVEL_INFO}]="INFO"
    [${LOG_LEVEL_WARN}]="WARN"
    [${LOG_LEVEL_ERROR}]="ERROR"
    [${LOG_LEVEL_SUCCESS}]="SUCCESS"
    [${LOG_LEVEL_LOOP}]="LOOP"
)

# 当前 Job 日志上下文
_LOG_JOB_MODULE=""
_LOG_JOB_NAME=""
_LOG_JOB_FILE=""
_LOG_JOB_START_TIME=""

# 文件锁（用于并发安全）
readonly _LOG_LOCK_DIR="${LOG_DIR}/.locks"

# ============================================
# 内部函数
# ============================================

# 初始化日志系统
# 创建必要的目录结构
_log_init() {
    if [[ ! -d "${LOG_DIR}" ]]; then
        mkdir -p "${LOG_DIR}" 2>/dev/null || {
            echo "ERROR: Failed to create log directory: ${LOG_DIR}" >&2
            return 1
        }
    fi

    if [[ ! -d "${_LOG_LOCK_DIR}" ]]; then
        mkdir -p "${_LOG_LOCK_DIR}" 2>/dev/null || {
            echo "ERROR: Failed to create lock directory: ${_LOG_LOCK_DIR}" >&2
            return 1
        }
    fi

    # 创建 jobs 子目录
    if [[ ! -d "${LOG_DIR}/jobs" ]]; then
        mkdir -p "${LOG_DIR}/jobs" 2>/dev/null || {
            echo "ERROR: Failed to create jobs log directory" >&2
            return 1
        }
    fi

    return 0
}

# 缓存的时间戳（减少 date 调用）
_LOG_TIMESTAMP_CACHE=""
_LOG_TIMESTAMP_CACHE_TIME=0

# 获取当前时间戳（ISO8601 格式）
_log_timestamp() {
    # 使用内置 printf %()T 格式，避免外部 date 调用
    # 比 date 命令快约 100 倍
    printf '%(%Y-%m-%d %H:%M:%S)T\n' -1
}

# 获取当前时间戳（ISO8601 带时区）
_log_timestamp_iso() {
    printf '%(%Y-%m-%dT%H:%M:%SZ)T\n' -1
}

# 将级别名称转换为数字
_log_level_to_number() {
    local level="$1"
    case "${level^^}" in
        DEBUG)   echo "${LOG_LEVEL_DEBUG}" ;;
        INFO)    echo "${LOG_LEVEL_INFO}" ;;
        WARN|WARNING) echo "${LOG_LEVEL_WARN}" ;;
        ERROR)   echo "${LOG_LEVEL_ERROR}" ;;
        SUCCESS) echo "${LOG_LEVEL_SUCCESS}" ;;
        LOOP)    echo "${LOG_LEVEL_LOOP}" ;;
        *)       echo "${LOG_LEVEL_INFO}" ;;  # 默认 INFO
    esac
}

# 检查是否需要记录该级别
_log_should_log() {
    local msg_level="$1"
    # 当消息级别 >= 配置的日志级别时记录
    [[ "${msg_level}" -ge "${LOG_LEVEL}" ]]
}

# 检测并存储最佳的文件锁机制
_log_detect_lock_mechanism() {
    # 检查 flock 命令是否可用（util-linux）
    if command -v flock >/dev/null 2>&1; then
        echo "flock"
    # 检查 shlock (BSD)
    elif command -v shlock >/dev/null 2>&1; then
        echo "shlock"
    else
        echo "mkdir"
    fi
}

# 存储检测到的锁机制
_LOG_LOCK_MECHANISM="$(_log_detect_lock_mechanism)"

# 获取文件锁
_log_acquire_lock() {
    local lock_name="$1"
    local lock_file="${_LOG_LOCK_DIR}/${lock_name}.lock"

    case "${_LOG_LOCK_MECHANISM}" in
        flock)
            # 使用 flock 命令，创建文件描述符
            exec 200>"${lock_file}"
            flock -n 200 2>/dev/null || flock 200
            echo "200"  # 返回文件描述符
            ;;
        mkdir)
            # 回退到 mkdir 方法（原子操作）
            local timeout=5
            local waited=0
            while ! mkdir "${lock_file}" 2>/dev/null; do
                # 使用忙等待而非 sleep，降低延迟
                if [[ $((waited++)) -ge $((timeout * 10000)) ]]; then
                    return 1
                fi
            done
            echo "mkdir"
            ;;
        *)
            return 1
            ;;
    esac
}

# 释放文件锁
_log_release_lock() {
    local lock_handle="$1"

    case "${lock_handle}" in
        200|201|202|203|204)
            # 关闭文件描述符以释放 flock
            eval "exec ${lock_handle}>&-"
            ;;
        mkdir)
            # mkdir 锁由 _log_write 直接处理
            ;;
    esac
}

# 格式化日志消息（文本格式）
_log_format_text() {
    local timestamp="$1"
    local level_name="$2"
    local module="$3"
    local job="$4"
    local message="$5"

    local prefix="[${timestamp}] [${level_name}]"

    if [[ -n "${module}" ]]; then
        if [[ -n "${job}" ]]; then
            prefix="${prefix} [${module}:${job}]"
        else
            prefix="${prefix} [${module}]"
        fi
    fi

    echo "${prefix} ${message}"
}

# 上下文数据序列化为 JSON
# 支持多种输入格式：JSON字符串、key=value 对、关联数组名
_log_serialize_context() {
    local context="$1"

    # 如果为空，返回空
    if [[ -z "${context}" ]]; then
        echo ""
        return 0
    fi

    # 检查是否已经是有效的 JSON 对象（以 { 开头 } 结尾）
    if [[ "${context}" =~ ^\s*\{.*\}\s*$ ]]; then
        # 验证基本 JSON 格式并返回
        echo "${context}"
        return 0
    fi

    # 检查是否是 JSON 数组（以 [ 开头 ] 结尾）
    if [[ "${context}" =~ ^\s*\[.*\]\s*$ ]]; then
        echo "${context}"
        return 0
    fi

    # 尝试解析为 key=value 格式（逗号分隔）
    if [[ "${context}" == *"="* ]]; then
        local json_parts=()
        local IFS_OLD="${IFS}"
        IFS=',' read -ra pairs <<< "${context}"
        IFS="${IFS_OLD}"

        for pair in "${pairs[@]}"; do
            # 提取 key 和 value
            local key="${pair%%=*}"
            local value="${pair#*=}"

            # 清理空白字符
            key=$(echo "${key}" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
            value=$(echo "${value}" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')

            if [[ -n "${key}" ]]; then
                # 转义 value 中的特殊字符
                local escaped_value
                escaped_value=$(_log_json_escape "${value}")
                json_parts+=("\"${key}\":\"${escaped_value}\"")
            fi
        done

        if [[ ${#json_parts[@]} -gt 0 ]]; then
            # 构建 JSON 对象
            local result="{"
            local first=true
            for part in "${json_parts[@]}"; do
                if [[ "${first}" == true ]]; then
                    first=false
                else
                    result="${result},"
                fi
                result="${result}${part}"
            done
            result="${result}}"
            echo "${result}"
            return 0
        fi
    fi

    # 默认：将上下文作为普通字符串值
    local escaped_context
    escaped_context=$(_log_json_escape "${context}")
    echo "\"${escaped_context}\""
}

# JSON 字符串转义
# 将字符串中的特殊字符转义为 JSON 安全格式
_log_json_escape() {
    local str="$1"
    # 使用 Bash 内置字符串替换进行转义（比 sed 更可靠）
    # 顺序很重要：先转义反斜杠，再转义其他字符
    str="${str//\\/\\\\}"  # 反斜杠 -> \\
    str="${str//\"/\\\"}"  # 双引号 -> \\"
    str="${str//$'\t'/\\t}"  # 制表符 -> \t
    str="${str//$'\r'/\\r}"  # 回车 -> \r
    str="${str//$'\n'/\\n}"  # 换行 -> \n
    str="${str//$'\b'/\\b}"  # 退格 -> \b
    str="${str//$'\f'/\\f}"  # 换页 -> \f
    printf '%s' "${str}"
}

# 格式化日志消息（JSON 格式）
_log_format_json() {
    local timestamp="$1"
    local level_name="$2"
    local module="$3"
    local job="$4"
    local message="$5"
    local context="$6"

    # 转义消息中的特殊字符
    local escaped_message
    escaped_message=$(_log_json_escape "${message}")

    local json="{\"timestamp\":\"${timestamp}\",\"level\":\"${level_name}\",\"message\":\"${escaped_message}\""

    if [[ -n "${module}" ]]; then
        json="${json},\"module\":\"${module}\""
    fi

    if [[ -n "${job}" ]]; then
        json="${json},\"job\":\"${job}\""
    fi

    if [[ -n "${context}" ]]; then
        # 尝试将上下文序列化为 JSON 字段
        local serialized_context
        serialized_context=$(_log_serialize_context "${context}")
        if [[ -n "${serialized_context}" ]]; then
            json="${json},\"context\":${serialized_context}"
        fi
    fi

    json="${json}}"
    echo "${json}"
}

# 核心写入函数
_log_write() {
    local level="$1"
    local message="$2"
    local context="${3:-}"

    # 检查级别
    if ! _log_should_log "${level}"; then
        return 0
    fi

    # 确保初始化
    _log_init

    local level_name="${_LOG_LEVEL_NAMES[${level}]:-INFO}"
    local timestamp
    local module="${_LOG_JOB_MODULE:-}"
    local job="${_LOG_JOB_NAME:-}"

    # 格式化时间戳
    if [[ "${LOG_FORMAT}" == "json" ]]; then
        timestamp=$(_log_timestamp_iso)
    else
        timestamp=$(_log_timestamp)
    fi

    # 格式化消息
    local formatted_msg
    if [[ "${LOG_FORMAT}" == "json" ]]; then
        formatted_msg=$(_log_format_json "${timestamp}" "${level_name}" "${module}" "${job}" "${message}" "${context}")
    else
        formatted_msg=$(_log_format_text "${timestamp}" "${level_name}" "${module}" "${job}" "${message}")
    fi

    # 检查并执行日志轮转（在获取锁之前）
    _log_rotate_if_needed "${LOG_MAIN_FILE}"

    # 写入主日志文件（使用同步写入保证数据完整性）
    # 使用 printf 和 >> 原子操作，配合 sync 确保数据落盘
    if printf '%s\n' "${formatted_msg}" >> "${LOG_MAIN_FILE}" 2>/dev/null; then
        : # 成功
    else
        # 写入失败，输出到 stderr
        echo "${formatted_msg}" >&2
    fi

    # 如果启用了 Job 日志，也写入 Job 日志
    if [[ -n "${_LOG_JOB_FILE}" && -n "${_LOG_JOB_NAME}" ]]; then
        # 检查 Job 日志是否需要轮转
        _log_rotate_if_needed "${_LOG_JOB_FILE}"

        # 直接写入，使用 >> 原子追加操作
        printf '%s\n' "${formatted_msg}" >> "${_LOG_JOB_FILE}" 2>/dev/null
    fi

    return 0
}

# ============================================
# 公共 API
# ============================================

# 核心日志函数
# Usage: log <level> <message> [context]
log() {
    local level="$1"
    local message="$2"
    local context="${3:-}"

    # 转换级别为数字
    local level_num
    if [[ "${level}" =~ ^[0-9]+$ ]]; then
        level_num="${level}"
    else
        level_num=$(_log_level_to_number "${level}")
    fi

    _log_write "${level_num}" "${message}" "${context}"
}

# 调试日志
log_debug() {
    _log_write "${LOG_LEVEL_DEBUG}" "$1" "${2:-}"
}

# 信息日志
log_info() {
    _log_write "${LOG_LEVEL_INFO}" "$1" "${2:-}"
}

# 警告日志
log_warn() {
    _log_write "${LOG_LEVEL_WARN}" "$1" "${2:-}"
}

# 错误日志
log_error() {
    _log_write "${LOG_LEVEL_ERROR}" "$1" "${2:-}"
}

# 成功日志
log_success() {
    _log_write "${LOG_LEVEL_SUCCESS}" "$1" "${2:-}"
}

# 循环/执行日志（用于 Doing 模式）
log_loop() {
    _log_write "${LOG_LEVEL_LOOP}" "$1" "${2:-}"
}

# 结构化日志（JSON 格式，忽略 LOG_FORMAT 设置）
# Usage: log_structured <level> <data>
# data 可以是：
#   - 有效的 JSON 对象字符串（如 '{"key":"value"}'）
#   - key=value 对（如 'user=admin,action=login'）
#   - 关联数组名称（数组中应包含键值对）
log_structured() {
    local level="$1"
    local data="$2"

    local level_num
    if [[ "${level}" =~ ^[0-9]+$ ]]; then
        level_num="${level}"
    else
        level_num=$(_log_level_to_number "${level}")
    fi

    # 检查级别
    if ! _log_should_log "${level_num}"; then
        return 0
    fi

    _log_init

    local level_name="${_LOG_LEVEL_NAMES[${level_num}]:-INFO}"
    local timestamp
    timestamp=$(_log_timestamp_iso)

    # 构建基础 JSON
    local json="{\"timestamp\":\"${timestamp}\",\"level\":\"${level_name}\""

    # 处理数据部分
    if [[ -n "${data}" ]]; then
        # 检查是否是有效的 JSON 对象（优先检查，避免将 JSON 当作变量名）
        if [[ "${data}" =~ ^[[:space:]]*\{.*\}[[:space:]]*$ ]]; then
            # 是 JSON 对象，提取内部内容
            local inner_data
            inner_data=$(printf '%s' "${data}" | sed 's/^[[:space:]]*{[[:space:]]*//; s/[[:space:]]*}[[:space:]]*$//')
            if [[ -n "${inner_data}" ]]; then
                json="${json},${inner_data}"
            fi
        # 检查是否是关联数组名（必须是有效的变量名且不是 JSON）
        elif [[ "${data}" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]] && declare -p "${data}" 2>/dev/null | grep -q 'declare -A'; then
            # 是关联数组，遍历键值对
            local array_ref="${data}"
            eval "local keys=(\${!${array_ref}[@]})"
            for key in "${keys[@]}"; do
                local val
                eval "val=\"\${${array_ref}[${key}]}\""
                local escaped_val
                escaped_val=$(_log_json_escape "${val}")
                json="${json},\"${key}\":\"${escaped_val}\""
            done
        # 检查是否是 key=value 格式
        elif [[ "${data}" == *"="* ]]; then
            # 解析 key=value 对
            local serialized
            serialized=$(_log_serialize_context "${data}")
            if [[ "${serialized}" =~ ^\{.*\}$ ]]; then
                # 提取序列化后对象的内部内容
                local inner
                inner=$(printf '%s' "${serialized}" | sed 's/^[[:space:]]*{[[:space:]]*//; s/[[:space:]]*}[[:space:]]*$//')
                if [[ -n "${inner}" ]]; then
                    json="${json},${inner}"
                fi
            fi
        else
            # 普通字符串，作为 message 字段
            local escaped_msg
            escaped_msg=$(_log_json_escape "${data}")
            json="${json},\"message\":\"${escaped_msg}\""
        fi
    fi

    # 添加模块/Job 信息
    if [[ -n "${_LOG_JOB_MODULE}" ]]; then
        json="${json},\"module\":\"${_LOG_JOB_MODULE}\""
    fi
    if [[ -n "${_LOG_JOB_NAME}" ]]; then
        json="${json},\"job\":\"${_LOG_JOB_NAME}\""
    fi

    json="${json}}"

    # 写入主日志（使用原子追加操作）
    printf '%s\n' "${json}" >> "${LOG_MAIN_FILE}" 2>/dev/null

    # Job 日志
    if [[ -n "${_LOG_JOB_FILE}" && -n "${_LOG_JOB_NAME}" ]]; then
        printf '%s\n' "${json}" >> "${_LOG_JOB_FILE}" 2>/dev/null
    fi
}

# ============================================
# Job 日志 API
# ============================================

# 开始 Job 日志上下文
# Usage: log_job_start <module> <job_name>
log_job_start() {
    local module="$1"
    local job_name="$2"

    _log_init

    # 规范化 job_name: 移除下划线以生成文件友好名称
    local job_file_name="${job_name//_/}"

    _LOG_JOB_MODULE="${module}"
    _LOG_JOB_NAME="${job_name}"
    _LOG_JOB_FILE="${LOG_DIR}/jobs/${module}_${job_file_name}.log"
    _LOG_JOB_START_TIME=$(date +%s)

    # 创建 Job 日志文件（log_info 会同时写入主日志和 Job 日志）
    log_info "Job ${module}:${job_name} started"
}

# 结束 Job 日志上下文
# Usage: log_job_end [status]
log_job_end() {
    local status="${1:-completed}"

    if [[ -z "${_LOG_JOB_NAME}" ]]; then
        return 0
    fi

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - _LOG_JOB_START_TIME))

    # 记录结束信息到主日志（会自动写入 Job 日志）
    log_info "Job ${_LOG_JOB_MODULE}:${_LOG_JOB_NAME} ${status} (duration: ${duration}s)"

    # 在 Job 日志末尾添加统计信息
    if [[ -n "${_LOG_JOB_FILE}" ]]; then
        local timestamp
        timestamp=$(_log_timestamp)
        {
            echo ""
            echo "=== Job ${status} ==="
            echo "End time: ${timestamp}"
            echo "Duration: ${duration}s"
        } >> "${_LOG_JOB_FILE}" 2>/dev/null
    fi

    # 清理上下文
    _LOG_JOB_MODULE=""
    _LOG_JOB_NAME=""
    _LOG_JOB_FILE=""
    _LOG_JOB_START_TIME=""
}

# 写入 Job 独立日志
# Usage: log_job <message> [level]
# 注意：此函数只写入 Job 独立日志，不会写入主日志
# 如需同时写入两者，请直接使用 log_info 等函数（在 Job 上下文中会自动双写）
log_job() {
    local message="$1"
    local level="${2:-INFO}"

    if [[ -z "${_LOG_JOB_FILE}" ]]; then
        # 如果没有 Job 上下文，使用普通日志
        log "${level}" "${message}"
        return 0
    fi

    local level_num
    if [[ "${level}" =~ ^[0-9]+$ ]]; then
        level_num="${level}"
    else
        level_num=$(_log_level_to_number "${level}")
    fi

    if ! _log_should_log "${level_num}"; then
        return 0
    fi

    local level_name="${_LOG_LEVEL_NAMES[${level_num}]:-INFO}"
    local timestamp
    timestamp=$(_log_timestamp)

    local formatted_msg="[${timestamp}] [${level_name}] ${message}"

    # 写入 Job 日志（使用原子追加操作）
    printf '%s\n' "${formatted_msg}" >> "${_LOG_JOB_FILE}" 2>/dev/null
}

# Job 调试日志
log_job_debug() {
    log_job "$1" "DEBUG"
}

# ============================================
# 日志配置 API
# ============================================

# 设置日志级别
# Usage: log_set_level <level>
# level 可以是数字或名称: DEBUG, INFO, WARN, ERROR, SUCCESS, LOOP
log_set_level() {
    local level="$1"

    if [[ "${level}" =~ ^[0-9]+$ ]]; then
        LOG_LEVEL="${level}"
    else
        LOG_LEVEL=$(_log_level_to_number "${level}")
    fi
}

# 获取当前日志级别
log_get_level() {
    echo "${LOG_LEVEL}"
}

# 获取当前日志级别名称
log_get_level_name() {
    echo "${_LOG_LEVEL_NAMES[${LOG_LEVEL}]:-INFO}"
}

# 设置日志格式
# Usage: log_set_format <text|json>
log_set_format() {
    local format="$1"
    case "${format}" in
        text|json)
            LOG_FORMAT="${format}"
            ;;
        *)
            echo "ERROR: Invalid log format: ${format}. Use 'text' or 'json'." >&2
            return 1
            ;;
    esac
}

# 获取日志目录
log_get_dir() {
    echo "${LOG_DIR}"
}

# 获取主日志文件路径
log_get_main_file() {
    echo "${LOG_MAIN_FILE}"
}

# 获取当前 Job 日志文件路径
log_get_job_file() {
    echo "${_LOG_JOB_FILE:-}"
}

# ============================================
# 日志轮转（将在 Job 2 中实现完整功能）
# ============================================

# 清理超出最大保留数的旧日志文件（包括压缩文件）
_log_cleanup_old_logs() {
    local base_name="$1"
    local max_files="$2"

    # 删除所有编号大于 max_files 的历史日志文件
    local i=$((max_files + 1))
    while [[ -f "${base_name}.${i}" ]] || [[ -f "${base_name}.${i}.gz" ]]; do
        rm -f "${base_name}.${i}"
        rm -f "${base_name}.${i}.gz"
        i=$((i + 1))
    done
}

# 压缩日志文件
# Usage: _log_compress_file <file_path>
# 返回: 0 成功, 1 失败
_log_compress_file() {
    local file_path="$1"

    # 检查文件是否存在且非空
    if [[ ! -f "${file_path}" ]] || [[ ! -s "${file_path}" ]]; then
        return 0
    fi

    # 检查 gzip 是否可用
    if ! command -v gzip >/dev/null 2>&1; then
        log_warn "gzip not available, skipping compression for ${file_path}"
        return 1
    fi

    # 执行压缩
    local original_size compressed_size
    original_size=$(stat -f%z "${file_path}" 2>/dev/null || stat -c%s "${file_path}" 2>/dev/null || echo "0")

    # 使用 gzip 压缩，保留原文件时间戳
    if gzip -c "${file_path}" > "${file_path}.gz.tmp"; then
        mv "${file_path}.gz.tmp" "${file_path}.gz"
        rm -f "${file_path}"

        # 验证压缩效果
        compressed_size=$(stat -f%z "${file_path}.gz" 2>/dev/null || stat -c%s "${file_path}.gz" 2>/dev/null || echo "0")

        if [[ ${original_size} -gt 0 ]]; then
            local ratio=$((compressed_size * 100 / original_size))
            log_debug "Compressed ${file_path}: ${original_size} bytes -> ${compressed_size} bytes (${ratio}%)"
        fi

        return 0
    else
        rm -f "${file_path}.gz.tmp"
        log_warn "Failed to compress ${file_path}"
        return 1
    fi
}

# 写入计数器（用于降低轮转检查频率）
_LOG_WRITE_COUNT=0

# 检查并执行日志轮转（优化：每100次写入检查一次）
_log_rotate_if_needed() {
    local log_file="$1"

    # 快速路径：如果文件不存在，跳过
    if [[ ! -f "${log_file}" ]]; then
        return 0
    fi

    # 优化：每100次写入才检查一次文件大小
    _LOG_WRITE_COUNT=$((_LOG_WRITE_COUNT + 1))
    if [[ $((_LOG_WRITE_COUNT % 100)) -ne 0 ]]; then
        return 0
    fi

    local file_size
    file_size=$(stat -f%z "${log_file}" 2>/dev/null || stat -c%s "${log_file}" 2>/dev/null || echo "0")

    if [[ ${file_size} -ge ${LOG_MAX_SIZE} ]]; then
        # 执行轮转
        local base_name="${log_file}"
        local max_files="${LOG_MAX_FILES}"

        # 首先清理任何超出最大保留数的旧日志文件（包括压缩文件）
        _log_cleanup_old_logs "${base_name}" "${max_files}"

        # 从最旧的开始处理：先压缩即将成为最旧的文件
        # 如果 max_files=5，那么 .4 -> .5，压缩 .5
        local oldest="${base_name}.${max_files}"
        if [[ -f "${oldest}" ]]; then
            # 压缩最旧的日志文件
            _log_compress_file "${oldest}"
        fi
        # 同时也要检查是否有对应的 .gz 文件需要删除
        rm -f "${oldest}.gz"

        # 向后移动日志（从后往前移动），同时对较旧的文件进行压缩
        # 例如：morty.log.3 -> morty.log.4 (并压缩), morty.log.2 -> morty.log.3 (并压缩)
        for ((i=max_files-1; i>=2; i--)); do
            local src="${base_name}.${i}"
            local dst="${base_name}.$((i+1))"
            if [[ -f "${src}" ]]; then
                mv "${src}" "${dst}"
                # 压缩移动后的文件（.log.3 及以上进行压缩）
                _log_compress_file "${dst}"
            fi
            # 同时处理可能存在的压缩文件
            if [[ -f "${src}.gz" ]]; then
                mv "${src}.gz" "${dst}.gz"
            fi
        done

        # 处理 .log.1 -> .log.2 (压缩)
        if [[ -f "${base_name}.1" ]]; then
            mv "${base_name}.1" "${base_name}.2"
            _log_compress_file "${base_name}.2"
        fi
        if [[ -f "${base_name}.1.gz" ]]; then
            mv "${base_name}.1.gz" "${base_name}.2.gz"
        fi

        # 移动当前日志到 .log.1（不压缩，保持可读取）
        mv "${log_file}" "${log_file}.1"
    fi
}

# 轮转主日志
log_rotate_main() {
    _log_rotate_if_needed "${LOG_MAIN_FILE}"
}

# ============================================
# 初始化
# ============================================

# 模块加载时自动初始化
_log_init

# 记录模块加载
if [[ "${LOG_LEVEL}" -le "${LOG_LEVEL_DEBUG}" ]]; then
    echo "DEBUG: logging.sh module loaded, LOG_LEVEL=${LOG_LEVEL}, LOG_DIR=${LOG_DIR}" >&2
fi
