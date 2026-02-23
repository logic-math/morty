# Plan: Errors 错误码定义

## 模块概述

**模块职责**: 定义 Morty 统一的错误码体系和错误处理规范，确保错误信息一致、可追踪、可处理。

**对应 Research**: 无

**依赖模块**: 无

**被依赖模块**: 所有模块

---

## 错误码设计原则

1. **唯一性**: 每个错误码唯一标识一种错误类型
2. **层级性**: 按模块和功能分层，便于定位
3. **可读性**: 错误码命名清晰，附带人类可读的错误信息
4. **可追踪**: 支持错误链和堆栈信息
5. **可恢复**: 区分可恢复错误和致命错误

---

## 错误码体系

### 错误码格式

```
M[模块][功能][序号]

示例:
- M0001 - 通用成功
- M1001 - Config 模块错误
- M2001 - State 模块错误
```

### 模块前缀

| 模块 | 前缀 | 范围 |
|------|------|------|
| 通用 | M0 | 0-999 |
| Config | M1 | 1000-1999 |
| State | M2 | 2000-2999 |
| Git | M3 | 3000-3999 |
| Parser | M4 | 4000-4999 |
| Call CLI | M5 | 5000-5999 |
| CLI | M6 | 6000-6999 |
| Executor | M7 | 7000-7999 |
| Cmd (命令) | M8 | 8000-8999 |
| Deploy | M9 | 9000-9999 |

---

## 错误码定义

### 通用错误 (M0xxx)

| 错误码 | 名称 | 说明 | HTTP 状态码类比 |
|--------|------|------|-----------------|
| M0000 | Success | 成功 | 200 |
| M0001 | ErrGeneral | 通用错误 | 500 |
| M0002 | ErrInvalidArgs | 参数错误 | 400 |
| M0003 | ErrNotFound | 资源未找到 | 404 |
| M0004 | ErrAlreadyExists | 资源已存在 | 409 |
| M0005 | ErrPermission | 权限不足 | 403 |
| M0006 | ErrTimeout | 操作超时 | 504 |
| M0007 | ErrCancelled | 操作被取消 | 499 |
| M0008 | ErrInterrupted | 被中断 | - |
| M0009 | ErrNotImplemented | 未实现 | 501 |

### Config 错误 (M1xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M1001 | ErrConfigNotFound | 配置文件不存在 |
| M1002 | ErrConfigParse | 配置文件解析失败 |
| M1003 | ErrConfigInvalid | 配置值无效 |
| M1004 | ErrConfigVersion | 配置版本不兼容 |
| M1005 | ErrConfigRequired | 缺少必需配置项 |

### State 错误 (M2xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M2001 | ErrStateNotFound | 状态文件不存在 |
| M2002 | ErrStateParse | 状态文件解析失败 |
| M2003 | ErrStateCorrupted | 状态文件损坏 |
| M2004 | ErrStateTransition | 无效的状态转换 |
| M2005 | ErrStateModuleNotFound | 模块不存在 |
| M2006 | ErrStateJobNotFound | Job 不存在 |

### Git 错误 (M3xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M3001 | ErrGitNotRepo | 非 Git 仓库 |
| M3002 | ErrGitCommit | Git 提交失败 |
| M3003 | ErrGitStatus | 获取 Git 状态失败 |
| M3004 | ErrGitDirtyWorktree | 工作区不干净 |
| M3005 | ErrGitNoCommits | 没有提交历史 |

### Parser 错误 (M4xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M4001 | ErrParserNotFound | 解析器未找到 |
| M4002 | ErrParserFileNotFound | Plan 文件不存在 |
| M4003 | ErrParserParse | 解析失败 |
| M4004 | ErrParserInvalidFormat | 格式无效 |
| M4005 | ErrParserNoJobs | 未找到 Jobs |

### Call CLI 错误 (M5xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M5001 | ErrCallCLINotFound | AI CLI 命令不存在 |
| M5002 | ErrCallCLIExec | 执行失败 |
| M5003 | ErrCallCLITimeout | 执行超时 |
| M5004 | ErrCallCLIKilled | 进程被终止 |
| M5005 | ErrCallCLIOutput | 输出读取失败 |
| M5006 | ErrCallCLISignal | 信号处理失败 |

### CLI 错误 (M6xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M6001 | ErrCLIUnknownCommand | 未知命令 |
| M6002 | ErrCLIInvalidFlag | 无效选项 |
| M6003 | ErrCLIMissingArg | 缺少参数 |
| M6004 | ErrCLIFlagConflict | 选项冲突 |

### Executor 错误 (M7xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M7001 | ErrExecutorPrecondition | 前置条件不满足 |
| M7002 | ErrExecutorJobFailed | Job 执行失败 |
| M7003 | ErrExecutorMaxRetry | 超过最大重试次数 |
| M7004 | ErrExecutorPromptBuild | 提示词构建失败 |
| M7005 | ErrExecutorResultParse | 结果解析失败 |
| M7006 | ErrExecutorBlocked | Job 被阻塞 |

### Cmd 错误 (M8xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M8001 | ErrCmdPlanNotFound | Plan 目录不存在 |
| M8002 | ErrCmdResearchNotFound | Research 目录不存在 |
| M8003 | ErrCmdDoingRunning | Doing 正在运行 |
| M8004 | ErrCmdNoPendingJobs | 没有待执行的 Jobs |
| M8005 | ErrCmdModuleNotFound | 指定模块不存在 |
| M8006 | ErrCmdJobNotFound | 指定 Job 不存在 |
| M8007 | ErrCmdResetFailed | 回滚失败 |
| M8008 | ErrCmdStatFailed | 状态查询失败 |

### Deploy 错误 (M9xxx)

| 错误码 | 名称 | 说明 |
|--------|------|------|
| M9001 | ErrDeployBuild | 构建失败 |
| M9002 | ErrDeployInstall | 安装失败 |
| M9003 | ErrDeployUninstall | 卸载失败 |
| M9004 | ErrDeployUpgrade | 升级失败 |
| M9005 | ErrDeployVersion | 版本检查失败 |

---

## Go 错误定义

```go
package errors

import "errors"

// MortyError Morty 错误结构
type MortyError struct {
    Code    string
    Message string
    Module  string
    Cause   error
    Details map[string]interface{}
}

func (e *MortyError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *MortyError) Unwrap() error {
    return e.Cause
}

// 通用错误
var (
    ErrSuccess         = &MortyError{Code: "M0000", Message: "success"}
    ErrGeneral         = &MortyError{Code: "M0001", Message: "general error"}
    ErrInvalidArgs     = &MortyError{Code: "M0002", Message: "invalid arguments"}
    ErrNotFound        = &MortyError{Code: "M0003", Message: "not found"}
    ErrAlreadyExists   = &MortyError{Code: "M0004", Message: "already exists"}
    ErrPermission      = &MortyError{Code: "M0005", Message: "permission denied"}
    ErrTimeout         = &MortyError{Code: "M0006", Message: "timeout"}
    ErrCancelled       = &MortyError{Code: "M0007", Message: "cancelled"}
    ErrInterrupted     = &MortyError{Code: "M0008", Message: "interrupted"}
    ErrNotImplemented  = &MortyError{Code: "M0009", Message: "not implemented"}
)

// Config 错误
var (
    ErrConfigNotFound  = &MortyError{Code: "M1001", Message: "config file not found", Module: "config"}
    ErrConfigParse     = &MortyError{Code: "M1002", Message: "config parse failed", Module: "config"}
    ErrConfigInvalid   = &MortyError{Code: "M1003", Message: "invalid config value", Module: "config"}
    ErrConfigVersion   = &MortyError{Code: "M1004", Message: "config version mismatch", Module: "config"}
    ErrConfigRequired  = &MortyError{Code: "M1005", Message: "required config missing", Module: "config"}
)

// State 错误
var (
    ErrStateNotFound        = &MortyError{Code: "M2001", Message: "state file not found", Module: "state"}
    ErrStateParse           = &MortyError{Code: "M2002", Message: "state parse failed", Module: "state"}
    ErrStateCorrupted       = &MortyError{Code: "M2003", Message: "state file corrupted", Module: "state"}
    ErrStateTransition      = &MortyError{Code: "M2004", Message: "invalid state transition", Module: "state"}
    ErrStateModuleNotFound  = &MortyError{Code: "M2005", Message: "module not found in state", Module: "state"}
    ErrStateJobNotFound     = &MortyError{Code: "M2006", Message: "job not found in state", Module: "state"}
)

// Git 错误
var (
    ErrGitNotRepo         = &MortyError{Code: "M3001", Message: "not a git repository", Module: "git"}
    ErrGitCommit          = &MortyError{Code: "M3002", Message: "git commit failed", Module: "git"}
    ErrGitStatus          = &MortyError{Code: "M3003", Message: "git status failed", Module: "git"}
    ErrGitDirtyWorktree   = &MortyError{Code: "M3004", Message: "worktree is dirty", Module: "git"}
    ErrGitNoCommits       = &MortyError{Code: "M3005", Message: "no commits found", Module: "git"}
)

// 更多错误定义...

// New 创建新的 MortyError
func New(code, message, module string) *MortyError {
    return &MortyError{
        Code:    code,
        Message: message,
        Module:  module,
        Details: make(map[string]interface{}),
    }
}

// Wrap 包装错误
func Wrap(err error, code, message, module string) *MortyError {
    return &MortyError{
        Code:    code,
        Message: message,
        Module:  module,
        Cause:   err,
        Details: make(map[string]interface{}),
    }
}

// WithDetail 添加详情
func (e *MortyError) WithDetail(key string, value interface{}) *MortyError {
    e.Details[key] = value
    return e
}

// Is 判断错误类型
func Is(err error, target *MortyError) bool {
    if err == nil || target == nil {
        return err == target
    }

    var me *MortyError
    if errors.As(err, &me) {
        return me.Code == target.Code
    }
    return false
}
```

---

## 使用示例

### 创建错误

```go
// 简单错误
err := errors.New(errors.ErrConfigNotFound)

// 包装错误
err := errors.Wrap(io.ErrNotExist, "M1001", "config file not found", "config")

// 带详情
err := errors.New(errors.ErrStateParse).
    WithDetail("file", ".morty/status.json").
    WithDetail("line", 42)
```

### 错误处理

```go
if err != nil {
    var me *errors.MortyError
    if errors.As(err, &me) {
        switch me.Code {
        case "M1001": // Config not found
            // 创建默认配置
        case "M2001": // State not found
            // 初始化新状态
        case "M0008": // Interrupted
            // 保存状态并退出
        default:
            return err
        }
    }
}
```

### 错误检查

```go
if errors.Is(err, errors.ErrNotFound) {
    // 处理未找到错误
}

if errors.Is(err, errors.ErrTimeout) {
    // 处理超时错误
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: 错误码定义实现

**目标**: 实现统一的错误码体系和错误结构

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] Task 1: 定义 MortyError 结构体
- [ ] Task 2: 定义所有错误码常量
- [ ] Task 3: 实现 Error() 和 Unwrap() 方法
- [ ] Task 4: 实现 New() 和 Wrap() 构造器
- [ ] Task 5: 实现 WithDetail() 和 Is() 方法
- [ ] Task 6: 实现错误链支持
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 所有错误码定义完整
- [ ] 错误信息格式正确
- [ ] 错误链支持正常工作
- [ ] Is() 方法正确判断错误类型
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

## 文件清单

- `pkg/errors/errors.go` - 错误码定义
- `plan/errors.md` - 本文件
