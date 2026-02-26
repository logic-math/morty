# Plan: Doing Command

## 模块概述

**模块职责**: 实现 `morty doing` 命令，执行 Plan 中定义的 Jobs，是 Morty 的核心执行命令

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.6 节 Executor 模块接口定义
- `morty-project-research.md` 第 3.3 节 Loop 模式分析

**依赖模块**: Config, Logging, State, Git, Parser, Call CLI, Executor

**被依赖模块**: CLI (命令注册)

---

## 命令行接口

### 用法

```bash
# 执行下一个未完成的 Job
morty doing

# 仅执行指定模块
morty doing --module cli

# 仅执行指定 Job
morty doing --module cli --job job_1

# 重置后执行
morty doing --restart
morty doing --restart --module cli
morty doing --restart --module cli --job job_1
```

### 选项

| 选项 | 简写 | 说明 |
|------|------|------|
| `--restart` | `-r` | 重置状态后执行 |
| `--module` | `-m` | 指定模块 |
| `--job` | `-j` | 指定 Job（需配合 --module）|

---

## 工作流程

```
1. 前置检查
   └─ 检查 .morty/plan/ 是否存在
      └─ 不存在 → 报错 "请先运行 morty plan"

2. 加载状态
   └─ 读取 .morty/status.json
      └─ 不存在 → 初始化新状态

3. 处理 --restart
   └─ 重置指定范围的状态为 PENDING
      └─ 不删除 Git 历史

4. 选择目标 Job
   ├─ 无参数: 找到第一个 PENDING Job
   ├─ --module: 找到该模块第一个 PENDING Job
   └─ --module + --job: 指定具体 Job

5. 检查前置条件
   └─ 依赖的 Jobs 是否已完成
      └─ 未完成 → 报错 "前置条件不满足"

6. 执行 Job
   └─ 调用 Executor 执行
      ├─ 加载 Plan 文件
      ├─ 提取当前 Job 的 Tasks
      ├─ 构建提示词
      ├─ 调用 AI CLI
      └─ 解析执行结果

7. 更新状态
   ├─ 标记 Job 为 COMPLETED/FAILED
   ├─ 更新 status.json
   └─ 记录执行日志

8. Git 提交
   └─ 创建循环提交
      └─ morty[loop:N]: [模块/job: 状态]

9. 输出摘要
   └─ 显示执行结果和下一步操作
```

---

## 数据模型

```go
// DoingHandler doing 命令处理器
type DoingHandler struct {
    config        config.Manager
    logger        logging.Logger
    stateManager  state.Manager
    gitManager    git.Manager
    executor      executor.Engine
    parserFactory parser.Factory
    cliCaller     callcli.AICliCaller
}

// DoingOptions doing 命令选项
type DoingOptions struct {
    Restart bool
    Module  string
    Job     string
}

// ExecutionSummary 执行摘要
type ExecutionSummary struct {
    Module      string
    Job         string
    Status      string
    Duration    time.Duration
    TasksTotal  int
    TasksDone   int
    NextAction  string
}
```

---

## 接口定义

### 输入接口
- 命令行参数
- `.morty/plan/*.md` Plan 文件
- `.morty/status.json` 状态文件

### 输出接口
- 执行日志输出
- 更新的 `status.json`
- Git 提交
- 执行摘要

---

## Jobs (Loop 块列表)

---

### Job 1: Doing 命令框架

**目标**: 实现 doing 命令的基础框架和参数解析

**前置条件**:
- Config, Logging 模块完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/cmd/doing.go` 文件
- [x] Task 2: 实现 `DoingHandler` 结构体
- [x] Task 3: 实现 `NewDoingHandler()` 构造函数
- [x] Task 4: 实现参数解析 (`--restart`, `--module`, `--job`)
- [x] Task 5: 实现前置检查（Plan 目录存在性）
- [x] Task 6: 友好的错误提示
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 无 Plan 文件时提示 "请先运行 morty plan"
- [x] 正确解析所有选项
- [x] `--job` 单独使用时提示需要 `--module`
- [x] 返回码正确 (0=成功, 1=失败)
- [x] 所有单元测试通过

**调试日志**:
- explore1: [探索发现] 项目使用标准 cmd handler 模式, internal/cmd/plan.go 和 research.go 为参考实现, handler 包含 cfg/logger/paths/cliCaller 字段, Execute 方法返回 Result 和 error, 已记录
- debug1: TestDoingHandler_SetPlanDir 测试失败, 期望 GetPlanDir() 返回自定义路径, 实际返回默认路径, 猜想: 1)SetPlanDir 设置 paths.workDir 但 getPlanDir 优先使用 cfg.GetPlanDir() 2)mockConfig 返回默认路径, 验证: 检查 plan_test.go 发现应使用 cfg.SetWorkDir(), 修复: 修改测试使用 cfg.SetWorkDir(tmpDir), 已修复
- debug2: doing_test.go:178 语法错误 illegal character U+005C, 行尾有非法制表符, 猜想: 编辑时意外插入的字符, 验证: 读取文件发现 \t 字符, 修复: 删除非法字符并正确格式化代码, 已修复
- debug3: doing.go:8 编译错误 imported and not used path/filepath, 猜想: 导入的包未被使用, 验证: 检查代码发现 filepath 未使用, 修复: 删除未使用的导入, 已修复

---

### Job 2: 状态管理集成

**目标**: 集成 State 模块，实现状态加载和更新

**前置条件**:
- Job 1 完成
- State 模块完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `loadStatus()` 加载状态
- [x] Task 2: 实现 `--restart` 状态重置逻辑
- [x] Task 3: 实现 `selectTargetJob()` 选择目标 Job
- [x] Task 4: 实现前置条件检查
- [x] Task 5: 实现 `updateStatus()` 更新状态
- [x] Task 6: 状态持久化到文件
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 正确加载现有状态
- [x] `--restart` 正确重置状态
- [x] 正确选择下一个 PENDING Job
- [x] 前置条件不满足时报错
- [x] 状态更新后正确持久化
- [x] 所有单元测试通过

**调试日志**:
- debug1: TestDoingHandler_Execute_success 测试失败, 执行时返回错误"模块不存在", 猜想: 1)测试未设置状态模块 2)测试未创建计划文件, 验证: 添加setupTestState和setupTestPlanFile调用, 修复: 更新测试以正确设置状态和计划文件, 已修复

---

### Job 3: Plan 文件加载与解析

**目标**: 使用 Markdown Parser 加载和解析 Plan 文件

**前置条件**:
- Job 2 完成
- Markdown Parser 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `loadPlan(module)` 加载模块 Plan
- [ ] Task 2: 使用 Markdown Parser 解析 Plan 文件
- [ ] Task 3: 提取目标 Job 的定义
- [ ] Task 4: 提取 Job 的 Tasks 列表
- [ ] Task 5: 提取验证器定义
- [ ] Task 6: 处理 Plan 文件不存在错误
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 正确加载指定模块的 Plan 文件
- [ ] 正确解析 Job 定义
- [ ] 正确提取 Tasks 列表
- [ ] 正确提取验证器
- [ ] Plan 不存在时友好报错
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

### Job 4: Executor 集成与执行

**目标**: 集成 Executor 执行 Job

**前置条件**:
- Job 3 完成
- Executor 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 初始化 Executor
- [ ] Task 2: 实现 `executeJob(module, job)`
- [ ] Task 3: 构建执行上下文
- [ ] Task 4: 调用 Executor 执行
- [ ] Task 5: 处理执行结果
- [ ] Task 6: 超时控制
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] Executor 正确初始化
- [ ] Job 正确传递给 Executor
- [ ] 执行上下文包含必要信息
- [ ] 正确处理执行结果
- [ ] 超时后终止执行
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

### Job 5: Git 提交集成

**目标**: Job 完成后创建 Git 提交

**前置条件**:
- Job 4 完成
- Git 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `createGitCommit(summary)`
- [ ] Task 2: 生成提交信息
  - 格式: `morty[loop:N]: [模块/job: 状态]`
- [ ] Task 3: 检查是否有变更
- [ ] Task 4: 添加所有变更到暂存区
- [ ] Task 5: 创建提交
- [ ] Task 6: 处理提交错误
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 提交信息格式正确
- [ ] 包含循环编号
- [ ] 包含模块/Job/状态
- [ ] 无变更时不提交（或创建空提交）
- [ ] 提交失败时记录错误
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

### Job 6: 执行摘要与输出

**目标**: 实现执行结果摘要输出

**前置条件**:
- Job 5 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `generateSummary()` 生成摘要
- [ ] Task 2: 计算执行耗时
- [ ] Task 3: 统计 Tasks 完成情况
- [ ] Task 4: 确定下一步操作提示
- [ ] Task 5: 格式化输出摘要
- [ ] Task 6: 彩色输出支持
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 摘要包含模块/Job 名称
- [ ] 摘要包含执行状态
- [ ] 摘要包含耗时
- [ ] 摘要包含 Tasks 统计
- [ ] 提示下一步操作
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

### Job 7: 错误处理与重试

**目标**: 实现错误处理和重试机制

**前置条件**:
- Job 6 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 定义错误类型
- [ ] Task 2: 实现错误分类
  - 前置条件错误
  - Plan 文件错误
  - 执行错误
  - Git 错误
- [ ] Task 3: 实现友好错误提示
- [ ] Task 4: 实现重试逻辑（最多 3 次）
- [ ] Task 5: 记录错误日志
- [ ] Task 6: 状态恢复机制
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 不同类型的错误有明确的提示
- [ ] 重试机制正常工作
- [ ] 超过重试次数后标记 FAILED
- [ ] 错误正确记录到日志
- [ ] 状态保持一致性
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的 doing 流程: 检查 → 加载 → 执行 → 提交 → 输出
- [ ] 多 Job 顺序执行
- [ ] 失败重试机制
- [ ] 中断后恢复
- [ ] 集成测试通过

**调试日志**:
- 待填充

---

## 使用示例

```bash
# 执行下一个 Job
$ morty doing
正在执行: config/job_1
...
✓ 执行完成
提交: morty[loop:1]: [config/job_1: COMPLETED]
下一步: 运行 `morty doing` 继续

# 执行指定模块
$ morty doing --module cli
正在执行: cli/job_2
...
✓ 执行完成

# 重置后执行
$ morty doing --restart
重置状态...
正在执行: config/job_1
...
```

---

## 文件清单

- `internal/cmd/doing.go` - doing 命令实现
- `prompts/doing.md` - doing 模式系统提示词
