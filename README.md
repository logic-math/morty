# Morty

上下文优先的 AI Coding Agent 编排框架

## 概述

Morty 是一个上下文优先的 AI 开发系统,帮助你:
1. **Research 研究** - 深入理解问题和需求
2. **Plan 规划** - 制定模块化的开发计划
3. **Doing 执行** - 基于 Plan 执行分层 TDD 开发
4. **版本管理** - Git 自动提交和回滚支持

## 核心特性

### 🔬 Research 模式 - 深度研究
- 启动交互式 Claude Code 会话
- 深入理解问题空间
- 记录研究事实到 `.morty/research/`
- 为 Plan 阶段提供知识基础

### 📋 Plan 模式 - 结构化规划
- 基于 Research 结果制定开发计划
- 模块化设计，支持分层开发
- 生成 `.morty/plan/*.md` 计划文档
- 定义清晰的 Jobs 和 Tasks

### 🚀 Doing 模式 - 执行开发
- 执行 Plan 制定的开发计划
- 支持分层 TDD 开发范式
- 自动状态管理和断点恢复
- Job 级别 Git 自动提交

### 🔄 版本管理(Git 自动提交)
- 自动 Git 初始化(首次运行时)
- 每个 Job 完成后自动创建 commit
- 完整的变更历史记录
- 支持回滚到任意状态
- 支持人工干预后继续执行

### 📁 项目管理
- 在现有项目中启用 Morty
- 自动检测项目类型
- 生成构建/测试命令
- 在 `.morty/` 目录中维护完整上下文

## Installation

### 一键安装（推荐）

```bash
curl -sSL https://get.morty.dev | bash
```

### 本地安装

```bash
cd morty
./bootstrap.sh install
```

### 自定义路径安装

```bash
./bootstrap.sh install --prefix /opt/morty --bin-dir /usr/local/bin
```

Ensure `~/.local/bin` (or your custom bin dir) is in your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

## 快速开始

### 步骤 1: Research 研究

```bash
morty research "创建一个命令行 todo 应用"
```

这会启动一个 **交互式 Claude Code 会话**:
- Claude 分析你的需求
- 提出澄清问题
- 深入探索问题空间
- 记录研究事实到 `.morty/research/`

### 步骤 2: Plan 规划

```bash
morty plan
```

基于 Research 结果制定开发计划:
- 模块化设计
- 定义 Jobs 和 Tasks
- 生成 `.morty/plan/*.md`

### 步骤 3: Doing 执行

```bash
morty doing
```

执行开发计划:
- 按顺序执行 Jobs
- 支持断点自动恢复
- 每个 Job 完成后自动提交
- 实时显示执行状态

## 命令

### `morty research <topic>`
研究模式 - 深入理解问题空间。

**功能:**
1. 使用 research 模式系统提示词启动 Claude Code
2. 通过对话深入理解需求
3. 记录研究事实到 `.morty/research/`
4. 为 Plan 阶段提供知识基础

**示例:**
```bash
morty research "创建一个 REST API"
morty research "优化数据库查询性能"
```

### `morty plan [options]`
规划模式 - 制定结构化开发计划。

**功能:**
- 读取 `.morty/research/` 中的研究结果
- 制定模块化的开发计划
- 生成 `.morty/plan/[模块名].md`
- 定义清晰的 Jobs 和 Tasks

**示例:**
```bash
morty plan                      # 基于 research 生成计划
```

### `morty doing [options]`
执行模式 - 执行开发计划。

**功能:**
- 读取 `.morty/plan/*.md` 中的开发计划
- 按顺序逐个执行 Job
- 支持断点自动恢复
- 分层 TDD 开发（单元测试 → 集成测试 → 端到端测试）

**选项:**
- `--module <name>` - 只执行指定模块
- `--job <name>` - 只执行指定 Job
- `--restart` - 强制从头开始（忽略已有状态）

**示例:**
```bash
morty doing                     # 执行所有待完成的 Jobs
morty doing --module install    # 只执行 install 模块
morty doing --job job_1         # 只执行 job_1
morty doing --restart           # 强制重新开始
```

### `morty reset [options]`
版本回滚和循环管理。

**功能:**
- 查看循环提交历史
- 回滚到指定 commit
- 关闭运行中的 tmux 会话
- 保留所有日志文件
- 支持人工干预后继续循环

**选项:**
- `-l, --list [N]` - 显示最近 N 次循环提交(默认: 20)
- `-c, --commit <id>` - 回滚到指定 commit
- `-s, --status` - 显示当前状态

**示例:**
```bash
morty reset -l              # 查看循环提交历史
morty reset -c abc123       # 回滚到 commit abc123
morty reset -s              # 查看当前状态
```

**工作流程:**
1. 运行 `morty reset -l` 查看历史
2. 找到目标 commit ID
3. 运行 `morty reset -c <commit-id>` 回滚
4. 可选: 手动修改代码进行干预
5. 运行 `morty doing` 从当前状态继续

## Git Auto-Commit

Morty automatically commits changes after each successful loop iteration:

**Features:**
- **Auto-commit after each job**: Creates a snapshot with job metadata
- **Rollback capability**: Use `morty reset <commit>` to revert to any state
- **Job history**: Use `morty reset -l` to view all job commits
- **Commit metadata**: Each commit includes:
  - Job name
  - Timestamp (ISO format)
  - Task completion status
  - Auto-commit marker

**Example commit message:**
```
feat(install): complete Job 3 - installation functions

- Implemented bootstrap_cmd_install()
- Implemented bootstrap_cmd_reinstall()
- Added config backup and restore functionality

Job: install/job_3
Tasks: 6/6 completed
Timestamp: 2024-01-15T10:30:45Z

🤖 Generated with Claude Code
```

**Benefits:**
- **Safety**: Every loop creates a restore point
- **Debugging**: Easily identify when issues were introduced
- **Experimentation**: Try changes knowing you can rollback
- **Transparency**: Clear history of what Morty did in each loop

**Requirements:**
- Project must be a git repository
- Git must be installed and available

**Notes:**
- Only commits if there are changes (doesn't create empty commits)
- Commits are local (not pushed to remote)
- Uses `git add -A` to stage all changes

## Workflow Deep Dive

### How Morty Works

Morty uses a 3-phase workflow:

1. **Research** - Understand the problem space
2. **Plan** - Create structured development plans
3. **Doing** - Execute plans with state management

### Research Mode

Research mode uses a system prompt that enables Claude Code to:

1. **Deep Exploration**
   - Ask probing questions
   - Challenge assumptions
   - Explore edge cases
   - Identify dependencies

2. **Knowledge Recording**
   - Record facts to `.morty/research/`
   - Maintain research context
   - Build domain understanding

### Plan Mode

Plan mode creates structured development plans:

1. **Modular Design**
   - Break down into modules
   - Define Jobs and Tasks
   - Set clear dependencies

2. **Output Structure**
   - Generates `.morty/plan/[module].md`
   - Defines validation criteria
   - Creates executable specifications

### Doing Mode

Doing mode executes the plan:

1. **State Management**
   - Track task completion in `.morty/status.json`
   - Support breakpoint resume
   - Handle failures and retries

2. **Git Integration**
   - Auto-commit after each Job
   - Support rollback to any state
   - Preserve full history

## Project Structure

After running `morty research` and `morty plan`, you get:

```
my-project/
├── .morty/
│   ├── status.json            # Execution state
│   ├── research/              # Research facts
│   │   └── *.md               # Research documents
│   ├── plan/                  # Development plans
│   │   └── [module].md        # Module plans
│   ├── doing/                 # Execution context
│   │   └── logs/              # Execution logs
│   └── logs/                  # System logs
├── src/                       # Source code
├── README.md
└── .gitignore
```

### Key Files

**`.morty/status.json`**
- Current execution state
- Task completion tracking
- Module and Job status
- Debug logs

**`.morty/research/*.md`**
- Research findings
- Problem understanding
- Technical constraints
- Domain knowledge

**`.morty/plan/*.md`**
- Module development plans
- Jobs and Tasks definition
- Validation criteria
- Dependencies

**`.morty/doing/logs/`**
- Execution logs
- Prompt and output history
- Error logs

## Development Workflow

```
research → plan → doing
   ↑         ↑      |
   └────────┴──────┘
```

**States:**
- **research**: Understanding the problem space
- **plan**: Creating structured development plans
- **doing**: Executing plans with state management

**Exit Conditions:**
- All Jobs completed
- Error detected (with retry logic)
- User interrupt

## 状态监控

### `morty stat` - 监控大盘

显示当前执行状态和进度:

```bash
morty stat
```

**显示内容:**
- 当前模块和 Job
- Task 完成进度
- 整体完成百分比
- 最近的执行日志

**特性:**
- 自动刷新（可配置间隔）
- 彩色输出
- 简洁摘要或详细视图

## Configuration

**Environment Variables:**
```bash
# Custom Claude Code CLI command (default: "claude")
export CLAUDE_CODE_CLI="ai_cli"     # Use your custom CLI wrapper

# Loop configuration
export MAX_LOOPS=100                # Maximum iterations (default: 50)
export LOOP_DELAY=10                # Seconds between loops (default: 5)
```

**Example: Using Custom CLI Wrapper**
```bash
# If you have a custom enterprise CLI wrapper
export CLAUDE_CODE_CLI="/path/to/ai_cli"

# Or with additional configuration
export CLAUDE_CODE_CLI="ai_cli --config enterprise"

# Then use Morty normally
morty research "your topic"
morty plan
morty doing
```

**Project Files:**
- `.morty/status.json` - View and manage execution state
- `.morty/plan/*.md` - Review and modify development plans
- `.morty/doing/logs/` - Review execution history

## Requirements

- Bash 4.0+
- Claude Code CLI (`claude` command)
- tmux (optional, for monitoring)
- jq (optional, for status display)
- Git

## Testing

```bash
# Run all tests
./tests/run_all_tests.sh

# Or run individual tests
./tests/test_git_autocommit.sh     # Git auto-commit tests
./tests/test_json_logging.sh       # JSON logging tests
```

See [tests/README.md](tests/README.md) for detailed test documentation.

## Examples

### Example 1: Web API Project

```bash
# Research the problem space
morty research "Create a REST API for a blogging platform"

# Claude will ask questions like:
# - What's the expected load?
# - How should comments be moderated?
# - What's the permission model?
# - Should we support markdown?

# After research, create the plan
morty plan

# Execute the development plan
morty doing
```

### Example 2: CLI Tool

```bash
# Research
morty research "Build a CLI tool to organize files by rules"

# Plan
morty plan

# Execute
morty doing
```

## Tips

1. **Research first** - Spend time understanding the problem before planning
2. **Be specific in dialogue** - Answer Claude's questions thoughtfully
3. **Review generated plans** - Customize `.morty/plan/*.md` as needed
4. **Monitor progress** - Use `morty stat` to check execution status
5. **Check logs** - `.morty/doing/logs/` for detailed execution history
6. **Use reset when needed** - `morty reset` to rollback if something goes wrong

## Troubleshooting

### "Claude command not found"
Install Claude Code CLI:
```bash
npm install -g @anthropic-ai/claude-code
```

### Plan mode doesn't start
Ensure:
- Research phase is completed (`.morty/research/` exists)
- Claude CLI is installed
- `prompts/plan.md` exists

### Project not generated
Check if Claude created `problem_description.md` in the working directory during plan mode.

## Architecture

**Core Components:**
- `morty` - Main command router
- `morty_research.sh` - Research mode implementation
- `morty_plan.sh` - Plan mode implementation
- `morty_doing.sh` - Doing mode execution
- `morty_reset.sh` - Version management and rollback
- `lib/common.sh` - Shared utilities
- `lib/config.sh` - Configuration management
- `lib/logging.sh` - Logging system
- `lib/version_manager.sh` - Git integration
- `prompts/research.md` - Research mode system prompt
- `prompts/plan.md` - Plan mode system prompt
- `prompts/doing.md` - Doing mode system prompt
- `bootstrap.sh` - Installation script

**Design Principles:**
- Context-first over prompt-first
- Structured workflow over free-form
- State management over stateless
- Modular design over monolithic

## License

MIT License

## Documentation

For detailed documentation, see the `docs/` directory:

- **[Configuration Guide](docs/CONFIGURATION.md)** - Environment variables and project configuration
- **[Plan Mode Guide](docs/PLAN_MODE_GUIDE.md)** - Comprehensive guide to interactive PRD refinement
- **[Git Auto-Commit Feature](docs/GIT_AUTOCOMMIT_FEATURE.md)** - Loop rollback and history management
- **[Changelog](docs/CHANGELOG.md)** - Version history and migration guides

## Acknowledgments

Inspired by [Ralph for Claude Code](https://github.com/frankbria/ralph-claude-code) by Frank Bria.

---

**Version**: 2.0.0 (Context-First Framework)
**Status**: Production Ready
