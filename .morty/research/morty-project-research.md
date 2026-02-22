# Morty 项目重构调研报告

**调查主题**: morty-project-research
**调研日期**: 2026-02-20
**调研目标**: 理解 Morty 项目架构，为重构做准备

---

## 1. 项目概述

Morty 是一个**AI 驱动的开发循环管理系统**，帮助开发者通过迭代式对话改进 PRD（产品需求文档），并自主执行开发任务。

### 1.1 核心定位
- **Fix 模式**: 迭代式 PRD 改进与知识积累
- **Loop 模式**: 自主 AI 开发循环执行
- **Research 模式**: 代码库/文档库研究（当前正在使用）
- **Reset 模式**: 版本管理与回滚

### 1.2 设计理念
- 简化 AI 开发循环
- 交互式对话驱动
- 模块化知识管理
- 完整的可追溯性（Git 集成）

---

## 2. 目录结构

```
morty/
├── morty                    # 主命令入口（统一接口）
├── morty_fix.sh            # Fix 模式实现
├── morty_loop.sh           # Loop 模式实现
├── morty_research.sh       # Research 模式实现
├── morty_reset.sh          # Reset 模式实现
├── install.sh              # 安装脚本
│
├── lib/                    # 共享库
│   ├── common.sh           # 通用工具函数
│   ├── loop_monitor.sh     # tmux 监控集成
│   └── git_manager.sh      # Git 版本管理
│
├── prompts/                # 系统提示词
│   ├── fix_mode_system.md  # Fix 模式提示词
│   └── research.md         # Research 模式提示词
│
├── tests/                  # 测试套件
│   ├── run_all_tests.sh
│   ├── test_fix_mode.sh
│   └── test_git_autocommit.sh
│
└── docs/                   # 文档
    ├── CHANGELOG.md
    ├── CONFIGURATION.md
    └── examples/
```

---

## 3. 核心模块分析

### 3.1 主入口 (morty)

**功能**: 命令路由与分发

```bash
# 支持命令
morty fix <prd.md>       # 迭代式 PRD 改进
morty loop [options]     # 启动开发循环
morty reset [options]    # 版本回滚
morty research [topic]   # 研究模式
morty version            # 显示版本
```

**特点**:
- 简单的 case 语句路由
- 调用 `$MORTY_HOME/` 下的实际脚本
- 版本号硬编码 (0.3.0)

### 3.2 Fix 模式 (morty_fix.sh)

**功能**: 启动交互式 Claude Code 会话改进 PRD

**工作流程**:
1. 检查 PRD 文件（只读）
2. 创建工作目录 `.morty_fix_work/`
3. 如果是再次运行，复制已有 specs/
4. 构建交互式提示词（包含系统提示词 + PRD 内容）
5. 启动 Claude Code 会话
6. 验证生成的 `.morty/` 目录结构

**两种运行模式**:
- **首次运行**: 创建全新的 `.morty/` 结构
- **再次运行**: 合并修改，保留已有 specs，添加演进历史

**关键文件处理策略**:
| 文件 | 首次运行 | 再次运行 |
|------|----------|----------|
| PROMPT.md | 创建 | 重建 |
| fix_plan.md | 创建 | 重建 |
| AGENT.md | 创建 | 重建 |
| specs/*.md | 创建 | 合并（保留+更新+新增） |

### 3.3 Loop 模式 (morty_loop.sh)

**功能**: 自主开发循环执行

**核心流程**:
```
检查项目结构 → 读取配置文件 → 循环执行 Claude
     ↓
构建循环提示词 → 执行 Claude → 检查退出信号
     ↓
Git 自动提交 → 延迟 → 下一循环
```

**退出条件**:
- 检测到 `EXIT_SIGNAL: true`
- 所有任务完成（fix_plan.md 中没有未完成任务）
- 达到最大循环次数
- Claude 执行失败

**监控模式** (`--no-monitor` 控制):
- 默认启动 tmux 三面板监控
- 可直接运行而不启动 tmux

### 3.4 Loop 监控 (lib/loop_monitor.sh)

**功能**: tmux 集成监控

**三面板布局**:
```
┌──────────────────┬───────────────┐
│                  │ 交互终端(30%) │
│  循环日志(50%)   │ status/help   │
│  (满屏)          ├───────────────┤
│                  │ Fix模式(70%)  │
│                  │ morty fix     │
└──────────────────┴───────────────┘
```

**左侧面板**: 直接执行 `morty_loop.sh --no-monitor`
**右上(30%)**: 交互式终端（status/progress/logs/plan 命令）
**右下(70%)**: Fix 模式终端（运行 `morty fix` 进行干预）

### 3.5 Research 模式 (morty_research.sh)

**功能**: 交互式代码库/文档库研究

**工作流程** (由 prompts/research.md 定义):
1. 理解用户输入，确定调查主题
2. 定义搜索路径（搜索源 + 关键词）
3. 搜索相关资源并记录
4. 搜索工作空间（目录结构、关键文件）
5. 批判性追问（价值、事实、逻辑）
6. 综合理解，更新研究文档

**验证器**:
- 检查 `.morty/` 目录存在
- 检查 `.morty/research/` 目录存在
- 检查 `.morty/research/[主题].md` 文件存在

### 3.6 Reset 模式 (morty_reset.sh)

**功能**: 版本回滚和循环管理

**子命令**:
- `morty reset -l [N]`: 显示最近 N 次循环提交
- `morty reset -c <id>`: 回滚到指定 commit
- `morty reset -s`: 显示当前状态

**工作流程**:
1. 关闭运行中的 tmux 会话
2. 执行 `git reset --hard <commit>`
3. 清理未跟踪文件（保留 .morty/logs/）
4. 下次 loop 从回滚状态继续

### 3.7 Git 管理 (lib/git_manager.sh)

**功能**: Git 版本管理集成

**核心函数**:
- `init_git_if_needed()`: 自动初始化 git 仓库
- `create_loop_commit()`: 每次循环后自动提交
- `show_loop_history()`: 显示循环提交历史
- `get_current_loop_number()`: 获取当前循环编号
- `has_uncommitted_changes()`: 检查未提交变更

**提交信息格式**:
```
morty: Loop #N - <status>

自动提交由 Morty 开发循环创建。

循环信息:
- 循环编号: #N
- 状态: completed
- 时间戳: ISO8601
- 父提交: <hash>

变更统计:
- 文件数: N
- 新增行: +N
- 删除行: -N

变更文件:
  - file1
  - file2
```

### 3.8 通用工具 (lib/common.sh)

**功能**: 共享工具函数

**核心功能**:
- **日志**: `log()` 支持多级别日志（INFO/WARN/ERROR/SUCCESS/LOOP）
- **项目检测**: `is_morty_project()` 检查 `.morty/PROMPT.md`
- **项目类型检测**: `detect_project_type()` 支持 nodejs/python/rust/go
- **命令检测**: `detect_build_command()` / `detect_test_command()`
- **上下文更新**: `update_prompt_context()` 更新 PROMPT.md
- **Git 自动提交**: `git_auto_commit()` 循环后自动提交
- **项目结构验证**: `morty_check_project_structure()` 完整验证

---

## 4. 核心数据结构

### 4.1 项目配置文件

#### .morty/PROMPT.md
开发指令文件，包含：
- 问题理解（PRD 引用、模块规范引用）
- 开发原则
- 工作流程
- 质量标准
- **RALPH_STATUS 块格式**（循环结束时输出）

#### .morty/fix_plan.md
任务列表文件，格式：
```markdown
# 任务列表

## 阶段 1: XXX
- [ ] 任务 1 - 参考 specs/xxx.md
- [x] 任务 2 - 已完成
```

#### .morty/AGENT.md
构建和测试指令，格式：
```markdown
# 构建和运行指令

## 项目类型
nodejs/python/rust/go

## 安装
```bash
npm install
```

## 测试
```bash
npm test
```
```

#### .morty/specs/*.md
模块规范文件，包含：
- 目的、范围
- 技术规范
- 集成点
- 质量属性
- 已知问题与解决方案
- **演进历史**（关键，用于追踪变更）

### 4.2 状态文件

#### .morty/status.json
运行时状态：
```json
{
  "state": "running|completed|error|max_loops_reached",
  "loop_count": 5,
  "max_loops": 50,
  "message": "执行循环 5",
  "timestamp": "2026-02-20T14:30:00Z"
}
```

#### .morty/.session_id
Claude Code 会话 ID（可选）

---

## 5. 处理流程

### 5.1 完整工作流程

```
┌─────────────┐
│  创建 PRD   │
└──────┬──────┘
       ↓
┌─────────────┐     ┌─────────────┐
│ morty fix   │────→│ 交互式对话  │
└──────┬──────┘     │ 改进 PRD    │
       ↓            └──────┬──────┘
┌─────────────┐            ↓
│ 生成 .morty/│←────┌──────────────┐
│ 目录结构    │     │ 工作目录隔离 │
└──────┬──────┘     │ .morty_fix_work/
       ↓            └──────────────┘
┌─────────────┐
│ morty loop  │
└──────┬──────┘
       ↓
┌─────────────┐     ┌─────────────┐
│ tmux 监控   │────→│ 三面板布局  │
└──────┬──────┘     └─────────────┘
       ↓
┌─────────────┐
│ 开发循环    │←──── 自动 Git 提交
│ 执行        │
└──────┬──────┘
       ↓
┌─────────────┐
│ morty reset │←──── 版本回滚
│ (可选)      │
└─────────────┘
```

### 5.2 状态机抽象

Morty 项目本身有一个简单的生命周期状态机：

```
[init] → [loop] → [completed]
           ↓
         [error]
```

但实际上，各个模式之间是独立的：
- **Fix 模式**: 独立运行，生成/更新 `.morty/` 结构
- **Loop 模式**: 依赖 `.morty/` 结构，执行开发循环
- **Research 模式**: 独立运行，研究工作空间
- **Reset 模式**: 管理 Git 历史，可独立运行

---

## 6. 部署与安装

### 6.1 安装流程 (install.sh)

1. 创建安装目录 `$HOME/.morty/`
2. 创建命令目录 `$HOME/.local/bin/`
3. 复制脚本文件到安装目录
4. 设置可执行权限
5. 生成 `morty` 主命令（内嵌在 install.sh 中）

### 6.2 依赖项

**必需**:
- Bash 4.0+
- Claude Code CLI (`claude` 命令或自定义 `ai_cli`)
- Git

**可选**:
- tmux（用于监控）
- jq（用于状态显示）

### 6.3 环境变量

```bash
# 自定义 Claude Code CLI 命令
export CLAUDE_CODE_CLI="ai_cli"

# 循环配置
export MAX_LOOPS=100        # 默认 50
export LOOP_DELAY=10        # 默认 5 秒
```

---

## 7. 测试方法

### 7.1 测试结构

```
tests/
├── run_all_tests.sh        # 测试主入口
├── test_fix_mode.sh        # Fix 模式测试
└── test_git_autocommit.sh  # Git 自动提交测试
```

### 7.2 测试覆盖

- Fix 模式初始化流程
- Git 自动提交功能
- 项目结构验证
- 循环执行流程

### 7.3 运行测试

```bash
./tests/run_all_tests.sh
```

---

## 8. 关键设计决策

### 8.1 工作目录隔离

Fix 模式使用 `.morty_fix_work/` 隔离对话过程：
- **优点**: 不污染项目目录，可随时清理
- **缺点**: 需要额外的复制步骤

### 8.2 模块化知识管理

Specs 目录用于维护功能模块规范：
- **优点**: 活文档，持续演进
- **演进历史**: 每次变更都有记录

### 8.3 Git 自动提交

每次循环自动创建 commit：
- **优点**: 完整可追溯，支持回滚
- **缺点**: 可能产生大量 commit

### 8.4 tmux 集成监控

三面板布局提供实时监控：
- **优点**: 实时查看进度，可交互干预
- **缺点**: 依赖 tmux，学习成本

---

## 9. 潜在改进点（重构方向）

### 9.1 架构层面

1. **配置管理**: 当前配置分散在多个地方（环境变量、硬编码），建议统一配置管理
2. **错误处理**: 部分脚本使用 `set -e`，但错误处理不够统一
3. **日志系统**: 日志分散在多个文件，建议统一日志管理

### 9.2 功能层面

1. **插件系统**: 当前项目类型检测硬编码，建议支持插件扩展
2. **状态持久化**: status.json 比较简单，建议增强状态管理
3. **并行执行**: 当前是单循环，可考虑支持并行任务

### 9.3 用户体验

1. **命令补全**: 缺少 shell 自动补全
2. **配置文件**: 缺少用户级配置文件（如 `~/.mortyrc`）
3. **文档生成**: 可自动生成项目文档

### 9.4 代码质量

1. **测试覆盖**: 测试覆盖不够全面
2. **代码复用**: 部分代码重复（如颜色定义）
3. **参数验证**: 参数验证可以更严格

---

## 10. 总结

Morty 是一个设计精良的 AI 开发辅助系统，具有以下特点：

**优势**:
- 清晰的模式分离（Fix/Loop/Research/Reset）
- 完整的 Git 集成，支持版本回滚
- 模块化知识管理
- tmux 集成监控，提供良好可视化

**特点**:
- 以对话驱动的方式改进需求
- 工作目录隔离保护原始文件
- 自动检测项目类型
- 灵活的配置机制

**重构机会**:
- 统一配置管理
- 增强错误处理
- 扩展测试覆盖
- 添加插件系统

---

## 11. 重构设计：新架构 research → plan → doing

### 11.1 重构目标

将原有 `fix → loop` 模式替换为 `research → plan → doing` 三层架构，实现**分层 TDD 开发范式**。

### 11.2 新模式架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Morty 2.0 架构                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   research   │───→│     plan     │───→│    doing     │       │
│  │   (调研)     │    │   (规划)     │    │   (执行)     │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│         │                   │                   │                │
│         ▼                   ▼                   ▼                │
│    .morty/research/    .morty/plan/        .morty/doing/        │
│    [主题].md           [模块].md           logs/               │
│                        [生产测试].md       status.json         │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 11.3 分层 TDD 验证模型

```
Layer 3: [生产测试].md ──→ 端到端测试 + 部署验证
              ↑
Layer 2: [模块].md ──────→ 集成测试（模块内所有 Job）
              ↑
Layer 1: Job ────────────→ 单元测试（Job 执行前生成）
```

### 11.4 Plan 模式设计决策

| 设计项 | 决策 |
|--------|------|
| **验证器格式** | 纯文本 Markdown 自然语言描述 |
| **测试生成** | 全自动生成，用户必要时手动修复 |
| **环境同构** | 灵活自定义，Plan 中定义策略 |
| **失败处理** | 重试 3 次后跳过，日志记录 |

### 11.5 Plan 文件输出结构

```
.morty/plan/
├── README.md              # Plan 索引
├── [模块A].md             # 功能模块 A 计划
│   ├── 模块概述
│   ├── 接口定义
│   ├── 数据模型
│   └── Jobs (Loop 块)
│       ├── Job 1: Tasks + 验证器(自然语言)
│       ├── Job 2: Tasks + 验证器(自然语言)
│       └── 集成测试验证器
├── [模块B].md             # 功能模块 B 计划
└── [生产测试].md          # 端到端测试计划
    ├── 部署架构
    ├── 环境同构策略
    ├── 开发环境启动验证 Job
    ├── 端到端功能测试 Job
    └── 全局回滚策略
```

### 11.6 Job (Loop 块) 结构

每个 Job 包含：
- **目标**: 一句话描述
- **前置条件**: 依赖的 Job 或环境
- **Tasks**: Todo 列表
- **验证器**: 自然语言描述验收标准
- **回滚策略**: 重试次数 + 失败后动作（跳过）

### 11.7 生成的设计文档

- **plan.md**: Plan 模式系统提示词 (`prompts/plan.md`)
- **plan-mode-design.md**: 详细设计文档 (`.morty/research/plan-mode-design.md`)

### 11.8 待实现组件

1. `morty_plan.sh` - Plan 模式脚本
2. `morty_doing.sh` - Doing 模式脚本（取代 loop）
3. `prompts/doing.md` - Doing 模式系统提示词
4. 更新 `morty` 主命令路由
5. 更新 `install.sh` 安装脚本

---

## 12. 研究总结

本次研究完成了：

1. **现有项目分析**: 深入理解了 Morty 1.0 的架构（fix/loop/research/reset）
2. **重构方向确定**: 设计了三层架构（research/plan/doing）
3. **Plan 模式设计**: 完成了系统提示词和详细设计文档
4. **TDD 范式定义**: 确立了单元测试→集成测试→端到端测试的分层验证模型

**下一步**: 实现 `morty_plan.sh` 脚本和 `doing` 模式设计。

---

**文档版本**: 1.1
**研究完成时间**: 2026-02-20
**状态**: 已完成
