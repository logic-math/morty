# Doing 模式 Prompt 构建分析

## 概述

本文档详细分析 `morty doing` 命令中 AI CLI 的输入 prompt 是如何构建的，以及它的组成结构。

---

## 调用流程

```
morty doing
    ↓
DoingHandler.Execute()
    ↓
Engine.ExecuteJob(module, job)
    ↓
engine.executeTasks(module, job)
    ↓
engine.buildJobPrompt(module, job)
    ↓
AI CLI 执行
```

---

## Prompt 构建逻辑

### 核心方法：`buildJobPrompt()`

位置：`internal/executor/engine.go:577-653`

```go
func (e *engine) buildJobPrompt(module, job string) (string, error) {
    // 1. 加载 doing 提示词模板
    doingPromptPath := filepath.Join(e.config.PromptsDir, "doing.md")
    promptTemplate, err := os.ReadFile(doingPromptPath)

    // 2. 加载 plan 文件内容
    planFilePath := filepath.Join(e.config.PlanDir, planFileName)
    planContent, err := os.ReadFile(planFilePath)

    // 3. 获取 job 状态信息
    jobState, err := e.getJobState(module, job)

    // 4. 构建 task 列表（带完成状态）
    taskList := ""
    for i, task := range jobState.Tasks {
        status := "[ ]"  // 未完成
        if task.Status == state.StatusCompleted {
            status = "[x]"  // 已完成
        }
        taskList += fmt.Sprintf("- %s Task %d: %s\n", status, i+1, task.Description)
    }

    // 5. 组装完整 prompt
    prompt := fmt.Sprintf(`%s

# Current Job

**Module**: %s
**Job**: %s

## Job Tasks

%s

## Job Details

**Tasks Total**: %d
**Tasks Completed**: %d

# Plan Context

%s

# Job-Level Execution Instructions

You are executing the entire job "%s" in module "%s". This is a job-level execution where you should:

1. Review all tasks listed above
2. Execute each task in sequence
3. Skip tasks that are already marked as completed [x]
4. Follow the doing prompt template for task execution
5. Ensure all validation criteria are met before completing
6. Update the plan file with any issues encountered in the debug logs section
7. Mark the job as complete when all tasks are done and validated

Execute the job autonomously and handle all tasks. Report any issues or blockers encountered.
`, string(promptTemplate), module, job, taskList, len(jobState.Tasks), jobState.TasksCompleted, string(planContent), job, module)

    return prompt, nil
}
```

---

## Prompt 组成结构

一个完整的 AI CLI 输入 prompt 由以下部分组成：

### 1. Doing 提示词模板（doing.md）

**来源**: `prompts/doing.md`

**内容结构**:
```markdown
# Doing

在满足`执行意图`的约束下不断执行`循环`中的工作步骤...

---

# 精简上下文格式
[定义精简上下文的结构]

---

# 循环
loop:[验证器]
    step0: [加载精简上下文]
    step1: [理解Job]
    step1.5: [探索代码库]
    step2: [执行Task]
    step3: [验证Job]
    step4: [更新Plan调试日志]
    step5: [输出RALPH]

---

# 验证器
[Job 完成检查器的规则]

---

# 执行意图
[Task 执行规范、探索子代理使用规范、调试日志记录规范]

---

# RALPH_STATUS 格式
[输出格式定义]

---

# 示例
[完整的执行示例]

---

# 重要提醒
[关键注意事项]
```

**作用**:
- 定义 AI 的执行流程（循环、验证器）
- 规范 Task 执行方式
- 定义调试日志记录格式
- 定义输出格式（RALPH_STATUS）

---

### 2. Current Job 信息

**格式**:
```markdown
# Current Job

**Module**: logging
**Job**: job_3

## Job Tasks

- [ ] Task 1: 实现 JSON 格式输出
- [x] Task 2: 支持上下文数据序列化
- [ ] Task 3: 实现日志格式切换

## Job Details

**Tasks Total**: 3
**Tasks Completed**: 1
```

**作用**:
- 告诉 AI 当前要执行的模块和 Job
- 列出所有 Tasks 及其完成状态
- 提供统计信息（总数、已完成数）

**关键点**:
- `[ ]` 表示未完成的 Task
- `[x]` 表示已完成的 Task（AI 应跳过）
- Task 编号从 1 开始

---

### 3. Plan Context（完整的 plan 文件内容）

**来源**: `.morty/plan/[模块名].md`

**格式**:
```markdown
# Plan: logging

## 模块概述

**模块职责**: 实现日志系统

**对应 Research**: ...

**依赖模块**: 无

**被依赖模块**: ...

## 接口定义

### 输入接口
...

### 输出接口
...

## 数据模型
...

## Jobs

---

### Job 1: 实现日志核心框架

#### 目标
...

#### 前置条件
无

#### Tasks
- [x] Task 1: ...
- [x] Task 2: ...

#### 验证器
- ...

#### 调试日志
- debug1: ...

#### 完成状态
✅ 已完成

---

### Job 2: 日志轮转和归档

[同上格式]

---

### Job 3: 结构化 JSON 日志

#### 目标
实现结构化 JSON 日志支持

#### 前置条件
- job_2 - 日志轮转和归档完成

#### Tasks
- [ ] Task 1: 实现 JSON 格式输出
- [ ] Task 2: 支持上下文数据序列化
- [ ] Task 3: 实现日志格式切换

#### 验证器
- 当配置 log_format: json 时，日志输出应为有效的 JSON 格式

#### 调试日志
无

#### 完成状态
⏳ 待开始

---

[其他 Jobs...]
```

**作用**:
- 提供完整的模块上下文
- 包含所有 Jobs 的定义（目标、Tasks、验证器）
- 显示已完成的 Jobs（作为参考）
- 显示当前 Job 的详细信息
- 包含之前的调试日志（供参考）

**关键点**:
- AI 可以看到整个模块的 plan
- 可以参考已完成 Jobs 的实现
- 可以查看之前的调试日志
- 可以理解当前 Job 的前置条件和依赖

---

### 4. Job-Level Execution Instructions

**格式**:
```markdown
# Job-Level Execution Instructions

You are executing the entire job "job_3" in module "logging". This is a job-level execution where you should:

1. Review all tasks listed above
2. Execute each task in sequence
3. Skip tasks that are already marked as completed [x]
4. Follow the doing prompt template for task execution
5. Ensure all validation criteria are met before completing
6. Update the plan file with any issues encountered in the debug logs section
7. Mark the job as complete when all tasks are done and validated

Execute the job autonomously and handle all tasks. Report any issues or blockers encountered.
```

**作用**:
- 强调这是 Job 级别的执行（不是单个 Task）
- 提醒 AI 要自主完成所有 Tasks
- 强调要更新 plan 文件的调试日志
- 要求 AI 报告问题和阻塞

---

## 完整 Prompt 示例

```markdown
# Doing

在满足`执行意图`的约束下不断执行`循环`中的工作步骤,结合[精简Job上下文]对[任务列表]进行执行,直到满足`验证器`中的约束,才能结束循环,完成Job。

---

# 精简上下文格式
[... doing.md 的完整内容 ...]

---

# 循环
[... doing.md 的完整内容 ...]

---

# 验证器
[... doing.md 的完整内容 ...]

---

# 执行意图
[... doing.md 的完整内容 ...]

---

# RALPH_STATUS 格式
[... doing.md 的完整内容 ...]

---

# 示例
[... doing.md 的完整内容 ...]

---

# 重要提醒
[... doing.md 的完整内容 ...]

# Current Job

**Module**: logging
**Job**: job_3

## Job Tasks

- [ ] Task 1: 实现 JSON 格式输出
- [ ] Task 2: 支持上下文数据序列化
- [ ] Task 3: 实现日志格式切换

## Job Details

**Tasks Total**: 3
**Tasks Completed**: 0

# Plan Context

# Plan: logging

## 模块概述

**模块职责**: 实现日志系统

**对应 Research**:
- `.morty/research/logging_research.md` - 日志系统调研

**依赖模块**: 无

**被依赖模块**: 无

## 接口定义

### 输入接口
- 日志消息字符串
- 日志级别（DEBUG/INFO/WARN/ERROR）

### 输出接口
- 格式化的日志文件
- 控制台输出

## 数据模型

无

## Jobs

---

### Job 1: 实现日志核心框架

#### 目标

实现基础的日志写入和格式化功能

#### 前置条件

无

#### Tasks

- [x] Task 1: 创建日志目录结构
- [x] Task 2: 实现基础日志写入函数
- [x] Task 3: 实现日志级别过滤
- [x] Task 4: 实现时间戳格式化
- [x] Task 5: 添加单元测试

#### 验证器

- 日志文件正确创建
- 日志内容包含时间戳和级别
- 级别过滤正常工作
- 所有测试通过

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 2: 日志轮转和归档

#### 目标

实现日志文件的自动轮转和归档功能

#### 前置条件

- job_1 - 日志核心框架完成

#### Tasks

- [x] Task 1: 实现基于大小的日志轮转
- [x] Task 2: 实现基于时间的日志轮转
- [x] Task 3: 实现日志归档压缩
- [x] Task 4: 实现旧日志自动清理
- [x] Task 5: 添加轮转测试

#### 验证器

- 日志文件达到大小限制时自动轮转
- 日志文件按时间轮转正常工作
- 归档文件正确压缩
- 旧日志按保留策略清理
- 所有测试通过

#### 调试日志

- debug1: 日志轮转时丢失消息, 高频写入时触发轮转, 猜想: 1)文件句柄未同步 2)并发竞争, 验证: 添加文件锁测试, 修复: 使用 flock 同步, 已修复

#### 完成状态

✅ 已完成

---

### Job 3: 结构化 JSON 日志

#### 目标

实现结构化 JSON 日志支持

#### 前置条件

- job_2 - 日志轮转和归档完成

#### Tasks

- [ ] Task 1: 实现 JSON 格式输出
- [ ] Task 2: 支持上下文数据序列化
- [ ] Task 3: 实现日志格式切换

#### 验证器

- 当配置 log_format: json 时，日志输出应为有效的 JSON 格式
- JSON 日志包含所有必需字段（timestamp, level, message）
- 复杂对象正确序列化
- 格式切换不影响现有功能

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 4: 集成测试

#### 目标

验证日志模块的完整性和所有功能协同工作

#### 前置条件

- job_1 - 日志核心框架完成
- job_2 - 日志轮转和归档完成
- job_3 - 结构化 JSON 日志完成

#### Tasks

- [ ] Task 1: 验证模块所有公开接口可以被正常调用
- [ ] Task 2: 验证模块内部各 Job 协同工作产生正确结果
- [ ] Task 3: 验证处理典型业务场景时表现符合预期
- [ ] Task 4: 验证错误处理机制正常工作

#### 验证器

- 模块所有公开接口可以被正常调用
- 模块内部各 Job 协同工作产生正确结果
- 处理典型业务场景时表现符合预期
- 错误处理机制正常工作

#### 调试日志

无

#### 完成状态

⏳ 待开始

# Job-Level Execution Instructions

You are executing the entire job "job_3" in module "logging". This is a job-level execution where you should:

1. Review all tasks listed above
2. Execute each task in sequence
3. Skip tasks that are already marked as completed [x]
4. Follow the doing prompt template for task execution
5. Ensure all validation criteria are met before completing
6. Update the plan file with any issues encountered in the debug logs section
7. Mark the job as complete when all tasks are done and validated

Execute the job autonomously and handle all tasks. Report any issues or blockers encountered.
```

---

## AI CLI 调用参数

### 命令行参数

```go
// 构建参数
baseArgs := e.cliCaller.BuildArgs()
args := append([]string{"--permission-mode", "bypassPermissions", "-p"}, baseArgs...)

// 执行命令
result, err := e.cliCaller.GetBaseCaller().CallWithOptions(
    ctx,
    e.cliCaller.GetCLIPath(),
    args,
    opts
)
```

**完整命令**:
```bash
claude --permission-mode bypassPermissions -p [其他参数]
```

**参数说明**:
- `--permission-mode bypassPermissions`: 绕过权限检查，自动执行
- `-p`: 非交互模式，通过 stdin 传入 prompt
- `[其他参数]`: 从配置中读取的额外参数

---

### 执行选项

```go
opts := callcli.Options{
    Timeout:    0,                          // 无超时限制
    Stdin:      prompt,                     // 通过 stdin 传入 prompt
    WorkingDir: e.config.WorkingDir,        // 工作目录
    Output: callcli.OutputConfig{
        Mode:       callcli.OutputCapture,  // 捕获输出到内存
        OutputFile: logFilePath,            // 同时写入日志文件
    },
}
```

**选项说明**:
- **Timeout**: 设置为 0，表示无超时限制（Job 可能需要很长时间）
- **Stdin**: 完整的 prompt 通过标准输入传递给 AI CLI
- **WorkingDir**: AI CLI 的工作目录（通常是项目根目录）
- **Output.Mode**: `OutputCapture` 表示捕获输出到内存，不污染控制台
- **Output.OutputFile**: 同时将输出写入日志文件（`.morty/logs/[module]_[job]_[timestamp].log`）

---

## Prompt 的关键特点

### 1. Job 级别执行

- **不是 Task 级别**: 一次 AI CLI 调用执行整个 Job 的所有 Tasks
- **自主执行**: AI 需要自己决定如何执行每个 Task
- **状态管理**: AI 需要跳过已完成的 Tasks

### 2. 完整上下文

- **doing.md 模板**: 提供执行流程和规范
- **Current Job 信息**: 提供当前要执行的 Job 和 Tasks
- **Plan Context**: 提供完整的模块 plan，包括所有 Jobs
- **执行指令**: 明确告诉 AI 这是 Job 级别执行

### 3. 调试日志机制

- AI 必须将问题记录到 plan 文件的调试日志中
- 调试日志格式: `debug1: 现象, 复现, 猜想, 验证, 修复, 进展`
- 后续执行可以查看之前的调试日志

### 4. 验证器驱动

- 每个 Job 都有验证器（自然语言描述）
- AI 需要根据验证器生成测试并执行
- 验证通过后才能标记 Job 为完成

### 5. 探索子代理支持

- AI 可以调用 Explore subagent 了解代码库
- 探索结果记录到调试日志
- 用于不熟悉的代码库

---

## 与 Plan 模式的对比

| 维度 | Plan 模式 | Doing 模式 |
|------|-----------|-----------|
| **Prompt 来源** | `prompts/plan.md` | `prompts/doing.md` |
| **输入上下文** | Research 文件 + 用户需求 | Plan 文件 + Job 状态 |
| **执行模式** | 交互式（用户确认） | 自动化（bypassPermissions） |
| **输出** | 生成 plan 文件 | 执行 Tasks，修改代码 |
| **验证** | `morty plan validate` | Job 验证器 + 测试 |
| **权限模式** | `--permission-mode plan` | `--permission-mode bypassPermissions` |
| **超时** | 0（无限制） | 0（无限制） |
| **输出捕获** | `OutputStream`（流式） | `OutputCapture`（捕获） |

---

## 执行流程总结

```
1. morty doing 启动
   ↓
2. 选择下一个可执行的 Job（检查依赖）
   ↓
3. 检查前置条件
   ↓
4. 转换状态: PENDING → RUNNING
   ↓
5. 构建 Prompt:
   - 加载 doing.md
   - 加载 plan 文件
   - 获取 Job 状态
   - 构建 Task 列表
   - 组装完整 prompt
   ↓
6. 调用 AI CLI:
   - 命令: claude --permission-mode bypassPermissions -p
   - Stdin: 完整 prompt
   - 输出: 捕获到内存 + 写入日志文件
   ↓
7. AI 执行:
   - 理解 Job 和 Tasks
   - 可选：调用 Explore subagent
   - 执行每个 Task
   - 运行验证器
   - 更新 plan 文件调试日志
   - 输出 RALPH_STATUS
   ↓
8. 标记所有 Tasks 为完成
   ↓
9. 验证 plan 文件中的完成标记
   ↓
10. 转换状态: RUNNING → COMPLETED
    ↓
11. 创建 Git commit（如果配置）
    ↓
12. 清除当前 Job
    ↓
13. 选择下一个 Job（循环）
```

---

## 优化建议

### 1. Prompt 大小优化

**当前问题**:
- 完整的 plan 文件可能很大（包含所有 Jobs）
- 对于大型项目，prompt 可能超过 context window

**建议**:
- 只包含当前 Job 和相关的前置 Jobs
- 已完成的 Jobs 只保留摘要
- 使用精简格式（类似 doing.md 中提到的精简上下文）

### 2. 调试日志的利用

**当前**:
- 调试日志在 plan 文件中
- AI 可以查看但不一定会利用

**建议**:
- 在 prompt 中突出显示相关的调试日志
- 如果是重试，明确告诉 AI 上次失败的原因

### 3. 验证器的增强

**当前**:
- 验证器是自然语言描述
- AI 需要自己理解并实现

**建议**:
- 提供验证器的示例代码
- 或者提供可执行的验证脚本

### 4. 进度反馈

**当前**:
- AI 执行过程中没有进度反馈
- 只能等待完成或失败

**建议**:
- 使用流式输出（OutputStream）
- 实时显示 AI 的执行进度

---

## 相关文件

- `internal/executor/engine.go` - Prompt 构建和执行逻辑
- `prompts/doing.md` - Doing 提示词模板
- `.morty/plan/[模块名].md` - Plan 文件（上下文来源）
- `.morty/status.json` - 状态文件（Job 状态来源）
- `.morty/logs/[module]_[job]_[timestamp].log` - 执行日志

---

## 总结

Doing 模式的 Prompt 构建是一个精心设计的系统：

1. **分层结构**: 模板 + 当前信息 + 完整上下文 + 执行指令
2. **自主执行**: AI 负责完成整个 Job 的所有 Tasks
3. **上下文丰富**: 提供完整的 plan 文件，AI 可以参考已完成的工作
4. **调试机制**: 强制要求 AI 记录问题到 plan 文件
5. **验证驱动**: 每个 Job 都有明确的验证标准

这种设计使得 morty 能够实现真正的自动化开发流程，AI 可以自主完成复杂的开发任务。
