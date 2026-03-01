# Event Log Format - 事件日志格式

## 概述

从此版本开始，morty 会自动将 Claude Code 返回的 JSON 事件流格式化为易读的文本格式，每个事件一行，包含：
- 事件编号
- 时间戳
- 事件类型
- 内容摘要

## 日志格式示例

### 新格式（Text - 易读）

```
=== Claude Code Event Stream ===
Timestamp: 2026-02-28 17:45:30

[0001] 2026-02-28 17:45:30 | system/init          | Session initialized
[0002] 2026-02-28 17:45:31 | assistant            | Text: I'll execute the job... | Tokens(in:1234 out:56 cache:890)
[0003] 2026-02-28 17:45:32 | assistant            | Tools: [Read] | Tokens(in:234 out:12 cache:456)
[0004] 2026-02-28 17:45:33 | user                 | Tool results: [Read]
[0005] 2026-02-28 17:45:34 | assistant            | Text: Based on the file content... | Tools: [Write, Bash] | Tokens(in:345 out:78 cache:567)
[0006] 2026-02-28 17:45:35 | user                 | Tool results: [Write, Bash]
[0007] 2026-02-28 17:45:36 | assistant            | Text: Task completed successfully | Tokens(in:123 out:45 cache:234)
[0008] 2026-02-28 17:45:37 | result               | Execution completed: end_turn

=== Total Events: 8 ===

==========================================
Exit Code: 0
Completed: 2026-02-28 17:45:37
```

### 旧格式（JSON - 难读）

```
[{"type":"system","subtype":"init","cwd":"/path","session_id":"abc123",...},{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll execute..."}],"usage":{"input_tokens":1234,"output_tokens":56}}},...]
```

## 事件类型说明

### system 事件
- `system/init` - 会话初始化
- `system/error` - 系统错误

### assistant 事件
Claude AI 的消息，包含：
- **Text** - AI 的文本回复
- **Tools** - AI 调用的工具（Read, Write, Bash, Edit 等）
- **Tokens** - Token 使用统计
  - `in` - 输入 tokens
  - `out` - 输出 tokens
  - `cache` - 缓存命中 tokens

### user 事件
工具执行结果返回给 AI

### result 事件
执行完成事件

## 实现细节

### 格式化器位置
- `internal/executor/event_formatter.go` - 事件格式化器实现
- `internal/executor/engine.go:writeJobLog()` - 自动检测并格式化 JSON 事件流

### 格式化逻辑

```go
// 检测 stdout 是否为 JSON 事件流
if stdout != "" && strings.HasPrefix(strings.TrimSpace(stdout), "[") {
    // 格式化为文本
    formatter := NewEventFormatter(logFile)
    if err := formatter.FormatEventStream(stdout); err != nil {
        // 格式化失败则使用原始输出
        fmt.Fprintf(logFile, "\n=== STDOUT (Raw) ===\n")
        fmt.Fprintf(logFile, "%s\n", stdout)
    }
} else {
    // 非 JSON，直接输出
    fmt.Fprintf(logFile, "\n=== STDOUT ===\n")
    fmt.Fprintf(logFile, "%s\n", stdout)
}
```

### 事件摘要提取

每种事件类型都有专门的摘要提取逻辑：

#### Assistant 事件摘要
```go
// 提取文本内容（截断到 100 字符）
// 提取工具调用列表
// 添加 token 使用统计
// 格式: "Text: ... | Tools: [Read, Write] | Tokens(in:123 out:45 cache:67)"
```

#### System 事件摘要
```go
// system/init -> "Session initialized"
// system/error -> "System error occurred"
```

#### User 事件摘要
```go
// 提取工具结果列表
// 格式: "Tool results: [Read, Write]"
```

## 使用场景

### 1. 实时监控
查看 AI 执行过程中调用了哪些工具，处理了多少 tokens：
```bash
tail -f .morty/logs/module_job_timestamp.log
```

### 2. 审计追踪
检查特定 job 的完整执行历史：
```bash
cat .morty/logs/测试模块_创建文件_20260228_163000.log
```

### 3. 性能分析
统计 token 使用情况：
```bash
grep "Tokens(in:" .morty/logs/module_job.log | \
  awk -F'in:|out:|cache:' '{in+=$2; out+=$3; cache+=$4} END {print "Total - In:"in" Out:"out" Cache:"cache}'
```

### 4. 工具使用统计
查看最常用的工具：
```bash
grep "Tools:" .morty/logs/*.log | \
  grep -oP 'Tools: \[\K[^\]]+' | \
  tr ',' '\n' | \
  sort | uniq -c | sort -rn
```

## 格式化现有日志

如果你有旧版本生成的 JSON 格式日志，可以使用脚本转换：

```bash
# 格式化所有日志
bash scripts/format_existing_logs.sh .morty/logs

# 格式化单个日志
bash scripts/format_existing_logs.sh .morty/logs/specific_job.log
```

格式化后的日志文件会保存为 `{original}.formatted`。

## 配置选项

目前格式化是自动的，无需配置。未来版本可能会添加配置选项：

```json
{
  "logging": {
    "job_log_format": "text",  // "text" 或 "json"
    "include_timestamps": true,
    "max_text_length": 100,
    "show_token_usage": true
  }
}
```

## 故障排除

### 问题 1: 日志文件仍然是 JSON 格式

**原因**: 使用的是旧版本 morty

**解决**:
```bash
morty -version  # 检查版本
# 如果版本早于 2026-02-28 的构建，重新安装
bash scripts/install.sh --force
```

### 问题 2: 格式化失败，显示原始 JSON

**原因**: JSON 格式不标准或损坏

**检查**:
```bash
# 检查 JSON 是否有效
head -1 .morty/logs/job.log | jq . > /dev/null
```

**解决**: 这是预期行为，morty 会在格式化失败时回退到原始输出

### 问题 3: 缺少部分事件

**原因**: 日志文件被截断或写入不完整

**检查**:
```bash
# 检查文件大小
ls -lh .morty/logs/job.log

# 检查是否有错误日志
grep "ERROR" .morty/logs/job.log
```

## 最佳实践

1. **定期清理旧日志**
   ```bash
   # 删除 7 天前的日志
   find .morty/logs -name "*.log" -mtime +7 -delete
   ```

2. **归档重要日志**
   ```bash
   # 归档特定 module 的日志
   tar -czf logs_archive_$(date +%Y%m%d).tar.gz .morty/logs/module_*
   ```

3. **监控 token 使用**
   ```bash
   # 创建监控脚本
   watch -n 5 'grep "Tokens(in:" .morty/logs/*.log | tail -10'
   ```

4. **提取错误事件**
   ```bash
   # 查找所有错误事件
   grep -h "ERROR\|error\|failed" .morty/logs/*.log
   ```

## 参考

- 事件格式化器实现: `internal/executor/event_formatter.go`
- 日志写入逻辑: `internal/executor/engine.go:writeJobLog()`
- Claude Code 事件流文档: https://docs.anthropic.com/claude/docs/events
