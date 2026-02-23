# Plan: Executor

## 模块概述

**模块职责**: 实现执行引擎，负责 Job/Task 的执行流程控制、提示词构建和结果解析。与 Call CLI 模块分层协作：Executor 负责高层执行逻辑，Call CLI 负责底层进程调用。

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.6 节 Executor 模块接口定义
- `morty-go-refactor-plan.md` 第 6.1 节 Job 执行流程
- `morty-project-research.md` 第 3.3 节 Loop 模式分析

**现有实现参考**:
- 原 Shell 版本: `morty_doing.sh`，执行循环和状态管理

**依赖模块**: State, Git, Parser, Call CLI

**被依赖模块**: doing_cmd

---

## 与 Call CLI 的分工

```
┌─────────────────────────────────────────────────────────────┐
│                     执行流程分层架构                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Executor (本模块)                   │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │ Job 执行调度  │  │ 提示词构建    │  │ 结果解析    │ │  │
│  │  │ • 状态流转   │  │ • 精简上下文  │  │ • RALPH解析 │ │  │
│  │  │ • Task循环   │  │ • Prompt拼接  │  │ • 调试日志  │ │  │
│  │  │ • 重试逻辑   │  │ • 变量替换    │  │ • 状态更新  │ │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬──────┘ │  │
│  └─────────┼────────────────┼────────────────┼────────┘  │
│            │                │                │           │
│            └────────────────┴────────────────┘           │
│                           │                              │
│                           ▼                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                Call CLI (调用模块)                   │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │ 进程管理      │  │ 超时控制      │  │ 信号处理    │ │  │
│  │  │ • Start      │  │ • Timeout    │  │ • SIGINT   │ │  │
│  │  │ • Wait       │  │ • Kill       │  │ • SIGTERM  │ │  │
│  │  │ • Kill       │  │ • Context    │  │ • 优雅退出  │ │  │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                              │
│                           ▼                              │
│                    ┌──────────────┐                      │
│                    │  ai_cli      │                      │
│                    │  (外部进程)   │                      │
│                    └──────────────┘                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Executor 职责** (高层):
- Job 执行流程控制（状态流转、Task 循环）
- 提示词构建（精简上下文、Prompt 拼接）
- 结果解析（RALPH_STATUS 解析、调试日志更新）
- 重试逻辑和错误处理

**Call CLI 职责** (底层):
- 子进程管理（启动、等待、终止）
- 超时控制（定时器、强制终止）
- 信号处理（转发信号、优雅退出）
- 输出捕获（stdout/stderr 捕获）

---

## 接口定义

### 输入接口
- 模块名和 Job 名
- Plan 文件内容
- 当前状态

### 输出接口
- `Engine` 接口实现
- 执行结果
- 更新后的状态

---

## 数据模型

```go
// Engine 执行引擎接口
type Engine interface {
    ExecuteJob(ctx context.Context, module, job string) error
    ExecuteTask(ctx context.Context, module, job string, taskIndex int, taskDesc string) error
    ResumeJob(ctx context.Context, module, job string) error
}

// PromptBuilder 提示词构建接口
type PromptBuilder interface {
    BuildPrompt(module, job string, taskIndex int, taskDesc string) (string, error)
    BuildCompactContext(module, job string) (map[string]interface{}, error)
}

// ResultParser 结果解析接口
type ResultParser interface {
    Parse(outputFile string) (*ExecutionResult, error)
}

// ExecutionResult 执行结果
type ExecutionResult struct {
    Status         string
    TasksCompleted int
    TasksTotal     int
    Summary        string
    Module         string
    Job            string
}

// JobRunner Job 执行器
type JobRunner struct {
    stateManager StateManager
    gitManager   GitManager
    logger       Logger
    promptBuilder PromptBuilder
    resultParser  ResultParser
}

// TaskRunner Task 执行器
type TaskRunner struct {
    logger  Logger
    aiCli   string
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: 执行引擎核心实现

**目标**: 实现执行引擎，管理 Job 执行生命周期

**前置条件**:
- State 模块完成
- Git 模块完成
- Plan Parser 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/executor/engine.go` 文件结构
- [ ] Task 2: 实现 `Engine` 接口
- [ ] Task 3: 实现 `ExecuteJob(ctx, module, job)` 方法
- [ ] Task 4: 实现前置条件检查
- [ ] Task 5: 实现状态转换 (PENDING → RUNNING → COMPLETED/FAILED)
- [ ] Task 6: 实现重试逻辑 (最多 3 次)
- [ ] Task 7: 集成 Git 提交 (Job 完成后)
- [ ] Task 8: 编写单元测试 `engine_test.go`

**验证器**:
- [ ] ExecuteJob 正确执行单个 Job
- [ ] 前置条件不满足时返回错误
- [ ] 状态正确转换 PENDING → RUNNING → COMPLETED
- [ ] 失败时重试最多 3 次
- [ ] Job 完成后创建 Git 提交
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 2: Job 执行器实现

**目标**: 实现 Job 级别的执行逻辑

**前置条件**:
- Job 1 完成 (执行引擎核心)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/executor/job_runner.go` 文件结构
- [ ] Task 2: 实现 `JobRunner` 结构体
- [ ] Task 3: 实现 `Run(ctx, module, job)` 方法
- [ ] Task 4: 实现 Tasks 循环执行
- [ ] Task 5: 实现 Task 完成状态更新
- [ ] Task 6: 实现 Job 级别错误处理
- [ ] Task 7: 实现跳过已完成的 Tasks
- [ ] Task 8: 编写单元测试 `job_runner_test.go`

**验证器**:
- [ ] JobRunner 正确执行 Job 的所有 Tasks
- [ ] 已完成的 Tasks 自动跳过
- [ ] 每个 Task 完成后更新状态
- [ ] Task 失败时停止并标记 Job 失败
- [ ] 所有 Tasks 完成后标记 Job 完成
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: Task 执行器实现

**目标**: 实现 Task 级别的执行逻辑，调用 AI CLI

**前置条件**:
- Job 2 完成 (Job 执行器)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/executor/task_runner.go` 文件结构
- [ ] Task 2: 实现 `TaskRunner` 结构体
- [ ] Task 3: 实现 `Run(ctx, taskDesc, prompt)` 方法
- [ ] Task 4: 实现 AI CLI 调用 (使用 `os/exec`)
- [ ] Task 5: 实现超时控制
- [ ] Task 6: 实现输出捕获和记录
- [ ] Task 7: 实现退出码处理
- [ ] Task 8: 编写单元测试 `task_runner_test.go`

**验证器**:
- [ ] TaskRunner 正确执行单个 Task
- [ ] AI CLI 调用成功并返回结果
- [ ] 超时后终止进程并返回错误
- [ ] 输出正确捕获并记录到日志
- [ ] 退出码 0 表示成功，非 0 表示失败
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 4: 提示词构建器实现

**目标**: 实现提示词构建，支持动态上下文生成

**前置条件**:
- Job 3 完成 (Task 执行器)
- Plan Parser 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/executor/prompt_builder.go` 文件结构
- [ ] Task 2: 实现 `PromptBuilder` 接口
- [ ] Task 3: 实现 `BuildPrompt(module, job, taskIndex, taskDesc)` 方法
- [ ] Task 4: 读取 Plan 文件内容
- [ ] Task 5: 读取系统提示词 (prompts/doing.md)
- [ ] Task 6: 实现 `BuildCompactContext(module, job)` 精简上下文
- [ ] Task 7: 实现模板变量替换
- [ ] Task 8: 编写单元测试 `prompt_builder_test.go`

**验证器**:
- [ ] BuildPrompt 生成完整的提示词
- [ ] 包含系统提示词、Plan 内容、当前 Task
- [ ] BuildCompactContext 返回精简的上下文信息
- [ ] 模板变量正确替换
- [ ] 提示词长度控制在合理范围
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 5: 结果解析器实现

**目标**: 实现执行结果解析，支持 RALPH_STATUS 格式

**前置条件**:
- Job 4 完成 (提示词构建器)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/executor/result_parser.go` 文件结构
- [ ] Task 2: 实现 `ResultParser` 接口
- [ ] Task 3: 实现 `Parse(outputFile)` 方法，读取 AI CLI 输出
- [ ] Task 4: 实现 RALPH_STATUS JSON 解析
- [ ] Task 5: 支持嵌套和扁平两种 JSON 格式
- [ ] Task 6: 实现错误输出提取和记录
- [ ] Task 7: 实现调试日志自动更新（调用 Parser 更新 Plan 文件）
- [ ] Task 8: 编写单元测试 `result_parser_test.go`

**验证器**:
- [ ] 正确解析 RALPH_STATUS JSON 块
- [ ] 支持嵌套格式 `ralph_status: {...}`
- [ ] 支持扁平格式 `status, tasks_completed, ...`
- [ ] 解析失败时返回友好错误
- [ ] 自动提取错误信息到调试日志
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 6: 与 Call CLI 集成

**目标**: 集成 Call CLI 模块，实现完整的 Task 执行流程

**前置条件**:
- Job 5 完成 (结果解析器)
- Call CLI 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 更新 `TaskRunner` 使用 `callcli.AICliCaller`
- [ ] Task 2: 实现 `ExecuteTask` 调用 Call CLI 执行 AI 命令
- [ ] Task 3: 传递提示词文件路径给 Call CLI
- [ ] Task 4: 配置超时参数（默认 10 分钟）
- [ ] Task 5: 处理 Call CLI 返回结果（退出码、输出文件）
- [ ] Task 6: 集成结果解析器解析输出
- [ ] Task 7: 实现执行中断处理（转发信号给 Call CLI）
- [ ] Task 8: 编写集成测试 `executor_integration_test.go`

**验证器**:
- [ ] TaskRunner 正确调用 Call CLI 执行 AI 命令
- [ ] 提示词正确传递给 AI CLI
- [ ] 超时配置生效（默认 10 分钟）
- [ ] 执行结果正确解析
- [ ] 中断信号正确转发给子进程
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的执行流程: 加载 Plan → 执行 Job → 更新状态 → Git 提交
- [ ] 多 Job 顺序执行正确
- [ ] 失败重试机制正常工作
- [ ] 断点恢复功能正常
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充
