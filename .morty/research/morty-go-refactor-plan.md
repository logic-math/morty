# Morty Go 重构计划

**调查主题**: morty-go-refactor
**调研日期**: 2026-02-23

---

## 1. 项目概述

### 1.1 项目类型
代码仓库 - AI Coding Agent 编排框架

### 1.2 重构目标
将基于 Shell 的 morty 项目重构为 Go 语言实现，遵循 TDD 原则（先设计单元测试用例，再实现代码），并设计流程图。

### 1.3 技术栈
- Go 1.21+
- slog (标准库结构化日志)
- testify (测试框架)
- go-git (Git 操作，可选)

---

## 2. 核心发现

### 2.1 原 Shell 架构分析

Morty 是一个**上下文优先的 AI Coding Agent 编排框架**，具有以下核心模块：

| 模块 | 文件 | 职责 |
|-----|------|------|
| CLI 系统 | `morty`, `lib/cli_*.sh` | 命令注册、解析、路由、执行 |
| 配置管理 | `lib/config.sh` | settings.json 管理，支持 dot notation |
| 日志系统 | `lib/logging.sh` | 多级别日志、JSON/文本格式、日志轮转、Job 上下文 |
| 版本管理 | `lib/version_manager.sh` | Git 集成、循环提交、版本回滚 |
| 状态管理 | `morty_doing.sh` | status.json、Job 状态机、断点恢复 |
| 执行引擎 | `morty_doing.sh` | Plan 解析、Job/Task 执行、AI CLI 调用 |

### 2.2 关键数据结构

**status.json 结构:**
```json
{
  "version": "2.0",
  "state": "running",
  "current": { "module": "...", "job": "...", "status": "..." },
  "session": { "start_time": "...", "last_update": "...", "total_loops": 0 },
  "modules": {
    "module_name": {
      "status": "pending",
      "jobs": {
        "job_1": {
          "status": "PENDING",
          "loop_count": 0,
          "retry_count": 0,
          "tasks_total": 5,
          "tasks_completed": 0,
          "debug_logs": []
        }
      }
    }
  }
}
```

**Job 状态机:**
```
PENDING ──▶ RUNNING ──▶ COMPLETED
              │
              ├──▶ FAILED ──▶ PENDING (重试)
              │
              └──▶ BLOCKED ──▶ PENDING (解除阻塞)
```

---

## 3. Go 项目结构设计

### 3.1 目录结构

```
morty-go/
├── cmd/morty/main.go              # 主入口
├── internal/
│   ├── cli/                       # CLI 模块
│   │   ├── command.go             # 命令结构定义
│   │   ├── parser.go              # 参数解析
│   │   ├── parser_test.go         # 参数解析测试
│   │   ├── router.go              # 命令路由
│   │   ├── router_test.go         # 路由测试
│   │   ├── executor.go            # 命令执行
│   │   └── registry.go            # 命令注册表
│   ├── config/                    # 配置模块
│   │   ├── config.go              # 配置接口
│   │   ├── loader.go              # 配置加载器
│   │   ├── settings.go            # settings.json 管理
│   │   ├── paths.go               # 路径管理
│   │   └── manager_test.go        # 配置管理测试
│   ├── logging/                   # 日志模块
│   │   ├── logger.go              # 日志接口
│   │   ├── slog_adapter.go        # slog 实现
│   │   ├── slog_adapter_test.go   # 日志测试
│   │   ├── rotator.go             # 日志轮转
│   │   ├── rotator_test.go        # 轮转测试
│   │   ├── job_logger.go          # Job 上下文日志
│   │   └── format.go              # 格式定义
│   ├── state/                     # 状态管理模块
│   │   ├── state.go               # 状态接口和类型
│   │   ├── manager.go             # 状态管理器
│   │   ├── manager_test.go        # 状态管理测试
│   │   ├── transitions.go         # 状态转换规则
│   │   └── status_json.go         # status.json 操作
│   ├── executor/                  # 执行引擎模块
│   │   ├── engine.go              # 执行引擎
│   │   ├── engine_test.go         # 引擎测试
│   │   ├── job_runner.go          # Job 执行器
│   │   ├── task_runner.go         # Task 执行器
│   │   ├── prompt_builder.go      # 提示词构建
│   │   ├── prompt_builder_test.go # 提示词测试
│   │   └── result_parser.go       # 结果解析
│   ├── git/                       # Git 模块
│   │   ├── git.go                 # Git 接口
│   │   ├── manager.go             # Git 管理器
│   │   ├── manager_test.go        # Git 测试
│   │   ├── commit.go              # 提交操作
│   │   └── version.go             # 版本管理
│   ├── plan/                      # Plan 模块
│   │   ├── parser.go              # Plan 文件解析
│   │   ├── parser_test.go         # 解析测试
│   │   ├── loader.go              # Plan 加载器
│   │   └── types.go               # Plan 数据结构
│   └── errors/                    # 错误定义
│       └── errors.go
├── pkg/
│   ├── types/types.go             # 公共类型定义
│   └── utils/                     # 工具函数
│       ├── files.go
│       └── strings.go
├── tests/mocks/                   # Mock 实现
│   ├── mock_config.go
│   ├── mock_state.go
│   ├── mock_git.go
│   └── mock_logger.go
├── configs/settings.json          # 默认配置
├── prompts/                       # 系统提示词
│   ├── doing.md
│   ├── plan.md
│   └── research.md
├── go.mod
├── go.sum
└── Makefile
```

---

## 4. 核心接口定义

### 4.1 CLI 模块

```go
// Command 表示一个 CLI 命令
type Command struct {
    Name        string
    Description string
    Handler     CommandHandler
    Options     []Option
}

type CommandHandler func(ctx context.Context, args []string) error

type Parser interface {
    Parse(args []string) (*ParseResult, error)
}

type ParseResult struct {
    Command    string
    Positional []string
    Options    map[string]string
    Flags      map[string]bool
}

type Router interface {
    Register(cmd Command) error
    Route(ctx context.Context, result *ParseResult) error
    GetHandler(name string) (CommandHandler, bool)
}
```

### 4.2 Config 模块

```go
type Manager interface {
    Load(path string) error
    Get(key string, defaultValue ...interface{}) (interface{}, error)
    GetString(key string, defaultValue ...string) string
    GetInt(key string, defaultValue ...int) int
    GetBool(key string, defaultValue ...bool) bool
    Set(key string, value interface{}) error
    Save() error
    GetWorkDir() string
    GetLogDir() string
    GetResearchDir() string
    GetPlanDir() string
    GetStatusFile() string
}
```

### 4.3 Logging 模块

```go
type Level int
const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
    SUCCESS
    LOOP
)

type Logger interface {
    Debug(msg string, attrs ...slog.Attr)
    Info(msg string, attrs ...slog.Attr)
    Warn(msg string, attrs ...slog.Attr)
    Error(msg string, attrs ...slog.Attr)
    Success(msg string, attrs ...slog.Attr)
    Loop(msg string, attrs ...slog.Attr)
    WithContext(ctx context.Context) Logger
    WithJob(module, job string) Logger
    WithAttrs(attrs ...slog.Attr) Logger
    SetLevel(level Level)
    GetLevel() Level
}

type Rotator interface {
    Rotate(logFile string) error
    ShouldRotate(logFile string) bool
}
```

### 4.4 State 模块

```go
type Status string
const (
    StatusPending   Status = "PENDING"
    StatusRunning   Status = "RUNNING"
    StatusCompleted Status = "COMPLETED"
    StatusFailed    Status = "FAILED"
    StatusBlocked   Status = "BLOCKED"
)

type Manager interface {
    Load() error
    Save() error
    GetJobStatus(module, job string) (Status, error)
    UpdateJobStatus(module, job string, status Status) error
    IsValidTransition(from, to Status) bool
    GetCurrent() (*CurrentJob, error)
    SetCurrent(module, job string, status Status) error
    GetSummary() (*Summary, error)
    GetPendingJobs() []JobRef
}
```

### 4.5 Git 模块

```go
type Manager interface {
    InitIfNeeded(dir string) error
    HasUncommittedChanges(dir string) (bool, error)
    GetRepoRoot(dir string) (string, error)
    CreateLoopCommit(loopNumber int, status string, dir string) (string, error)
    GetCurrentLoopNumber(dir string) (int, error)
    GetChangeStats(dir string) (*ChangeStats, error)
    ResetToCommit(commitHash string, dir string) error
    ShowLoopHistory(n int, dir string) ([]LoopCommit, error)
}
```

### 4.6 Executor 模块

```go
type Engine interface {
    ExecuteJob(ctx context.Context, module, job string) error
    ExecuteTask(ctx context.Context, module, job string, taskIndex int, taskDesc string) error
    ResumeJob(ctx context.Context, module, job string) error
}

type PromptBuilder interface {
    BuildPrompt(module, job string, taskIndex int, taskDesc string) (string, error)
    BuildCompactContext(module, job string) (map[string]interface{}, error)
}

type ResultParser interface {
    Parse(outputFile string) (*ExecutionResult, error)
}

type ExecutionResult struct {
    Status         string
    TasksCompleted int
    TasksTotal     int
    Summary        string
    Module         string
    Job            string
}
```

---

## 5. 单元测试用例设计

### 5.1 CLI Parser 测试

```go
func TestParser_Parse(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        expected *ParseResult
        wantErr  bool
    }{
        {
            name: "simple command",
            args: []string{"doing"},
            expected: &ParseResult{
                Command:    "doing",
                Positional: []string{},
                Options:    map[string]string{},
                Flags:      map[string]bool{},
            },
        },
        {
            name: "command with options",
            args: []string{"doing", "--module", "config", "--job", "job_1"},
            expected: &ParseResult{
                Command:    "doing",
                Positional: []string{},
                Options: map[string]string{
                    "--module": "config",
                    "--job":    "job_1",
                },
                Flags: map[string]bool{},
            },
        },
        {
            name: "command with flags",
            args: []string{"doing", "--restart", "-w"},
            expected: &ParseResult{
                Command:    "doing",
                Positional: []string{},
                Options:    map[string]string{},
                Flags: map[string]bool{
                    "--restart": true,
                    "-w":        true,
                },
            },
        },
    }
    // ... 测试实现
}
```

**测试场景总结:**

| 测试场景 | 输入 | 预期输出 |
|---------|------|---------|
| simple command | `[]string{"doing"}` | Command="doing" |
| command with options | `[]string{"doing", "--module", "config"}` | Options={"--module":"config"} |
| command with flags | `[]string{"doing", "--restart"}` | Flags={"--restart":true} |
| mixed args | `[]string{"doing", "--module=config", "extra"}` | Positional=["extra"] |

### 5.2 Config Manager 测试

| 测试场景 | 输入 | 预期输出 |
|---------|------|---------|
| simple key | key="cli.command" | "claude" |
| nested key | key="defaults.paths.work_dir" | ".morty" |
| key not found | key="cli.nonexistent", default="default" | "default" |
| integer value | key="defaults.max_loops" | 50 |
| set simple key | Set("cli.command", "claude") | 成功设置 |
| set nested key | Set("defaults.max_loops", 100) | 自动创建嵌套结构 |

### 5.3 State Manager 测试

| 测试场景 | 输入 | 预期输出 |
|---------|------|---------|
| valid transition PENDING->RUNNING | from=PENDING, to=RUNNING | true |
| valid transition RUNNING->COMPLETED | from=RUNNING, to=COMPLETED | true |
| valid transition FAILED->PENDING | from=FAILED, to=PENDING | true (重试) |
| invalid transition PENDING->COMPLETED | from=PENDING, to=COMPLETED | false |
| update status | UpdateJobStatus("test", "job_1", RUNNING) | 状态更新，loop_count++ |
| get pending jobs | modules包含PENDING和COMPLETED | 只返回PENDING |

### 5.4 Logging Rotator 测试

| 测试场景 | 输入 | 预期输出 |
|---------|------|---------|
| small file | fileSize=50, maxSize=100 | ShouldRotate=false |
| over limit | fileSize=101, maxSize=100 | ShouldRotate=true |
| rotate file | content="test", rotate | 原文件清空，.1文件包含内容 |
| compress old | rotate 多次 | .2及以上文件被 gzip 压缩 |

### 5.5 Git Manager 测试

| 测试场景 | 输入 | 预期输出 |
|---------|------|---------|
| no uncommitted changes | 空 git 仓库 | false |
| has uncommitted changes | 创建未跟踪文件 | true |
| create loop commit | loopNumber=1, status="completed" | hash非空，提交信息包含"morty[loop:1" |
| get change stats | 创建多个文件 | staged + unstaged + untracked 正确统计 |
| reset to commit | commitHash=abc123 | 成功回滚，创建备份分支 |

### 5.6 Result Parser 测试

| 测试场景 | 输入 | 预期输出 |
|---------|------|---------|
| valid JSON result | `{"ralph_status":{"status":"COMPLETED"}}` | Status="COMPLETED" |
| old format JSON | `{"status":"FAILED"}` | Status="FAILED" |
| invalid JSON | `{invalid` | 回退到文本解析 |

---

## 6. 流程图设计

### 6.1 Job 执行流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Job 执行流程                                 │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────┐
│   开始执行    │
└──────┬───────┘
       │
       ▼
┌──────────────────┐     否     ┌─────────────────┐
│ 检查前置条件      │──────────▶│  返回错误        │
│ (Plan 文件存在?)  │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 是
       ▼
┌──────────────────┐     否     ┌─────────────────┐
│ 加载 status.json  │──────────▶│  初始化状态      │
│                  │            │  (首次运行)      │
└──────┬───────────┘            └─────────────────┘
       │ 是
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ --restart 标志?  │──────────▶│  重置 Job 状态   │
│                  │            │  (保留 git 历史) │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ 检查 Job 状态     │──────────▶│  跳过已完成      │
│ == COMPLETED?    │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ 检查重试次数      │──────────▶│  标记为 FAILED   │
│ >= MAX_RETRY?    │            │  返回失败        │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐
│ 更新状态为 RUNNING│◄─────────────────────────────┐
│ (loop_count++)   │                              │
└──────┬───────────┘                              │
       │                                           │
       ▼                                           │
┌──────────────────┐     失败    ┌────────────────┐│
│ 执行 Tasks 循环   │──────────▶│  标记为 FAILED  ││
│ (逐个执行)       │            │  检查是否重试   ││
└──────┬───────────┘            │  是: 重置 PENDING│
       │ 成功                    │  否: 彻底失败   ││
       ▼                        └───────┬────────┘│
┌──────────────────┐                   │         │
│ 标记为 COMPLETED │◄──────────────────┘         │
│ 创建 Git 提交    │                              │
└──────┬───────────┘                              │
       │                                           │
       ▼                                           │
┌──────────────┐                                   │
│   执行完成    │───────────────────────────────────┘
└──────────────┘
       │
       ▼
┌──────────────────┐
│  处理下一个 Job   │
└──────────────────┘
```

### 6.2 状态转换状态机

```
┌─────────────────────────────────────────────────────────────────────┐
│                       状态转换状态机                                 │
└─────────────────────────────────────────────────────────────────────┘

                              ┌─────────────┐
                              │   PENDING   │
                              │   (初始状态) │
                              └──────┬──────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                │                ▼
            ┌─────────────┐          │        ┌─────────────┐
            │   RUNNING   │◄─────────┘        │   BLOCKED   │
            │  (执行中)    │                   │   (阻塞)     │
            └──────┬──────┘                   └──────┬──────┘
                   │                                 │
         ┌─────────┼─────────┐                       │
         │         │         │                       ▼
         ▼         ▼         ▼                ┌─────────────┐
   ┌─────────┐ ┌─────────┐ ┌─────────┐        │   PENDING   │
   │COMPLETED│ │  FAILED │ │ BLOCKED │        │  (解除阻塞)  │
   │ (完成)   │ │ (失败)   │ │ (阻塞)   │        └─────────────┘
   └────┬────┘ └────┬────┘ └─────────┘
        │           │
        │    ┌──────┴──────┐
        │    │             │
        │    ▼             ▼
        │ ┌─────────┐  ┌─────────┐
        │ │ PENDING │  │ RUNNING │  (重试)
        │ │ (重试)   │  │ (重试)   │
        │ └─────────┘  └─────────┘
        │
        ▼
   ┌─────────┐
   │ PENDING │  (重置)
   │ (重置)   │
   └─────────┘

═══════════════════════════════════════════════════════════════════════
有效转换规则:
═══════════════════════════════════════════════════════════════════════

PENDING ──▶ RUNNING    ✓ 开始执行
PENDING ──▶ BLOCKED    ✓ 前置条件不满足

RUNNING ──▶ COMPLETED  ✓ 成功完成
RUNNING ──▶ FAILED     ✓ 执行失败
RUNNING ──▶ PENDING    ✓ 重置/重试
RUNNING ──▶ BLOCKED    ✓ 执行中遇到阻塞

FAILED  ──▶ PENDING    ✓ 重试 (retry_count < MAX)
FAILED  ──▶ RUNNING    ✓ 立即重试

BLOCKED ──▶ PENDING    ✓ 阻塞解除

COMPLETED ─▶ PENDING   ✓ 重置 (--restart)
```

### 6.3 日志轮转流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        日志轮转流程                                  │
└─────────────────────────────────────────────────────────────────────┘

┌────────────────┐
│   写入日志      │
│ (每100次检查)  │
└───────┬────────┘
        │
        ▼
┌──────────────────┐     否     ┌─────────────────┐
│ 检查文件大小      │──────────▶│   直接写入       │
│ >= MAX_SIZE?     │            │                 │
│ (默认 10MB)      │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 是
       ▼
┌──────────────────┐
│  执行日志轮转     │
└──────┬───────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    标准轮转模式                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  morty.log ──▶ morty.log.1 ──▶ morty.log.2 ──▶ [压缩]      │
│     │              │               │                        │
│     │              │               └── morty.log.3.gz       │
│     │              │                                        │
│     │              └────────────────── morty.log.2.gz       │
│     │                                                       │
│     └─────────────────────────────────────── [新空文件]      │
│                                                             │
│  清理: 删除 morty.log.(MAX_FILES+1) 及更旧的日志             │
└─────────────────────────────────────────────────────────────┘

       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    日期归档模式 (可选)                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  morty.log ──▶ archive/morty-2024-01-15.log.gz              │
│                                                             │
│  文件名格式: {basename}-{date}.log[.{counter}].gz           │
│                                                             │
│  如果文件已存在，添加序号:                                   │
│  morty-2024-01-15.log ──▶ morty-2024-01-15-1.log            │
│                                                             │
└─────────────────────────────────────────────────────────────┘

       │
       ▼
┌────────────────┐
│  压缩旧日志     │
│ (gzip)         │
└───────┬────────┘
        │
        ▼
┌────────────────┐
│   轮转完成      │
└────────────────┘
```

### 6.4 CLI 命令路由流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                      CLI 命令路由流程                                │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────┐
│  morty [args] │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ 解析全局选项      │
│ --verbose        │
│ --debug          │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ 剩余参数为空?     │──────────▶│  显示全局帮助    │
│                  │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐
│ 获取第一个参数    │
│ 作为 command     │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ command ==       │──────────▶│ 显示全局帮助     │
│ "help" (无参数)? │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ command ==       │──────────▶│ 显示特定命令帮助 │
│ "help <cmd>"?    │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ command ==       │──────────▶│ 显示版本信息     │
│ "version"?       │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐     否     ┌─────────────────┐
│ 命令是否已注册?   │──────────▶│  显示错误        │
│                  │            │ "未知命令"       │
└──────┬───────────┘            └─────────────────┘
       │ 是
       ▼
┌──────────────────┐
│ 解析命令特定参数  │
│ (剩余参数)       │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ Handler 是       │──────────▶│ 执行脚本文件     │
│ 脚本文件?        │            │ (exec)          │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐     是     ┌─────────────────┐
│ Handler 是       │──────────▶│ 调用函数         │
│ 函数?            │            │                 │
└──────┬───────────┘            └─────────────────┘
       │ 否
       ▼
┌──────────────────┐
│ 尝试作为外部命令  │
│ 执行             │
└──────────────────┘
```

---

## 7. 实现步骤

### Phase 1: 基础框架
- [ ] 创建项目结构和 go.mod
- [ ] 实现 errors 包
- [ ] 实现公共 types 和 utils

### Phase 2: Config 模块
- [ ] 实现配置接口和加载器
- [ ] 实现 settings.json 管理
- [ ] 编写单元测试

### Phase 3: Logging 模块
- [ ] 实现 Logger 接口
- [ ] 实现 slog 适配器
- [ ] 实现日志轮转
- [ ] 编写单元测试

### Phase 4: State 模块
- [ ] 实现状态类型定义
- [ ] 实现状态管理器
- [ ] 实现状态转换规则
- [ ] 编写单元测试

### Phase 5: Git 模块
- [ ] 实现 Git 管理器
- [ ] 实现循环提交
- [ ] 实现版本回滚
- [ ] 编写单元测试

### Phase 6: Plan 模块
- [ ] 实现 Plan 文件解析器
- [ ] 实现 Plan 加载器
- [ ] 编写单元测试

### Phase 7: CLI 模块
- [ ] 实现参数解析器
- [ ] 实现命令路由器
- [ ] 实现命令注册表
- [ ] 编写单元测试

### Phase 8: Executor 模块
- [ ] 实现执行引擎
- [ ] 实现 Job/Task 执行器
- [ ] 实现提示词构建器
- [ ] 实现结果解析器
- [ ] 编写单元测试

### Phase 9: 集成
- [ ] 实现依赖注入容器
- [ ] 实现主入口 main.go
- [ ] 集成所有模块
- [ ] 编写集成测试

### Phase 10: 验证
- [ ] 运行所有单元测试
- [ ] 运行集成测试
- [ ] 端到端测试
- [ ] 性能测试

---

## 8. 依赖库

```go
// go.mod
module github.com/morty/morty-go

go 1.21

require (
    github.com/stretchr/testify v1.8.4
    github.com/go-git/go-git/v5 v5.11.0  // 可选，Git 操作
    gopkg.in/yaml.v3 v3.0.1               // 可选，YAML 配置
    github.com/BurntSushi/toml v1.3.2     // 可选，TOML 配置
)
```

---

## 9. 验证方法

```bash
# 运行单元测试
go test ./...

# 运行覆盖率测试
go test -cover ./...

# 构建可执行文件
go build -o morty ./cmd/morty

# 功能验证
./morty version           # 显示版本
./morty help              # 显示帮助
./morty doing --help      # 显示 doing 命令帮助
./morty doing             # 执行 Plan
./morty doing --restart   # 重置并执行
```

---

## 10. 关键文件参考

| 原 Shell 文件 | Go 对应实现 | 说明 |
|--------------|------------|------|
| `morty` | `cmd/morty/main.go`, `internal/cli/` | 主入口和 CLI 路由 |
| `lib/cli_parse_args.sh` | `internal/cli/parser.go` | 参数解析 |
| `morty_doing.sh` | `internal/executor/` | 执行引擎（最复杂） |
| `lib/logging.sh` | `internal/logging/` | 日志系统 |
| `lib/version_manager.sh` | `internal/git/` | Git 管理 |
| `lib/config.sh` | `internal/config/` | 配置管理 |
| `lib/common.sh` | `pkg/utils/` | 通用工具 |

---

**文档版本**: 1.0
**研究完成时间**: 2026-02-23
**状态**: 已完成
**探索子代理使用**: 是
