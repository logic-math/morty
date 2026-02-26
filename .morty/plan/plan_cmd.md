# Plan: Plan Command

## 模块概述

**模块职责**: 实现 `morty plan` 命令，启动交互式规划模式，基于研究结果生成分层 TDD 开发计划

**对应 Research**:
- `plan-mode-design.md` - Plan 模式详细设计
- `morty-project-research.md` 第 11 节 重构设计：新架构 research → plan → doing

**依赖模块**: Config, Logging, Parser, Call CLI

**被依赖模块**: 无（顶层交互命令）

---

## 命令行接口

### 用法

```bash
# 启动规划模式
morty plan

# 强制重新生成（覆盖已有 Plan）
morty plan --force
```

### 工作流程

```
1. 检查 .morty/plan/ 目录
   └─ 已存在且非空时提示是否覆盖（除非 --force）

2. 检查 .morty/research/ 目录
   └─ 存在 → 加载所有研究文件作为事实输入
   └─ 不存在 → 提示 "将通过对话理解需求"

3. 加载 prompts/plan.md 作为系统提示词

4. 调用 Claude Code Plan 模式
   └─ claude --permission-mode plan -p "$(cat prompts/plan.md)"

5. 用户与 Claude 交互完成架构设计

6. 确认后生成 .morty/plan/ 文件
   ├─ README.md - Plan 索引
   ├─ [模块A].md - 模块 A 计划
   ├─ [模块B].md - 模块 B 计划
   └─ [生产测试].md - 端到端测试计划
```

---

## 数据模型

```go
// PlanHandler plan 命令处理器
type PlanHandler struct {
    config        config.Manager
    logger        logging.Logger
    parserFactory parser.Factory
    cliCaller     callcli.AICliCaller
}

// PlanOptions plan 命令选项
type PlanOptions struct {
    Force bool
}

// PlanResult 规划结果
type PlanResult struct {
    Modules      []string
    TotalJobs    int
    OutputDir    string
    Timestamp    time.Time
}
```

---

## 接口定义

### 输入接口
- 命令行参数: `morty plan [--force]`
- 输入文件: `.morty/research/*.md`（可选）
- 环境变量: `CLAUDE_CODE_CLI`

### 输出接口
- 返回码: 0=成功, 1=失败
- 输出目录: `.morty/plan/`

---

## Jobs (Loop 块列表)

---

### Job 1: plan 命令框架

**目标**: 实现 plan 命令的基础框架

**前置条件**:
- Config 模块完成
- Logging 模块完成
- Plan Parser 模块完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/cmd/plan.go` 文件
- [x] Task 2: 实现 `PlanHandler` 结构体
- [x] Task 3: 实现 `NewPlanHandler(cfg, logger, parser)` 构造函数
- [x] Task 4: 实现 `Execute(ctx, args) error` 方法
- [x] Task 5: 解析 `--force` 选项
- [x] Task 6: 检查并创建 `.morty/plan/` 目录
- [x] Task 7: 检查已有 Plan 文件是否存在
- [x] Task 8: 编写单元测试 `plan_test.go`

**验证器**:
- [x] 无 `--force` 时，已有 Plan 文件提示确认
- [x] `--force` 时直接覆盖
- [x] 自动创建 `.morty/plan/` 目录
- [x] 返回正确的 PlanResult
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: mockConfig 和 mockLogger 在多个测试文件中重复定义导致编译错误, 同时加载 plan_test.go 和 research_test.go 时报 redeclared 错误, 猜想: 1)包级别类型重名 2)Go测试文件共享同一包命名空间, 验证: 检查 research_test.go 发现已定义 mockConfig/mockLogger, 修复: 移除 plan_test.go 中的重复定义，使用 research_test.go 中的 mock 类型, 已修复
- debug2: 测试失败因为 GetPlanDir() 返回相对路径 .morty/plan 而非绝对路径, 测试期望返回基于 tmpDir 的绝对路径, 猜想: 1)mockConfig.GetPlanDir() 实现问题 2)Paths 配置未正确设置, 验证: 检查 mockConfig 实现发现返回固定字符串, 修复: 更新 mockConfig 支持 SetWorkDir 和 SetPlanDir 方法，GetPlanDir 基于 workDir 计算, 已修复
- explore1: [探索发现] 项目使用单文件日志实现, lib/logging.sh 为核心模块, 使用文件追加模式写入, 已记录

---

### Job 2: 研究文件加载

**目标**: 加载已有研究文件作为事实输入

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `loadResearchFacts() ([]string, error)`
- [x] Task 2: 扫描 `.morty/research/` 目录
- [x] Task 3: 读取所有 `.md` 文件内容
- [x] Task 4: 按文件名排序
- [x] Task 5: 无研究文件时给出提示
- [x] Task 6: 将研究内容格式化为提示词输入
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 正确读取所有研究文件
- [x] 文件内容按顺序组合
- [x] 无研究文件时返回空列表并提示
- [x] 损坏文件时返回错误
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: 测试失败因为 setupTestDir 只创建临时目录而没有创建 research 子目录，创建文件时报 "no such file or directory" 错误, 猜想: 1)setupTestDir 应该自动创建所有子目录 2)测试应该自己创建所需目录, 验证: 检查 setupTestDir 实现发现只创建 tmpDir, 修复: 在需要创建文件的测试中使用 os.MkdirAll 创建 research 目录, 已修复
- debug2: TestPlanHandler_loadResearchFacts_emptyDir 返回 nil 而非空切片，测试期望返回空列表 []string{}, 猜想: 1)返回 nil 是预期行为 2)变量声明方式导致返回 nil, 验证: 检查代码发现 var facts []string 初始为 nil，当无文件时返回 nil, 修复: 使用 facts := make([]string, 0, len(mdFiles)) 确保返回空切片而非 nil, 已修复
- debug3: 测试覆盖率未达到 80%，需要添加更多边界情况测试, 猜想: 1)缺少错误处理测试 2)缺少边界情况测试, 验证: 运行覆盖率检查显示 loadResearchFacts 覆盖率为 95.7%，已超过要求, 修复: 无需修复，覆盖率达到要求, 已解决

---

### Job 3: Claude Code 调用实现

**目标**: 实现调用 Claude Code Plan 模式

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `loadPlanPrompt() string` 加载系统提示词
- [x] Task 2: 实现 `buildClaudeCommand(prompt, facts) []string`
- [x] Task 3: 构建包含研究事实的完整提示词
- [x] Task 4: 使用 `os/exec` 执行 Claude Code
- [x] Task 5: 以 Plan 模式启动 (`--permission-mode plan`)
- [x] Task 6: 处理执行错误和退出码
- [x] Task 7: 记录执行日志
- [x] Task 8: 编写单元测试

**验证器**:
- [x] 正确读取 `prompts/plan.md`
- [x] 研究事实正确嵌入提示词
- [x] 构建正确的 Claude Code 命令
- [x] 以 Plan 模式启动
- [x] 执行失败时返回错误
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: mockAICliCaller 和 mockCaller 在 plan_test.go 和 research_test.go 中重复定义导致编译错误, 同时加载两个测试文件时报 redeclared 错误, 猜想: 1)包级别类型重名 2)Go测试文件共享同一包命名空间, 验证: 检查 research_test.go 发现已定义 mockAICliCaller/mockCaller, 修复: 移除 plan_test.go 中的重复定义，使用 research_test.go 中的 mock 类型, 已修复
- debug2: 覆盖率最初为 79.1% 未达到 80% 要求, 缺少 executeClaudeCode 测试, 猜想: 1)未测试执行成功/失败场景 2)缺少 mock 测试, 验证: 添加 executeClaudeCode 的多种场景测试, 修复: 添加 TestPlanHandler_executeClaudeCode_success/failure/nonZeroExit 等测试, 已修复 (覆盖率提升至 84.7%)

---

### Job 4: Plan 文件验证

**目标**: 验证生成的 Plan 文件

**前置条件**:
- Job 3 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `validatePlanResult() error`
- [x] Task 2: 检查 `README.md` 是否存在
- [x] Task 3: 检查至少有一个模块 Plan 文件
- [x] Task 4: 使用 Plan Parser 验证文件格式
- [x] Task 5: 统计模块数和 Jobs 数
- [x] Task 6: 输出 Plan 摘要
- [x] Task 7: 提示下一步操作（运行 `morty doing`）
- [x] Task 8: 编写单元测试

**验证器**:
- [x] README.md 存在且格式正确
- [x] 至少有一个 [模块].md 文件
- [x] 所有 Plan 文件可被正确解析
- [x] 成功时提示用户运行 `morty doing`
- [x] 失败时给出友好错误提示
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: 探索发现所有 Tasks 已在代码中实现，validatePlanResult() 函数存在于 plan.go:602-620，ValidatePlanResult() 完整实现在 plan.go:492-594，统计功能在 plan.go:561-585，PrintPlanSummary 在 plan.go:622-669，测试覆盖在 plan_test.go:1141-1583，猜想: 1)Job 4 在之前 Jobs 开发时已被实现 2)代码结构完整，验证: 读取代码确认所有功能已实现，修复: 无需修复，将 Tasks 标记为完成，已修复

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的 plan 流程: 检查 → 加载研究 → 规划 → 生成文件
- [ ] 无研究文件时也能正常工作
- [ ] `--force` 正确覆盖已有 Plan
- [ ] 生成的 Plan 文件可被 Plan Parser 正确解析
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 文件清单

- `internal/cmd/plan.go` - plan 命令实现
- `prompts/plan.md` - plan 模式系统提示词
- `.morty/plan/README.md` - Plan 索引（生成）
- `.morty/plan/[模块].md` - 模块计划（生成）
- `.morty/plan/[生产测试].md` - 端到端测试计划（生成）

---

## 使用示例

```bash
# 示例 1: 首次规划
$ morty plan
检查研究文件...
找到 2 个研究文件:
  - morty-architecture.md
  - morty-cli-design.md

正在启动规划模式...
提示词: prompts/plan.md

[Claude Code Plan 模式启动，用户交互规划]

规划完成！
生成 Plan 文件:
  - .morty/plan/README.md
  - .morty/plan/cli.md (3 Jobs)
  - .morty/plan/config.md (3 Jobs)
  - .morty/plan/logging.md (4 Jobs)
  - .morty/plan/state.md (3 Jobs)
  - .morty/plan/git.md (3 Jobs)
  - .morty/plan/plan_parser.md (2 Jobs)
  - .morty/plan/executor.md (4 Jobs)
  - .morty/plan/生产测试.md (2 Jobs)

总计: 8 个模块, 27 个 Jobs

下一步: 运行 `morty doing` 开始执行
```

```bash
# 示例 2: 强制重新生成
$ morty plan
发现已有 Plan 文件，是否覆盖? [y/N]: n
取消操作。

$ morty plan --force
强制重新生成 Plan...
[Claude Code Plan 模式启动]
规划完成！
```

```bash
# 示例 3: 无研究文件
$ morty plan
未找到研究文件，将通过对话理解需求。

正在启动规划模式...
提示词: prompts/plan.md

[Claude Code Plan 模式启动]

规划完成！
生成 Plan 文件: ...
```
