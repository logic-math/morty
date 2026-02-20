# Plan: doing

## 模块概述

**模块职责**: 实现 Doing 模式（取代原有的 Loop 模式），执行 Plan 中定义的开发任务，支持分层 TDD 开发范式（单元测试 → 集成测试 → 端到端测试）。通过 `.morty/status.json` 维护所有执行状态和进度。

**对应 Research**: [重构设计] doing 模式取代 loop；分层 TDD 验证模型

**依赖模块**: config, logging, git_manager, plan_mode

**被依赖模块**: cli

## 接口定义

### 输入接口
- `morty doing [options]`: 启动开发执行（默认从断点自动恢复）
- `.morty/plan/*.md`: Plan 模式输出的计划文件
- `--module <name>`: 只执行指定模块
- `--job <name>`: 只执行指定 Job
- `--restart`: 强制从头开始（忽略已有状态）

### 输出接口
- `.morty/status.json`: 完整的执行状态和进度（核心状态文件）
- `.morty/doing/logs/`: 执行日志
- Git 提交：每次 Job 完成后自动提交

### Job 状态机
```
[PENDING] → [RUNNING] → [TEST_GENERATED] → [TEST_PASSED] → [COMPLETED]
                ↓              ↓
            [FAILED] ← [TEST_FAILED] ← [RETRYING]
                ↓
            [BLOCKED]
```

## 数据模型

### 核心状态文件 (`.morty/status.json`)

```json
{
  "version": "2.0",
  "state": "running|completed|error|blocked|paused",
  "current": {
    "module": "config",
    "job": "job_2",
    "status": "RUNNING",
    "start_time": "2026-02-20T14:25:00Z"
  },
  "session": {
    "start_time": "2026-02-20T10:00:00Z",
    "last_update": "2026-02-20T14:30:00Z",
    "total_loops": 15
  },
  "modules": {
    "config": {
      "status": "completed",
      "started_at": "2026-02-20T10:00:00Z",
      "completed_at": "2026-02-20T11:30:00Z",
      "jobs": {
        "job_1": {
          "status": "COMPLETED",
          "loop_count": 1,
          "started_at": "2026-02-20T10:00:00Z",
          "completed_at": "2026-02-20T10:30:00Z",
          "tasks_total": 5,
          "tasks_completed": 5,
          "debug_logs": []
        },
        "job_2": {
          "status": "COMPLETED",
          "loop_count": 3,
          "started_at": "2026-02-20T10:30:00Z",
          "completed_at": "2026-02-20T11:30:00Z",
          "tasks_total": 5,
          "tasks_completed": 5,
          "debug_logs": [
            {
              "id": 1,
              "timestamp": "2026-02-20T10:45:00Z",
              "phenomenon": "测试生成失败：验证器解析错误",
              "reproduction": "验证器包含不支持的语法",
              "hypotheses": ["验证器格式不符合规范", "解析器 bug"],
              "fix": "修正验证器描述为自然语言格式",
              "resolved": true
            },
            {
              "id": 2,
              "timestamp": "2026-02-20T11:15:00Z",
              "phenomenon": "测试执行超时",
              "reproduction": "任务 4 执行超过 60 秒",
              "hypotheses": ["网络请求阻塞", "无限循环"],
              "fix": "添加超时保护和网络超时设置",
              "resolved": true
            }
          ]
        }
      }
    },
    "logging": {
      "status": "running",
      "started_at": "2026-02-20T11:30:00Z",
      "jobs": {
        "job_1": {
          "status": "COMPLETED",
          "loop_count": 1,
          "debug_logs": []
        },
        "job_2": {
          "status": "RUNNING",
          "loop_count": 2,
          "started_at": "2026-02-20T13:00:00Z",
          "tasks_total": 4,
          "tasks_completed": 2,
          "current_task": "Task 3: 实现日志轮转",
          "retry_count": 1,
          "debug_logs": [
            {
              "id": 1,
              "timestamp": "2026-02-20T13:30:00Z",
              "phenomenon": "日志轮转时丢失消息",
              "reproduction": "高频写入时触发轮转",
              "hypotheses": ["文件句柄未正确同步", "并发写入竞争"],
              "fix": "待修复",
              "resolved": false
            }
          ]
        }
      }
    }
  },
  "summary": {
    "total_modules": 8,
    "completed_modules": 1,
    "running_modules": 1,
    "pending_modules": 6,
    "blocked_modules": 0,
    "total_jobs": 42,
    "completed_jobs": 4,
    "running_jobs": 1,
    "failed_jobs": 0,
    "blocked_jobs": 0,
    "progress_percentage": 12
  },
  "git": {
    "current_commit": "abc123",
    "total_commits": 15
  }
}
```

### debug_log 结构说明

每个 Job 的 `debug_logs` 数组记录该 Job 执行过程中遇到的所有问题：

```json
{
  "id": 1,
  "timestamp": "ISO8601时间戳",
  "phenomenon": "错误现象描述",
  "reproduction": "复现方法",
  "hypotheses": ["猜想原因1", "猜想原因2"],
  "verification_todo": ["验证步骤1", "验证步骤2"],
  "fix": "修复方法描述",
  "fix_progress": "修复进展",
  "resolved": true|false
}
```

### 进度统计规则

- **模块进度**: 模块内所有 Jobs 完成则模块完成
- **整体进度**: `completed_jobs / total_jobs * 100%`
- **循环计数**: 每个 Job 执行一次算一个循环，重试也算新循环
- **total_loops**: 所有 Job 的 `loop_count` 总和

## 执行调度策略

### 1. 断点自动恢复（默认行为）

`morty doing` 默认行为：
1. 读取 `.morty/status.json` 获取上次执行状态
2. 从未完成的 Job 开始继续执行
3. Job 内从未完成的 Task 开始继续执行
4. 如需强制从头开始，使用 `--restart` 选项

### 2. 拓扑排序执行

#### 模块级拓扑排序
- 根据模块依赖关系（`dependency_modules`）进行拓扑排序
- 优先执行被依赖的模块（底层优先）
- 无依赖的模块可以并行（但按顺序执行）

**示例执行顺序**:
```
config (无依赖) → logging (依赖 config) → git_manager (依赖 config, logging)
```

#### Job 级拓扑排序
- 在每个模块内，根据 Job 的 `前置条件` 进行拓扑排序
- 优先执行 0 依赖（无前置条件）的 Job
- 前置条件完成的 Job 才能执行

**示例**:
```
Job 1 (无前置条件) → Job 2 (前置: Job 1) → Job 3 (前置: Job 2)
```

### 3. 执行选择算法

```
doitng_execute_strategy():
    1. 加载 Plan 和当前状态
    2. 对模块进行拓扑排序
    3. 对每个模块:
       a. 对模块内 Jobs 进行拓扑排序
       b. 找到第一个状态非 COMPLETED 的 Job
       c. 如果 Job 有未解决的 debug_log:
          - 从 debug_log 处恢复执行（重试）
       d. 否则:
          - 从第一个未完成的 Task 开始执行
       e. 执行完成后更新状态
    4. 所有模块完成后执行集成测试
```

### 4. Job 恢复执行逻辑

```
doing_resume_job(module, job):
    status = 读取 status.json 中该 Job 的状态

    if status == "COMPLETED":
        return  # Job 已完成，跳过

    if status == "FAILED" 或 有未解决的 debug_log:
        # 从上次失败点恢复
        log_info "从 debug_log 恢复执行: ${module}/${job}"
        增加 loop_count
        重新执行该 Job（从第一个未完成的 Task）

    elif status == "PENDING" 或 status == null:
        # 新 Job，从头执行
        设置 status = "RUNNING"
        loop_count = 1
        从 Task 1 开始执行

    elif status == "RUNNING":
        # 上次中断，从当前 Task 恢复
        从 current_task 继续执行
```

### 5. Task 级执行与标记

```
doing_execute_task(module, job, task_index):
    task = 读取 Plan 中该 Job 的第 task_index 个 Task

    # 执行前检查
    if task 已标记为完成:
        log_debug "Task ${task_index} 已完成，跳过"
        return

    # 执行任务
    log_info "执行 Task ${task_index}: ${task.description}"
    result = 执行 task

    if result 成功:
        在 status.json 中标记 task 为完成
        tasks_completed += 1
        log_success "Task ${task_index} 完成"
    else:
        log_error "Task ${task_index} 失败"
        创建 debug_log 记录失败信息
        throw 终止当前 Job
```

### 6. 批式推理调用 (ai_cli)

Doing 模式通过批式推理方式调用 ai_cli 执行每个 Job 循环:

#### 调用流程

```
doing_execute_job_loop(module, job):
    1. 读取当前 Job 的 Plan 定义
    2. 读取 status.json 获取执行状态
    3. 构建 Doing 提示词（系统提示词 + Job 上下文）
    4. 调用 ai_cli 执行当前 Job 的一个循环
    5. 解析 RALPH_STATUS 输出
    6. 更新 status.json
    7. 创建 Git 提交
```

#### 提示词构建

```bash
build_doing_prompt(module, job):
    # 1. 读取系统提示词
    SYSTEM_PROMPT=$(cat prompts/doing.md)

    # 2. 读取 Job 定义
    JOB_PLAN=$(cat .morty/plan/${module}.md)

    # 3. 读取当前状态
    CURRENT_STATUS=$(cat .morty/status.json)

    # 4. 组合提示词
    cat << EOF
${SYSTEM_PROMPT}

---

# 当前 Job 上下文

**模块**: ${module}
**Job**: ${job}
**Loop**: ${loop_count}

## Job 定义

${JOB_PLAN}

## 当前状态

\`\`\`json
${CURRENT_STATUS}
\`\`\`

开始执行!
EOF
```

#### ai_cli 调用

```bash
execute_ai_cli(prompt_file):
    AI_CMD=$(config_get ai_cli "claude")  # 默认使用 claude

    # 构建命令参数
    ARGS=(
        "${AI_CMD}"
        "--verbose"
    )

    # 执行
    cat "${prompt_file}" | "${ARGS[@]}" 2>&1
```

#### 与旧 loop 模式的区别

| 特性 | 旧 loop 模式 | 新 doing 模式 |
|------|-------------|--------------|
| 循环粒度 | 整个项目 | 单个 Job |
| 状态管理 | 简单 status.json | 详细的 Job/Task 状态 |
| 恢复能力 | 循环级 | Task 级 |
| 调试记录 | 无 | debug_log 数组 |
| 拓扑排序 | 无 | 模块/Job 级 |

### 7. 完整执行流程示例

**Plan 结构**:
```
modules:
  config:
    jobs: [job1, job2, job3]  # job2 依赖 job1
  logging:
    jobs: [job1, job2]        # 无依赖
    依赖模块: [config]
```

**执行流程**:
```
第1次执行 morty doing:
  → config/job1 (新, 0依赖)
    → 构建提示词 (prompts/doing.md + job定义 + 状态)
    → 调用 ai_cli 执行 Loop 1
    → ai_cli 执行 Task1-5 → COMPLETED
    → 解析 RALPH_STATUS → 更新 status.json
    → Git 提交 "config/job1: COMPLETED"

  → config/job2 (新, 依赖job1)
    → 构建提示词
    → 调用 ai_cli 执行 Loop 1
    → ai_cli 执行 Task1-2 → Task3 FAILED
    → 记录 debug_log
    → 解析 RALPH_STATUS → 更新 status.json
    → Git 提交 "config/job2: FAILED (loop 1)"

第2次执行 morty doing (自动恢复):
  → config/job1 (COMPLETED) → 跳过

  → config/job2 (FAILED, 有未解决 debug_log)
    → 构建提示词 (包含 debug_log 上下文)
    → 调用 ai_cli 执行 Loop 2 (重试)
    → ai_cli 跳过 Task1-2 → 重试 Task3 → 成功 → Task4-5
    → 解析 RALPH_STATUS → 更新 status.json
    → Git 提交 "config/job2: COMPLETED (loop 2)"

  → config/job3 (新, 依赖job2) → 执行...
```

### 执行日志结构

```
.morty/doing/
├── logs/
│   ├── doing.log           # 主执行日志
│   ├── doing_job1.log     # Job 独立日志
│   └── config_job2.log
└── tests/                  # 生成的测试文件
    └── ...
```

## Jobs (Loop 块列表)

---

### Job 1: Doing 模式基础架构

**目标**: 建立 Doing 模式的核心框架，支持读取 Plan 和管理执行状态

**前置条件**: config, logging, git_manager, plan_mode 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_doing.sh` 脚本
- [ ] 实现 `doing_load_plan()`: 读取 Plan 文件
- [ ] 实现 `doing_init_status()`: 初始化 `.morty/status.json`
- [ ] 实现 `doing_save_status()`: 保存状态到 status.json
- [ ] 实现 `doing_check_prerequisites()`: 检查 Plan 存在且有效

**验证器**:
- 当 `.morty/plan/` 不存在时，应提示用户先运行 `morty plan`
- 当 Plan 验证失败时，应显示错误详情并退出
- `doing_load_plan()` 应正确解析所有模块和 Jobs
- `doing_init_status()` 应创建有效的初始状态文件（包含所有模块和 Jobs 的 PENDING 状态）
- 状态保存后，可以从文件中完整恢复执行状态

**调试日志**:
- 无

---

### Job 2: Job 执行引擎

**目标**: 实现单个 Job 的执行引擎，支持任务执行、状态追踪和 Task 级恢复

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_execute_job(module, job)`: 执行单个 Job（支持恢复逻辑）
- [ ] 实现 `doing_resume_job(module, job)`: 从断点恢复 Job 执行
- [ ] 实现 `doing_run_task(task)`: 执行单个 Task
- [ ] 实现 Task 完成标记（跳过已完成的 Task）
- [ ] 实现 `doing_record_loop()`: 记录一次循环执行

**验证器**:
- 执行 Job 时，status.json 中对应 Job 的状态应从 PENDING 变为 RUNNING 再变为 COMPLETED
- 已标记完成的 Task 应被跳过，未完成的 Task 按顺序执行
- 从断点恢复时，应从第一个未完成的 Task 开始执行
- `loop_count` 应在每次执行 Job 时递增（包括重试）
- `start_time` 和 `completed_at` 应正确记录
- 状态变更应实时写入 status.json

**调试日志**:
- 无

---

### Job 3: 验证器执行与测试生成

**目标**: 基于验证器描述自动生成和执行测试

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_generate_test(job)`: 根据验证器生成测试
- [ ] 实现 `doing_run_test(test_file)`: 执行生成的测试
- [ ] 实现验证器解析器（从自然语言提取测试条件）
- [ ] 实现测试结果解析器
- [ ] 支持多种验证类型（函数返回值、文件存在、命令执行等）

**验证器**:
- 验证器 "当输入 X 时应输出 Y" 应生成对应的测试代码
- 测试失败后应提供清晰的错误信息
- 测试生成应支持 bash 函数测试和文件系统测试
- 生成的测试文件应保存在 `.morty/doing/tests/` 目录
- 测试执行结果应正确更新 Job 状态（TEST_PASSED/TEST_FAILED）

**调试日志**:
- 无

---

### Job 4: 重试与回滚机制

**目标**: 实现 Job 失败后的重试和回滚策略

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_retry_job(job)`: 重试失败的 Job
- [ ] 实现 `doing_rollback_job(job)`: 回滚 Job 变更
- [ ] 实现 `doing_add_debug_log()`: 添加调试日志到 status.json
- [ ] 实现重试计数器和上限检查
- [ ] 实现回滚策略选择（skip/fix/terminate）

**验证器**:
- Job 失败后应自动重试，最多达到配置的 max_retries
- 每次重试应增加 `loop_count` 和 `retry_count`
- 失败时应自动添加 debug_log 到 status.json
- debug_log 应包含：现象、复现方法、猜想原因、修复方法
- 重试耗尽后应根据策略执行（skip/fix/terminate）

**调试日志**:
- 无

---

### Job 5: 模块集成测试执行

**目标**: 模块内所有 Jobs 完成后，执行模块级集成测试

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_run_module_integration_test(module)`: 执行模块集成测试
- [ ] 实现集成测试生成器
- [ ] 检测模块内所有 Jobs 完成状态
- [ ] 执行模块 Plan 文件中定义的集成测试验证器
- [ ] 更新 status.json 中模块的 `completed_at` 和 `status`

**验证器**:
- 所有 Jobs 完成后应自动触发集成测试
- 集成测试应验证模块内各 Job 的协同工作
- 集成测试失败应标记模块为 FAILED，阻止继续
- status.json 中模块状态应正确更新为 completed
- 集成测试通过后应创建 Git 提交

**调试日志**:
- 无

---

### Job 6: 端到端测试执行（生产测试）

**目标**: 所有模块完成后，执行端到端测试验证完整业务流程

**前置条件**: Job 5 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_run_e2e_test()`: 执行端到端测试
- [ ] 读取 `[生产测试].md` 中的测试定义
- [ ] 实现环境一致性检查
- [ ] 执行端到端功能测试
- [ ] 更新 status.json 中整体 `state` 为 completed

**验证器**:
- 所有模块完成后应自动触发端到端测试
- 端到端测试应覆盖完整的业务流程
- 环境一致性检查应验证开发与生产环境等价
- 端到端测试失败应触发回滚机制
- 通过后应标记 `state` 为 COMPLETED，并记录完成时间

**调试日志**:
- 无

---

### Job 7: 拓扑调度与断点恢复

**目标**: 实现拓扑排序调度器和断点自动恢复机制

**前置条件**: Job 6 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_topology_sort_modules()`: 模块拓扑排序
- [ ] 实现 `doing_topology_sort_jobs(module)`: Job 拓扑排序（基于前置条件）
- [ ] 实现 `doing_get_next_job()`: 获取下一个待执行 Job（0 依赖优先）
- [ ] 实现 `doing_check_prerequisites_satisfied()`: 检查前置条件是否满足
- [ ] 实现断点自动恢复逻辑（默认行为）
- [ ] 实现 `--restart` 选项支持

**验证器**:
- 模块应按依赖关系拓扑排序执行（被依赖的模块先执行）
- 模块内 Jobs 应按前置条件拓扑排序执行（0 依赖 Job 优先）
- 有前置依赖未满足的 Job 应保持在 PENDING 状态
- 默认启动应从 status.json 中第一个未完成 Job 恢复
- `--restart` 选项应重置所有状态并从头执行
- 有未解决 debug_log 的 Job 应自动重试（增加 loop_count）
- 调度应正确处理循环依赖（检测并报错）

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 完整的 doing 流程可以从 Plan 执行到完成
- Job 失败、重试、回滚流程正常工作
- debug_log 正确记录到 status.json
- 集成测试和端到端测试正确触发
- 所有状态变化正确持久化到 status.json
- `morty stat` 可以正确读取和显示状态

---

## 待实现方法签名

```bash
# morty_doing.sh

# 入口
doing_main(options)

# 初始化和状态
doing_load_plan()
doing_init_status()
doing_save_status()
doing_check_prerequisites()
doing_load_status()  # 读取现有状态

# Job 执行
doing_execute_job(module, job)
doing_run_task(task)
doing_update_job_status(module, job, status)
doing_mark_task_complete(module, job, task_index)
doing_record_loop(module, job)  # 记录一次循环

# 测试与验证
doing_generate_test(job)
doing_run_test(test_file)
doing_run_module_integration_test(module)
doing_run_e2e_test()
doing_parse_validator(validator_text)

# 重试与回滚
doing_retry_job(job)
doing_rollback_job(job)
doing_should_retry(job)
doing_apply_rollback_strategy(job)
doing_add_debug_log(module, job, phenomenon, reproduction, hypotheses, fix)  # 添加调试日志

# 调度
doing_scheduler()
doing_topology_sort_modules()  # 模块拓扑排序
doing_topology_sort_jobs(module)  # Job 拓扑排序
doing_get_next_job()  # 获取下一个待执行 Job（0 依赖优先）
doing_check_prerequisites_satisfied(module, job)  # 检查前置条件
doing_resolve_dependencies()
doing_update_summary()  # 更新进度统计
doing_has_unresolved_debug_log(module, job)  # 检查是否有未解决的 debug_log

# 恢复
doing_resume_job(module, job)  # 从断点恢复 Job
doing_find_resume_point()  # 找到恢复起点
doing_is_task_completed(module, job, task_index)  # 检查 Task 是否已完成

# 状态查询
doing_get_status_summary()  # 获取状态摘要（供 stat 命令使用）
doing_get_module_status(module)
doing_get_job_status(module, job)

# 控制
doing_pause()
doing_resume()
doing_graceful_shutdown()
```

---

## stat 命令支持

Doing 模块提供状态查询接口，供 `morty stat` 命令使用：

```bash
# morty stat 显示示例

Morty 开发状态
==============

整体进度: 12% (5/42 Jobs)
当前状态: running
运行时间: 4小时30分钟
总循环数: 15

模块进度:
  [✓] config     100% (3/3 Jobs)  耗时: 1h30m
  [>] logging     50% (2/4 Jobs)  耗时: 2h00m
  [ ] git_manager  0% (0/5 Jobs)
  ...

当前执行:
  模块: logging
  Job: job_2 (日志轮转)
  状态: RUNNING
  循环: 第2次
  任务: Task 3/4 - 实现日志轮转

最近调试记录:
  logging/job_2: 日志轮转时丢失消息 (13:30)
    状态: 待修复
    猜想: 文件句柄未正确同步, 并发写入竞争
```

状态接口：
- `doing_get_status_summary()`: 返回整体进度摘要（JSON）
- `doing_get_module_status(module)`: 返回模块状态
- `doing_get_job_status(module, job)`: 返回 Job 状态和 debug_logs
```
