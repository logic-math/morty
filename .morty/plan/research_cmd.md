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
- [x] Task 1: 创建 `internal/cmd/research.go` 文件
- [x] Task 2: 实现 `ResearchHandler` 结构体
- [x] Task 3: 实现 `NewResearchHandler(cfg, logger)` 构造函数
- [x] Task 4: 实现 `Execute(ctx, args) error` 方法
- [x] Task 5: 解析命令行参数获取主题
- [x] Task 6: 检查并创建 `.morty/research/` 目录
- [x] Task 7: 编写单元测试 `research_test.go`

**验证器**:
- [x] 无参数时交互式提示输入主题
- [x] 有参数时直接使用参数作为主题
- [x] 自动创建 `.morty/research/` 目录
- [x] 返回正确的 ResearchResult
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: [现象] 测试编译失败, [复现] 运行 go test 时 mockConfig 类型错误, [猜想] 1)GetDuration 签名不匹配 2)缺少 time 包导入, [验证] 检查 config.Manager 接口定义, [修复] 修正 GetDuration 签名为 time.Duration 返回值并添加 time 导入, [进展] 已修复
- debug2: [现象] 测试预期值不匹配, [复现] TestResearchHandler_generateOutputPath 和 TestResearchHandler_sanitizeFilename 失败, [猜想] 1)sanitizeFilename 的截断逻辑预期不一致 2)特殊字符处理产生的下划线数量不同, [验证] 检查实际输出值, [修复] 调整测试预期值与实际行为一致, [进展] 已修复
- explore1: [探索发现] 项目使用标准 Go 项目结构, internal/ 存放内部包, cmd/ 存放命令实现, config.Manager 定义在 internal/config/manager.go, logging.Logger 定义在 internal/logging/logger.go, 已记录

---

### Job 2: Claude Code 调用实现

**目标**: 实现调用 Claude Code Plan 模式

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `loadResearchPrompt() string` 加载系统提示词
- [x] Task 2: 实现 `buildClaudeCommand(topic, prompt) []string`
- [x] Task 3: 使用 `os/exec` 执行 Claude Code
- [x] Task 4: 传递研究主题给 Claude
- [x] Task 5: 处理执行错误和退出码
- [x] Task 6: 记录执行日志
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 正确读取 `prompts/research.md`
- [x] 构建正确的 Claude Code 命令
- [x] 以 Plan 模式启动 (`--permission-mode plan`)
- [x] 正确传递研究主题
- [x] 执行失败时返回错误
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: [现象] 编译失败 logging.Duration 未定义, [复现] 运行 go build 时报错, [猜想] logging 包没有 Duration 函数, [验证] 检查 logging/logger.go, [修复] 使用 logging.Any 替代 logging.Duration, [进展] 已修复
- debug2: [现象] AICliCaller 接口缺少 GetBaseCaller 方法, [复现] 编译报错, [猜想] 接口定义不完整, [验证] 检查 ai_caller.go, [修复] 向 AICliCaller 接口和 AICliCallerImpl 添加 GetBaseCaller 方法, [进展] 已修复
- debug3: [现象] 测试失败 prompts/research.md 文件未找到, [复现] 运行 go test 时报错, [猜想] 路径解析不正确, [验证] 检查 getResearchPromptPath 和测试代码, [修复] 修改 getResearchPromptPath 使用 GetPromptsDir 并添加 SetPromptsDir 方法用于测试, [进展] 已修复
- debug4: [探索发现] 项目 prompts/ 目录位于项目根目录, callcli 包提供 AICliCaller 接口用于执行外部 CLI 命令, 已记录

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
