# Plan: Logging

## 模块概述

**模块职责**: 实现结构化日志系统，支持多级别日志、日志轮转和 Job 上下文追踪

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.3 节 Logging 模块接口定义
- `morty-project-research.md` 第 3.8 节通用工具分析 (原 logging.sh)

**现有实现参考**:
- 原 Shell 版本: `lib/logging.sh`，支持多级别、JSON/文本格式、日志轮转

**依赖模块**: Config (获取日志路径配置)

**被依赖模块**: Executor, State

---

## 接口定义

### 输入接口
- 日志消息和属性
- 日志级别设置
- Job 上下文 (module, job)

### 输出接口
- `Logger` 接口实现
- `Rotator` 接口实现
- 日志文件写入

---

## 数据模型

```go
// Level 日志级别
type Level int
const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
    SUCCESS
    LOOP
)

// Logger 日志接口
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

// Rotator 日志轮转接口
type Rotator interface {
    Rotate(logFile string) error
    ShouldRotate(logFile string) bool
}

// JobLogger Job 上下文日志
type JobLogger struct {
    Module string
    Job    string
    logger Logger
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: Logger 接口与 slog 适配器实现

**目标**: 实现 Logger 接口，基于 slog 标准库

**前置条件**:
- Config 模块完成 (获取日志配置)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/logging/logger.go` 定义 Logger 接口
- [ ] Task 2: 创建 `internal/logging/slog_adapter.go` 实现 slog 适配器
- [ ] Task 3: 实现所有日志级别方法 (Debug, Info, Warn, Error, Success, Loop)
- [ ] Task 4: 支持结构化属性 (attrs)
- [ ] Task 5: 实现 `WithContext` 添加上下文信息
- [ ] Task 6: 实现 `WithJob` 添加 Job 上下文
- [ ] Task 7: 实现 `SetLevel` 和 `GetLevel`
- [ ] Task 8: 编写单元测试 `slog_adapter_test.go`

**验证器**:
- [ ] 各日志级别输出正确 (DEBUG, INFO, WARN, ERROR, SUCCESS, LOOP)
- [ ] 结构化属性正确输出为 JSON
- [ ] `WithContext` 返回的 Logger 包含上下文信息
- [ ] `WithJob` 返回的 Logger 包含 Job 信息
- [ ] 设置日志级别后低于该级别的日志不输出
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 2: 日志轮转器实现

**目标**: 实现基于文件大小的日志轮转功能

**前置条件**:
- Job 1 完成 (Logger 基础)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/logging/rotator.go` 文件结构
- [ ] Task 2: 实现 `ShouldRotate(logFile string) bool` 检查文件大小
- [ ] Task 3: 实现 `Rotate(logFile string) error` 执行轮转
- [ ] Task 4: 实现标准轮转模式 (morty.log → morty.log.1 → ...)
- [ ] Task 5: 实现旧日志 gzip 压缩 (保留最近 5 个)
- [ ] Task 6: 实现清理过期日志文件
- [ ] Task 7: 编写单元测试 `rotator_test.go`

**验证器**:
- [ ] 小文件 (50 bytes < max 100 bytes) 不触发轮转
- [ ] 大文件 (101 bytes > max 100 bytes) 触发轮转
- [ ] 轮转后原文件清空，.1 文件包含原内容
- [ ] 多次轮转后 .2+ 文件被 gzip 压缩
- [ ] 超过最大保留数的旧日志被删除
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: Job 上下文日志实现

**目标**: 实现支持 Job 上下文的日志记录

**前置条件**:
- Job 1 完成 (Logger 基础)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/logging/job_logger.go` 文件结构
- [ ] Task 2: 实现 `JobLogger` 结构体，包含 module 和 job 字段
- [ ] Task 3: 实现 Job 开始/结束日志自动记录
- [ ] Task 4: 实现 Task 级别的日志记录
- [ ] Task 5: 支持从 JobLogger 获取标准 Logger 接口
- [ ] Task 6: 实现日志文件按 Job 分离 (可选)
- [ ] Task 7: 编写单元测试 `job_logger_test.go`

**验证器**:
- [ ] JobLogger 正确记录 module 和 job 信息
- [ ] Job 开始日志包含模块名、Job 名、开始时间
- [ ] Job 结束日志包含执行结果、耗时
- [ ] Task 日志包含 Task 编号和描述
- [ ] 所有日志条目包含一致的 Job 上下文
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 4: 日志格式与输出配置

**目标**: 实现文本和 JSON 两种日志格式，支持控制台和文件输出

**前置条件**:
- Job 1, Job 2 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/logging/format.go` 定义格式相关类型
- [ ] Task 2: 实现文本格式输出 (人类可读)
- [ ] Task 3: 实现 JSON 格式输出 (结构化)
- [ ] Task 4: 实现多输出目标 (控制台 + 文件)
- [ ] Task 5: 实现根据环境自动选择格式 (开发=文本, 生产=JSON)
- [ ] Task 6: 配置文件支持设置日志格式和级别
- [ ] Task 7: 编写单元测试 `format_test.go`

**验证器**:
- [ ] 文本格式输出包含时间、级别、消息、属性
- [ ] JSON 格式输出是有效的 JSON 对象
- [ ] 同时输出到控制台和文件
- [ ] 配置文件正确控制日志行为
- [ ] 开发环境默认文本格式，生产环境默认 JSON 格式
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的日志生命周期: 创建 Logger → 记录日志 → 轮转 → 清理
- [ ] Job 上下文正确传递到所有日志条目
- [ ] 多输出目标同时工作
- [ ] 轮转时不丢失日志
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充
