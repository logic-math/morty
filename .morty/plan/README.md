# Plan 索引

**生成时间**: 2026-02-23T00:00:00Z

**对应 Research**:
- `.morty/research/morty-project-research.md` - Morty 项目重构调研报告
- `.morty/research/morty-go-refactor-plan.md` - Go 重构计划
- `.morty/research/morty-claude-code-best-practices-improvements.md` - 最佳实践改进
- `.morty/research/plan-mode-design.md` - Plan 模式设计
- `.morty/research/best-practices.md` - Claude Code 最佳实践

**现有实现探索**: 是，基于 Shell 版本的 Morty 1.0 架构分析

---

## 模块列表

### Phase 0: 环境准备（最高优先级）

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| Go 环境搭建 | go_env_setup.md | 5 | 无 | **必须先完成** |

### Phase 1: 基础框架 (无依赖)

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| Config | config.md | 4 | 无 | 规划中 |
| Logging | logging.md | 4 | Config | 规划中 |
| Git | git.md | 3 | 无 | 规划中 |
| Parser | parser.md | 7 | 无 | 规划中 |
| Call CLI | call_cli.md | 7 | Config, Logging | 规划中 |
| Deploy | deploy.md | 4 | 无 | 规划中 |
| Prompts | prompts.md | 无 | 无 | 规划中 |
| Errors | errors.md | 1 | 无 | 规划中 |

**说明**:
- **Config**: 配置管理（配置结构定义、加载/读取/保存、层级合并）
- **Deploy**: 提供构建和安装脚本 (`scripts/build.sh`, `scripts/install.sh`, `scripts/uninstall.sh`, `scripts/upgrade.sh`)
- **Prompts**: 定义系统提示词 (`prompts/research.md`, `prompts/plan.md`, `prompts/doing.md`)
- **Errors**: 定义统一错误码体系

### Phase 2: 核心依赖模块

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| State | state.md | 3 | Config | 规划中 |
| CLI | cli.md | 3 | Config, Logging | 规划中 |

### Phase 3: 命令模块 (自动化/交互式)

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| research_cmd | research_cmd.md | 3 | Config, Logging, Call CLI | 规划中 |
| plan_cmd | plan_cmd.md | 4 | Config, Logging, Parser, Call CLI | 规划中 |
| doing_cmd | doing_cmd.md | 7 | Config, Logging, State, Git, Parser, Call CLI | 规划中 |
| stat_cmd | stat_cmd.md | 5 | Config, Logging, State, Git, Parser | 规划中 |
| reset_cmd | reset_cmd.md | 5 | Config, Logging, State, Git | 规划中 |

### Phase 4: 验证

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| E2E 功能测试 | e2e_test.md | 7 | 所有模块 | 规划中 |

**说明**: E2E 测试覆盖完整用户旅程（安装→Research→Plan→Doing→Stat→Reset）和生产部署验证。

---

## 整体架构

### 架构概览

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Morty Go 架构 v2.0                              │
│                     Research → Plan → Doing 工作流引擎                       │
└─────────────────────────────────────────────────────────────────────────────┘

【Phase 0】环境准备                    【Phase 1】基础组件层 (可并行开发)
┌─────────────────┐                   ┌─────────────────────────────────────────┐
│  Go 环境搭建     │                   │  ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│   (5 Jobs)      │                   │  │ Config  │ │  Git    │ │ Parser  │  │
│                 │                   │  │ 3 Jobs  │ │ 3 Jobs  │ │ 7 Jobs  │  │
│ • Go 1.21+      │                   │  └────┬────┘ └────┬────┘ └────┬────┘  │
│ • 环境变量       │                   │       │         │         │         │
│ • 项目结构       │                   │       └─────────┴────┬────┴─────────┘  │
└────────┬────────┘                   │                      │                │
         │                            │                      ▼                │
         │                            │               ┌─────────────┐         │
         ▼                            │               │   Logging   │         │
┌─────────────────────────────────────┤               │   4 Jobs    │         │
│                                     │               └──────┬──────┘         │
│  ┌─────────┐      ┌─────────┐      │                      │                │
│  │ Call CLI│◄────►│  Parser │      │                      │                │
│  │ 7 Jobs  │      │ 7 Jobs  │      │                      │                │
│  └────┬────┘      └────┬────┘      │                      │                │
│       │                │           │                      │                │
│       └────────────────┴───────────┴──────────────────────┘                │
│                                     │                                       │
│  ┌─────────────────────────────────────────────────────────┐               │
│  │                     Deploy (4 Jobs)                      │               │
│  │  ┌─────────┐ ┌───────────┐ ┌─────────────┐ ┌─────────┐ │               │
│  │  │build.sh │ │install.sh │ │uninstall.sh │ │upgrade.sh│ │               │
│  │  └─────────┘ └───────────┘ └─────────────┘ └─────────┘ │               │
│  └─────────────────────────────────────────────────────────┘               │
│                                     │                                       │
└─────────────────────────────────────┴───────────────────────────────────────┘
                                      │
                                      ▼
【Phase 2】核心层                     【Phase 3】命令层
┌─────────────────────────────────┐   ┌─────────────────────────────────────────┐
│  ┌─────────────┐ ┌───────────┐ │   │         交互式命令 (Plan Mode)           │
│  │    State    │ │    CLI    │ │   │  ┌─────────────┐   ┌─────────────┐      │
│  │  3 Jobs     │ │  3 Jobs   │ │   │  │ research_cmd│   │  plan_cmd   │      │
│  │ (依赖Config)│ │(依赖Config│ │   │  │   3 Jobs    │   │   4 Jobs    │      │
│  │             │ │ /Logging) │ │   │  │             │   │             │      │
│  └──────┬──────┘ └─────┬─────┘ │   │  │ • 启动研究   │   │ • 加载研究   │      │
│         │              │       │   │  │ • 调用AI    │   │ • 调用AI    │      │
│         └──────────────┘       │   │  │ • 生成报告   │   │ • 生成Plan  │      │
│                                │   │  └─────────────┘   └─────────────┘      │
└────────────────────────────────┘   │                                         │
                                      │         自动化命令 (Normal Mode)         │
                                      │  ┌───────────┬───────────┬──────────┐   │
                                      │  │ doing_cmd │ stat_cmd  │reset_cmd │   │
                                      │  │   7 Jobs  │  5 Jobs   │  5 Jobs  │   │
                                      │  │           │           │          │   │
                                      │  │• 加载Plan │• 读取状态 │• 查询历史│   │
                                      │  │• 执行Job  │• 显示进度 │• 回滚版本│   │
                                      │  │• 更新状态 │• 监控模式 │• 状态同步│   │
                                      │  └───────────┴───────────┴──────────┘   │
                                      └─────────────────────────────────────────┘
                                                        │
                                                        ▼
                                      ┌─────────────────────────────────────────┐
                                      │  【Phase 4】验证层                      │
                                      │  ┌─────────────────────────────────────┐  │
                                      │  │        E2E 功能测试 (7 Jobs)         │  │
                                      │  │    完整用户旅程 + 部署验证            │  │
                                      │  └─────────────────────────────────────┘  │
                                      └─────────────────────────────────────────┘
```

### 模块依赖关系

```text
                              ┌──────────────┐
                              │  Phase 0     │
                              │ Go环境搭建   │
                              └──────┬───────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Phase 1: 基础组件层                              │
│                                                                              │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│   │   Config    │    │    Git      │    │   Parser    │    │  Call CLI   │  │
│   │  ├─ settings│    │  ├─ commit  │    │  ├─ markdown│    │  ├─ exec    │  │
│   │  ├─ defaults│    │  ├─ branch  │    │  ├─ plan    │    │  ├─ async   │  │
│   │  └─ env     │    │  └─ log     │    │  └─ prompt  │    │  └─ timeout │  │
│   └──────┬──────┘    └─────────────┘    └─────────────┘    └─────────────┘  │
│          │                                                                    │
│          └────────────────────────┬───────────────────────────────────────────┘
│                                   │
│                                   ▼
│                            ┌──────────────┐
│                            │   Logging    │
│                            │  ├─ slog     │
│                            │  ├─ rotate   │
│                            │  └─ level    │
│                            └──────┬───────┘
└───────────────────────────────────┼──────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Phase 2: 核心层                                  │
│                                                                              │
│   ┌──────────────────────────┐          ┌──────────────────────────┐        │
│   │         State            │          │          CLI             │        │
│   │  ├─ status.json 管理     │          │  ├─ 参数解析             │        │
│   │  ├─ 状态机转换            │          │  ├─ 命令路由             │        │
│   │  └─ 断点恢复             │          │  └─ 全局选项             │        │
│   └──────────────────────────┘          └──────────────────────────┘        │
└──────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Phase 3: 命令层                                  │
│                                                                              │
│   ┌──────────────────────────────────────────────────────────────────────┐  │
│   │                        交互式命令                                     │  │
│   │  ┌────────────────────┐              ┌────────────────────┐         │  │
│   │  │   research_cmd     │              │     plan_cmd       │         │  │
│   │  │  (Call CLI ──► AI) │              │  (Call CLI ──► AI) │         │  │
│   │  └────────────────────┘              └────────────────────┘         │  │
│   └──────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│   ┌──────────────────────────────────────────────────────────────────────┐  │
│   │                        自动化命令                                     │  │
│   │                                                                      │  │
│   │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │  │
│   │  │    doing_cmd     │  │    stat_cmd      │  │   reset_cmd      │   │  │
│   │  │  ├─ 加载 Plan    │  │  ├─ 读取 State   │  │  ├─ Git 历史     │   │  │
│   │  │  ├─ 调用AI执行   │  │  ├─ 格式化输出   │  │  ├─ 版本回滚     │   │  │
│   │  │  ├─ 更新状态     │  │  └─ 监控模式     │  │  └─ 状态同步     │   │  │
│   │  │  └─ Git 提交     │  │                  │  │                  │   │  │
│   │  └──────────────────┘  └──────────────────┘  └──────────────────┘   │  │
│   └──────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 数据流图

```text
【Research 流程】
用户输入 ──► research_cmd ──► Call CLI ──► ai_cli (Plan Mode)
                                      │
                                      ▼
                              .morty/research/
                              └── [topic].md

【Plan 流程】
Research 结果 ──► plan_cmd ──► Parser (读取 research)
                          │
                          ├──► Call CLI ──► ai_cli (Plan Mode)
                          │
                          ▼
                   .morty/plan/
                   ├── README.md
                   └── [module].md

【Doing 流程】
Plan 文件 ──► doing_cmd ──► Parser (解析 Plan)
                        │
                        ├──► State (读取/更新状态)
                        │
                        ├──► Call CLI ──► ai_cli (Normal Mode)
                        │       │
                        │       └── 执行 Task
                        │
                        ├──► Git (创建提交)
                        │
                        └──► State (保存结果)

【Stat 流程】
无 ──► stat_cmd ──► State (读取状态)
                │
                ├──► Parser (解析 Plan 统计)
                │
                ├──► Git (获取提交历史)
                │
                └──► 格式化输出 (Table/JSON)

【Reset 流程】
用户输入 ──► reset_cmd ──► Git (查询历史/回滚)
                        │
                        └──► State (同步状态)
```

---

## 执行顺序

### Phase 0: 环境准备 (必须先完成)
1. **Go 环境搭建** - 安装 Go 1.21+，配置环境变量，初始化项目结构

### Phase 1: 基础框架 (无依赖，可并行)
2. **Config** - 配置管理模块（结构定义、加载/保存、层级合并）
3. **Git** - Git 操作模块
4. **Parser** - 通用文件解析模块
5. **Call CLI** - CLI 调用器模块
6. **Deploy** - 部署模块（build/install/uninstall/upgrade）
7. **Prompts** - 系统提示词定义
8. **Errors** - 错误码体系定义
9. **Logging** - 日志模块（依赖 Config）

### Phase 2: 核心模块
10. **State** - 状态管理模块（依赖 Config）
11. **CLI** - CLI 框架（依赖 Config, Logging）
12. **Executor** - 执行引擎（依赖 State, Parser, Call CLI）

### Phase 3: 命令模块
13. **research_cmd** - 研究命令（依赖 Config, Logging, Call CLI）
14. **plan_cmd** - 规划命令（依赖 Config, Logging, Parser, Call CLI）
15. **doing_cmd** - 执行命令（依赖 Config, Logging, State, Git, Parser, Call CLI, Executor）
16. **stat_cmd** - 状态命令（依赖 Config, Logging, State, Git, Parser）
17. **reset_cmd** - 回滚命令（依赖 Config, Logging, State, Git）

### Phase 4: 验证
18. **E2E 功能测试** - 端到端测试（完整用户旅程 + 部署验证）

---

## 统计信息

- **总模块数**: 18
- **总 Jobs 数**: 81
- **预计执行轮次**: 5 (含环境准备和E2E测试)
- **探索子代理使用**: 是

---

## 关键设计决策

1. **接口优先**: 每个模块先定义接口，再实现
2. **依赖注入**: 使用构造函数注入依赖
3. **测试驱动**: 每个 Job 先写测试再实现
4. **错误处理**: 使用自定义错误类型，支持错误链
5. **结构化日志**: 使用 slog 标准库
6. **模块化命令**: CLI 框架与命令实现分离，各自独立开发测试
7. **抽象解析器**: Parser 模块采用抽象设计，支持 Markdown/JSON/YAML 等多种格式扩展
8. **子进程管理**: Call CLI 模块统一封装外部 CLI 调用，支持超时、信号、异步等特性
9. **分层架构**: 基础组件 → 核心层 → 命令层 → 验证层，逐层构建

---

## 文件清单

### Phase 0: 环境准备

- `plan/go_env_setup.md` - **Go 环境搭建 (必须先完成)**

### Phase 1: 基础框架

- `plan/config.md` - 配置模块（结构定义、加载/保存、层级合并）
- `plan/logging.md` - 日志模块 (结构化日志、轮转)
- `plan/git.md` - Git 模块 (版本管理、循环提交)
- `plan/parser.md` - 通用文件解析模块
- `plan/call_cli.md` - CLI 调用器模块（子进程管理）
- `plan/executor.md` - 执行引擎（Job/Task 执行调度）
- `plan/deploy.md` - 部署模块（build.sh/install.sh/uninstall.sh/upgrade.sh）
- `plan/prompts.md` - 系统提示词定义
- `plan/errors.md` - 错误码体系定义

### Phase 2: 核心模块

- `plan/state.md` - 状态管理模块 (status.json、状态机)
- `plan/cli.md` - CLI 框架模块 (命令注册、路由)

### Phase 3: 命令模块

- `plan/research_cmd.md` - research 命令实现（交互式）
- `plan/plan_cmd.md` - plan 命令实现（交互式）
- `plan/doing_cmd.md` - doing 命令实现（自动化）
- `plan/stat_cmd.md` - stat 命令实现（状态监控）
- `plan/reset_cmd.md` - reset 命令实现（版本回滚）

### Phase 4: 验证

- `plan/e2e_test.md` - E2E 测试（完整用户旅程 + 部署验证）

---

## 命令行接口设计

### 主入口

```bash
morty <command> [options] [args]
```

### 支持的命令

#### 1. research - 研究模式

```bash
# 启动研究模式，交互式研究指定主题
morty research [topic]

# 示例
morty research                    # 交互式输入主题
morty research morty-architecture # 直接指定主题
```

研究模式工作流程:
1. 检查 `.morty/research/` 目录，加载已有研究文件
2. 启动 Claude Code Plan 模式会话
3. 读取 `prompts/research.md` 作为系统提示词
4. 交互式研究，生成 `.morty/research/[主题].md`

#### 2. plan - 规划模式

```bash
# 启动规划模式，基于研究结果生成 Plan
morty plan

# 强制重新生成（覆盖已有 Plan）
morty plan --force
```

规划模式工作流程:
1. 检查 `.morty/research/` 是否有研究文件（可选）
2. 如没有 research，提示 "将通过对话理解需求"
3. 启动 Claude Code Plan 模式会话
4. 读取 `prompts/plan.md` 作为系统提示词
5. 交互式确认模块划分和 Job 设计
6. 生成 `.morty/plan/README.md` 和模块 Plan 文件

#### 3. doing - 执行 Plan

```bash
# 执行下一个未完成的 Job
morty doing

# 仅执行指定模块的下一个未完成 Job
morty doing --module cli

# 仅执行指定的单个 Job
morty doing --module cli --job job_1

# 重置后执行（从第一个 Job 开始）
morty doing --restart

# 重置指定模块
morty doing --restart --module cli

# 重置并执行指定 Job
morty doing --restart --module cli --job job_1
```

执行模式工作流程:
1. 检查 `.morty/plan/` 是否存在（必须）
2. 读取 `status.json` 获取当前状态
3. 选择下一个 PENDING 状态的 Job
4. 执行 Job（调用 AI CLI）
5. 更新状态并创建 Git 提交
6. 单循环执行，完成后退出

#### 4. stat - 状态监控

```bash
# 显示当前状态（默认表格格式）
morty stat

# 监控模式，每60秒刷新
morty stat -w

# JSON 格式输出
morty stat --json
```

#### 5. reset - 版本回滚

```bash
# 显示最近10次循环提交
morty reset -l

# 显示最近5次循环提交
morty reset -l 5

# 回滚到指定提交
morty reset -c abc1234
```

#### 6. version - 显示版本

```bash
morty version
```

#### 7. help - 帮助信息

```bash
# 显示全局帮助
morty help

# 显示指定命令帮助
morty help doing
```

### 全局选项

```bash
--verbose    # 详细输出模式
--debug      # 调试模式
```

---

## 项目目录结构

### 源代码结构

```
morty/                                    # 项目根目录（原地更新）
├── cmd/morty/main.go                     # 主入口
├── internal/                             # 内部模块
│   ├── cli/                              # CLI 框架（注册、路由、解析）
│   │   ├── parser.go                     # 参数解析
│   │   ├── router.go                     # 命令路由
│   │   ├── options.go                    # 全局选项
│   │   └── command.go                    # 命令定义
│   ├── cmd/                              # 命令实现（各子命令）
│   │   ├── register.go                   # 命令注册入口
│   │   ├── research.go                   # research 命令
│   │   ├── plan.go                       # plan 命令
│   │   ├── doing.go                      # doing 命令
│   │   ├── stat.go                       # stat 命令
│   │   ├── reset.go                      # reset 命令
│   │   ├── version.go                    # version 命令
│   │   └── help.go                       # help 命令
│   ├── config/                           # 配置模块
│   ├── logging/                          # 日志模块
│   ├── state/                            # 状态管理模块
│   ├── git/                              # Git 模块
│   ├── parser/                           # 通用解析模块
│   │   ├── interface.go                  # 核心接口
│   │   ├── factory.go                    # 解析器工厂
│   │   ├── markdown/                     # Markdown 解析
│   │   │   ├── parser.go
│   │   │   ├── section.go
│   │   │   ├── task.go
│   │   │   └── metadata.go
│   │   ├── plan/                         # Plan 专用解析
│   │   │   └── parser.go
│   │   └── prompt/                       # Prompt 专用解析
│   │       └── parser.go
│   └── callcli/                          # CLI 调用器模块
│       ├── interface.go                  # 核心接口
│       ├── caller.go                     # 基础调用实现
│       ├── ai_caller.go                  # AI CLI 专用封装
│       ├── async.go                      # 异步调用
│       └── timeout.go                    # 超时控制
├── pkg/                                  # 公共包
│   ├── types/                            # 公共类型
│   └── utils/                            # 工具函数
├── configs/                              # 默认配置
│   └── settings.json                     # 默认配置模板
├── prompts/                              # 系统提示词
│   ├── research.md                       # research 模式提示词
│   ├── plan.md                           # plan 模式提示词
│   └── doing.md                          # doing 模式提示词
├── scripts/                              # 构建和部署脚本
│   ├── build.sh                          # 编译脚本
│   ├── install.sh                        # 安装脚本
│   ├── uninstall.sh                      # 卸载脚本
│   └── upgrade.sh                        # 升级脚本
├── go.mod
└── go.sum
```

### 用户安装目录

```
~/.morty/                                 # 用户级安装目录
├── bin/
│   └── morty                             # 可执行文件
└── config.json                           # 用户全局配置文件
```

### 项目级配置目录（每个项目独立）

```
./.morty/                                 # 项目级配置（随项目创建）
├── plan/                                 # Plan 文件目录
│   ├── README.md
│   ├── [module].md
│   └── ...
├── research/                             # 研究文件目录
│   └── [topic].md
├── doing/                                # 执行日志目录
│   ├── logs/                             # 执行日志
│   └── tests/                            # 测试输出
├── status.json                           # 执行状态
└── .git/                                 # Git 版本控制
```

---

## 工作流程

### 完整使用流程

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   research  │───→│    plan     │───→│   doing     │
│  (交互研究)  │    │  (交互规划)  │    │  (自动执行)  │
└─────────────┘    └─────────────┘    └─────────────┘
      │                   │                   │
      ▼                   ▼                   ▼
.morty/research/    .morty/plan/       .morty/doing/
[主题].md           README.md           logs/
                    [模块].md           status.json
```

### 命令说明

| 命令 | 模式 | 自动化 | 说明 |
|------|------|--------|------|
| `research` | Plan 模式 | 否 | 交互式研究，生成调研报告 |
| `plan` | Plan 模式 | 否 | 交互式规划，生成 Plan 文件 |
| `doing` | Normal 模式 | 是 | 自动执行 Plan，单循环 |
| `stat` | Normal 模式 | 是 | 显示状态，可监控模式 |
| `reset` | Normal 模式 | 是 | 版本回滚 |

**注意**: `research` 和 `plan` 是交互式命令，由用户手动执行，Claude Code 以 Plan 模式运行；`doing`、`stat`、`reset` 是自动化命令。

---

## 模块职责边界

| 模块 | 职责 | 不负责 |
|------|------|--------|
| **CLI** | 参数解析、命令路由、全局选项 | 具体命令业务逻辑 |
| **research_cmd** | research 命令的完整实现 | 其他命令 |
| **plan_cmd** | plan 命令的完整实现 | 其他命令 |
| **doing_cmd** | doing 命令的完整实现（执行 Plan） | 其他命令 |
| **stat_cmd** | stat 命令的完整实现（状态监控） | 其他命令 |
| **reset_cmd** | reset 命令的完整实现（版本回滚） | 其他命令 |
| **Parser** | 通用文件解析框架（MD/JSON/YAML） | 业务逻辑 |
| **Call CLI** | 子进程管理、超时控制、信号处理 | AI CLI 业务逻辑 |
| **Executor** | Job/Task 执行调度、提示词构建、结果解析 | 底层进程调用 |
| **State** | status.json 管理、状态机、断点恢复 | 业务状态判断 |
| **Git** | Git 操作、循环提交、版本管理 | 业务逻辑 |
| **Config** | 配置结构定义、加载/保存、层级合并 | 业务配置值 |
| **Logging** | 结构化日志、日志轮转、级别控制 | 日志消费 |
| **Prompts** | 系统提示词定义（research/plan/doing） | AI 执行逻辑 |
| **Errors** | 错误码体系、错误处理规范 | 具体错误场景 |

### 模块间调用规范

```
【允许调用】
• 命令模块 → 基础组件 (Config, Logging, Parser, Call CLI, State, Git)
• 核心模块 → 基础组件
• 基础组件之间: Parser ↔ Call CLI (互相独立)

【禁止调用】
• 基础组件 → 命令模块 (避免循环依赖)
• 命令模块之间 (通过 CLI 路由解耦)
• State → Git (State 只管理状态，Git 操作由命令模块协调)

【特殊规则】
• Call CLI 调用 ai_cli 时通过环境变量 `CLAUDE_CODE_CLI` 获取命令路径
• Parser 通过工厂模式支持扩展，不直接依赖具体解析器
• CLI 路由通过接口注册命令，不依赖具体实现
```

---

---

## 技术栈

| 层级 | 技术选择 | 说明 |
|------|----------|------|
| **语言** | Go 1.21+ | 使用泛型、slog 等现代特性 |
| **日志** | slog (标准库) | 结构化日志，支持 JSON 输出 |
| **配置** | encoding/json | 原生 JSON 支持，无外部依赖 |
| **测试** | testing + testify | 标准测试框架 + 断言库 |
| **CLI 框架** | 自研 | 轻量级，满足特定需求 |
| **AI CLI** | ai_cli / claude | 通过环境变量配置 |
| **版本控制** | Git | 循环提交、版本回滚 |

---

## 扩展性设计

### 1. 解析器扩展

```go
// 添加新的解析器只需实现 Parser 接口
factory.Register(&YAMLParser{})
factory.Register(&TOMLParser{})
```

### 2. 命令扩展

```go
// 添加新命令只需注册到 CLI 路由
router.Register(cli.Command{
    Name:    "newcmd",
    Handler: NewNewCmdHandler(...).Execute,
})
```

### 3. AI CLI 适配

```go
// 通过环境变量切换不同的 AI CLI
export CLAUDE_CODE_CLI="claude"    # 使用官方 CLI
export CLAUDE_CODE_CLI="ai_cli"    # 使用本地 CLI
```

---

**下一步**: 运行 `morty doing` 开始分层 TDD 开发
