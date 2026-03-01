# Conversation Logging

Morty 自动解析 Claude Code 返回的对话 JSON 数据，并格式化保存到 `.morty/logs` 目录。

## 功能概述

当 Morty 执行任务时，它会：
1. 调用 Claude Code AI CLI
2. 捕获返回的 JSON 对话数据
3. 解析对话内容（用户消息、AI 响应、工具调用等）
4. 格式化并保存为可读的日志文件

## 日志文件格式

### 文件命名

日志文件按照以下格式命名：
```
{module}_{job}_{timestamp}.log
```

例如：
```
测试模块_创建测试文件结构_20260228_150405.log
```

### 日志内容结构

日志文件包含以下部分：

#### 1. 头部信息
```
=== Claude Code Conversation Log ===
Generated: 2026-02-28 15:04:05
Model: claude-opus-4-6
Tokens Used: 5000 / 200000
Total Messages: 12
=====================================
```

#### 2. 对话记录

每条消息包含时间戳、角色和内容：

**用户消息：**
```
[15:04:05] USER:
请帮我创建一个 hello_world.py 文件
```

**AI 响应：**
```
[15:04:10] ASSISTANT:
好的，我来帮你创建 hello_world.py 文件
```

**工具调用：**
```
[15:04:12] TOOL CALL: Write
  Parameters:
  {
    "file_path": "/path/to/hello_world.py",
    "content": "#!/usr/bin/env python3\nprint('Hello World!')"
  }
```

**工具结果：**
```
[15:04:13] TOOL RESULT: Write
  Result: File created successfully at /path/to/hello_world.py
```

**错误信息：**
```
[15:04:15] ERROR:
Failed to execute command: file not found
```

#### 3. 统计信息

日志末尾包含统计摘要：
```
=====================================
=== Statistics ===
=====================================
Message Types:
  - user_message: 3
  - assistant_text: 5
  - tool_call: 8
  - tool_result: 8

Tool Usage:
  - Write: 3
  - Read: 2
  - Bash: 3
```

## 使用方式

### 自动日志记录

在 `doing` 命令执行时，日志会自动记录：

```bash
morty doing
```

日志文件会保存到 `.morty/logs/` 目录。

### 手动解析日志

如果你有 Claude Code 的 JSON 输出文件，可以手动解析：

```go
import "github.com/morty/morty/internal/callcli"

parser := callcli.NewConversationParser(".morty/logs")
logPath, err := parser.ParseAndSave(jsonData, "module_name", "job_name")
```

### 保存格式化 JSON

除了文本日志，你也可以保存格式化的 JSON：

```go
conversation, err := parser.Parse(jsonData)
if err != nil {
    // handle error
}

jsonPath, err := parser.SaveFormattedJSON(conversation, "module_name", "job_name")
```

## Claude Code JSON 格式

Morty 支持解析以下 Claude Code JSON 格式：

### 基本格式

```json
{
  "messages": [
    {
      "role": "user",
      "content": "用户消息内容"
    },
    {
      "role": "assistant",
      "content": "AI 响应内容"
    }
  ],
  "model": "claude-opus-4-6",
  "tokens_used": 5000,
  "tokens_limit": 200000
}
```

### 带工具调用的格式

```json
{
  "messages": [
    {
      "role": "assistant",
      "content": [
        {
          "type": "text",
          "text": "我来读取这个文件"
        }
      ],
      "tool_use": {
        "type": "tool_use",
        "id": "toolu_123",
        "name": "Read",
        "input": {
          "file_path": "test.txt"
        }
      }
    }
  ]
}
```

### 内容块格式

支持多种内容格式：

**字符串格式：**
```json
{
  "role": "user",
  "content": "简单文本消息"
}
```

**数组格式：**
```json
{
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "第一部分"
    },
    {
      "type": "text",
      "text": "第二部分"
    }
  ]
}
```

## 配置选项

### 日志目录

默认日志目录是 `.morty/logs`，可以通过以下方式自定义：

```go
taskRunner := executor.NewTaskRunnerWithConfig(
    logger,
    aiCliCaller,
    timeout,
    "/custom/logs/path",
)
```

### 在代码中使用

在 `task_runner.go` 中使用带日志功能的执行方法：

```go
result, err := taskRunner.RunWithLogging(
    ctx,
    "module_name",
    "job_name",
    "task_description",
    promptContent,
)

if result != nil && result.ConversationLogPath != "" {
    fmt.Printf("Conversation log saved to: %s\n", result.ConversationLogPath)
}
```

## 日志解析器 API

### ConversationParser

主要方法：

#### ParseAndSave
```go
func (cp *ConversationParser) ParseAndSave(
    jsonData string,
    module, job string,
) (string, error)
```
解析 JSON 并保存为文本日志，返回日志文件路径。

#### Parse
```go
func (cp *ConversationParser) Parse(
    jsonData string,
) (*ConversationData, error)
```
解析 JSON 为结构化数据。

#### ExtractLogs
```go
func (cp *ConversationParser) ExtractLogs(
    conversation *ConversationData,
) []FormattedLog
```
从对话数据中提取格式化的日志条目。

#### ParseFromFile
```go
func (cp *ConversationParser) ParseFromFile(
    jsonFile string,
) (*ConversationData, error)
```
从文件读取并解析 JSON。

#### SaveFormattedJSON
```go
func (cp *ConversationParser) SaveFormattedJSON(
    conversation *ConversationData,
    module, job string,
) (string, error)
```
保存格式化的 JSON 文件。

## 数据结构

### ConversationData
```go
type ConversationData struct {
    Messages    []ConversationMessage
    Model       string
    SystemInfo  map[string]interface{}
    Metadata    map[string]interface{}
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    TokensUsed  int
    TokensLimit int
}
```

### ConversationMessage
```go
type ConversationMessage struct {
    Role      string      // "user" or "assistant"
    Content   interface{} // string or array of content blocks
    Timestamp time.Time
    Type      string
    ToolUse   *ToolUseBlock
    ToolCalls []ToolCall
    Metadata  map[string]interface{}
}
```

### FormattedLog
```go
type FormattedLog struct {
    Timestamp   time.Time
    MessageType string // "user_message", "assistant_text", "tool_call", "tool_result", "error"
    Role        string
    Content     string
    ToolName    string
    ToolParams  map[string]interface{}
    ToolResult  string
    Error       string
    Metadata    map[string]interface{}
}
```

## 故障排除

### 日志文件未生成

**可能原因：**
1. Claude Code 没有返回 JSON 数据
2. JSON 格式不符合预期
3. 日志目录权限问题

**解决方法：**
1. 检查 Claude Code 输出格式
2. 查看 Morty 的调试日志
3. 确保 `.morty/logs` 目录可写

### JSON 解析失败

**可能原因：**
1. JSON 格式不完整或损坏
2. 包含特殊字符或转义问题

**解决方法：**
1. 使用 JSON 验证工具检查格式
2. 查看详细错误信息
3. 手动解析并调试

### 日志内容不完整

**可能原因：**
1. 对话被截断
2. 某些消息类型未被识别

**解决方法：**
1. 检查原始 JSON 数据
2. 更新 ConversationParser 以支持新格式
3. 查看统计信息了解缺失的部分

## 示例

### 完整的日志示例

```
=== Claude Code Conversation Log ===
Generated: 2026-02-28 15:04:05
Model: claude-opus-4-6
Tokens Used: 8500 / 200000
Total Messages: 15
=====================================

[15:04:05] USER:
请创建一个 Python Hello World 程序

[15:04:10] ASSISTANT:
好的，我来帮你创建一个 Python Hello World 程序

[15:04:12] TOOL CALL: Write
  Parameters:
  {
    "file_path": "/home/user/hello_world.py",
    "content": "#!/usr/bin/env python3\n\ndef main():\n    print('Hello World!')\n\nif __name__ == '__main__':\n    main()"
  }

[15:04:13] TOOL RESULT: Write
  Result: File created successfully

[15:04:15] ASSISTANT:
文件已创建成功！现在让我运行它来验证

[15:04:17] TOOL CALL: Bash
  Parameters:
  {
    "command": "python3 /home/user/hello_world.py",
    "description": "Run the hello world program"
  }

[15:04:18] TOOL RESULT: Bash
  Result: Hello World!

[15:04:20] ASSISTANT:
完成！程序运行成功，输出了 "Hello World!"

=====================================
=== Statistics ===
=====================================
Message Types:
  - user_message: 1
  - assistant_text: 3
  - tool_call: 2
  - tool_result: 2

Tool Usage:
  - Write: 1
  - Bash: 1
```

## 扩展功能

### 自定义日志格式

你可以扩展 `ConversationParser` 来支持自定义日志格式：

```go
// 继承 ConversationParser 并重写 writeLogEntry 方法
type CustomParser struct {
    *callcli.ConversationParser
}

func (cp *CustomParser) writeLogEntry(file *os.File, log callcli.FormattedLog) {
    // 自定义日志格式
}
```

### 日志过滤

你可以过滤特定类型的日志：

```go
logs := parser.ExtractLogs(conversation)
filteredLogs := []callcli.FormattedLog{}

for _, log := range logs {
    if log.MessageType == "tool_call" {
        filteredLogs = append(filteredLogs, log)
    }
}
```

### 日志分析

你可以基于日志进行分析：

```go
conversation, _ := parser.Parse(jsonData)
logs := parser.ExtractLogs(conversation)

// 统计工具使用次数
toolUsage := make(map[string]int)
for _, log := range logs {
    if log.MessageType == "tool_call" {
        toolUsage[log.ToolName]++
    }
}
```

## 相关文件

- `internal/callcli/conversation_parser.go` - 对话解析器实现
- `internal/callcli/conversation_parser_test.go` - 测试文件
- `internal/executor/task_runner.go` - 任务执行器（集成日志功能）
- `.morty/logs/` - 日志文件存储目录

## 参考

- [Claude Code 文档](https://docs.anthropic.com/claude-code)
- [Morty 架构设计](./architecture.md)
- [任务执行流程](./task-execution.md)
