# Morty Go 项目深度分析报告

**调查主题**: morty-go-project-analysis  
**调研日期**: 2026-02-27  
**项目版本**: 2.0.0

---

## 1. 项目概述

### 1.1 项目定位
**Morty** 是一个基于 Go 语言实现的 AI 驱动的上下文优先编码编排框架（Context-First Coding Orchestration Framework）。它通过结构化的工作流系统，利用 Claude AI 完成开发任务。

### 1.2 核心工作流
```
Research (研究) → Plan (计划) → Doing (执行)
              ↓
        Git Auto-commit (每个 Job 完成后自动提交)
```

### 1.3 技术栈
- **语言**: Go 1.21.6+
- **架构**: 纯 Go 标准库实现（无外部依赖）
- **模块**: `github.com/morty/morty`
- **集成**: Claude AI CLI, Git

---

## 2. 目录结构分析

### 2.1 Internal 目录结构（核心实现）

```
internal/
├── cmd/                    # 命令处理器 (13,110 LOC)
│   ├── doing.go           # Doing 模式执行 (31KB) ⭐
│   ├── plan.go            # Plan 模式实现 (20KB)
│   ├── research.go        # Research 模式 (11KB)
│   ├── reset.go           # 状态回滚 (30KB)
│   └── stat.go            # 状态展示 (30KB)
│
├── executor/              # 任务执行引擎 ⭐⭐⭐
│   ├── engine.go          # Job 执行生命周期管理
│   ├── job_runner.go      # 单个 Job 执行器
│   ├── task_runner.go     # Task 执行器
│   ├── prompt_builder.go  # Prompt 构建
│   └── result_parser.go   # 输出解析
│
├── state/                 # 状态管理 ⭐⭐⭐
│   ├── manager.go         # 状态持久化 (278 LOC)
│   ├── state.go           # 状态定义 (164 LOC)
│   ├── transitions.go     # 状态转换 (223 LOC)
│   └── status_json.go     # JSON 序列化 (375 LOC)
│
├── callcli/               # AI CLI 集成 ⭐⭐
│   ├── ai_caller.go       # AI CLI 调用器
│   ├── async.go           # 异步执行支持
│   ├── signal.go          # 信号处理
│   ├── timeout.go         # 超时管理
│   └── execution_log.go   # 执行日志
│
├── parser/                # 文档解析
│   ├── factory.go         # 解析器工厂
│   ├── markdown/          # Markdown 解析
│   ├── plan/              # Plan 文档解析
│   └── prompt/            # Prompt 解析
│
├── git/                   # Git 版本控制 ⭐
│   ├── manager.go         # Git 操作
│   ├── commit.go          # 自动提交
│   └── version.go         # 版本控制与重置
│
├── config/                # 配置管理
│   ├── paths.go           # 路径管理 (243 LOC)
│   ├── loader.go          # 配置加载
│   └── validation.go      # 配置验证
│
└── logging/               # 日志系统
    ├── logger.go          # 主日志接口
    ├── job_logger.go      # Job 专用日志
    ├── format.go          # 输出格式化
    └── rotator.go         # 日志轮转
```

### 2.2 Scripts 目录结构

```
scripts/
├── build.sh          # 7步构建流程 ⭐
├── install.sh        # 安装脚本
├── uninstall.sh      # 卸载脚本
└── upgrade.sh        # 升级脚本
```

**build.sh 7步构建流程**:
1. 检测 Go 环境 (Go >= 1.21)
2. 解析构建参数
3. 执行 `go mod tidy`
4. 使用 ldflags 注入版本信息编译
5. 验证编译结果
6. 测试版本输出
7. 输出构建信息

---

## 3. 模块架构图

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI Layer (cmd/morty)                   │
│  morty research | plan | doing | stat | reset | version     │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                   Command Handlers (internal/cmd/)           │
│  ResearchHandler │ PlanHandler │ DoingHandler │ StatHandler │
└──────┬───────────────────┬───────────────┬──────────────────┘
       │                   │               │
       │                   │               │
┌──────▼───────┐  ┌────────▼────────┐  ┌──▼─────────────────┐
│   Config     │  │  State Manager  │  │   Executor Engine  │
│   Manager    │  │  (状态机)        │  │   (Job执行)        │
└──────────────┘  └────────┬────────┘  └──┬─────────────────┘
                           │              │
                  ┌────────▼────────┐     │
                  │  status.json    │     │
                  │  (持久化状态)    │     │
                  └─────────────────┘     │
                                          │
        ┌─────────────────────────────────┼─────────────────┐
        │                                 │                 │
┌───────▼────────┐  ┌──────────────▼─────────┐  ┌─────────▼────────┐
│   AI CLI       │  │    Task Runner         │  │   Git Manager    │
│   Caller       │  │    (任务执行)           │  │   (自动提交)      │
│ (Claude集成)    │  └────────────────────────┘  └──────────────────┘
└────────────────┘
        │
        ▼
┌────────────────┐
│   Claude AI    │
│   (外部进程)    │
└────────────────┘
```

### 3.2 状态机架构

```
Job 状态转换图:

    ┌──────────┐
    │ PENDING  │ ◄──────────┐
    └────┬─────┘            │
         │                  │
         │ (开始执行)        │ (重试)
         ▼                  │
    ┌──────────┐            │
    │ RUNNING  │            │
    └────┬─────┘            │
         │                  │
    ┌────┴────┐             │
    │         │             │
    ▼         ▼             │
┌────────┐  ┌────────┐     │
│COMPLETED│  │ FAILED │─────┘
└────────┘  └────────┘
    │            │
    │            │ (Max Retries: 3)
    ▼            ▼
 [终态]       [终态]

BLOCKED 状态:
  PENDING ⇄ BLOCKED (依赖未满足时)
```

### 3.3 执行流程图

```
用户输入: morty doing
    │
    ▼
┌─────────────────────────┐
│ DoingHandler.Execute()  │
│ 1. 解析参数             │
│ 2. 加载状态             │
│ 3. 选择目标 Job         │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│ Engine.ExecuteJob()     │
│ 1. 检查前置条件         │
│ 2. PENDING→RUNNING      │
│ 3. 执行所有 Tasks       │
│ 4. RUNNING→COMPLETED    │
│ 5. Git Auto-commit      │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│ TaskRunner.Execute()    │
│ 1. 构建 Prompt          │
│ 2. 调用 Claude CLI      │
│ 3. 解析结果             │
│ 4. 更新任务状态         │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│ GitManager.CreateCommit │
│ morty: module/job       │
│        - COMPLETED      │
└─────────────────────────┘
```

---

## 4. 不符合预期的代码

### 4.1 发现的 TODO

**位置**: `cmd/morty/main.go:170`
```go
var executor interface{} // TODO: create actual executor if needed
```

**分析**:
- 在 `handlePlan()` 中创建 PlanHandler 时传入了 `nil` executor
- PlanHandler 的构造函数接受 executor 参数但未使用
- **影响**: 不影响功能，Plan 模式不需要 executor
- **建议**: 可以移除 executor 参数或明确文档说明

### 4.2 备份文件（未清理）

发现 2 个备份测试文件:
1. `internal/executor/task_runner_test.go.bak`
2. `internal/executor/test_check_test.go.bak`

**分析**:
- 这些是开发过程中的备份文件
- 包含 "not implemented" 错误返回的测试桩
- **影响**: 不影响功能，但增加代码库混乱
- **建议**: 删除或移到 `.gitignore`

### 4.3 Config Manager 适配器模式问题

**位置**: `cmd/morty/main.go:304-396`

```go
type pathsConfigManager struct {
    paths *config.Paths
}
// 实现了 config.Manager 接口但大部分方法返回空值
func (p *pathsConfigManager) Load(path string) error { return nil }
func (p *pathsConfigManager) Get(key string, ...) (interface{}, error) { return nil, nil }
```

**分析**:
- 创建了一个适配器将 `*config.Paths` 适配为 `config.Manager` 接口
- 大部分方法返回空值/nil，只实现了路径相关方法
- **影响**: 功能正常但设计不够清晰
- **建议**: 
  - 方案1: 创建专门的 PathsManager 类型
  - 方案2: 让 Handlers 直接接受 Paths 而非 Manager 接口

---

## 5. 未完成的代码

### 5.1 Executor Engine 的任务更新

**位置**: `internal/executor/engine.go:351-362`

```go
func (e *engine) updateTasksCompleted(module, job string, count int) error {
    // Note: In the current state.Manager, we don't have a direct method
    // to update TasksCompleted. This would need to be added...
    e.logger.Debug("Tasks completed updated", ...)
    return nil
}
```

**分析**:
- 方法存在但未实际更新状态
- 只记录日志，不更新 state.Manager 中的任务完成计数
- **影响**: 任务完成进度可能不准确
- **建议**: 在 state.Manager 中添加 `UpdateTasksCompleted()` 方法

### 5.2 失败原因更新

**位置**: `internal/executor/engine.go:364-373`

```go
func (e *engine) updateFailureReason(module, job, reason string) error {
    // Similar to updateTasksCompleted, this would need state access
    e.logger.Debug("Failure reason updated", ...)
    return nil
}
```

**分析**:
- 失败原因未持久化到状态文件
- 只记录到日志
- **影响**: 无法从状态文件中查看失败原因
- **建议**: 扩展 JobState 结构添加 FailureReason 字段

### 5.3 任务级别状态标记

**位置**: `internal/executor/engine.go:338-349`

```go
func (e *engine) markTaskCompleted(module, job string, taskIndex int) error {
    // This would typically be implemented by accessing the state directly
    // For now, we use the UpdateJobStatus which saves the state
    e.logger.Debug("Task marked as completed", ...)
    return nil
}
```

**分析**:
- 单个任务完成状态未更新
- ExecuteTask 调用此方法但实际未标记
- **影响**: 任务级别的状态跟踪不完整
- **建议**: 实现真正的任务状态更新逻辑

---

## 6. 执行主路径卡点

### 6.1 状态转换验证卡点 ⚠️

**位置**: `internal/state/transitions.go:76-127`

**问题**:
- 状态转换规则严格，COMPLETED 是终态（无法转换）
- 如果 Job 已完成但需要重新执行，必须先手动重置

```go
var TransitionRules = map[Status][]Status{
    StatusCompleted: {}, // Terminal state - no outgoing transitions
}
```

**影响**:
- 用户无法直接重跑已完成的 Job
- 必须使用 `morty reset` 或 `--restart` 标志
- 增加操作复杂度

**建议**:
- 添加 `COMPLETED → PENDING` 转换（用于重新执行）
- 或在 DoingHandler 中特殊处理已完成 Job

### 6.2 AI CLI 进程创建开销 ⚠️⚠️

**位置**: `internal/callcli/ai_caller.go`

**问题**:
- 每个 Task 都会创建新的 `claude` CLI 进程
- 进程创建 + AI 初始化开销大
- 串行执行，无并发优化

**性能影响**:
```
单个 Task 耗时 = 进程启动 (0.5-1s) + AI处理 (5-30s) + 进程清理 (0.1s)
10个 Tasks = 至少 60-310秒
```

**建议**:
- 实现 Task 批处理（一次调用处理多个 Tasks）
- 使用长连接模式与 Claude CLI 通信
- 支持并发执行独立 Tasks

### 6.3 Plan 文档解析依赖 ⚠️

**位置**: `internal/parser/plan/parser.go`

**问题**:
- Plan 文档格式必须严格遵循 Markdown 结构
- 解析器基于正则表达式，容错性差
- 格式错误会导致整个 Doing 流程失败

**影响**:
- Plan 文档编写门槛高
- 小的格式错误导致执行失败
- 错误提示不够友好

**建议**:
- 增强解析器容错性
- 提供 Plan 文档验证工具 (`morty plan --validate`)
- 改进错误提示，指出具体格式问题

### 6.4 状态文件锁竞争 ⚠️

**位置**: `internal/state/manager.go`

**问题**:
- 使用 `sync.RWMutex` 保护状态文件
- 每次状态更新都需要 Save() 到磁盘
- 高频更新时可能产生 I/O 瓶颈

```go
func (m *Manager) TransitionJobStatus(...) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    // ... 更新状态
    m.mu.Unlock()
    err := m.Save()  // 磁盘 I/O
    m.mu.Lock()
}
```

**影响**:
- 大型项目（100+ Jobs）状态更新慢
- 频繁磁盘写入影响性能

**建议**:
- 实现批量状态更新
- 延迟写入（定期刷盘）
- 使用 WAL（Write-Ahead Log）模式

### 6.5 前置条件检查限制 ⚠️

**位置**: `internal/executor/engine.go:246-279`

**问题**:
- 前置条件检查简单，只检查 Job 状态
- 不支持跨 Module 依赖
- 不支持复杂依赖关系（如 A 依赖 B 或 C）

```go
func (e *engine) checkPrerequisites(ctx context.Context, module, job string) error {
    // 只检查当前 Job 的状态
    // 不检查依赖的其他 Jobs
}
```

**影响**:
- 无法表达复杂的任务依赖
- 必须手动控制执行顺序
- 容易出现依赖未满足的执行错误

**建议**:
- 在 Plan 文档中添加 `depends_on` 字段
- 实现依赖图解析和验证
- 支持自动依赖排序执行

### 6.6 错误恢复机制不足 ⚠️

**位置**: `internal/executor/engine.go:108-207`

**问题**:
- 失败后只支持简单重试（最多3次）
- 重试策略固定，无指数退避
- 部分失败无法跳过继续执行

```go
if jobState.RetryCount >= e.config.MaxRetries {
    return fmt.Errorf("max retries exceeded (%d)", e.config.MaxRetries)
}
```

**影响**:
- 临时性错误（网络超时）浪费重试次数
- 无法跳过非关键 Job 继续执行
- 错误恢复策略不灵活

**建议**:
- 实现指数退避重试策略
- 支持 `--skip-failed` 选项
- 添加 Job 级别的重试配置

---

## 7. 架构优势

### 7.1 设计模式运用

1. **Factory Pattern** (`parser/factory.go`)
   - 解析器工厂，支持多种文档类型

2. **Manager Pattern** (贯穿整个项目)
   - StateManager, GitManager, ConfigManager
   - 封装领域逻辑

3. **Handler Pattern** (`cmd/`)
   - 每个命令一个 Handler
   - 统一接口: `Execute(ctx, args) -> Result`

4. **Interface-Based Design**
   - 高度解耦，易于测试
   - 支持多种实现

### 7.2 状态管理

- 完整的状态机实现
- 严格的状态转换验证
- 持久化到 JSON 文件
- 线程安全（RWMutex）

### 7.3 测试覆盖

- 58 个测试文件
- 单元测试 + 集成测试 + Shell 测试
- 测试驱动开发模式
- 高测试覆盖率

---

## 8. 性能瓶颈总结

| 瓶颈点 | 严重程度 | 影响 | 优化优先级 |
|--------|---------|------|-----------|
| AI CLI 进程创建开销 | ⚠️⚠️⚠️ | 执行时间长 | 🔥 高 |
| 状态文件频繁 I/O | ⚠️⚠️ | 大项目慢 | 🔥 高 |
| Plan 解析容错性差 | ⚠️⚠️ | 易出错 | 🔥 高 |
| 前置条件检查简单 | ⚠️ | 依赖管理弱 | 🔶 中 |
| 错误恢复机制不足 | ⚠️ | 重试不灵活 | 🔶 中 |
| 状态转换限制 | ⚠️ | 操作复杂 | 🔷 低 |

---

## 9. 改进建议

### 9.1 短期优化（1-2周）

1. **清理代码**
   - 删除 `.bak` 备份文件
   - 移除 TODO 注释或实现功能
   - 完善 pathsConfigManager

2. **完善状态更新**
   - 实现 `updateTasksCompleted()`
   - 实现 `updateFailureReason()`
   - 实现 `markTaskCompleted()`

3. **增强错误提示**
   - Plan 解析错误详细提示
   - 状态转换错误友好提示

### 9.2 中期优化（2-4周）

1. **性能优化**
   - 实现 Task 批处理
   - 状态文件批量更新
   - 添加缓存机制

2. **依赖管理**
   - 支持 Job 依赖声明
   - 实现依赖图解析
   - 自动依赖排序

3. **错误恢复**
   - 指数退避重试
   - 支持跳过失败 Job
   - Job 级别重试配置

### 9.3 长期优化（1-2月）

1. **并发执行**
   - 支持独立 Job 并发执行
   - 实现任务调度器
   - 资源限制控制

2. **监控和可观测性**
   - 实时进度展示
   - 性能指标收集
   - 执行链路追踪

3. **扩展性**
   - 插件系统
   - 自定义 Parser
   - 多 AI 后端支持

---

## 10. 关键文件清单

| 文件 | 行数 | 重要性 | 说明 |
|------|------|--------|------|
| `cmd/morty/main.go` | 396 | ⭐⭐⭐ | CLI 入口 |
| `internal/executor/engine.go` | 448 | ⭐⭐⭐ | 执行引擎核心 |
| `internal/state/manager.go` | 278 | ⭐⭐⭐ | 状态管理核心 |
| `internal/state/transitions.go` | 224 | ⭐⭐⭐ | 状态转换逻辑 |
| `internal/cmd/doing.go` | ~1000 | ⭐⭐⭐ | Doing 命令处理 |
| `internal/callcli/ai_caller.go` | ~500 | ⭐⭐ | AI CLI 集成 |
| `internal/git/manager.go` | ~300 | ⭐⭐ | Git 自动提交 |
| `internal/parser/plan/parser.go` | ~800 | ⭐⭐ | Plan 解析 |
| `scripts/build.sh` | 261 | ⭐ | 构建脚本 |

---

## 11. 结论

### 11.1 项目成熟度
- ✅ **架构设计**: 优秀，模块化清晰
- ✅ **代码质量**: 良好，测试覆盖高
- ⚠️ **功能完整性**: 部分功能未实现（状态更新）
- ⚠️ **性能**: 存在瓶颈（AI CLI 调用、状态 I/O）
- ✅ **可维护性**: 优秀，接口设计清晰

### 11.2 主要卡点
1. **AI CLI 进程开销** - 最大性能瓶颈
2. **状态文件 I/O** - 大项目性能问题
3. **Plan 解析容错** - 易用性问题
4. **依赖管理缺失** - 功能限制
5. **部分功能未完成** - 状态更新不完整

### 11.3 总体评价
Morty 是一个**架构优秀、设计清晰**的 AI 编码编排框架，具有良好的扩展性和可维护性。主要问题集中在**性能优化**和**功能完善**方面，通过上述改进建议可以显著提升用户体验。

---

**文档版本**: 1.0  
**研究完成时间**: 2026-02-27T11:05:00Z  
**状态**: ✅ 已完成  
**探索子代理使用**: ✅ 是 (Agent ID: a8417b7a1b1ad5545)
