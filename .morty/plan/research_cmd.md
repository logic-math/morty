# Plan: Research

## 模块概述

**模块职责**: 实现 `morty research` 命令，启动交互式研究模式，生成结构化调研报告

**对应 Research**:
- `morty-project-research.md` 第 3.5 节 Research 模式分析
- `plan-mode-design.md` 第 6.1 节 Plan 模式实现要点

**依赖模块**: Config, Logging, Call CLI

**被依赖模块**: 无（顶层交互命令）

---

## 命令行接口

### 用法

```bash
# 交互式输入主题
morty research

# 直接指定主题
morty research morty-architecture

# 带选项
morty research --topic morty-architecture
```

### 工作流程

```
1. 检查 .morty/research/ 目录存在
   └─ 不存在则创建

2. 获取研究主题
   └─ 从命令行参数或交互式输入

3. 加载 prompts/research.md 作为系统提示词

4. 调用 Claude Code Plan 模式
   └─ claude --permission-mode plan -p "$(cat prompts/research.md)"

5. 用户与 Claude 交互完成研究

6. 生成 .morty/research/[主题].md
```

---

## 数据模型

```go
// ResearchHandler research 命令处理器
type ResearchHandler struct {
    config    config.Manager
    logger    logging.Logger
    cliCaller callcli.AICliCaller
}

// ResearchOptions research 命令选项
type ResearchOptions struct {
    Topic string
}

// ResearchResult 研究结果
type ResearchResult struct {
    Topic     string
    FilePath  string
    Timestamp time.Time
}
```

---

## 接口定义

### 输入接口
- 命令行参数: `morty research [topic]`
- 环境变量: `CLAUDE_CODE_CLI`

### 输出接口
- 返回码: 0=成功, 1=失败
- 输出文件: `.morty/research/[主题].md`

---

## Jobs (Loop 块列表)

---

### Job 1: research 命令框架

**目标**: 实现 research 命令的基础框架

**前置条件**:
- Config 模块完成
- Logging 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/cmd/research.go` 文件
- [ ] Task 2: 实现 `ResearchHandler` 结构体
- [ ] Task 3: 实现 `NewResearchHandler(cfg, logger)` 构造函数
- [ ] Task 4: 实现 `Execute(ctx, args) error` 方法
- [ ] Task 5: 解析命令行参数获取主题
- [ ] Task 6: 检查并创建 `.morty/research/` 目录
- [ ] Task 7: 编写单元测试 `research_test.go`

**验证器**:
- [ ] 无参数时交互式提示输入主题
- [ ] 有参数时直接使用参数作为主题
- [ ] 自动创建 `.morty/research/` 目录
- [ ] 返回正确的 ResearchResult
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 2: Claude Code 调用实现

**目标**: 实现调用 Claude Code Plan 模式

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `loadResearchPrompt() string` 加载系统提示词
- [ ] Task 2: 实现 `buildClaudeCommand(topic, prompt) []string`
- [ ] Task 3: 使用 `os/exec` 执行 Claude Code
- [ ] Task 4: 传递研究主题给 Claude
- [ ] Task 5: 处理执行错误和退出码
- [ ] Task 6: 记录执行日志
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 正确读取 `prompts/research.md`
- [ ] 构建正确的 Claude Code 命令
- [ ] 以 Plan 模式启动 (`--permission-mode plan`)
- [ ] 正确传递研究主题
- [ ] 执行失败时返回错误
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: 研究结果验证

**目标**: 验证研究结果文件生成

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `validateResearchResult(topic) error`
- [ ] Task 2: 检查 `.morty/research/[主题].md` 是否存在
- [ ] Task 3: 验证文件内容非空
- [ ] Task 4: 输出研究结果摘要
- [ ] Task 5: 提示下一步操作（运行 `morty plan`）
- [ ] Task 6: 编写单元测试

**验证器**:
- [ ] 研究结果文件存在且非空
- [ ] 文件内容包含有效的 Markdown
- [ ] 成功时提示用户运行 `morty plan`
- [ ] 失败时给出友好错误提示
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的 research 流程: 启动 → 研究 → 生成报告
- [ ] 重复研究同一主题覆盖旧文件
- [ ] 中文主题正确处理
- [ ] 特殊字符主题正确处理
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 文件清单

- `internal/cmd/research.go` - research 命令实现
- `prompts/research.md` - research 模式系统提示词
- `.morty/research/[主题].md` - 生成的研究报告

---

## 使用示例

```bash
# 示例 1: 研究 Morty 架构
$ morty research morty-architecture
正在启动研究模式...
主题: morty-architecture
提示词: prompts/research.md

[Claude Code Plan 模式启动，用户交互研究]

研究完成！
生成文件: .morty/research/morty-architecture.md

下一步: 运行 `morty plan` 生成开发计划
```

```bash
# 示例 2: 无参数启动
$ morty research
请输入研究主题: morty-cli-design
正在启动研究模式...

[Claude Code Plan 模式启动]

研究完成！
生成文件: .morty/research/morty-cli-design.md
```
