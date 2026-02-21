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
DOING_PROMPT="$SCRIPT_DIR/prompts/doing.md"

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

# 加载现有状态
doing_load_status() {
    if [[ ! -f "$STATUS_FILE" ]]; then
        return 1
    fi

    # 验证 JSON 格式
    if command -v jq &> /dev/null; then
        if ! jq empty "$STATUS_FILE" 2>/dev/null; then
            log ERROR "状态文件格式无效: $STATUS_FILE"
            return 1
        fi
    fi

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

# 更新 Job 状态
doing_update_job_status() {
    local module="$1"
    local job="$2"
    local status="$3"

    if ! command -v jq &> /dev/null; then
        log WARN "jq 未安装，无法更新状态"
        return 1
    fi

    local timestamp=$(get_iso_timestamp)
    local temp_file=$(mktemp)

    # 如果状态是 RUNNING，记录 started_at
    # 如果状态是 COMPLETED/FAILED，记录 completed_at
    if [[ "$status" == "RUNNING" ]]; then
        jq --arg mod "$module" \
           --arg job "$job" \
           --arg status "$status" \
           --arg ts "$timestamp" \
           '.modules[$mod].jobs[$job].status = $status |
            .modules[$mod].jobs[$job].started_at = $ts |
            .current.module = $mod |
            .current.job = $job |
            .current.status = $status |
            .current.start_time = $ts |
            .session.last_update = $ts' \
           "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
    elif [[ "$status" == "COMPLETED" || "$status" == "FAILED" ]]; then
        jq --arg mod "$module" \
           --arg job "$job" \
           --arg status "$status" \
           --arg ts "$timestamp" \
           '.modules[$mod].jobs[$job].status = $status |
            .modules[$mod].jobs[$job].completed_at = $ts |
            .current.module = $mod |
            .current.job = $job |
            .current.status = $status |
            .session.last_update = $ts' \
           "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
    else
        jq --arg mod "$module" \
           --arg job "$job" \
           --arg status "$status" \
           --arg ts "$timestamp" \
           '.modules[$mod].jobs[$job].status = $status |
            .current.module = $mod |
            .current.job = $job |
            .current.status = $status |
            .session.last_update = $ts' \
           "$STATUS_FILE" > "$temp_file" && mv "$temp_file" "$STATUS_FILE"
    fi

    log INFO "状态更新: $module/$job → $status"
}

# 标记 Task 完成
doing_mark_task_complete() {
    local module="$1"
    local job="$2"
    local task_index="$3"

    if ! command -v jq &> /dev/null; then
        return 1
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

# 获取 Job 状态
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
        --param phenomenon "$phenomenon" \
        --param reproduction "$reproduction" \
        --param hypotheses "$hypotheses" \
        --param fix "$fix" \
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

# 从断点恢复 Job 执行
doing_resume_job() {
    local module="$1"
    local job="$2"

    log INFO "恢复 Job: $module/$job"

    local status=$(doing_get_job_status "$module" "$job")

    # 如果已完成，跳过
    if [[ "$status" == "COMPLETED" ]]; then
        log INFO "  Job 已完成，跳过"
        return 0
    fi

    # 检查是否有未解决的 debug_log
    if doing_has_unresolved_debug_log "$module" "$job"; then
        log WARN "  发现未解决的 debug_log，进入重试模式"
        doing_record_loop "$module" "$job"
    fi

    # 设置状态为 RUNNING
    doing_update_job_status "$module" "$job" "RUNNING"

    # 增加 loop_count
    doing_record_loop "$module" "$job"

    # 执行 Job
    doing_execute_job "$module" "$job"
}

# 执行单个 Job
doing_execute_job() {
    local module="$1"
    local job="$2"

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

    # 所有 Task 完成，标记 Job 为 COMPLETED
    doing_update_job_status "$module" "$job" "COMPLETED"
    log SUCCESS "Job $module/$job 完成"

    return 0
}

# 构建 Task 执行提示词
build_task_prompt() {
    local module="$1"
    local job="$2"
    local task_index="$3"
    local task_desc="$4"

    # 读取 doing.md 提示词
    local doing_prompt=""
    if [[ -f "$DOING_PROMPT" ]]; then
        doing_prompt=$(cat "$DOING_PROMPT")
    else
        log ERROR "Doing 提示词文件不存在: $DOING_PROMPT"
        return 1
    fi

    # 读取当前 Job 的所有 Tasks
    local job_tasks=""
    local tasks=$(get_job_tasks "$module" "$job")
    local idx=1
    while IFS= read -r task_line; do
        if [[ -n "$task_line" ]]; then
            local status=$(echo "$task_line" | cut -d':' -f3)
            local desc=$(echo "$task_line" | cut -d':' -f2)
            if [[ "$status" == "completed" ]]; then
                job_tasks="${job_tasks}- [x] ${desc}\n"
            else
                job_tasks="${job_tasks}- [ ] ${desc}\n"
            fi
        fi
        ((idx++))
    done <<< "$tasks"

    # 读取验证器
    local validators=""
    local val_list=$(get_job_validators "$module" "$job")
    while IFS= read -r val_line; do
        if [[ -n "$val_line" ]]; then
            local vidx=$(echo "$val_line" | cut -d':' -f1)
            local vdesc=$(echo "$val_line" | cut -d':' -f2)
            validators="${validators}- ${vdesc}\n"
        fi
    done <<< "$val_list"

    # 构建完整提示词
    cat << EOF
$doing_prompt

---

# 当前 Job 上下文

**模块**: $module
**Job**: $job
**当前 Task**: #$task_index
**Task 描述**: $task_desc

## 任务列表

$job_tasks

## 验证器

$validators

## 执行指令

请按照 Doing 模式的循环步骤执行：
1. 读取 .morty/status.json 了解当前状态
2. 执行当前 Task: $task_desc
3. 如有问题，记录 debug_log
4. 更新状态文件
5. 输出 RALPH_STATUS

开始执行!
EOF
}

# 执行单个 Task
doing_run_task() {
    local module="$1"
    local job="$2"
    local task_index="$3"
    local task_desc="$4"

    log INFO "    调用 ai_cli 执行 Task..."

    # 构建提示词
    local task_prompt
    if ! task_prompt=$(build_task_prompt "$module" "$job" "$task_index" "$task_desc"); then
        log ERROR "构建提示词失败"
        return 1
    fi

    # 保存提示词到临时文件
    local prompt_file="$DOING_LOGS/${module}_${job}_task${task_index}_prompt.md"
    echo "$task_prompt" > "$prompt_file"
    log INFO "    提示词已保存: $prompt_file"

    # 构建 ai_cli 命令
    local ai_args=(
        "$CLAUDE_CMD"
        "--verbose"
        "--debug"
        "--dangerously-skip-permissions"
    )

    # 执行 ai_cli
    local output_file="$DOING_LOGS/${module}_${job}_task${task_index}_output.log"
    log INFO "    执行 ai_cli..."

    if cat "$prompt_file" | "${ai_args[@]}" 2>&1 | tee "$output_file"; then
        local exit_code=0
    else
        local exit_code=$?
    fi

    log INFO "    ai_cli 退出码: $exit_code"

    # 检查输出中是否有成功标记
    if [[ $exit_code -eq 0 ]]; then
        # 检查 RALPH_STATUS 中的状态
        if grep -q '"status":.*"COMPLETED"' "$output_file" 2>/dev/null; then
            log SUCCESS "    Task 执行完成"
            return 0
        elif grep -q '"status":.*"FAILED"' "$output_file" 2>/dev/null; then
            log ERROR "    Task 执行失败 (RALPH 报告 FAILED)"
            return 1
        else
            # 默认认为成功
            log SUCCESS "    Task 执行完成"
            return 0
        fi
    else
        log ERROR "    ai_cli 执行失败 (退出码: $exit_code)"
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
    # 解析命令行参数
    doing_parse_args "$@"

    # 显示欢迎信息
    log INFO "╔════════════════════════════════════════════════════════════╗"
    log INFO "║              MORTY DOING 模式 - 分层 TDD 开发              ║"
    log INFO "╚════════════════════════════════════════════════════════════╝"
    log INFO ""

    # 检查前置条件
    doing_check_prerequisites || exit 1

    # 加载 Plan
    doing_load_plan

    # 处理 --restart 选项
    if [[ "$OPTION_RESTART" == true ]]; then
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

            if ! doing_resume_job "$module" "$job"; then
                log ERROR "Job $module/$job 执行失败，停止执行"
                exit 1
            fi
        done
    fi

    log INFO ""
    log SUCCESS "Doing 模式执行完成"
}

# 如果不是被 source，则执行主函数
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    doing_main "$@"
fi
