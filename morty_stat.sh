#!/bin/bash
# Morty Stat - 执行状态和进度监控脚本
#
# 显示：当前执行、上一个 Job 摘要、Debug 问题、整体进度、累计时间

set -e

# ============================================================================
# 配置和常量
# ============================================================================

MORTY_DIR=".morty"
STATUS_FILE="$MORTY_DIR/status.json"
LOGS_DIR="$MORTY_DIR/doing/logs"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m'

# 表格字符 (使用ASCII字符避免乱码)
TABLE_TOP_LEFT='+='
TABLE_TOP_RIGHT='=+'
TABLE_BOTTOM_LEFT='+='
TABLE_BOTTOM_RIGHT='=+'
TABLE_HORIZONTAL='-'
TABLE_VERTICAL='|'
TABLE_CROSS='|'
TABLE_LEFT_T='|'
TABLE_RIGHT_T='|'
TABLE_TOP_T='|'
TABLE_BOTTOM_T='|'

# 状态图标 (使用ASCII字符避免乱码)
ICON_PENDING='o'
ICON_RUNNING='>'
ICON_COMPLETED='ok'
ICON_FAILED='X'
ICON_BLOCKED='-'

# 脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MORTY_HOME="${MORTY_HOME:-$(dirname "$SCRIPT_DIR")}"

# 加载公共库
source "$MORTY_HOME/lib/common.sh" 2>/dev/null || true

# ============================================================================
# 全局变量
# ============================================================================

declare -A STATUS_DATA
declare -A CURRENT_JOB
declare -A PREVIOUS_JOB
declare -a DEBUG_ISSUES
TOTAL_TASKS=0
COMPLETED_TASKS=0
PROGRESS_PERCENT=0
WATCH_MODE=0
WATCH_INTERVAL=60

# ============================================================================
# 核心函数
# ============================================================================

# 加载 status.json 文件
# Usage: stat_load_status()
# Returns: 0 on success, 1 on failure
stat_load_status() {
    if [[ ! -f "$STATUS_FILE" ]]; then
        echo -e "${RED}错误: 状态文件不存在${NC}" >&2
        echo -e "${YELLOW}提示: 请先运行 morty doing 初始化项目${NC}" >&2
        return 1
    fi

    # 使用 jq 解析 JSON (如果可用)
    if command -v jq &>/dev/null; then
        STATUS_DATA["version"]=$(jq -r '.version // "unknown"' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["state"]=$(jq -r '.state // "unknown"' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["current_module"]=$(jq -r '.current.module // ""' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["current_job"]=$(jq -r '.current.job // ""' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["current_status"]=$(jq -r '.current.status // ""' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["session_start"]=$(jq -r '.session.start_time // ""' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["session_update"]=$(jq -r '.session.last_update // ""' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["total_loops"]=$(jq -r '.session.total_loops // 0' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["total_modules"]=$(jq -r '.summary.total_modules // 0' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["completed_modules"]=$(jq -r '.summary.completed_modules // 0' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["total_jobs"]=$(jq -r '.summary.total_jobs // 0' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["completed_jobs"]=$(jq -r '.summary.completed_jobs // 0' "$STATUS_FILE" 2>/dev/null)
        STATUS_DATA["progress_percentage"]=$(jq -r '.summary.progress_percentage // 0' "$STATUS_FILE" 2>/dev/null)
    else
        # 使用 grep/sed 作为备选
        STATUS_DATA["version"]=$(grep -o '"version": *"[^"]*"' "$STATUS_FILE" | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
        STATUS_DATA["state"]=$(grep -o '"state": *"[^"]*"' "$STATUS_FILE" | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
        STATUS_DATA["current_module"]=$(grep -o '"module": *"[^"]*"' "$STATUS_FILE" | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
        STATUS_DATA["current_job"]=$(grep -o '"job": *"[^"]*"' "$STATUS_FILE" | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
        STATUS_DATA["current_status"]=$(grep -o '"status": *"[^"]*"' "$STATUS_FILE" | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
    fi

    return 0
}

# 加载日志信息
# Usage: stat_load_logs()
stat_load_logs() {
    DEBUG_ISSUES=()

    if [[ ! -d "$LOGS_DIR" ]]; then
        return 0
    fi

    # 统计日志文件
    local log_count=0
    if [[ -d "$LOGS_DIR" ]]; then
        log_count=$(find "$LOGS_DIR" -name "*.log" 2>/dev/null | wc -l)
    fi
    STATUS_DATA["log_count"]=$log_count
}

# 获取当前 Job 信息
# Usage: stat_get_current_job()
# Sets: CURRENT_JOB associative array
stat_get_current_job() {
    local module="${STATUS_DATA["current_module"]}"
    local job="${STATUS_DATA["current_job"]}"

    CURRENT_JOB["module"]="$module"
    CURRENT_JOB["job"]="$job"
    CURRENT_JOB["status"]="${STATUS_DATA["current_status"]}"

    if [[ -z "$module" || -z "$job" ]]; then
        CURRENT_JOB["exists"]=false
        return 0
    fi

    CURRENT_JOB["exists"]=true

    if command -v jq &>/dev/null && [[ -f "$STATUS_FILE" ]]; then
        # 获取当前 job 的详细信息
        CURRENT_JOB["loop_count"]=$(jq -r --arg m "$module" --arg j "$job" '.modules[$m].jobs[$j].loop_count // 0' "$STATUS_FILE" 2>/dev/null)
        CURRENT_JOB["tasks_total"]=$(jq -r --arg m "$module" --arg j "$job" '.modules[$m].jobs[$j].tasks_total // 0' "$STATUS_FILE" 2>/dev/null)
        CURRENT_JOB["tasks_completed"]=$(jq -r --arg m "$module" --arg j "$job" '.modules[$m].jobs[$j].tasks_completed // 0' "$STATUS_FILE" 2>/dev/null)
        CURRENT_JOB["started_at"]=$(jq -r --arg m "$module" --arg j "$job" '.modules[$m].jobs[$j].started_at // ""' "$STATUS_FILE" 2>/dev/null)
        CURRENT_JOB["retry_count"]=$(jq -r --arg m "$module" --arg j "$job" '.modules[$m].jobs[$j].retry_count // 0' "$STATUS_FILE" 2>/dev/null)
        CURRENT_JOB["error_count"]=$(jq -r --arg m "$module" --arg j "$job" '.modules[$m].jobs[$j].error_count // 0' "$STATUS_FILE" 2>/dev/null)
    fi

    # 计算当前 job 的进度
    local total="${CURRENT_JOB["tasks_total"]:-0}"
    local completed="${CURRENT_JOB["tasks_completed"]:-0}"
    if [[ "$total" -gt 0 ]]; then
        CURRENT_JOB["progress"]=$(echo "scale=1; $completed * 100 / $total" | bc 2>/dev/null || echo "0")
    else
        CURRENT_JOB["progress"]="0"
    fi
}

# 获取上一个完成的 Job 信息
# Usage: stat_get_previous_job()
# Sets: PREVIOUS_JOB associative array
stat_get_previous_job() {
    PREVIOUS_JOB=("exists"=false)

    if ! command -v jq &>/dev/null || [[ ! -f "$STATUS_FILE" ]]; then
        return 0
    fi

    # 查找最近完成的 job
    local prev_module=""
    local prev_job=""
    local prev_time=""

    # 遍历所有模块和 jobs，找到 completed_at 最新的
    local modules=$(jq -r '(.modules // {}) | keys[]' "$STATUS_FILE" 2>/dev/null)
    for mod in $modules; do
        local jobs=$(jq -r --arg m "$mod" '(.modules[$m].jobs // {}) | keys[]' "$STATUS_FILE" 2>/dev/null)
        for jb in $jobs; do
            local status=$(jq -r --arg m "$mod" --arg j "$jb" '.modules[$m].jobs[$j].status // ""' "$STATUS_FILE" 2>/dev/null)
            local completed_at=$(jq -r --arg m "$mod" --arg j "$jb" '.modules[$m].jobs[$j].completed_at // ""' "$STATUS_FILE" 2>/dev/null)

            if [[ "$status" == "COMPLETED" && -n "$completed_at" ]]; then
                if [[ -z "$prev_time" || "$completed_at" > "$prev_time" ]]; then
                    prev_module="$mod"
                    prev_job="$jb"
                    prev_time="$completed_at"
                fi
            fi
        done
    done

    if [[ -n "$prev_module" && -n "$prev_job" ]]; then
        PREVIOUS_JOB["exists"]=true
        PREVIOUS_JOB["module"]="$prev_module"
        PREVIOUS_JOB["job"]="$prev_job"
        PREVIOUS_JOB["completed_at"]="$prev_time"
        PREVIOUS_JOB["tasks_total"]=$(jq -r --arg m "$prev_module" --arg j "$prev_job" '.modules[$m].jobs[$j].tasks_total // 0' "$STATUS_FILE" 2>/dev/null)
        PREVIOUS_JOB["tasks_completed"]=$(jq -r --arg m "$prev_module" --arg j "$prev_job" '.modules[$m].jobs[$j].tasks_completed // 0' "$STATUS_FILE" 2>/dev/null)
        PREVIOUS_JOB["started_at"]=$(jq -r --arg m "$prev_module" --arg j "$prev_job" '.modules[$m].jobs[$j].started_at // ""' "$STATUS_FILE" 2>/dev/null)
        PREVIOUS_JOB["last_summary"]=$(jq -r --arg m "$prev_module" --arg j "$prev_job" '.modules[$m].jobs[$j].last_summary // ""' "$STATUS_FILE" 2>/dev/null)

        # 计算耗时
        if [[ -n "${PREVIOUS_JOB["started_at"]}" && -n "${PREVIOUS_JOB["completed_at"]}" ]]; then
            PREVIOUS_JOB["elapsed"]=$(stat_calculate_elapsed "${PREVIOUS_JOB["started_at"]}" "${PREVIOUS_JOB["completed_at"]}")
        fi
    fi
}

# 获取所有需要 debug 的问题列表
# Usage: stat_get_debug_issues()
# Sets: DEBUG_ISSUES array
stat_get_debug_issues() {
    DEBUG_ISSUES=()

    if ! command -v jq &>/dev/null || [[ ! -f "$STATUS_FILE" ]]; then
        return 0
    fi

    local idx=0
    local modules=$(jq -r '(.modules // {}) | keys[]' "$STATUS_FILE" 2>/dev/null)
    for mod in $modules; do
        local jobs=$(jq -r --arg m "$mod" '(.modules[$m].jobs // {}) | keys[]' "$STATUS_FILE" 2>/dev/null)
        for jb in $jobs; do
            local debug_count=$(jq -r --arg m "$mod" --arg j "$jb" '.modules[$m].jobs[$j].debug_logs | length // 0' "$STATUS_FILE" 2>/dev/null)
            if [[ "$debug_count" -gt 0 ]]; then
                for i in $(seq 0 $((debug_count - 1))); do
                    local resolved=$(jq -r --arg m "$mod" --arg j "$jb" --argjson i "$i" '.modules[$m].jobs[$j].debug_logs[$i].resolved // false' "$STATUS_FILE" 2>/dev/null)
                    if [[ "$resolved" != "true" ]]; then
                        local phenomenon=$(jq -r --arg m "$mod" --arg j "$jb" --argjson i "$i" '.modules[$m].jobs[$j].debug_logs[$i].phenomenon // ""' "$STATUS_FILE" 2>/dev/null)
                        DEBUG_ISSUES[$idx]="[$mod/$jb] $phenomenon"
                        idx=$((idx + 1))
                    fi
                done
            fi
        done
    done
}

# 获取整体进度百分比
# Usage: stat_get_progress()
# Sets: PROGRESS_PERCENT variable
stat_get_progress() {
    PROGRESS_PERCENT="${STATUS_DATA["progress_percentage"]:-0}"

    # 如果没有百分比，手动计算
    if [[ "$PROGRESS_PERCENT" == "0" ]]; then
        local total="${STATUS_DATA["total_jobs"]:-0}"
        local completed="${STATUS_DATA["completed_jobs"]:-0}"
        if [[ "$total" -gt 0 ]]; then
            PROGRESS_PERCENT=$(echo "scale=1; $completed * 100 / $total" | bc 2>/dev/null || echo "0")
        fi
    fi
}

# 获取累计运行时间
# Usage: stat_get_elapsed_time()
# Returns: 格式化的 elapsed time string
stat_get_elapsed_time() {
    local session_start="${STATUS_DATA["session_start"]}"
    local current_time
    current_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    if [[ -z "$session_start" ]]; then
        echo "00:00:00"
        return 0
    fi

    echo "$(stat_calculate_elapsed "$session_start" "$current_time")"
}

# 计算两个 ISO 时间戳之间的差值
# Usage: stat_calculate_elapsed <start_time> <end_time>
# Returns: HH:MM:SS format
stat_calculate_elapsed() {
    local start="$1"
    local end="$2"

    # 转换到秒
    local start_sec end_sec

    if command -v date &>/dev/null; then
        start_sec=$(date -d "$start" +%s 2>/dev/null || date -u -d "$start" +%s 2>/dev/null || echo "0")
        end_sec=$(date -d "$end" +%s 2>/dev/null || date -u -d "$end" +%s 2>/dev/null || echo "0")
    else
        echo "00:00:00"
        return 0
    fi

    local diff=$((end_sec - start_sec))
    if [[ $diff -lt 0 ]]; then
        diff=0
    fi

    local hours=$((diff / 3600))
    local minutes=$(((diff % 3600) / 60))
    local seconds=$((diff % 60))

    printf "%02d:%02d:%02d" "$hours" "$minutes" "$seconds"
}

# ============================================================================
# 显示格式化函数
# ============================================================================

# 获取状态颜色
stat_status_color() {
    case "$1" in
        COMPLETED) echo "$GREEN" ;;
        RUNNING)   echo "$BLUE" ;;
        FAILED|INTERRUPTED) echo "$RED" ;;
        BLOCKED)   echo "$YELLOW" ;;
        PENDING)   echo "$GRAY" ;;
        *)         echo "$NC" ;;
    esac
}

# 获取状态图标
stat_status_icon() {
    case "$1" in
        COMPLETED) echo "$ICON_COMPLETED" ;;
        RUNNING)   echo "$ICON_RUNNING" ;;
        FAILED|INTERRUPTED) echo "$ICON_FAILED" ;;
        BLOCKED)   echo "$ICON_BLOCKED" ;;
        PENDING)   echo "$ICON_PENDING" ;;
        *)         echo "$ICON_PENDING" ;;
    esac
}

# 格式化表格输出
# Usage: stat_format_table()
stat_format_table() {
    local output=""

    # 获取数据
    stat_get_current_job
    stat_get_previous_job
    stat_get_debug_issues
    stat_get_progress

    local elapsed=$(stat_get_elapsed_time)

    # 标题
    echo ""
    echo -e "${CYAN}+==============================================================+${NC}"
    echo -e "${CYAN}|${NC}              ${YELLOW}Morty 执行状态监控${NC}                             ${CYAN}|${NC}"
    echo -e "${CYAN}+==============================================================+${NC}"
    echo ""

    # 当前执行状态
    echo -e "${BLUE}▸ 当前执行${NC}"
    echo -e "${TABLE_TOP_LEFT}$(printf '%*s' 60 '' | tr ' ' "$TABLE_HORIZONTAL")${TABLE_TOP_RIGHT}"

    if [[ "${CURRENT_JOB["exists"]}" == true ]]; then
        local status_color=$(stat_status_color "${CURRENT_JOB["status"]}")
        local status_icon=$(stat_status_icon "${CURRENT_JOB["status"]}")

        echo -e "${TABLE_VERTICAL} 模块         ${TABLE_VERTICAL} ${CURRENT_JOB["module"]:-N/A}"
        echo -e "${TABLE_VERTICAL} Job          ${TABLE_VERTICAL} ${CURRENT_JOB["job"]:-N/A}"
        echo -e "${TABLE_VERTICAL} 状态         ${TABLE_VERTICAL} ${status_color}${status_icon} ${CURRENT_JOB["status"]:-N/A}${NC}"
        echo -e "${TABLE_VERTICAL} 进度         ${TABLE_VERTICAL} ${CURRENT_JOB["tasks_completed"]:-0}/${CURRENT_JOB["tasks_total"]:-0} (${CURRENT_JOB["progress"]:-0}%)"
        echo -e "${TABLE_VERTICAL} 循环次数     ${TABLE_VERTICAL} ${CURRENT_JOB["loop_count"]:-0}"
    else
        printf "${TABLE_VERTICAL} %-58s ${TABLE_VERTICAL}\n" "  无正在执行的 Job"
    fi
    echo -e "${TABLE_LEFT_T}$(printf '%*s' 60 '' | tr ' ' "$TABLE_HORIZONTAL")${TABLE_RIGHT_T}"

    # 上一个完成的 Job
    echo -e "${GRAY}▸ 上一个 Job${NC}"
    if [[ "${PREVIOUS_JOB["exists"]}" == true ]]; then
        echo -e "${TABLE_VERTICAL} 模块         ${TABLE_VERTICAL} ${PREVIOUS_JOB["module"]:-N/A}"
        echo -e "${TABLE_VERTICAL} Job          ${TABLE_VERTICAL} ${PREVIOUS_JOB["job"]:-N/A}"
        echo -e "${TABLE_VERTICAL} 任务         ${TABLE_VERTICAL} ${PREVIOUS_JOB["tasks_completed"]:-0}/${PREVIOUS_JOB["tasks_total"]:-0}"
        echo -e "${TABLE_VERTICAL} 耗时         ${TABLE_VERTICAL} ${PREVIOUS_JOB["elapsed"]:-N/A}"

        # 截断摘要以适应行宽
        local summary="${PREVIOUS_JOB["last_summary"]:-}"
        if [[ -n "$summary" ]]; then
            summary="${summary:0:45}"
            [[ ${#PREVIOUS_JOB["last_summary"]} -gt 45 ]] && summary="${summary}..."
            echo -e "${TABLE_VERTICAL} 摘要         ${TABLE_VERTICAL} $summary"
        fi
    else
        echo -e "${TABLE_VERTICAL}   无已完成的 Job                                      ${TABLE_VERTICAL}"
    fi
    echo -e "${TABLE_LEFT_T}$(printf '%*s' 60 '' | tr ' ' "$TABLE_HORIZONTAL")${TABLE_RIGHT_T}"

    # Debug 问题
    echo -e "${GRAY}▸ Debug 问题${NC}"
    local issue_count=${#DEBUG_ISSUES[@]}
    if [[ $issue_count -gt 0 ]]; then
        local display_count=0
        for issue in "${DEBUG_ISSUES[@]}"; do
            if [[ $display_count -lt 5 ]]; then
                # 截断以适应行宽
                local display_issue="${issue:0:56}"
                [[ ${#issue} -gt 56 ]] && display_issue="${display_issue}..."
                echo -e "${TABLE_VERTICAL} ${RED}*${NC} $display_issue"
                display_count=$((display_count + 1))
            fi
        done
        if [[ $issue_count -gt 5 ]]; then
            echo -e "${TABLE_VERTICAL}   ${GRAY}... 还有 $((issue_count - 5)) 个问题 ...${NC}"
        fi
    else
        echo -e "${TABLE_VERTICAL}   ${GREEN}[ok]${NC} 没有待解决的 debug 问题                             ${TABLE_VERTICAL}"
    fi
    echo -e "${TABLE_LEFT_T}$(printf '%*s' 60 '' | tr ' ' "$TABLE_HORIZONTAL")${TABLE_RIGHT_T}"

    # 整体进度
    echo -e "${GRAY}▸ 整体进度${NC}"
    echo -e "${TABLE_VERTICAL} 模块         ${TABLE_VERTICAL} ${STATUS_DATA["completed_modules"]:-0}/${STATUS_DATA["total_modules"]:-0}"
    echo -e "${TABLE_VERTICAL} Jobs         ${TABLE_VERTICAL} ${STATUS_DATA["completed_jobs"]:-0}/${STATUS_DATA["total_jobs"]:-0}"
    echo -e "${TABLE_VERTICAL} 完成度       ${TABLE_VERTICAL} ${PROGRESS_PERCENT}%"

    # 进度条
    local bar_width=40
    local filled=$(echo "scale=0; $PROGRESS_PERCENT * $bar_width / 100" | bc 2>/dev/null || echo "0")
    local empty=$((bar_width - filled))
    local bar_filled=$(printf '%*s' "$filled" '' | tr ' ' '#')
    local bar_empty=$(printf '%*s' "$empty" '' | tr ' ' '-')
    echo -e "${TABLE_VERTICAL} 进度条       ${TABLE_VERTICAL} ${GREEN}${bar_filled}${GRAY}${bar_empty}${NC} ${PROGRESS_PERCENT}%"
    echo -e "${TABLE_LEFT_T}$(printf '%*s' 60 '' | tr ' ' "$TABLE_HORIZONTAL")${TABLE_RIGHT_T}"

    # 会话信息
    echo -e "${GRAY}▸ 会话信息${NC}"
    echo -e "${TABLE_VERTICAL} 累计时间     ${TABLE_VERTICAL} $elapsed"
    echo -e "${TABLE_VERTICAL} 总循环       ${TABLE_VERTICAL} ${STATUS_DATA["total_loops"]:-0}"
    echo -e "${TABLE_VERTICAL} 版本         ${TABLE_VERTICAL} ${STATUS_DATA["version"]:-unknown}"
    echo -e "${TABLE_BOTTOM_LEFT}$(printf '%*s' 60 '' | tr ' ' "$TABLE_HORIZONTAL")${TABLE_BOTTOM_RIGHT}"

    echo ""
}

# 监控模式
# Usage: stat_watch_mode()
stat_watch_mode() {
    WATCH_MODE=1

    # 清除屏幕并隐藏光标
    clear
    tput civis 2>/dev/null || true

    # 设置退出时的清理
    cleanup() {
        tput cnorm 2>/dev/null || true
        echo ""
        echo -e "${YELLOW}监控模式已退出${NC}"
        exit 0
    }
    trap cleanup INT TERM EXIT

    while true; do
        # 移动光标到左上角
        tput cup 0 0 2>/dev/null || clear

        # 显示时间戳
        echo -e "${CYAN}$(date '+%Y-%m-%d %H:%M:%S')${NC} - ${YELLOW}Morty 监控模式${NC} (按 Ctrl+C 退出)"
        echo ""

        # 重新加载并显示
        if stat_load_status &>/dev/null; then
            stat_format_table
        else
            echo -e "${RED}无法加载状态文件${NC}"
        fi

        echo ""
        echo -e "${GRAY}每 ${WATCH_INTERVAL} 秒刷新一次...${NC}"

        sleep "$WATCH_INTERVAL"
    done
}

# ============================================================================
# 帮助信息
# ============================================================================

show_help() {
    cat << 'EOF'
Usage: morty stat [OPTIONS]

显示 Morty 执行状态和进度监控大盘

Options:
    -t, --table     以表格形式输出（默认）
    -w, --watch     监控模式，每 60 秒原地刷新
    -h, --help      显示帮助信息

显示内容：
    ▸ 当前执行      当前正在执行的模块/Job/状态/进度
    ▸ 上一个 Job    最近完成的 Job 摘要和耗时
    ▸ Debug 问题    待解决的 debug 问题列表
    ▸ 整体进度      模块和 Job 完成度百分比
    ▸ 会话信息      累计运行时间和版本

Examples:
    morty stat              # 显示当前状态表格
    morty stat -t           # 显示当前状态表格（显式）
    morty stat -w           # 进入监控模式

EOF
}

# ============================================================================
# 主函数
# ============================================================================

main() {
    local mode="table"

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                show_help
                exit 0
                ;;
            -t|--table)
                mode="table"
                shift
                ;;
            -w|--watch)
                mode="watch"
                shift
                ;;
            *)
                echo -e "${RED}错误: 未知选项 $1${NC}" >&2
                show_help
                exit 1
                ;;
        esac
    done

    # 检查状态文件
    if ! stat_load_status; then
        exit 1
    fi

    # 执行对应模式
    case "$mode" in
        table)
            stat_format_table
            ;;
        watch)
            stat_watch_mode
            ;;
    esac
}

# 运行主函数
main "$@"
