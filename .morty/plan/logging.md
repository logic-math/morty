# Plan: logging

## 模块概述

**模块职责**: 提供统一的日志管理系统，支持多级别日志、结构化日志输出和日志轮转，解决原有日志分散在多个文件的问题。

**对应 Research**: [重构机会] 日志系统分散，建议统一日志管理

**依赖模块**: config

**被依赖模块**: git_manager, research, plan, doing, monitor, cli

## 接口定义

### 输入接口
- 配置项: `log_level`, `log_dir`, `log_format`, `log_max_size`, `log_max_files`
- 日志写入请求: 级别、消息、上下文

### 输出接口
- `log(level, message, context)`: 写入日志
- `log_info()`, `log_warn()`, `log_error()`, `log_success()`, `log_loop()`: 快捷方法
- `log_debug()`: 调试日志
- 日志文件: `.morty/logs/morty.log`, `.morty/logs/doing.log` 等

## 数据模型

### 日志级别
```
DEBUG < INFO < WARN < ERROR < SUCCESS < LOOP
```

### 日志格式
```
# 文本格式 (默认)
[2026-02-20 14:30:00] [INFO] [module:job] 消息内容

# JSON 格式 (可选)
{
  "timestamp": "2026-02-20T14:30:00Z",
  "level": "INFO",
  "module": "doing",
  "job": "job_1",
  "message": "任务开始执行",
  "context": { ... }
}
```

### 日志文件结构
```
.morty/logs/
├── morty.log          # 主日志文件
├── morty.log.1        # 轮转历史
├── doing.log          # doing 模式专用日志
├── doing.log.1
└── jobs/
    ├── config_job1.log    # 各 Job 独立日志
    ├── doing_job1.log
    └── ...
```

## Jobs (Loop 块列表)

---

### Job 1: 日志系统核心框架

**目标**: 建立日志系统的核心框架，支持多级别日志和文件输出

**前置条件**: config 模块 Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 创建 `lib/logging.sh` 日志管理模块
- [ ] 实现 `log()` 核心函数，支持级别过滤
- [ ] 实现日志文件写入和自动创建目录
- [ ] 实现日志级别快捷方法（log_info, log_warn 等）
- [ ] 实现日志格式化和时间戳

**验证器**:
- 调用 `log_info "测试消息"` 后，日志文件应包含带时间戳的 INFO 级别消息
- 当日志级别设置为 WARN 时，INFO 级别的消息不应被写入
- 日志目录不存在时应自动创建
- 并发写入日志不应导致内容交错或丢失
- 单条日志写入延迟应小于 10ms

**调试日志**:
- 无

---

### Job 2: 日志轮转和归档

**目标**: 实现日志轮转机制，防止日志文件无限增长

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现日志文件大小检查
- [ ] 实现日志轮转（当前日志 -> morty.log.1 -> morty.log.2）
- [ ] 实现最大保留文件数限制
- [ ] 支持按日期归档（可选）
- [ ] 实现日志压缩（对旧日志）

**验证器**:
- 当日志文件超过配置的大小限制（默认 10MB）时，应自动触发轮转
- 轮转后新日志应写入新的空文件，旧日志重命名为 .log.1
- 当历史日志文件数超过限制时，最旧的日志应被删除
- 轮转过程中不应丢失日志消息
- 压缩后的日志文件大小应小于原文件的 20%

**调试日志**:
- 无

---

### Job 3: Job 级独立日志

**目标**: 支持为每个 Job 创建独立的日志文件，便于调试

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `log_job_start(module, job)` 创建 Job 日志上下文
- [ ] 实现 `log_job_end()` 关闭 Job 日志上下文
- [ ] 实现 `log_job(message)` 写入 Job 独立日志
- [ ] 在 Job 日志中自动记录开始时间、结束时间、执行时长
- [ ] 支持 Job 日志与主日志同时写入

**验证器**:
- 调用 `log_job_start "doing" "job_1"` 后，应创建 `.morty/logs/jobs/doing_job1.log`
- Job 执行期间的所有日志应同时写入主日志和 Job 独立日志
- Job 独立日志应包含 Job 开始和结束的时间戳
- Job 失败时应记录错误详情和堆栈信息（如可用）
- Job 日志文件大小应可通过配置限制

**调试日志**:
- 无

---

### Job 4: 结构化日志支持

**目标**: 支持 JSON 格式的结构化日志，便于日志分析和监控集成

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 JSON 格式日志输出
- [ ] 支持上下文数据自动序列化为 JSON 字段
- [ ] 添加 `log_structured()` 函数用于机器可读日志
- [ ] 实现日志格式切换（文本/JSON）
- [ ] 确保特殊字符正确转义

**验证器**:
- 当配置 `log_format: json` 时，日志输出应为有效的 JSON 格式
- JSON 日志应包含 timestamp, level, module, message 字段
- 上下文对象应被正确序列化为 JSON 字段
- 包含特殊字符的消息应被正确转义
- JSON 日志应可被标准日志分析工具解析（如 jq）

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 所有其他模块可以正常导入和使用日志功能
- 日志级别动态调整可以实时生效
- 大量日志写入（1000条/秒）不会导致性能问题
- 日志轮转不会丢失消息
- 磁盘满时可以优雅降级（如输出到 stderr）

---

## 待实现方法签名

```bash
# lib/logging.sh

# 核心日志函数
log(level, message, context="")
log_debug(message, context="")
log_info(message, context="")
log_warn(message, context="")
log_error(message, context="")
log_success(message, context="")
log_loop(message, context="")

# 结构化日志
log_structured(level, data)

# Job 日志
log_job_start(module, job_name)
log_job_end(status="completed")
log_job(message, level="INFO")
log_job_debug(message)

# 日志配置
log_set_level(level)
log_get_level()
log_set_format(format)  # text/json

# 内部函数
_log_write(level, message, context)
_log_rotate_if_needed()
_log_archive_old_logs()
```
