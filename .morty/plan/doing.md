# Plan: doing

## 模块概述

**模块职责**: 实现 Doing 模式，循环执行 Plan 中定义的 Job。Agent 以黑箱方式后台运行，通过日志输出信息，所有修改在 plan 目录闭环。

**核心概念**:
- **Job**: AI coding agent 执行的最小任务单元
- **Task**: Agent 内部感知到的 todo list 的最小执行单元

**对应 Research**: doing 模式取代 loop；黑箱执行；日志驱动

**依赖模块**: config, logging, version_manager

**被依赖模块**: cli

## 接口定义

### 输入接口
```bash
morty doing [options]
  --restart                 # 重置模式（见下方详细说明）
  --module <name>           # 指定模块
  --job <name>              # 指定 Job（必须与 --module 一起使用）
```

### 执行模式详解

#### 模式 1: 默认执行（无参数）
```bash
morty doing
```
- 从 `status.json` 获取第一个未完成的 Job
- 执行该 Job 后退出（单循环）

#### 模式 2: 指定模块（无 --restart）
```bash
morty doing --module config
```
- 仅执行指定模块
- 从该模块第一个未完成的 Job 开始
- 执行一个 Job 后退出

#### 模式 3: 指定模块和 Job（无 --restart）
```bash
morty doing --module config --job job_1
```
- 仅执行指定的单个 Job
- 无论该 Job 之前状态如何，都执行一次
- 执行后退出

#### 模式 4: 完全重置（仅 --restart）
```bash
morty doing --restart
```
- 重置 `status.json` 中所有 Job 状态为 PENDING
- **不**进行 git 重置，保留之前工作变更
- 从第一个 Job 开始执行
- 执行一个 Job 后退出

#### 模式 5: 重置模块（--restart + --module）
```bash
morty doing --restart --module config
```
- 重置指定模块的所有 Job 状态为 PENDING
- 从该模块第一个 Job 开始执行
- 执行一个 Job 后退出

#### 模式 6: 重置指定 Job（--restart + --module + --job）
```bash
morty doing --restart --module config --job job_1
```
- 重置指定 Job 的状态为 PENDING
- 执行该 Job
- 执行后退出

### 输出接口
- `.morty/status.json`: 更新执行状态
- `.morty/logs/`: 执行日志（由 logging 模块处理）
- Git 提交：每次 Job 完成后自动提交
- `ai_cli --output-format json`: 结构化 JSON 输出（替代文本解析）

### 标准执行流程
```
1. 解析命令行参数，确定执行模式
2. 加载 status.json
3. 根据模式确定要执行的 Job
4. 更新 Job 状态为 RUNNING
5. 调用 ai_cli --output-format json 执行 Job（黑箱方式）
6. 等待执行完成，解析 JSON 结果
7. 更新 status.json（COMPLETED/FAILED）
8. 创建 Git 提交
9. 退出
```

## 数据模型

### status.json 结构
```json
{
  "version": "2.0",
  "state": "running|completed|error",
  "current": {
    "module": "config",
    "job": "job_1"
  },
  "modules": {
    "config": {
      "status": "running|completed|pending",
      "jobs": {
        "job_1": {
          "status": "PENDING|RUNNING|COMPLETED|FAILED",
          "loop_count": 1,
          "tasks_total": 5,
          "tasks_completed": 3
        }
      }
    }
  }
}
```

## Jobs (Loop 块列表)

---

### Job 1: Doing 基础框架

**目标**: 建立 Doing 模式的核心框架，支持参数解析和状态管理

**前置条件**: config, logging, version_manager 完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_doing.sh` 脚本
- [ ] 实现 `doing_parse_args()`: 解析命令行参数（--restart, --module, --job）
- [ ] 实现 `doing_load_status()`: 读取 status.json
- [ ] 实现 `doing_select_job()`: 根据参数选择要执行的 Job
- [ ] 实现 `doing_reset_status()`: 重置状态（--restart 时）

**验证器**:
- 解析 `--restart --module config --job job_1` 应正确识别所有参数
- `doing_select_job()` 无参数时应返回第一个未完成的 Job
- `doing_select_job()` 指定 `--module config` 时应返回 config 模块第一个未完成 Job
- `doing_reset_status()` 应仅重置状态，不影响 git 历史和工作目录
- 当 status.json 不存在时，应初始化并提示 "首次运行，已初始化状态"

**调试日志**:
- 无

---

### Job 2: 黑箱执行引擎

**目标**: 实现 Agent 黑箱执行引擎，通过 ai_cli 调用

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `doing_execute_job()`: 执行单个 Job
- [ ] 构建执行提示词（从 prompts/doing.md + plan 文件）
- [ ] 调用 ai_cli 后台执行
- [ ] 捕获执行结果（通过日志或输出文件）
- [ ] 解析执行结果，更新状态

**验证器**:
- 执行 Job 时，ai_cli 应以黑箱方式运行
- 所有执行输出应通过 logging 模块记录到日志
- 执行完成后应能正确解析结果（成功/失败）
- 失败时应记录失败原因到日志
- 执行过程中 Ctrl+C 应能优雅中断并保存状态

**调试日志**:
- 无

---

### Job 3: 提示词系统集成

**目标**: 将提示词收敛到 prompts/ 目录，doing 脚本不内置提示词

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 创建 `prompts/doing.md` 系统提示词
- [ ] 实现 `doing_build_prompt()`: 组合提示词 + Job 定义
- [ ] 实现 `doing_load_plan_context()`: 读取 Plan 文件内容
- [ ] 实现 `doing_save_prompt_to_file()`: 保存提示词到临时文件

**验证器**:
- `prompts/doing.md` 应存在且包含完整的系统提示词
- 构建的提示词应包含 Job 定义和当前状态
- 提示词应保存到临时文件供 ai_cli 使用
- doing 脚本本身不应包含任何硬编码提示词

**调试日志**:
- 无

---

### Job 4: 状态机管理

**目标**: 实现 Job 状态机，支持 PENDING→RUNNING→COMPLETED/FAILED 流转

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现状态转换逻辑
- [ ] 实现失败重试机制（最多 3 次）
- [ ] 实现 `doing_mark_completed()`: 标记 Job 完成
- [ ] 实现 `doing_mark_failed()`: 标记 Job 失败
- [ ] 实现中断恢复逻辑

**验证器**:
- Job 开始执行时状态应从 PENDING 变为 RUNNING
- 执行成功后状态应变为 COMPLETED
- 执行失败后状态应变为 FAILED，并增加 retry_count
- 重试 3 次后仍失败，应跳过该 Job 继续下一个
- 中断后重新运行 doing，应从断点继续

**调试日志**:
- 无

---

### Job 5: Git 提交集成

**目标**: 集成 version_manager，每次 Job 完成后自动提交

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 集成 `version_create_loop_commit()`
- [ ] 生成提交信息（包含 Job 名称、状态、变更摘要）
- [ ] 实现提交前的变更检测
- [ ] 处理提交失败情况

**验证器**:
- Job 完成后应自动创建 Git 提交
- 提交信息应包含 "morty: Loop #N - [模块/job_1: 状态]"
- 无变更时不应创建空提交
- 提交失败时应记录警告但不中断流程

**调试日志**:
- 无

---

### Job 6: 结构化 JSON 输出支持

**目标**: 实现 `ai_cli` 结构化 JSON 输出替代文本解析

**前置条件**: Job 2（黑箱执行引擎）完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 定义 RALPH_STATUS JSON Schema
- [ ] Task 2: 修改 `doing_run_task()` 使用 `--output-format json`
- [ ] Task 3: 实现 JSON 输出解析（替代 grep 文本解析）
- [ ] Task 4: 添加 JSON 解析失败时的 fallback 机制
- [ ] Task 5: 更新 `prompts/doing.md` 要求输出结构化 JSON

**验证器**:
- `ai_cli` 调用使用 `--output-format json` 参数
- JSON Schema 包含 job_status, tasks_completed, summary 字段
- 使用 `jq` 解析 JSON 输出替代 grep
- JSON 解析失败时能够回退到文本解析
- `prompts/doing.md` 中的 RALPH_STATUS 使用 JSON 格式

**调试日志**:
- 无

---

### Job 7: 精简上下文传递

**目标**: 实现精简上下文生成和传递，减少 context window 使用

**前置条件**: Job 3（提示词系统集成）完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 设计精简上下文格式（current + context）
- [ ] Task 2: 实现 `build_compact_context()` 函数
- [ ] Task 3: 从完整 status.json 生成精简上下文
- [ ] Task 4: 更新提示词模板以接收精简上下文

**验证器**:
- 精简上下文只包含当前 Job 和已完成 Job 的摘要
- 完整历史不包含在精简上下文中
- 上下文大小控制在合理范围（<1000 tokens）
- 提示词正确解析精简上下文

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 完整执行循环可以运行到所有 Job 完成
- 各种参数组合（--restart, --module, --job）工作正确
- 黑箱执行过程中日志正确记录
- 状态机在各级转换正确
- 每次循环后 Git 提交正常创建
- 中断后可以正确恢复

---

## 待实现方法签名

```bash
# morty_doing.sh

# 入口
doing_main(options)
doing_parse_args(args)

# 状态管理
doing_load_status()
doing_select_job(mode)
doing_reset_status(scope)  # scope: all/module/job
doing_update_status(module, job, status)
doing_save_status()

# 执行
doing_execute_job(module, job)
doing_build_prompt(module, job)
doing_load_plan_context(module)
doing_parse_result(output)

# 状态机
doing_mark_completed(module, job)
doing_mark_failed(module, job, reason)
doing_should_retry(module, job)

# Git 集成
doing_create_commit(module, job, status)

# 控制
doing_graceful_shutdown()
```

---

## 提示词文件

### prompts/doing.md

系统提示词文件，定义 Agent 的执行规范：
- 如何读取 Plan 文件
- 如何执行 Tasks
- 如何标记完成
- 输出格式要求
- RALPH_STATUS 格式

---

## 命令示例

```bash
# 执行下一个未完成的 Job
morty doing

# 仅执行 config 模块中的下一个未完成 Job
morty doing --module config

# 仅执行 config 模块的 job_1
morty doing --module config --job job_1

# 完全重置，从第一个 Job 开始
morty doing --restart

# 重置 config 模块，从该模块第一个 Job 开始
morty doing --restart --module config

# 重置并重新执行 config 模块的 job_1
morty doing --restart --module config --job job_1
```
