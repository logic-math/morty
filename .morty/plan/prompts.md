# Plan: Prompts 系统提示词

## 模块概述

**模块职责**: 定义 Claude Code 各工作模式的系统提示词，指导 AI 如何执行 research、plan、doing 三种模式。

**对应 Research**: 无

**依赖模块**: 无

**被依赖模块**: research_cmd, plan_cmd, doing_cmd (通过 Call CLI 传递)

---

## Prompts 目录结构

```
prompts/
├── research.md          # Research 模式提示词
├── plan.md              # Plan 模式提示词
└── doing.md             # Doing 模式提示词
```

---

## Prompt 文件说明

### research.md - 研究模式

**用途**: 指导 AI 对指定主题进行深入研究，生成调研报告

**核心循环**:
1. 理解用户输入的调查主题
2. 探索工作空间（如需要）
3. 定义搜索路径
4. 搜索并记录信息到 `.morty/research/[主题].md`
5. 深入搜索工作空间
6. 追问和验证信息
7. 综合理解并更新研究报告

**验证器**:
- 检查 `.morty/research/` 目录存在
- 检查 `[调查主题].md` 文件存在

**输出格式**: Markdown 研究报告，包含：
- 项目概述（类型、目录结构、技术栈）
- 核心发现（架构分析、关键代码）
- 潜在问题
- 改进建议
- 相关资源

---

### plan.md - 规划模式

**用途**: 基于研究结果生成可执行的分层 TDD 开发计划

**核心循环**:
1. 汇总调研（读取 `.morty/research/` 文件）
2. 询问用户需求
3. 探索现有代码（如需要）
4. 架构设计（模块划分、接口定义）
5. 生成计划（内存中，暂不写入）
6. 用户确认
7. 写入 Plan 文件

**验证器**:
- 检查 `.morty/plan/` 目录存在
- 检查至少一个 `[模块名].md` 文件
- 检查 `README.md` 索引存在

**输出格式**:
- `plan/README.md` - Plan 索引
- `plan/[模块名].md` - 模块计划

**Plan 文件模板**:
- 模块概述（职责、依赖）
- 接口定义（输入/输出）
- 数据模型
- Jobs（Loop 块列表，每个 Job 包含 Tasks、验证器、调试日志）
- 集成测试

---

### doing.md - 执行模式

**用途**: 执行 Plan 中定义的 Jobs，完成实际的开发工作

**核心循环**:
1. 加载精简上下文
2. 理解当前 Job
3. 探索代码库（如需要）
4. 执行 Tasks
5. 验证 Job
6. 更新 Plan 调试日志
7. 输出 RALPH_STATUS

**精简上下文格式**:
```json
{
  "current": {
    "module": "模块名",
    "job": "job_n",
    "status": "RUNNING",
    "loop_count": 1
  },
  "context": {
    "completed_jobs_summary": [...],
    "current_job": {
      "name": "job_n",
      "description": "...",
      "tasks": [...],
      "validator": "..."
    }
  }
}
```

**验证器**:
- 所有 Tasks 已完成
- 验证器通过
- 无未解决的 debug_log

**输出格式**:
- RALPH_STATUS JSON 块
- Plan 文件调试日志更新

---

## Prompt 设计原则

### 1. 结构化设计

每个 Prompt 包含：
- **目标声明**: 明确 AI 的工作目标
- **循环定义**: 标准的工作流程步骤
- **验证器**: 明确的完成检查条件
- **执行意图**: 详细的行为规范和约束

### 2. 可验证性

- 每个循环都有明确的验证器
- 验证失败时提供具体的改进方向
- 支持多次循环直到满足条件

### 3. 上下文管理

- Research/Plan 模式使用完整上下文
- Doing 模式使用精简上下文，提高效率
- 重要信息持久化到文件系统

### 4. 调试友好

- 每个 Job 都有调试日志区域
- 问题记录格式统一（现象/复现/猜想/验证/修复/进展）
- 支持探索子代理辅助调研

---

## 使用方式

### 调用时加载

```go
// research 模式
cliCaller.CallWithPrompt(ctx, "prompts/research.md", opts)

// plan 模式
cliCaller.CallWithPrompt(ctx, "prompts/plan.md", opts)

// doing 模式
cliCaller.CallWithPrompt(ctx, "prompts/doing.md", opts)
```

### 动态变量替换（可选）

```go
// 预留扩展：支持模板变量替换
type PromptVars struct {
    Module      string
    Job         string
    Topic       string
    ResearchDir string
}

prompt := parser.LoadPrompt("prompts/doing.md")
content := prompt.ReplaceVars(PromptVars{...})
```

---

## 文件清单

- `prompts/research.md` - Research 模式系统提示词
- `prompts/plan.md` - Plan 模式系统提示词
- `prompts/doing.md` - Doing 模式系统提示词
- `plan/prompts.md` - 本文件（Prompts 模块说明）
