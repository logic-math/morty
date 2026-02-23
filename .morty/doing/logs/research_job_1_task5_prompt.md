# Doing

在满足`执行意图`的约束下不断执行`循环`中的工作步骤,结合[精简Job上下文]对[任务列表]进行执行,直到满足`验证器`中的约束,才能结束循环,完成Job。

---

# 精简上下文格式

**重要**: Doing 模式接收的是精简后的上下文，而非完整的 status.json。这有助于保持 context window 精简，提高效率。

## 精简上下文结构

```json
{
  "current": {
    "module": "logging",
    "job": "job_3",
    "status": "RUNNING",
    "loop_count": 1
  },
  "context": {
    "completed_jobs_summary": [
      "logging/job_1: 实现日志核心框架 (5 tasks)",
      "logging/job_2: 日志轮转和归档 (5 tasks)"
    ],
    "current_job": {
      "name": "job_3",
      "description": "实现结构化 JSON 日志",
      "tasks": [
        "Task 1: 实现 JSON 格式输出",
        "Task 2: 支持上下文数据序列化",
        "Task 3: 实现日志格式切换"
      ],
      "dependencies": ["logging/job_2"],
      "validator": "彀�配置 log_format: json 时，日志输出应为有效的 JSON 格式"
    }
  }
}
```

## 上下文字段说明

| 字段 | 说明 |
|------|------|
| current.module | 当前执行的模块名称 |
| current.job | 当前执行的 Job 名称 |
| current.status | 当前 Job 状态 (RUNNING) |
| current.loop_count | 当前循环次数 |
| context.completed_jobs_summary | 已完成 Job 的摘要列表（只读参考） |
| context.current_job | 当前 Job 的完整定义 |
| context.current_job.tasks | 当前 Job 的 Task 列表 |
| context.current_job.dependencies | 当前 Job 的依赖 |
| context.current_job.validator | 验证器描述 |

---

# 循环

loop:[验证器]

    step0: [加载精简上下文] 读取传入的精简上下文，理解当前 Job 和已完成的依赖。

    step1: [理解Job] 基于精简上下文中的 current_job，理解目标、Tasks 和验证器要求。

    step1.5: [探索代码库] 如果 Job 涉及代码修改且对代码结构不熟悉，使用探索子代理：
           - 调用 `Task` 工具，subagent_type="Explore"
           - prompt: "Explore codebase structure for [模块名] to understand how to implement [Task目标]"
           - thoroughness: "medium" 或 "quick"
           - 等待探索结果，作为后续编码的参考
           - 将探索结果的关键发现记录到调试日志

    step2: [执行Task] 按顺序执行当前 Job 中未完成的 Tasks:
           - 检查每个 Task 的状态，跳过已完成的 Task
           - 执行未完成的 Task
           - 标记 Task 为完成状态
           - 记录执行过程中的问题和解决方案

    step3: [验证Job] 执行 Job 的验证器，检查所有验收标准是否满足:
           - 运行生成的测试
           - 检查结果是否符合预期
           - 如验证失败，记录问题到调试日志

    step4: [更新Plan调试日志] 将本次执行遇到的问题记录到 Plan 文件的调试日志中:
           - 读取 `.morty/plan/[模块名].md`
           - 在对应 Job 的 **调试日志** 部分添加 debug 条目
           - 保存更新后的 Plan 文件

    step5: [输出RALPH] 输出 RALPH_STATUS 块，包含本次循环的执行摘要

---

# 验证器

这是一个 Job 完成检查器

0. 如果当前 Job 的所有 Tasks 已完成且验证器通过，则检查通过，结束循环。
1. 如果当前 Job 存在未解决的 debug_log，则检查不通过，需要重试。
2. 如果验证器执行失败，则检查不通过，记录问题到调试日志并准备重试。
3. 如果达到最大重试次数，则标记 Job 为 BLOCKED，结束循环。
4. 其他情况下，继续执行下一个 Task 或重试当前 Task。

---

# 执行意图

## 精简上下文处理原则

1. **不依赖完整历史**: 只基于 completed_jobs_summary 了解已完成工作，不读取完整 status.json
2. **聚焦当前 Job**: 主要关注 context.current_job 的定义
3. **按需读取**: 如需更多信息，主动读取 `.morty/status.json` 或 Plan 文件
4. **及时输出**: 尽早输出 RALPH_STATUS，减少上下文累积

## Task 执行规范

1. **理解上下文**: 首先读取精简上下文，了解当前 Job 和已完成的依赖

2. **跳过已完成**: 检查每个 Task 的完成状态，已完成的 Task 直接跳过

3. **顺序执行**: 按顺序执行未完成的 Tasks，一次只执行一个 Task

4. **及时标记**: 每个 Task 完成后立即更新状态，标记为完成

5. **问题记录**: 遇到问题时记录到 Plan 文件的调试日志中

## 探索子什�理使用规范

**触发条件**:
- 需要对不熟悉的代码库进行调研时
- 需要理解项目架构和文件组织时
- 需要查找特定功能实现位置时

**使用方法**:
```
Task工具参数:
- description: "探索代码库结构"
- prompt: "Explore the codebase to understand [具体目标]. Find: 1) main entry points 2) key modules 3) test locations"
- subagent_type: "Explore"
```

**探索结果处理**:
- 将关键发现记录到当前 Job 的调试日志中（标记为探索发现）
- 根据探索结果制定 Task 执行策略
- 如需深入探索，可再次调用 Explore subagent

## 调试日志记录（重要）

**每个 Job 结束时，必须将执行过程中遇到的问题记录到 Plan 文件的对应 Job 的调试日志中。**

### 记录位置

在 `.morty/plan/[模块名].md` 中找到当前 Job，在 **调试日志** 部分添加条目：

```markdown
### Job N: [Job名称]

**目标**: ...

**前置条件**: ...

**Tasks (Todo 列表)**: ...

**验证器**: ...

**调诀�日志**:
- debug1: [现象], [复现], [猜想], [验证], [修复], [进展]
- debug2: [现象], [复现], [猜想], [验证], [修复], [进展]
```

### 记录格式

每个 debug 条目包含6个字段，用逗号分隔：

| 字段 | 说明 | 示例 |
|------|------|------|
| 现象 | 遇到的问题描述 | 日志轮转时丢失消息 |
| 复现 | 如何复现该问题 | 高频写入时触发轮转 |
| 猜想 | 可能的原因（按置信度排序）| 1)文件句柄未同步 2)并发竞争 |
| 验诀� | 验证猜想的待办事项 | 添加文件锁测试 |
| 修复 | 修复方法 | 使用 flock 同步 |
| 进展 | 修复进展 | 待修复/已修复 |

### 示例

```markdown
**调试日志**:
- debug1: 日志轮转时丢失消息, 高频写入时触发轮转, 猜想: 1)文件句柄未同步 2)并发竞争, 验证: 添加文件锁测试, 修复: 使用 flock 同步, 待修复
- debug2: Task 3 编译失败, 执行 make 时报错缺少头文件, 猜想: 1)缺少 libssl-dev, 验证: 检查依赖安装, 修复: 安装 libssl-dev, 已修复
- explore1: [探索发现] 项目使用 monorepo 结构, 核心代码在 packages/core, 测试使用 vitest, 配置: vitest.config.ts 在根目录, 已记录
```

## 验证器执行

1. 根据精简上下文中的 `context.current_job.validator` 描述生成测试
2. 执行测试并收集结果
3. 如测试通过，标记 Job 为 COMPLETED
4. 如测试失败，记录问题到 Plan 调试日志并标记为 FAILED (准备重试)

---

# RALPH_STATUS 格式

每个循环结束时必须输出 JSON 格式的 RALPH_STATUS。当使用 `--output-format json` 时，输出应为以下格式:

```json
{
  "ralph_status": {
    "module": "[模块名]",
    "job": "[Job名]",
    "status": "[RUNNING/COMPLETED/FAILED]",
    "tasks_completed": [N],
    "tasks_total": [M],
    "loop_count": [N],
    "debug_issues": [N],
    "debug_logs_in_plan": true,
    "explore_subagent_used": false,
    "summary": "[执行摘要，包含是否更新调试日志]"
  }
}
```

或者，如果无法使用嵌套格式，确保顶层包含以下字段:

```json
{
  "module": "[模块名]",
  "job": "[Job名]",
  "status": "[RUNNING/COMPLETED/FAILED]",
  "tasks_completed": [N],
  "tasks_total": [M],
  "loop_count": [N],
  "debug_issues": [N],
  "summary": "[执行摘要]"
}
```

**注意**: JSON Schema 必须包含 `status`, `tasks_completed`, `tasks_total`, `summary` 字段。

### 字段说明

| 字段 | 说明 |
|------|------|
| module | 当前模块名称 |
| job | 当前 Job 名称 |
| status | RUNNING/COMPLETED/FAILED |
| tasks_completed | 完成的 Task 数 |
| tasks_total | Task 总数 |
| loop_count | 当前循环次数 |
| debug_issues | 遇到的问题数量 |
| debug_logs_in_plan | 是否已记录到 Plan 调试日志 |
| explore_subagent_used | 是否使用了探索子代理 |
| summary | 执行摘要 |

---

# 示例

## 场景：执行 logging/job_3 遇到问题

### 接收的精简上下文

```json
{
  "current": {
    "module": "logging",
    "job": "job_3",
    "status": "RUNNING",
    "loop_count": 1
  },
  "context": {
    "completed_jobs_summary": [
      "logging/job_1: 实现日志核心框架 (5 tasks)",
      "logging/job_2: 日志轮转和归档 (5 tasks)"
    ],
    "current_job": {
      "name": "job_3",
      "description": "实现结构化 JSON 日志",
      "tasks": [
        "Task 1: 实现 JSON 格式输出",
        "Task 2: 支持上下文数据序列化",
        "Task 3: 实现日志格式切换"
      ],
      "dependencies": ["logging/job_2"],
      "validator": "当配置 log_format: json 时，日志输出应为有效的 JSON 格式"
    }
  }
}
```

### 执行前 Plan 文件状态

```markdown
### Job 3: 结构化 JSON 日志

**目标**: 实现结构化 JSON 日志支持

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 JSON 格式输出
- [ ] Task 2: 支持上下文数据序列化
- [ ] Task 3: 实现日志格式切换

**验证器**: 当配置 log_format: json 时，日志输出应为有效的 JSON 格式

**调试日志**:
- 无
```

### 执行过程

1. **理解上下文**: 从精简上下文了解 job_3 目标和依赖
2. **探索阶段**: 调用 Explore subagent 了解现有日志系统架构
3. Task 2 执行时发现 JSON 序列化问题
4. 继续完成 Task 3
5. 将问题记录到 Plan 调试日志

### 执行后 Plan 文件状态

```markdown
### Job 3: 结构化 JSON 日志

**目标**: 实现结构化 JSON 日志支持

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 JSON 格式输出
- [x] Task 2: 支持上下文数据序列化
- [x] Task 3: 实现日志格式切换

**验证器**: 当配置 log_format: json 时，日志输出应为有效的 JSON 格式

**调试日志**:
- explore1: [探索发现] 项目使用单文件日志实现, lib/logging.sh 为核心模块, 使用文件追加模式写入, 已记录
- debug1: JSON 序列化失败, 复杂对象循环引用, 猜想: 1)缺少循环引用处理 2)未使用 JSON.stringify 的 replacer, 验证: 添加 replacer 函数测试, 修复: 使用 WeakSet 检测循环引用, 待修复
```

### RALPH_STATUS 输出

```markdown
<!-- RALPH_STATUS -->
{
  "module": "logging",
  "job": "job_3",
  "status": "COMPLETED",
  "tasks_completed": 3,
  "tasks_total": 3,
  "loop_count": 1,
  "debug_issues": 1,
  "debug_logs_in_plan": true,
  "explore_subagent_used": true,
  "summary": "JSON 日志功能实现完成。使用 Explore subagent 了解架构，发现 JSON 序列化问题已记录到 Plan 调试日志 debug1"
}
<!-- END_RALPH_STATUS -->
```

---

# 重要提醒

1. **精简上下文**: Doing 模式只接收精简上下文，保持 context window 高效
2. **按需读取**: 如需更多信息，主动读取 `.morty/status.json` 或 Plan 文件
3. **Plan 文件必须更新**: 每个 Job 结束时，务必将问题记录到 `.morty/plan/[模块名].md` 的对应 Job 调试日志中
4. **调试日志是活的**: 后续 loop 可以查看之前的 debug 记录，修复后可以更新进展为"已修复"
5. **RALPH_STATUS 如实报告**: 包含 debug_issues 数量和 debug_logs_in_plan 标记
6. **善用 Explore Subagent**: 对不熟悉的代码库，先用 Explore subagent 调研，再执行 Tasks

---

# 精简上下文

```json
{
  "current": {
    "module": "research",
    "job": "job_1",
    "status": "RUNNING",
    "loop_count": 1
  },
  "context": {
    "completed_jobs_summary": ["cli/job_1: 完成 (5 tasks)","cli/job_2: 完成 (10 tasks)","cli/job_3: 完成 (7 tasks)","cli/job_4: 完成 (3 tasks)","cli/job_5: 完成 (3 tasks)","config/job_2: 完成 (4 tasks)","config/job_3: 完成 (5 tasks)","doing/job_1: 完成 (5 tasks)","doing/job_2: 完成 (5 tasks)","doing/job_3: 完成 (4 tasks)","doing/job_4: 完成 (5 tasks)","doing/job_5: 完成 (4 tasks)","doing/job_6: 完成 (5 tasks)","doing/job_7: 完成 (4 tasks)","install/job_1: 完成 (5 tasks)","install/job_2: 完成 (4 tasks)","install/job_3: 完成 (6 tasks)","install/job_4: 完成 (5 tasks)","install/job_5: 完成 (5 tasks)","install/job_6: 完成 (4 tasks)","install/job_7: 完成 (4 tasks)","logging/job_1: 完成 (5 tasks)","logging/job_2: 完成 (5 tasks)","logging/job_3: 完成 (5 tasks)","logging/job_4: 完成 (5 tasks)","plan/job_1: 完成 (7 tasks)","version_manager/job_1: 完成 (5 tasks)","version_manager/job_2: 完成 (5 tasks)","version_manager/job_3: 完成 (5 tasks)"],
    "current_job": {
      "name": "job_1",
      "description": "Job execution",
      "tasks": ["创建 `morty_research.sh` 脚本","读取 `prompts/research.md` 作为系统提示词","从 config 获取 ai_cli 命令：`AI_CLI=$(config_get "cli.command" "claude")`","构建 Claude 命令参数：","以 Plan 模式调用：`$AI_CLI --permission-mode plan -p "$PROMPT"`","创建 `.morty/research/` 目录","验证输出目录是否生成内容："],
      "dependencies": [],
      "validator": "`morty research` 能够启动研究流程"
    }
  }
}
```

---

# 当前 Job 上下文

**模块**: research
**Job**: job_1
**当前 Task**: #5
**Task 描述**: 以 Plan 模式调用：`$AI_CLI --permission-mode plan -p "$PROMPT"`

## 任务列表

- [ ] 创建 `morty_research.sh` 脚本\n- [ ] 读取 `prompts/research.md` 作为系统提示词\n- [ ] 从 config 获取 ai_cli 命令：`AI_CLI=$(config_get "cli.command" "claude")`\n- [ ] 构建 Claude 命令参数：\n- [ ] 以 Plan 模式调用：`$AI_CLI --permission-mode plan -p "$PROMPT"`\n- [ ] 创建 `.morty/research/` 目录\n- [ ] 验证输出目录是否生成内容：\n

## 验证器

- `morty research` 能够启动研究流程\n- 脚本从 config 读取 `cli.command` 作为 ai_cli 命令\n- 以 Plan 模式调用 ai_cli，传递系统提示词\n- 研究报告生成到 `.morty/research/[主题].md`\n- 无\n

## 执行指令

请按照 Doing 模式的循环步骤执行：
1. 读取精简上下文了解当前状态
2. 执行当前 Task: 以 Plan 模式调用：`$AI_CLI --permission-mode plan -p "$PROMPT"`
3. 如有问题，记录 debug_log
4. 更新状态文件
5. 输出 RALPH_STATUS

开始执行!
