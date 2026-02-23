#!/bin/bash
# Morty Doing Mode - 执行 Plan 的分层 TDD 开发
# 取代原有的 Loop 模式，支持 Job 级执行和断点恢复

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

# 版本和配置
VERSION="2.0"
MORTY_DIR=".morty"
PLAN_DIR="$MORTY_DIR/plan"
STATUS_FILE="$MORTY_DIR/status.json"
DOING_DIR="$MORTY_DIR/doing"
DOING_LOGS="$DOING_DIR/logs"
DOING_TESTS="$DOING_DIR/tests"
CLAUDE_CMD="${CLAUDE_CODE_CLI:-ai_cli}"
DOING_PROMPT="$SCRIPT_DIR/../prompts/doing.md"

# ============================================
# 帮助和入口函数
# ============================================

show_help() {
    cat << 'EOF'
Morty Doing 模式 - 执行 Plan 的分层 TDD 开发

用法: morty doing [选项]

选项:
    -h, --help              显示帮助信息
    --module <name>         只执行指定模块
    --job <name>            只执行指定 Job
    --restart               强制从头开始（忽略已有状态）

描述:
    Doing 模式执行 Plan 模式创建的开发计划，支持分层 TDD 开发范式
    （单元测试 → 集成测试 → 端到端测试）。

    工作流程:
    1. 读取 .morty/plan/*.md 中的开发计划
    2. 按顺序逐个执行 Job，支持断点自动恢复
    3. 所有 Jobs 完成后退出

    状态管理:
    - .morty/status.json 维护所有执行状态和进度
    - 默认从上次中断处自动恢复
    - 使用 --restart 强制从头开始

    执行流程:
    morty doing              # 从断点恢复执行（默认）
    morty doing --restart    # 强制从头开始
    morty doing --module config    # 只执行 config 模块
    morty doing --module config --job job_1  # 只执行指定 Job

示例:
    morty doing              # 启动 Doing 模式
    morty doing --restart    # 重置状态并从头执行

前置条件:
    - 必须先运行 morty plan 创建开发计划
    - .morty/plan/ 目录必须存在且包含有效的 Plan 文件

EOF
}

# ============================================
# 参数解析函数 (doing_parse_args)
# ============================================

# 命令行选项
OPTION_MODULE=""
OPTION_JOB=""
OPTION_RESTART=false

# 解析命令行参数
doing_parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            --module)
                if [[ -z "$2" || "$2" == --* ]]; then
                    log ERROR "--module 需要一个参数"
                    return 1
                fi
                OPTION_MODULE="$2"
                shift 2
                ;;
            --job)
                if [[ -z "$2" || "$2" == --* ]]; then
                    log ERROR "--job 需要一个参数"
                    return 1
                fi
                OPTION_JOB="$2"
                shift 2
                ;;
            --restart)
                OPTION_RESTART=true
                shift
                ;;
            *)
                log ERROR "未知参数: $1"
                show_help
                return 1
                ;;
        esac
    done
    return 0
}

# ============================================
# 状态管理函数 (doing_load_status)
# ============================================

# 定义有效的状态列表
DOING_VALID_STATES=("PENDING" "RUNNING" "COMPLETED" "FAILED" "BLOCKED")

# 状态转换规则定义
# 格式: "当前状态:目标状态" -> 是否允许
doing_is_valid_transition() {
    local from_state="$1"
    local to_state="$2"

    # 定义允许的状态转换
    case "${from_state}:${to_state}" in
        "PENDING:RUNNING")    return 0 ;;
        "PENDING:BLOCKED")    return 0 ;;
        "RUNNING:COMPLETED")  return 0 ;;
        "RUNNING:FAILED")     return 0 ;;
        "RUNNING:PENDING")    return 0 ;;  # 重试时允许重置为 PENDING
        "RUNNING:BLOCKED")    return 0 ;;
        "FAILED:PENDING")     return 0 ;;  # 重试时允许
        "FAILED:RUNNING")     return 0 ;;  # 重试时允许
        "BLOCKED:PENDING")    return 0 ;;  # 解除阻塞时允许
        "COMPLETED:PENDING")  return 0 ;;  # 重置时允许
        *)                    return 1 ;;  # 其他转换不允许
    esac
}

# 加载现有状态
doing_load_status() {
    if [[ ! -f "$STATUS_FILE" ]]; then
        log WARN "状态文件不存在: $STATUS_FILE"
        log INFO "首次运行，已初始化状态"
        return 1
    fi

    # 验证 JSON 格式
    if command -v jq &> /dev/null; then
        if ! jq empty "$STATUS_FILE" 2>/dev/null; then
            log ERROR "状态文件格式无效: $STATUS_FILE"
            return 1
        fi
    fi

    log INFO "状态文件加载成功: $STATUS_FILE"
    return 0
}

# ============================================
# Job 选择函数 (doing_select_job)
# ============================================

# 获取第一个未完成的 Job
doing_select_job() {
    local target_module="${1:-}"

    if ! command -v jq &> /dev/null; then
        echo ""
        return 1
    fi

    # 如果指定了模块，只查找该模块
    if [[ -n "$target_module" ]]; then
        jq -r --arg mod "$target_module" '
            .modules[$mod].jobs // {} | to_entries[] |
            select(.value.status == "PENDING" or .value.status == "FAILED") |
            "\($mod):\(.key)"
        ' "$STATUS_FILE" 2>/dev/null | head -1
    else
        # 按模块顺序查找第一个状态为 PENDING 或 FAILED 的 Job
        jq -r '
            .modules | to_entries[] |
            .key as $mod |
            .value.jobs | to_entries[] |
            select(.value.status == "PENDING" or .value.status == "FAILED") |
            "\($mod):\(.key)"
        ' "$STATUS_FILE" 2>/dev/null | head -1
    fi
}

# ============================================
# 状态重置函数 (doing_reset_status)
# ============================================

# 重置状态（仅重置状态文件，不影响 git 历史和工作目录）
doing_reset_status() {
    log WARN "重置状态..."

    # 只删除状态文件，保留目录结构和 git 历史
    if [[ -f "$STATUS_FILE" ]]; then
        rm -f "$STATUS_FILE"
        log INFO "状态文件已删除: $STATUS_FILE"
    fi

    # 保留 logs 和 tests 目录（历史记录有价值）
    # 只重置状态，不删除这些目录

    log SUCCESS "状态已重置"
    return 0
}

# ============================================
# 初始化和状态管理函数
# ============================================

# 检查前置条件
doing_check_prerequisites() {
    log INFO "检查 Doing 模式前置条件..."

    # 检查 .morty 目录
    if [[ ! -d "$MORTY_DIR" ]]; then
        log ERROR ".morty/ 目录不存在"
        log INFO ""
        log INFO "请先运行 morty research 和 morty plan 创建开发计划"
        return 1
    fi

    # 检查 plan 目录
    if [[ ! -d "$PLAN_DIR" ]]; then
        log ERROR ".morty/plan/ 目录不存在"
        log INFO ""
        log INFO "请先运行 morty plan 创建开发计划"
        return 1
    fi

    # 检查 plan 目录是否有 .md 文件
    local plan_files=$(find "$PLAN_DIR" -name "*.md" -type f 2>/dev/null || true)
    if [[ -z "$plan_files" ]]; then
        log ERROR ".morty/plan/ 目录中没有 Plan 文件"
        log INFO ""
        log INFO "请先运行 morty plan 创建开发计划"
        return 1
    fi

    # 检查提示词文件
    if [[ ! -f "$DOING_PROMPT" ]]; then
        log ERROR "Doing 模式提示词未找到: $DOING_PROMPT"
        log ERROR "请先运行安装: ./install.sh"
        return 1
    fi

    # 创建必要目录
    mkdir -p "$DOING_LOGS"
    mkdir -p "$DOING_TESTS"

    log SUCCESS "前置条件检查通过"
    return 0
}

# 解析单个 Plan 文件，提取模块和 Jobs 信息
parse_plan_file() {
    local plan_file="$1"
    local module_name=$(basename "$plan_file" .md)

    # 使用 awk 解析 markdown 文件
    awk -v module="$module_name" '
        BEGIN {
            in_job = 0
            in_tasks = 0
            in_validator = 0
            job_count = 0
            current_job = ""
        }

        # 检测 Job 标题
        /^### Job [0-9]+:/ {
            in_job = 1
            in_tasks = 0
            in_validator = 0
            job_count++
            # 提取 Job 名 (Job N: 名称)
            gsub(/^### /, "")
            current_job = "job_" job_count
            jobs[current_job, "title"] = $0
            jobs[current_job, "name"] = current_job
            next
        }

        # 检测前置条件
        in_job && /^\*\*前置条件\*\*:/ {
            in_tasks = 0
            in_validator = 0
            next
        }
        in_job && /^\*\*前置条件\*\*/ {
            in_tasks = 0
            in_validator = 0
            next
        }

        # 检测 Tasks 区域
        in_job && /Tasks.*Todo/ {
            in_tasks = 1
            in_validator = 0
            task_count = 0
            next
        }

        # 收集 Tasks
        in_tasks && /^- \[([ x])\] (.+)/ {
            match($0, /^- \[([ x])\] (.+)/, arr)
            status = arr[1]
            task = arr[2]
            task_count++
            tasks[current_job, task_count] = task
            task_status[current_job, task_count] = (status == "x") ? "completed" : "pending"
            task_module[current_job, task_count] = module
        }

        # 检测 Tasks 区域结束
        in_tasks && (/^---$/ || /^\*\*验证器\*\*/ || /^## /) {
            in_tasks = 0
        }

        # 检测验证器区域
        in_job && /^\*\*验证器\*\*:/ {
            in_validator = 1
            in_tasks = 0
            validator_count = 0
            next
        }
        in_job && /^\*\*验证器\*\*/ {
            in_validator = 1
            in_tasks = 0
            validator_count = 0
            next
        }

        # 收集验证器内容
        in_validator && /^- / {
            validator_count++
            validators[current_job, validator_count] = substr($0, 3)
            validator_module[current_job, validator_count] = module
        }

        # 检测 Job 区域结束
        /^---$/ && in_job {
            in_job = 0
            in_tasks = 0
            in_validator = 0
        }

        END {
            # 输出模块信息
            print "MODULE:name=" module
            print "MODULE:jobs=" job_count

            # 输出每个 Job 的信息
            for (i = 1; i <= job_count; i++) {
                job_name = "job_" i
                print "JOB:" job_name ":title=" jobs[job_name, "title"]
                print "JOB:" job_name ":module=" module

                # 输出 tasks
                tc = 0
                for (t = 1; t <= 100; t++) {
                    if ((job_name, t) in tasks) {
                        tc++
                    }
                }
                print "JOB:" job_name ":tasks=" tc

                for (t = 1; t <= tc; t++) {
                    print "TASK:" module ":" job_name ":" t ":" tasks[job_name, t] ":" task_status[job_name, t]
                }

                # 输出 validators
                vc = 0
                for (v = 1; v <= 100; v++) {
                    if ((job_name, v) in validators) {
                        vc++
                    }
                }
                print "JOB:" job_name ":validators=" vc

                for (v = 1; v <= vc; v++) {
                    print "VALIDATOR:" module ":" job_name ":" v ":" validators[job_name, v]
                }
            }
        }
    ' "$plan_file"
}

# 加载所有 Plan 文件
doing_load_plan() {
    log INFO "加载 Plan 文件..."

    local plan_files=$(find "$PLAN_DIR" -name "*.md" -type f 2>/dev/null | sort)
    local total_modules=0
    local total_jobs=0

    # 创建临时文件存储解析结果
    local temp_parse=$(mktemp)

    # 解析所有 Plan 文件
    while IFS= read -r plan_file; do
        [[ -f "$plan_file" ]] || continue

        local module_name=$(basename "$plan_file" .md)

        # 跳过 README.md 和生产测试文件（特殊处理）
        [[ "$module_name" == "README" ]] && continue
        [[ "$module_name" == "生产测试" ]] && continue
        [[ "$module_name" == "[生产测试]" ]] && continue

        log INFO "  解析: $(basename "$plan_file")"

        # 解析 Plan 文件内容
        parse_plan_file "$plan_file" >> "$temp_parse"

        ((total_modules++))
    done <<< "$plan_files"

    # 设置全局变量
    PLAN_MODULES="$total_modules"
    PLAN_PARSE_RESULT="$temp_parse"

    # 统计 Jobs 数量
    total_jobs=$(grep -c "^JOB:job_" "$temp_parse" 2>/dev/null || echo "0")
    PLAN_JOBS="$total_jobs"

    log SUCCESS "加载完成: $total_modules 个模块, $total_jobs 个 Jobs"
}

# 从解析结果中获取模块列表
get_modules_from_parse() {
    local parse_file="${1:-$PLAN_PARSE_RESULT}"
    grep "^MODULE:name=" "$parse_file" 2>/dev/null | cut -d'=' -f2 | sort -u
}

# 从解析结果中获取模块的 Jobs
get_module_jobs() {
    local module="$1"
    local parse_file="${2:-$PLAN_PARSE_RESULT}"

    awk -v mod="$module" -F':' '
        /^JOB:/ {
            job_name = $2
            # 提取 key=value 部分
            kv = $3
            for (i = 4; i <= NF; i++) {
                kv = kv ":" $i
            }
            # 分割 key=value
            split(kv, arr, "=")
            key = arr[1]
            value = arr[2]
            if (key == "module" && value == mod) {
                print job_name
            }
        }
    ' "$parse_file" | sort -u
}

# 从解析结果中获取 Job 的 Tasks
get_job_tasks() {
    local module="$1"
    local job="$2"
    local parse_file="${3:-$PLAN_PARSE_RESULT}"

    awk -v mod="$module" -v j="$job" -F':' '
        /^TASK:/ && $2 == mod && $3 == j {
            print $4 ":" $5 ":" $6
        }
    ' "$parse_file"
}

# 从解析结果中获取 Job 的验证器
get_job_validators() {
    local module="$1"
    local job="$2"
    local parse_file="${3:-$PLAN_PARSE_RESULT}"

    awk -v mod="$module" -v j="$job" -F':' '
        /^VALIDATOR:/ && $2 == mod && $3 == j {
            print $4 ":" $5
        }
    ' "$parse_file"
}

# 初始化 status.json
doing_init_status() {
    log INFO "初始化状态文件..."

    local timestamp=$(get_iso_timestamp)
    local modules_json=""
    local total_jobs=0

    # 构建模块和 Jobs 的初始状态
    local modules=$(get_modules_from_parse)

    while IFS= read -r module; do
        [[ -n "$module" ]] || continue

        local jobs_json=""
        local module_jobs=$(get_module_jobs "$module")
        local job_count=0

        while IFS= read -r job; do
            [[ -n "$job" ]] || continue

            # 获取该 Job 的 Tasks 数量
            local task_count=$(get_job_tasks "$module" "$job" | wc -l)

            jobs_json="$jobs_json
    \"$job\": {
      \"status\": \"PENDING\",
      \"loop_count\": 0,
      \"retry_count\": 0,
      \"tasks_total\": $task_count,
      \"tasks_completed\": 0,
      \"debug_logs\": []
    },"

            ((job_count++))
            ((total_jobs++))
        done <<< "$module_jobs"

        # 移除最后一个逗号
        jobs_json="${jobs_json%,}"

        modules_json="$modules_json
  \"$module\": {
    \"status\": \"pending\",
    \"jobs\": {$jobs_json
    }
  },"

    done <<< "$modules"

    # 移除最后一个逗号
    modules_json="${modules_json%,}"

    # 构建完整的 status.json
    cat > "$STATUS_FILE" << EOF
{
  "version": "$VERSION",
  "state": "running",
  "current": {
    "module": null,
    "job": null,
    "status": null,
    "start_time": "$timestamp"
  },
  "session": {
    "start_time": "$timestamp",
    "last_update": "$timestamp",
    "total_loops": 0
  },
  "modules": {$modules_json
  },
  "summary": {
    "total_modules": $PLAN_MODULES,
    "completed_modules": 0,
    "running_modules": 0,
    "pending_modules": $PLAN_MODULES,
    "blocked_modules": 0,
    "total_jobs": $total_jobs,
    "completed_jobs": 0,
    "running_jobs": 0,
    "failed_jobs": 0,
    "blocked_jobs": 0,
    "progress_percentage": 0
  }
}
EOF

    log SUCCESS "状态文件已创建: $STATUS_FILE"
}

# 保存状态到 status.json
doing_save_status() {
    # 更新最后更新时间
    local timestamp=$(get_iso_timestamp)

    # 使用 jq 更新 last_update 字段（如果可用）
    if command -v jq &> /dev/null; then
        local temp_file=$(mktemp)
        jq --arg ts "$timestamp" '.session.last_update = $ts' "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
    fi
}

# 获取 Job 当前状态
doing_get_job_status() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        echo "PENDING"
        return
    fi

    jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].status // "PENDING"' \
        "$STATUS_FILE" 2>/dev/null || echo "PENDING"
}

# 更新 Job 状态（带状态转换验证）
doing_update_job_status() {
    local module="$1"
    local job="$2"
    local new_status="$3"

    if ! command -v jq &> /dev/null; then
        log WARN "jq 未安装，无法更新状态"
        return 1
    fi

    # 获取当前状态
    local current_status=$(doing_get_job_status "$module" "$job")

    # 验证状态转换是否合法
    if ! doing_is_valid_transition "$current_status" "$new_status"; then
        log ERROR "无效的状态转换: $current_status → $new_status"
        return 1
    fi

    local timestamp=$(get_iso_timestamp)
    local temp_file=$(mktemp)

    # 根据目标状态更新不同字段
    case "$new_status" in
        "RUNNING")
            # PENDING/FAILED → RUNNING: 记录开始时间，增加 loop_count
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg status "$new_status" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].status = $status |
                .modules[$mod].jobs[$job].started_at = $ts |
                .modules[$mod].jobs[$job].loop_count += 1 |
                .current.module = $mod |
                .current.job = $job |
                .current.status = $status |
                .current.start_time = $ts |
                .session.last_update = $ts |
                .session.total_loops += 1' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
            ;;
        "COMPLETED")
            # RUNNING → COMPLETED: 记录完成时间
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg status "$new_status" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].status = $status |
                .modules[$mod].jobs[$job].completed_at = $ts |
                .current.module = $mod |
                .current.job = $job |
                .current.status = $status |
                .current.completed_at = $ts |
                .session.last_update = $ts' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
            ;;
        "FAILED")
            # RUNNING → FAILED: 记录完成时间和增加 retry_count
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg status "$new_status" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].status = $status |
                .modules[$mod].jobs[$job].completed_at = $ts |
                .modules[$mod].jobs[$job].retry_count += 1 |
                .current.module = $mod |
                .current.job = $job |
                .current.status = $status |
                .current.completed_at = $ts |
                .session.last_update = $ts' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
            ;;
        "BLOCKED")
            # 任何状态 → BLOCKED: 标记为阻塞
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg status "$new_status" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].status = $status |
                .modules[$mod].jobs[$job].blocked_at = $ts |
                .current.module = $mod |
                .current.job = $job |
                .current.status = $status |
                .session.last_update = $ts' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
            ;;
        *)
            # 其他状态直接更新
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg status "$new_status" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].status = $status |
                .current.module = $mod |
                .current.job = $job |
                .current.status = $status |
                .session.last_update = $ts' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
            ;;
    esac

    log INFO "状态更新: $module/$job $current_status → $new_status"
}

# 重置 Job 状态（用于 --restart 选项）
doing_reset_job_status() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        log WARN "jq 未安装，无法重置状态"
        return 1
    fi

    if [[ ! -f "$STATUS_FILE" ]]; then
        return 0
    fi

    local temp_file=$(mktemp)
    local timestamp=$(get_iso_timestamp)

    # 重置指定 job 的所有状态字段为初始值
    jq --arg mod "$module" \
       --arg job "$job" \
       --arg ts "$timestamp" \
       '.modules[$mod].jobs[$job].status = "PENDING" |
        .modules[$mod].jobs[$job].started_at = "" |
        .modules[$mod].jobs[$job].completed_at = "" |
        .modules[$mod].jobs[$job].loop_count = 0 |
        .modules[$mod].jobs[$job].retry_count = 0 |
        .modules[$mod].jobs[$job].tasks_completed = 0 |
        .modules[$mod].jobs[$job].error_count = 0 |
        .modules[$mod].jobs[$job].interrupted = false |
        .modules[$mod].jobs[$job].interrupted_at = "" |
        .modules[$mod].jobs[$job].last_summary = "" |
        .session.last_update = $ts' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"

    log INFO "Job 状态已重置: $module/$job"
}

# 标记 Task 完成
doing_mark_task_complete() {
    local module="$1"
    local job="$2"
    local task_index="$3"

    if ! command -v jq &> /dev/null; then
        return 1
    fi

    # 检查 Task 是否已经被标记完成，避免重复计数
    if doing_is_task_completed "$module" "$job" "$task_index"; then
        return 0
    fi

    local temp_file=$(mktemp)
    local timestamp=$(get_iso_timestamp)

    # 增加 tasks_completed 计数
    jq --arg mod "$module" \
       --arg job "$job" \
       --arg ts "$timestamp" \
       '.modules[$mod].jobs[$job].tasks_completed += 1 |
        .session.last_update = $ts' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
}

# ============================================
# 状态转换专用函数 (Job 4 Task 3 & 4)
# ============================================

# 获取最大重试次数
DOING_MAX_RETRY_COUNT=3

# 获取当前 retry_count
doing_get_retry_count() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        echo "0"
        return
    fi

    jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].retry_count // 0' \
        "$STATUS_FILE" 2>/dev/null || echo "0"
}

# 检查是否需要重试
doing_should_retry() {
    local module="$1"
    local job="$2"

    local retry_count=$(doing_get_retry_count "$module" "$job")

    if [[ "$retry_count" -lt "$DOING_MAX_RETRY_COUNT" ]]; then
        return 0  # 需要重试
    else
        return 1  # 不需要重试，已达到最大重试次数
    fi
}

# 标记 Job 为完成状态
doing_mark_completed() {
    local module="$1"
    local job="$2"
    local summary="${3:-Job completed successfully}"

    log INFO "标记 Job 完成: $module/$job"

    # 检查当前状态是否允许转换到 COMPLETED
    local current_status=$(doing_get_job_status "$module" "$job")

    # 如果已经是 COMPLETED，跳过状态更新和 Git 提交（避免重复）
    if [[ "$current_status" == "COMPLETED" ]]; then
        log INFO "Job 已经是 COMPLETED 状态，跳过完成标记"
        return 0
    fi

    if [[ "$current_status" != "RUNNING" ]]; then
        log WARN "Job 当前状态不是 RUNNING ($current_status)，但仍标记为 COMPLETED"
    fi

    # 更新状态为 COMPLETED
    if ! doing_update_job_status "$module" "$job" "COMPLETED"; then
        log ERROR "无法更新 Job 状态为 COMPLETED"
        return 1
    fi

    # 记录完成摘要
    if command -v jq &> /dev/null; then
        local temp_file=$(mktemp)
        local timestamp=$(get_iso_timestamp)
        jq --arg mod "$module" \
           --arg job "$job" \
           --arg summary "$summary" \
           --arg ts "$timestamp" \
           '.modules[$mod].jobs[$job].completion_summary = $summary |
            .modules[$mod].jobs[$job].completed_at = $ts' \
           "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
    fi

    # 创建 Git 提交记录本次 Job 完成
    version_create_loop_commit "$module" "$job" "COMPLETED" "$summary"

    log SUCCESS "Job $module/$job 已完成: $summary"
    return 0
}

# 标记 Job 为失败状态，并处理重试逻辑
doing_mark_failed() {
    local module="$1"
    local job="$2"
    local reason="${3:-Unknown error}"

    log ERROR "标记 Job 失败: $module/$job - $reason"

    # 获取当前重试次数
    local retry_count=$(doing_get_retry_count "$module" "$job")
    local new_retry_count=$((retry_count + 1))

    # 检查是否还可以重试
    if [[ "$new_retry_count" -lt "$DOING_MAX_RETRY_COUNT" ]]; then
        log WARN "Job 失败 ($new_retry_count/$DOING_MAX_RETRY_COUNT)，将重试..."

        # 记录失败信息但不改变状态（保持在 RUNNING 或重置为 PENDING 重试）
        if command -v jq &> /dev/null; then
            local temp_file=$(mktemp)
            local timestamp=$(get_iso_timestamp)
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg reason "$reason" \
               --arg retry_count "$new_retry_count" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].retry_count = ($retry_count | tonumber) |
                .modules[$mod].jobs[$job].last_failure_reason = $reason |
                .modules[$mod].jobs[$job].last_failure_at = $ts' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
        fi

        # 重置状态为 PENDING 以便重试
        if ! doing_update_job_status "$module" "$job" "PENDING"; then
            log ERROR "无法重置 Job 状态为 PENDING"
            return 1
        fi

        log INFO "Job 已重置为 PENDING，准备重试 ($new_retry_count/$DOING_MAX_RETRY_COUNT)"
        return 2  # 返回 2 表示需要重试
    else
        log ERROR "Job 已达到最大重试次数 ($DOING_MAX_RETRY_COUNT)，标记为 FAILED"

        # 更新状态为 FAILED
        if ! doing_update_job_status "$module" "$job" "FAILED"; then
            log ERROR "无法更新 Job 状态为 FAILED"
            return 1
        fi

        # 记录最终失败信息
        if command -v jq &> /dev/null; then
            local temp_file=$(mktemp)
            local timestamp=$(get_iso_timestamp)
            jq --arg mod "$module" \
               --arg job "$job" \
               --arg reason "$reason" \
               --arg retry_count "$new_retry_count" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].final_failure_reason = $reason |
                .modules[$mod].jobs[$job].failed_at = $ts |
                .modules[$mod].jobs[$job].retry_exhausted = true' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
        fi

        return 1  # 返回 1 表示彻底失败
    fi
}

# 重置失败 Job 以便重新执行（手动重试）
doing_reset_failed_job() {
    local module="$1"
    local job="$2"

    log WARN "手动重置 Job: $module/$job"

    # 只能重置 FAILED 或 BLOCKED 状态的 Job
    local current_status=$(doing_get_job_status "$module" "$job")
    if [[ "$current_status" != "FAILED" && "$current_status" != "BLOCKED" ]]; then
        log ERROR "只能重置 FAILED 或 BLOCKED 状态的 Job (当前: $current_status)"
        return 1
    fi

    if ! command -v jq &> /dev/null; then
        log WARN "jq 未安装，无法重置 Job"
        return 1
    fi

    local temp_file=$(mktemp)
    local timestamp=$(get_iso_timestamp)

    # 重置 Job 状态
    jq --arg mod "$module" \
       --arg job "$job" \
       --arg ts "$timestamp" \
       '.modules[$mod].jobs[$job].status = "PENDING" |
        .modules[$mod].jobs[$job].retry_count = 0 |
        .modules[$mod].jobs[$job].loop_count = 0 |
        .modules[$mod].jobs[$job].tasks_completed = 0 |
        .modules[$mod].jobs[$job].reset_at = $ts |
        del(.modules[$mod].jobs[$job].completed_at) |
        del(.modules[$mod].jobs[$job].failed_at) |
        del(.modules[$mod].jobs[$job].blocked_at) |
        del(.modules[$mod].jobs[$job].final_failure_reason) |
        del(.modules[$mod].jobs[$job].last_failure_reason) |
        del(.modules[$mod].jobs[$job].retry_exhausted) |
        del(.modules[$mod].jobs[$job].completion_summary)' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"

    log SUCCESS "Job $module/$job 已重置为 PENDING 状态"
    return 0
}

# 记录一次循环
doing_record_loop() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        return 1
    fi

    local temp_file=$(mktemp)
    local timestamp=$(get_iso_timestamp)

    jq --arg mod "$module" \
       --arg job "$job" \
       --arg ts "$timestamp" \
       '.modules[$mod].jobs[$job].loop_count += 1 |
        .session.total_loops += 1 |
        .session.last_update = $ts' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
}

# ============================================
# Job 执行引擎 (Job 2)
# ============================================

# 全局变量用于信号处理
declare -g CURRENT_MODULE=""
declare -g CURRENT_JOB=""
declare -g INTERRUPT_RECEIVED=false

# 信号处理函数 - Ctrl+C 优雅中断
doing_interrupt_handler() {
    log WARN ""
    log WARN "收到中断信号 (Ctrl+C)，正在优雅退出..."
    INTERRUPT_RECEIVED=true

    # 如果正在执行 Job，保存当前状态
    if [[ -n "$CURRENT_MODULE" && -n "$CURRENT_JOB" ]]; then
        log INFO "保存当前 Job 状态: $CURRENT_MODULE/$CURRENT_JOB"

        # 更新状态为中断前的状态（保持 RUNNING 以便下次恢复）
        local timestamp=$(get_iso_timestamp)
        local temp_file=$(mktemp)

        if command -v jq &> /dev/null; then
            # 记录中断信息，但保持 Job 状态为 RUNNING（以便恢复）
            jq --arg mod "$CURRENT_MODULE" \
               --arg job "$CURRENT_JOB" \
               --arg ts "$timestamp" \
               '.modules[$mod].jobs[$job].interrupted_at = $ts |
                .modules[$mod].jobs[$job].interrupted = true |
                .current.module = $mod |
                .current.job = $job |
                .current.status = "INTERRUPTED" |
                .current.interrupted_at = $ts |
                .session.last_update = $ts' \
               "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
        fi

        log INFO "状态已保存，可以稍后从断点恢复"
    fi

    # 清理临时文件
    if [[ -n "$PLAN_PARSE_RESULT" && -f "$PLAN_PARSE_RESULT" ]]; then
        rm -f "$PLAN_PARSE_RESULT"
    fi

    log INFO "退出 Doing 模式"
    exit 130  # 128 + SIGINT(2)
}

# 设置信号捕获
doing_setup_signal_handlers() {
    trap 'doing_interrupt_handler' INT TERM
}

# 检查 Task 是否已完成
doing_is_task_completed() {
    local module="$1"
    local job="$2"
    local task_index="$3"

    # 从 status.json 中读取 tasks_completed
    if ! command -v jq &> /dev/null; then
        return 1
    fi

    local tasks_completed=$(jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].tasks_completed // 0' \
        "$STATUS_FILE" 2>/dev/null || echo "0")

    if [[ "$tasks_completed" -ge "$task_index" ]]; then
        return 0  # 已完成
    else
        return 1  # 未完成
    fi
}

# 获取 Job 的 tasks_total
doing_get_tasks_total() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        echo "0"
        return
    fi

    jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].tasks_total // 0' \
        "$STATUS_FILE" 2>/dev/null || echo "0"
}

# 获取 Job 的 tasks_completed
# shellcheck disable=SC2317
# Note: This function is used dynamically in doing_show_job_report and doing_generate_html_report
doing_get_tasks_completed() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        echo "0"
        return
    fi

    jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].tasks_completed // 0' \
        "$STATUS_FILE" 2>/dev/null || echo "0"
}

# 获取当前 loop_count
doing_get_loop_count() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        echo "0"
        return
    fi

    jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].loop_count // 0' \
        "$STATUS_FILE" 2>/dev/null || echo "0"
}

# 解析执行结果 (RALPH_STATUS) - 使用 JSON 格式
# 首先尝试使用 jq 解析 JSON 输出，如果失败则回退到文本解析
doing_parse_execution_result() {
    local output_file="$1"

    # 检查输出文件是否存在
    if [[ ! -f "$output_file" ]]; then
        log ERROR "输出文件不存在: $output_file"
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

                log INFO "JSON 解析成功: ralph_status 对象"
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
                log INFO "JSON 解析成功: 顶层字段"
                echo "status=${result_status};tasks_completed=${tasks_completed};tasks_total=${tasks_total};summary=${summary};module=${module};job=${job}"
                return 0
            fi
        fi
    fi

    # JSON 解析失败或 jq 不可用，回退到文本解析
    log WARN "JSON 解析失败，回退到文本解析..."
    _doing_parse_execution_result_text "$output_file"
}

# 内部函数：文本解析方式（回退方案）
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
        log WARN "未找到 RALPH_STATUS 块，尝试解析其他格式..."
        # 如果没有 RALPH_STATUS，根据退出码判断
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

# 更新 Task 详细状态（包括描述和完成标记）
doing_update_task_detail() {
    local module="$1"
    local job="$2"
    local task_index="$3"
    local task_status="$4"
    local task_description="$5"

    if ! command -v jq &> /dev/null; then
        return 1
    fi

    local temp_file=$(mktemp)
    local timestamp=$(get_iso_timestamp)

    # 创建或更新 tasks 对象
    jq --arg mod "$module" \
       --arg job "$job" \
       --arg idx "$task_index" \
       --arg status "$task_status" \
       --arg desc "$task_description" \
       --arg ts "$timestamp" \
       '.modules[$mod].jobs[$job].tasks[$idx] = {
            "index": ($idx | tonumber),
            "status": $status,
            "description": $desc,
            "updated_at": $ts
        } |
        .session.last_update = $ts' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
}

# 从执行输出中提取 debug_log 信息
doing_extract_debug_logs() {
    local module="$1"
    local job="$2"
    local output_file="$3"

    if [[ ! -f "$output_file" ]]; then
        return 0
    fi

    # 查找错误信息
    local errors=$(grep -n "ERROR\|FAILED\|错误\|失败" "$output_file" 2>/dev/null | head -5)

    if [[ -n "$errors" ]]; then
        log WARN "从输出中检测到错误信息，准备记录 debug_log"

        # 提取错误详情
        local phenomenon=$(echo "$errors" | head -1 | cut -d':' -f2- | tr -d '"' | head -c 200)
        local reproduction="查看日志文件: $output_file"
        local hypotheses="执行过程中出现错误，可能原因: 1)代码问题 2)环境配置 3)依赖缺失"
        local fix="需要人工检查日志并修复问题"

        doing_add_debug_log "$module" "$job" "$phenomenon" "$reproduction" "$hypotheses" "$fix"
    fi
}

# 添加 debug_log
doing_add_debug_log() {
    local module="$1"
    local job="$2"
    local phenomenon="$3"
    local reproduction="$4"
    local hypotheses="$5"
    local fix="$6"

    if ! command -v jq &> /dev/null; then
        log WARN "jq 未安装，无法添加 debug_log"
        return 1
    fi

    local timestamp=$(get_iso_timestamp)
    local temp_file=$(mktemp)

    # 构建 debug_log 条目
    local debug_entry=$(jq -n \
        --arg ts "$timestamp" \
        --arg phenomenon "$phenomenon" \
        --arg reproduction "$reproduction" \
        --arg hypotheses "$hypotheses" \
        --arg fix "$fix" \
        '{
            id: now,
            timestamp: $ts,
            phenomenon: $phenomenon,
            reproduction: $reproduction,
            hypotheses: ($hypotheses | split(",")),
            fix: $fix,
            resolved: false
        }')

    # 添加到 status.json
    jq --arg mod "$module" \
       --arg job "$job" \
       --argjson entry "$debug_entry" \
       '.modules[$mod].jobs[$job].debug_logs += [$entry]' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"

    log INFO "添加 debug_log: $module/$job - $phenomenon"
}

# 检查是否有未解决的 debug_log
doing_has_unresolved_debug_log() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        return 1
    fi

    local unresolved=$(jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].debug_logs // [] | map(select(.resolved == false)) | length' \
        "$STATUS_FILE" 2>/dev/null || echo "0")

    if [[ "$unresolved" -gt 0 ]]; then
        return 0  # 有未解决的 debug_log
    else
        return 1  # 没有未解决的 debug_log
    fi
}

# 检查 Job 是否被中断过
doing_was_interrupted() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        return 1
    fi

    local interrupted=$(jq -r --arg mod "$module" --arg job "$job" \
        '.modules[$mod].jobs[$job].interrupted // false' \
        "$STATUS_FILE" 2>/dev/null || echo "false")

    if [[ "$interrupted" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# 清除中断标记
doing_clear_interrupt_flag() {
    local module="$1"
    local job="$2"

    if ! command -v jq &> /dev/null; then
        return 1
    fi

    local temp_file=$(mktemp)
    jq --arg mod "$module" \
       --arg job "$job" \
       'del(.modules[$mod].jobs[$job].interrupted) |
        del(.modules[$mod].jobs[$job].interrupted_at)' \
       "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
}

# 从断点恢复 Job 执行（改进版中断恢复逻辑）
doing_resume_job() {
    local module="$1"
    local job="$2"

    log INFO "恢复 Job: $module/$job"

    # 处理 --restart 选项：重置指定 job 的状态
    if [[ "$OPTION_RESTART" == true ]]; then
        log WARN "  --restart 选项已指定，重置 Job 状态: $module/$job"
        doing_reset_job_status "$module" "$job"
    fi

    local status=$(doing_get_job_status "$module" "$job")

    # 如果已完成，跳过
    if [[ "$status" == "COMPLETED" ]]; then
        log INFO "  Job 已完成，跳过"
        return 0
    fi

    # 检查是否已达到最大重试次数
    local retry_count=$(doing_get_retry_count "$module" "$job")
    if [[ "$retry_count" -ge "$DOING_MAX_RETRY_COUNT" ]]; then
        log ERROR "  Job 已达到最大重试次数 ($DOING_MAX_RETRY_COUNT)，跳过"
        # 确保状态为 FAILED
        if [[ "$status" != "FAILED" ]]; then
            doing_update_job_status "$module" "$job" "FAILED"
        fi
        return 1
    fi

    # 检查是否有未解决的 debug_log
    if doing_has_unresolved_debug_log "$module" "$job"; then
        log WARN "  发现未解决的 debug_log，进入重试模式"
        doing_record_loop "$module" "$job"
    fi

    # 检查是否是从中断恢复
    if doing_was_interrupted "$module" "$job"; then
        log WARN "  检测到上次执行被中断，从断点恢复..."
        doing_clear_interrupt_flag "$module" "$job"
        # 保持 RUNNING 状态，不增加 loop_count（因为是继续执行）
    else
        # 新的执行循环，设置状态为 RUNNING（这会触发状态转换验证和 loop_count 增加）
        doing_update_job_status "$module" "$job" "RUNNING"
    fi

    # 执行 Job
    doing_execute_job "$module" "$job"
    local result=$?

    # 根据执行结果处理
    case $result in
        0)
            # 执行成功，标记为 COMPLETED
            doing_mark_completed "$module" "$job" "Job executed successfully"
            return 0
            ;;
        130)
            # 被中断，保持当前状态（已经在信号处理器中保存）
            log WARN "  Job 执行被中断"
            return 130
            ;;
        *)
            # 执行失败，处理失败逻辑（包括重试）
            log ERROR "  Job 执行失败 (退出码: $result)"
            if doing_mark_failed "$module" "$job" "Execution failed with exit code $result"; then
                # 返回 2 表示需要重试
                log INFO "  Job 将重试..."
                return 2
            else
                # 返回 1 表示彻底失败
                log ERROR "  Job 彻底失败"
                return 1
            fi
            ;;
    esac
}

# 执行单个 Job
doing_execute_job() {
    local module="$1"
    local job="$2"

    # 设置全局变量用于信号处理
    CURRENT_MODULE="$module"
    CURRENT_JOB="$job"
    INTERRUPT_RECEIVED=false

    log INFO "执行 Job: $module/$job"

    local tasks_total=$(doing_get_tasks_total "$module" "$job")
    local current_loop=$(doing_get_loop_count "$module" "$job")

    log INFO "  总任务数: $tasks_total, 当前循环: $current_loop"

    # 执行每个 Task
    local task_index=1
    while [[ $task_index -le $tasks_total ]]; do
        if doing_is_task_completed "$module" "$job" "$task_index"; then
            log INFO "  Task $task_index 已完成，跳过"
        else
            log INFO "  执行 Task $task_index/$tasks_total..."

            # 获取 Task 描述
            local task_desc=$(get_job_tasks "$module" "$job" | sed -n "${task_index}p" | cut -d':' -f2)
            log INFO "    任务: $task_desc"

            # 执行 Task
            if doing_run_task "$module" "$job" "$task_index" "$task_desc"; then
                doing_mark_task_complete "$module" "$job" "$task_index"
                log SUCCESS "  Task $task_index 完成"
            else
                log ERROR "  Task $task_index 失败"
                doing_update_job_status "$module" "$job" "FAILED"
                return 1
            fi
        fi

        ((task_index++))
    done

    # 检查是否收到中断信号
    if [[ "$INTERRUPT_RECEIVED" == true ]]; then
        log WARN "Job 执行被中断"
        return 130
    fi

    # 所有 Task 完成，返回成功（状态更新由调用者 doing_mark_completed 处理）
    log SUCCESS "Job $module/$job 所有 Task 执行完成"

    # 重置全局变量
    CURRENT_MODULE=""
    CURRENT_JOB=""

    return 0
}

# ============================================
# 提示词构建和管理函数 (Job 3)
# ============================================

# 构建精简上下文
# 从 status.json 生成精简的上下文信息，用于传递给 AI
# 用法: doing_build_compact_context <module> <job>
# 输出: JSON 格式的精简上下文
doing_build_compact_context() {
    local module="$1"
    local job="$2"

    # 检查 status.json 是否存在
    if [[ ! -f "$STATUS_FILE" ]]; then
        log ERROR "状态文件不存在: $STATUS_FILE"
        return 1
    fi

    # 获取当前 Job 信息
    local job_status=$(doing_get_job_status "$module" "$job")
    local loop_count=$(doing_get_loop_count "$module" "$job")
    local tasks_total=$(doing_get_tasks_total "$module" "$job")
    local tasks_completed=$(doing_get_tasks_completed "$module" "$job")

    # 获取当前 Job 的任务列表
    local tasks=$(get_job_tasks "$module" "$job")
    local task_array=""
    local idx=1
    while IFS= read -r task_line; do
        if [[ -n "$task_line" ]]; then
            local task_desc=$(echo "$task_line" | cut -d':' -f2)
            if [[ -n "$task_array" ]]; then
                task_array="${task_array},"
            fi
            task_array="${task_array}\"${task_desc}\""
        fi
        ((idx++))
    done <<< "$tasks"

    # 获取验证器
    local validators=$(get_job_validators "$module" "$job")
    local validator_desc=""
    while IFS= read -r val_line; do
        if [[ -n "$val_line" ]]; then
            validator_desc=$(echo "$val_line" | cut -d':' -f2)
            break  # 只取第一个验证器描述
        fi
    done <<< "$validators"

    # 构建已完成 Job 的摘要列表
    local completed_jobs_summary=""
    local modules=$(jq -r '(.modules // {}) | keys[]' "$STATUS_FILE" 2>/dev/null)
    for mod in $modules; do
        local jobs=$(jq -r --arg m "$mod" '(.modules[$m].jobs // {}) | keys[]' "$STATUS_FILE" 2>/dev/null)
        for j in $jobs; do
            # 跳过当前 Job
            if [[ "$mod" == "$module" && "$j" == "$job" ]]; then
                continue
            fi

            local jstatus=$(jq -r --arg m "$mod" --arg j "$j" '.modules[$m].jobs[$j].status' "$STATUS_FILE" 2>/dev/null)
            if [[ "$jstatus" == "COMPLETED" ]]; then
                local jtasks=$(jq -r --arg m "$mod" --arg j "$j" '.modules[$m].jobs[$j].tasks_total // 0' "$STATUS_FILE" 2>/dev/null)
                if [[ -n "$completed_jobs_summary" ]]; then
                    completed_jobs_summary="${completed_jobs_summary},"
                fi
                completed_jobs_summary="${completed_jobs_summary}\"${mod}/${j}: 完成 (${jtasks} tasks)\""
            fi
        done
    done

    # 输出精简上下文 JSON
    cat << EOF
{
  "current": {
    "module": "$module",
    "job": "$job",
    "status": "$job_status",
    "loop_count": $loop_count
  },
  "context": {
    "completed_jobs_summary": [${completed_jobs_summary}],
    "current_job": {
      "name": "$job",
      "description": "Job execution",
      "tasks": [${task_array}],
      "dependencies": [],
      "validator": "${validator_desc}"
    }
  }
}
EOF
}

# 加载 Plan 上下文
# 读取 Plan 文件获取当前 Job 的完整上下文信息
# 用法: doing_load_plan_context <module> <job>
# 输出: JSON 格式的上下文信息
doing_load_plan_context() {
    local module="$1"
    local job="$2"

    # 查找对应的 Plan 文件
    local plan_file="$PLAN_DIR/${module}.md"
    if [[ ! -f "$plan_file" ]]; then
        log ERROR "Plan 文件不存在: $plan_file"
        return 1
    fi

    # 从解析结果中获取任务列表
    local tasks=$(get_job_tasks "$module" "$job")
    local task_list=""
    local idx=1
    while IFS= read -r task_line; do
        if [[ -n "$task_line" ]]; then
            local status=$(echo "$task_line" | cut -d':' -f3)
            local desc=$(echo "$task_line" | cut -d':' -f2)
            if [[ "$status" == "completed" ]]; then
                task_list="${task_list}- [x] ${desc}\n"
            else
                task_list="${task_list}- [ ] ${desc}\n"
            fi
        fi
        ((idx++))
    done <<< "$tasks"

    # 获取验证器
    local validators=$(get_job_validators "$module" "$job")
    local validator_list=""
    while IFS= read -r val_line; do
        if [[ -n "$val_line" ]]; then
            local vdesc=$(echo "$val_line" | cut -d':' -f2)
            validator_list="${validator_list}- ${vdesc}\n"
        fi
    done <<< "$validators"

    # 获取 Job 状态信息
    local job_status=$(doing_get_job_status "$module" "$job")
    local loop_count=$(doing_get_loop_count "$module" "$job")
    local tasks_total=$(doing_get_tasks_total "$module" "$job")
    local tasks_completed=$(doing_get_tasks_completed "$module" "$job")

    # 输出上下文信息
    cat << EOF
{
  "module": "$module",
  "job": "$job",
  "status": "$job_status",
  "loop_count": $loop_count,
  "tasks": {
    "total": $tasks_total,
    "completed": $tasks_completed
  },
  "task_list": "$task_list",
  "validators": "$validator_list"
}
EOF
}

# 构建完整的执行提示词
# 组合系统提示词和精简上下文
# 用法: doing_build_prompt <module> <job> <task_index> <task_desc>
# 输出: 完整的提示词内容
doing_build_prompt() {
    local module="$1"
    local job="$2"
    local task_index="${3:-1}"
    local task_desc="${4:-}"

    # 读取系统提示词
    local system_prompt=""
    if [[ -f "$DOING_PROMPT" ]]; then
        system_prompt=$(cat "$DOING_PROMPT")
    else
        log ERROR "系统提示词文件不存在: $DOING_PROMPT"
        return 1
    fi

    # 构建精简上下文
    local compact_context
    if ! compact_context=$(doing_build_compact_context "$module" "$job"); then
        log ERROR "构建精简上下文失败"
        return 1
    fi

    # 从精简上下文中提取任务列表用于显示
    local tasks=$(get_job_tasks "$module" "$job")
    local task_list=""
    while IFS= read -r task_line; do
        if [[ -n "$task_line" ]]; then
            local status=$(echo "$task_line" | cut -d':' -f3)
            local desc=$(echo "$task_line" | cut -d':' -f2)
            if [[ "$status" == "completed" ]]; then
                task_list="${task_list}- [x] ${desc}\n"
            else
                task_list="${task_list}- [ ] ${desc}\n"
            fi
        fi
    done <<< "$tasks"

    # 获取验证器列表
    local validators=$(get_job_validators "$module" "$job")
    local validator_list=""
    while IFS= read -r val_line; do
        if [[ -n "$val_line" ]]; then
            local vdesc=$(echo "$val_line" | cut -d':' -f2)
            validator_list="${validator_list}- ${vdesc}\n"
        fi
    done <<< "$validators"

    # 如果未提供 task_desc，从上下文中获取
    if [[ -z "$task_desc" ]]; then
        task_desc=$(get_job_tasks "$module" "$job" | sed -n "${task_index}p" | cut -d':' -f2)
    fi

    # 构建完整提示词
    cat << EOF
$system_prompt

---

# 精简上下文

\`\`\`json
$compact_context
\`\`\`

---

# 当前 Job 上下文

**模块**: $module
**Job**: $job
**当前 Task**: #$task_index
**Task 描述**: $task_desc

## 任务列表

$task_list

## 验证器

$validator_list

## 执行指令

请按照 Doing 模式的循环步骤执行：
1. 读取精简上下文了解当前状态
2. 执行当前 Task: $task_desc
3. 如有问题，记录 debug_log
4. 更新状态文件
5. 输出 RALPH_STATUS

开始执行!
EOF
}

# 保存提示词到临时文件
# 用法: doing_save_prompt_to_file <prompt_content> <output_file>
# 返回: 0 成功, 1 失败
doing_save_prompt_to_file() {
    local prompt_content="$1"
    local output_file="$2"

    # 检查参数
    if [[ -z "$prompt_content" ]]; then
        log ERROR "提示词内容为空"
        return 1
    fi

    if [[ -z "$output_file" ]]; then
        log ERROR "输出文件路径为空"
        return 1
    fi

    # 确保目录存在
    local output_dir=$(dirname "$output_file")
    if [[ ! -d "$output_dir" ]]; then
        mkdir -p "$output_dir" || {
            log ERROR "无法创建目录: $output_dir"
            return 1
        }
    fi

    # 保存提示词到文件
    echo "$prompt_content" > "$output_file" || {
        log ERROR "无法写入文件: $output_file"
        return 1
    }

    log INFO "提示词已保存: $output_file"
    return 0
}

# 构建 Task 执行提示词 (兼容旧接口，使用新的模块化函数)
build_task_prompt() {
    local module="$1"
    local job="$2"
    local task_index="$3"
    local task_desc="$4"

    # 使用新的模块化函数构建提示词
    doing_build_prompt "$module" "$job" "$task_index" "$task_desc"
}

# 执行单个 Task
doing_run_task() {
    local module="$1"
    local job="$2"
    local task_index="$3"
    local task_desc="$4"

    # 检查是否收到中断信号
    if [[ "$INTERRUPT_RECEIVED" == true ]]; then
        log WARN "    Task 执行被中断"
        return 130
    fi

    log INFO "    调用 ai_cli 执行 Task..."

    # 构建提示词
    local task_prompt
    if ! task_prompt=$(build_task_prompt "$module" "$job" "$task_index" "$task_desc"); then
        log ERROR "构建提示词失败"
        return 1
    fi

    # 保存提示词到临时文件
    local prompt_file="$DOING_LOGS/${module}_${job}_task${task_index}_prompt.md"
    if ! doing_save_prompt_to_file "$task_prompt" "$prompt_file"; then
        log ERROR "保存提示词失败"
        return 1
    fi

    # 构建 ai_cli 命令
    local ai_args=(
        "$CLAUDE_CMD"
        "--verbose"
        "--debug"
        "--dangerously-skip-permissions"
        "--output-format" "json"
    )

    # 执行 ai_cli
    local output_file="$DOING_LOGS/${module}_${job}_task${task_index}_output.log"
    log INFO "    执行 ai_cli..."
    log INFO "    输出日志: $output_file"

    # 使用子shell执行 ai_cli，以便可以捕获中断信号
    local exit_code=0
    (
        cat "$prompt_file" | "${ai_args[@]}" 2>&1
    ) > "$output_file" &
    local ai_pid=$!

    # 等待子进程完成，同时检查中断信号
    while kill -0 $ai_pid 2>/dev/null; do
        if [[ "$INTERRUPT_RECEIVED" == true ]]; then
            log WARN "    中断 Task 执行..."
            kill -TERM $ai_pid 2>/dev/null || true
            wait $ai_pid 2>/dev/null || true
            return 130
        fi
        sleep 0.5
    done

    wait $ai_pid
    exit_code=$?

    log INFO "    ai_cli 退出码: $exit_code"

    # 输出日志内容到控制台（用于实时查看）
    if [[ -f "$output_file" ]]; then
        cat "$output_file"
    fi

    # 解析执行结果
    log INFO "    解析执行结果..."
    local parsed_result
    parsed_result=$(doing_parse_execution_result "$output_file")

    # 提取解析结果
    local result_status=$(echo "$parsed_result" | grep -o 'status=[^;]*' | cut -d'=' -f2)
    local result_tasks_completed=$(echo "$parsed_result" | grep -o 'tasks_completed=[^;]*' | cut -d'=' -f2)
    local result_tasks_total=$(echo "$parsed_result" | grep -o 'tasks_total=[^;]*' | cut -d'=' -f2)
    local result_summary=$(echo "$parsed_result" | grep -o 'summary=[^;]*' | cut -d'=' -f2)

    log INFO "    解析结果: status=$result_status, tasks=$result_tasks_completed/$result_tasks_total"

    # 更新 Task 详细状态
    if [[ "$result_status" == "COMPLETED" ]]; then
        doing_update_task_detail "$module" "$job" "$task_index" "COMPLETED" "$task_desc"
    else
        doing_update_task_detail "$module" "$job" "$task_index" "$result_status" "$task_desc"
    fi

    # 检查输出中是否有成功标记
    if [[ $exit_code -eq 0 ]]; then
        # 根据解析的状态判断
        case "$result_status" in
            "COMPLETED")
                log SUCCESS "    Task 执行完成 (RALPH 报告 COMPLETED)"
                log INFO "    摘要: $result_summary"
                return 0
                ;;
            "FAILED")
                log ERROR "    Task 执行失败 (RALPH 报告 FAILED)"
                log ERROR "    失败原因: $result_summary"
                # 记录 debug_log
                doing_extract_debug_logs "$module" "$job" "$output_file"
                return 1
                ;;
            *)
                # 检查是否有错误输出
                if grep -qi "error\|failed\|失败" "$output_file" 2>/dev/null; then
                    log WARN "    输出中包含错误信息，请检查日志"
                    doing_extract_debug_logs "$module" "$job" "$output_file"
                fi
                # 默认认为成功
                log SUCCESS "    Task 执行完成 (退出码: 0)"
                return 0
                ;;
        esac
    elif [[ $exit_code -eq 130 ]]; then
        log WARN "    Task 被中断"
        return 130
    else
        log ERROR "    ai_cli 执行失败 (退出码: $exit_code)"
        # 尝试从输出中提取错误信息
        if [[ -f "$output_file" ]]; then
            local error_msg=$(tail -20 "$output_file" | grep -i "error\|failed" | head -1 || echo "")
            if [[ -n "$error_msg" ]]; then
                log ERROR "    错误信息: $error_msg"
            fi
        fi
        # 记录 debug_log
        doing_extract_debug_logs "$module" "$job" "$output_file"
        return 1
    fi
}

# ============================================
# 简单执行逻辑
# ============================================

# 获取第一个未完成的 Job（兼容函数，实际调用 doing_select_job）
doing_get_first_pending_job() {
    doing_select_job
}


# ============================================
# 主入口函数
# ============================================

doing_main() {
    # 设置信号处理程序（必须在最开始设置）
    doing_setup_signal_handlers

    # 解析命令行参数
    doing_parse_args "$@"

    # 验证参数：--job 必须和 --module 一起使用
    if [[ -n "$OPTION_JOB" && -z "$OPTION_MODULE" ]]; then
        log ERROR "--job 选项必须与 --module 选项一起使用"
        log ERROR "示例: morty doing --module research --job data-analysis"
        exit 1
    fi

    # 显示欢迎信息
    log INFO "╔════════════════════════════════════════════════════════════╗"
    log INFO "║              MORTY DOING 模式 - 分层 TDD 开发              ║"
    log INFO "╚════════════════════════════════════════════════════════════╝"
    log INFO ""
    log INFO "提示: 按 Ctrl+C 可优雅中断执行并保存状态"
    log INFO ""

    # 检查前置条件
    doing_check_prerequisites || exit 1

    # 加载 Plan
    doing_load_plan

    # 处理 --restart 选项
    # 注意：如果同时指定了 --module 或 --job，则在 doing_resume_job 中单独重置指定 job 的状态
    if [[ "$OPTION_RESTART" == true && -z "$OPTION_MODULE" && -z "$OPTION_JOB" ]]; then
        log WARN "--restart 选项已指定，重置所有状态"
        rm -f "$STATUS_FILE"
    fi

    # 初始化或加载状态
    if ! doing_load_status; then
        doing_init_status
    else
        log INFO "从现有状态恢复..."
    fi

    # 执行模式选择
    if [[ -n "$OPTION_MODULE" ]]; then
        # 只执行指定模块
        log INFO ""
        log INFO "执行指定模块: $OPTION_MODULE"

        if [[ -n "$OPTION_JOB" ]]; then
            # 只执行指定 Job
            doing_resume_job "$OPTION_MODULE" "$OPTION_JOB"
        else
            # 执行模块的所有 Jobs
            local module_jobs=$(get_module_jobs "$OPTION_MODULE")
            while IFS= read -r job; do
                [[ -n "$job" ]] || continue
                doing_resume_job "$OPTION_MODULE" "$job"
            done <<< "$module_jobs"
        fi
    else
        # 循环执行：按顺序执行第一个未完成的 Job
        log INFO ""
        log INFO "开始循环执行 Jobs..."

        while true; do
            # 获取第一个未完成的 Job
            local next_job=$(doing_get_first_pending_job)

            if [[ -z "$next_job" ]]; then
                log SUCCESS "所有 Jobs 已完成！"
                break
            fi

            local module=$(echo "$next_job" | cut -d':' -f1)
            local job=$(echo "$next_job" | cut -d':' -f2)

            log INFO ""
            log INFO "执行 Job: $module/$job"

            doing_resume_job "$module" "$job"
            local result=$?

            case $result in
                0)
                    # 执行成功，继续下一个
                    continue
                    ;;
                2)
                    # 需要重试，继续循环（不退出）
                    log INFO "Job $module/$job 将重试..."
                    continue
                    ;;
                130)
                    # 被中断
                    log WARN "Job $module/$job 被中断"
                    exit 130
                    ;;
                *)
                    # 彻底失败
                    log ERROR "Job $module/$job 执行失败，停止执行"
                    exit 1
                    ;;
            esac
        done
    fi

    log INFO ""
    log SUCCESS "Doing 模式执行完成"
}

# 如果不是被 source，则执行主函数
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    doing_main "$@"
fi
