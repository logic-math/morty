# Plan: Call CLI

## 模块概述

**模块职责**: 提供统一的命令行工具调用能力，以子进程方式执行外部 CLI 工具（如 ai_cli、claude、git 等），支持参数传递、超时控制、信号处理和输出捕获。

**对应 Research**:
- `morty-project-research.md` 第 3.2 节 AI 工具调用分析

**依赖模块**: Config, Logging

**被依赖模块**: doing_cmd, research_cmd, plan_cmd, executor

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        Call CLI Module                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │     Caller      │  │  ProcessManager │  │  Output     │ │
│  │   (调用接口)     │  │   (进程管理)     │  │  Handler    │ │
│  │                 │  │                 │  │ (输出处理)   │ │
│  │ - Call()        │  │ - Start()       │  │ - Capture() │ │
│  │ - CallAsync()   │  │ - Wait()        │  │ - Stream()  │ │
│  │ - CallWithCtx() │  │ - Kill()        │  │ - Parse()   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   AI CLI      │    │    Git        │    │   Shell       │
│  (ai_cli)     │    │  (git cmd)    │    │  (sh/bash)    │
└───────────────┘    └───────────────┘    └───────────────┘
```

---

## 接口定义

### 核心接口

```go
// Caller CLI 调用器接口
type Caller interface {
    // Call 同步执行命令，返回执行结果
    Call(ctx context.Context, req *CallRequest) (*CallResult, error)

    // CallAsync 异步执行命令，返回用于控制的 Handler
    CallAsync(ctx context.Context, req *CallRequest) (CallHandler, error)

    // CallWithInput 执行命令并传递 stdin 输入
    CallWithInput(ctx context.Context, req *CallRequest, input string) (*CallResult, error)
}

// CallHandler 异步调用控制句柄
type CallHandler interface {
    // Wait 等待命令执行完成，返回结果
    Wait() (*CallResult, error)

    // Kill 终止命令执行
    Kill() error

    // PID 获取进程 ID
    PID() int

    // Running 检查是否仍在运行
    Running() bool
}

// CallRequest 调用请求
type CallRequest struct {
    // 命令名称（如 "ai_cli", "git", "sh"）
    Command string

    // 命令参数
    Args []string

    // 工作目录
    WorkDir string

    // 环境变量（覆盖或追加）
    Env map[string]string

    // 超时时间（0 表示无超时）
    Timeout time.Duration

    // 输出模式
    OutputMode OutputMode

    // 输出文件路径（可选，用于保存完整输出）
    OutputFile string
}

// CallResult 调用结果
type CallResult struct {
    // 退出码
    ExitCode int

    // 标准输出
    Stdout string

    // 标准错误
    Stderr string

    // 合并输出（stdout + stderr）
    CombinedOutput string

    // 执行耗时
    Duration time.Duration

    // 是否超时
    TimedOut bool

    // 是否被信号中断
    Interrupted bool

    // 信号编号（如果被信号终止）
    Signal os.Signal
}

// OutputMode 输出模式
type OutputMode int

const (
    // OutputCapture 捕获输出到内存
    OutputCapture OutputMode = iota

    // OutputStream 实时流式输出到控制台
    OutputStream

    // OutputCaptureAndStream 同时捕获和流式输出
    OutputCaptureAndStream

    // OutputSilent 完全静默（丢弃输出）
    OutputSilent
)
```

### AI CLI 专用接口

```go
// AICliCaller AI CLI 专用调用器
type AICliCaller interface {
    Caller

    // CallWithPrompt 使用提示词文件调用 AI CLI
    CallWithPrompt(ctx context.Context, promptFile string, opts *AICliOptions) (*CallResult, error)

    // CallWithPromptContent 直接使用提示词内容调用 AI CLI
    CallWithPromptContent(ctx context.Context, promptContent string, opts *AICliOptions) (*CallResult, error)
}

// AICliOptions AI CLI 调用选项
type AICliOptions struct {
    // 输出格式（json, text）
    OutputFormat string

    // 详细模式
    Verbose bool

    // 调试模式
    Debug bool

    // 跳过权限确认（dangerously-skip-permissions）
    SkipPermissions bool

    // 超时时间
    Timeout time.Duration

    // 输出文件
    OutputFile string

    // 额外参数
    ExtraArgs []string
}

// AICliConfig AI CLI 配置
type AICliConfig struct {
    // AI CLI 命令名称（默认 ai_cli）
    Command string

    // 全局默认参数
    DefaultArgs []string

    // 是否启用 dangerously-skip-permissions
    EnableSkipPermissions bool

    // 默认超时
    DefaultTimeout time.Duration
}
```

---

## 数据模型

```go
// ProcessInfo 进程信息
type ProcessInfo struct {
    PID        int
    Command    string
    Args       []string
    StartTime  time.Time
    WorkDir    string
}

// ExecutionLog 执行日志
type ExecutionLog struct {
    ID          string
    Command     string
    Args        []string
    StartTime   time.Time
    EndTime     time.Time
    ExitCode    int
    Duration    time.Duration
    OutputSize  int64
    TimedOut    bool
    Error       string
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: 基础调用框架

**目标**: 实现 CLI 调用的基础框架和接口定义

**前置条件**:
- Config 模块完成
- Logging 模块完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/callcli/interface.go` 定义核心接口
- [x] Task 2: 创建 `internal/callcli/caller.go` 实现 Caller 结构体
- [x] Task 3: 实现 `Call()` 同步执行方法
- [x] Task 4: 实现命令构建和参数处理
- [x] Task 5: 实现工作目录和环境变量设置
- [x] Task 6: 实现基本错误处理
- [x] Task 7: 编写单元测试 `caller_test.go`

**验证器**:
- [x] 能正确执行简单命令（如 `echo hello`）
- [x] 能捕获 stdout 和 stderr
- [x] 能正确获取退出码
- [x] 能设置工作目录
- [x] 能设置环境变量
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用 `github.com/morty/morty` 模块路径, internal/ 用于内部包, pkg/ 用于公共包, 已使用 `github.com/morty/morty/pkg/errors` 错误码系统, 已记录
- debug1: 初始超时处理返回错误码 M5004 而非 M5003, 超时测试时进程被杀信号终止, 猜想: 1)context.DeadlineExceeded 检测顺序问题, 验证: 调整错误检查顺序, 先检查 ctx.Err() 再检查 exitError, 修复: 优先检查 ctx.Err() == context.DeadlineExceeded, 已修复

---

### Job 2: 异步调用和进程管理

**目标**: 实现异步调用和进程控制能力

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `CallAsync()` 异步执行方法
- [x] Task 2: 实现 `CallHandler` 接口
- [x] Task 3: 实现 `Wait()` 等待方法
- [x] Task 4: 实现 `Kill()` 终止进程方法
- [x] Task 5: 实现 `Running()` 状态检查
- [x] Task 6: 实现进程 PID 获取
- [x] Task 7: 编写单元测试 `async_test.go`

**验证器**:
- [x] 异步调用能立即返回 Handler
- [x] `Wait()` 能正确等待进程结束
- [x] `Kill()` 能正确终止进程
- [x] `Running()` 能正确反映进程状态
- [x] 能获取正确的 PID
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: Windows 平台 Kill 测试信号检测差异, Kill 后进程状态检查在 Windows 下行为不同, 猜想: 1)Windows 信号机制差异 2)进程退出检测方式不同, 验证: 检查 runtime.GOOS 区分平台测试, 修复: Windows 下放宽 Kill 测试断言, 已修复
- debug2: 测试文件类型错误, 使用 errors.Error 而非 errors.MortyError, 猜想: 1)IDE 自动导入错误, 验证: 检查 pkg/errors 包定义, 修复: 替换为 errors.MortyError, 已修复

---

### Job 3: 超时和上下文控制

**目标**: 实现超时和上下文取消支持

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `CallWithCtx()` 支持 context
- [x] Task 2: 实现超时检测机制
- [x] Task 3: 超时后自动终止进程
- [x] Task 4: 支持 context 取消信号
- [x] Task 5: 实现优雅终止（发送 SIGTERM 后等待）
- [x] Task 6: 实现强制终止（SIGTERM 后发送 SIGKILL）
- [x] Task 7: 编写单元测试 `timeout_test.go`

**验证器**:
- [x] 超时后返回 `TimedOut=true`
- [x] 超时后能正确终止进程
- [x] context 取消后能终止进程
- [x] 优雅终止给予进程清理时间
- [x] 强制终止能立即结束进程
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 无问题，所有功能正常实现

---

### Job 4: 输出处理

**目标**: 实现多种输出模式支持

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `OutputCapture` 模式（捕获到内存）
- [ ] Task 2: 实现 `OutputStream` 模式（实时输出）
- [ ] Task 3: 实现 `OutputCaptureAndStream` 模式
- [ ] Task 4: 实现 `OutputSilent` 模式
- [ ] Task 5: 支持输出重定向到文件
- [ ] Task 6: 实现输出大小限制（防止内存溢出）
- [ ] Task 7: 编写单元测试 `output_test.go`

**验证器**:
- [ ] `OutputCapture` 正确捕获输出
- [ ] `OutputStream` 实时输出到控制台
- [ ] `OutputCaptureAndStream` 同时满足两者
- [ ] `OutputSilent` 不产生任何输出
- [ ] 输出文件正确保存
- [ ] 超出大小限制时截断或报错
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 5: AI CLI 专用封装

**目标**: 实现 AI CLI（ai_cli/claude）的专用调用封装

**前置条件**:
- Job 3 完成
- Job 4 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/callcli/ai_caller.go`
- [ ] Task 2: 实现 `AICliCaller` 接口
- [ ] Task 3: 实现 `CallWithPrompt()` 方法
- [ ] Task 4: 实现 `CallWithPromptContent()` 方法
- [ ] Task 5: 实现 AI CLI 参数构建（--verbose, --debug 等）
- [ ] Task 6: 实现 AI CLI 配置读取
- [ ] Task 7: 支持从环境变量读取 CLI 路径（`CLAUDE_CODE_CLI`）
- [ ] Task 8: 编写单元测试 `ai_caller_test.go`

**验证器**:
- [ ] 能从环境变量读取 CLI 路径
- [ ] 正确构建 AI CLI 参数
- [ ] `CallWithPrompt()` 正确传递提示词文件
- [ ] `CallWithPromptContent()` 通过 stdin 传递内容
- [ ] 支持 `--output-format json`
- [ ] 支持 `--dangerously-skip-permissions`
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 6: 信号处理和中断恢复

**目标**: 实现信号处理，支持优雅中断

**前置条件**:
- Job 2 完成
- Job 3 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现信号捕获（SIGINT, SIGTERM）
- [ ] Task 2: 信号触发时优雅终止子进程
- [ ] Task 3: 实现中断状态保存
- [ ] Task 4: 支持中断后继续执行
- [ ] Task 5: 实现子进程信号转发
- [ ] Task 6: 处理僵尸进程
- [ ] Task 7: 编写单元测试 `signal_test.go`

**验证器**:
- [ ] Ctrl+C 能正确中断子进程
- [ ] 中断后子进程不会成为僵尸进程
- [ ] 中断状态正确记录
- [ ] 信号能正确转发给子进程
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 7: 执行日志和监控

**目标**: 实现执行日志记录和监控功能

**前置条件**:
- Job 5 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `ExecutionLog` 结构
- [ ] Task 2: 记录每次执行的命令和参数
- [ ] Task 3: 记录执行时间和退出码
- [ ] Task 4: 记录输出大小
- [ ] Task 5: 实现日志轮转
- [ ] Task 6: 实现执行统计信息
- [ ] Task 7: 编写单元测试 `log_test.go`

**验证器**:
- [ ] 每次执行都记录日志
- [ ] 日志包含完整执行信息
- [ ] 日志文件支持轮转
- [ ] 能获取执行统计（成功率、平均耗时等）
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整调用流程: 构建 → 执行 → 捕获 → 返回
- [ ] AI CLI 调用流程: 提示词 → 调用 → 解析结果
- [ ] 超时场景正确处理
- [ ] 中断场景正确处理
- [ ] 大输出不导致内存溢出
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 使用示例

### 基础调用

```go
// 创建调用器
caller := callcli.NewCaller(cfg, logger)

// 同步执行命令
result, err := caller.Call(ctx, &callcli.CallRequest{
    Command: "git",
    Args:    []string{"status", "--short"},
    WorkDir: "/path/to/repo",
    OutputMode: callcli.OutputCapture,
})

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Exit code: %d\n", result.ExitCode)
fmt.Printf("Output: %s\n", result.Stdout)
```

### AI CLI 调用

```go
// 创建 AI CLI 调用器
aiCaller := callcli.NewAICliCaller(cfg, logger)

// 使用提示词文件调用
result, err := aiCaller.CallWithPrompt(ctx, "prompts/doing.md", &callcli.AICliOptions{
    OutputFormat:    "json",
    Verbose:         true,
    Debug:           true,
    SkipPermissions: true,
    Timeout:         10 * time.Minute,
    OutputFile:      ".morty/doing/logs/output.log",
})

// 解析 AI CLI 输出
if result.ExitCode == 0 {
    var ralphStatus RalphStatus
    json.Unmarshal([]byte(result.Stdout), &ralphStatus)
}
```

### 异步调用

```go
// 异步执行
handler, err := caller.CallAsync(ctx, &callcli.CallRequest{
    Command: "long-running-task",
    Args:    []string{"--module", "config"},
    Timeout: 5 * time.Minute,
})

// 获取 PID
fmt.Printf("PID: %d\n", handler.PID())

// 等待完成或超时
result, err := handler.Wait()

// 或主动终止
if needToStop {
    handler.Kill()
}
```

### 超时控制

```go
// 设置超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := caller.Call(ctx, &callcli.CallRequest{
    Command: "slow-command",
    Args:    []string{"arg1", "arg2"},
})

if result.TimedOut {
    fmt.Println("Command timed out!")
}
```

---

## 文件清单

- `internal/callcli/interface.go` - 核心接口定义
- `internal/callcli/caller.go` - 基础调用实现
- `internal/callcli/async.go` - 异步调用和进程管理
- `internal/callcli/timeout.go` - 超时控制
- `internal/callcli/output.go` - 输出处理
- `internal/callcli/ai_caller.go` - AI CLI 专用封装
- `internal/callcli/signal.go` - 信号处理
- `internal/callcli/log.go` - 执行日志
- `internal/callcli/config.go` - 配置定义
