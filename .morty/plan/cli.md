# Plan: CLI

## 模块概述

**模块职责**: 实现命令行接口框架，包括参数解析、命令路由和命令注册。将具体命令实现委托给各 cmd 子模块（research_cmd, plan_cmd, doing_cmd, stat_cmd, reset_cmd）。

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.1 节 CLI 模块接口定义
- `morty-project-research.md` 第 3.1 节主入口分析

**依赖模块**: Config, Logging

**被依赖模块**: research_cmd, plan_cmd, doing_cmd, stat_cmd, reset_cmd（通过 CLI 注册到路由）

---

## 命令行调用设计

### 主入口 (cmd/morty/main.go)

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/morty/morty-go/internal/callcli"
    "github.com/morty/morty-go/internal/cli"
    "github.com/morty/morty-go/internal/cmd"
    "github.com/morty/morty-go/internal/config"
    "github.com/morty/morty-go/internal/git"
    "github.com/morty/morty-go/internal/logging"
    "github.com/morty/morty-go/internal/parser"
    "github.com/morty/morty-go/internal/state"
)

func main() {
    ctx := context.Background()

    // 初始化依赖
    cfg, err := config.NewManager()
    if err != nil {
        fmt.Fprintf(os.Stderr, "配置初始化失败: %v\n", err)
        os.Exit(1)
    }

    logger := logging.NewLogger(cfg)
    stateMgr := state.NewManager(cfg)
    gitMgr := git.NewManager(cfg)
    parserFactory := parser.NewFactory()
    cliCaller := callcli.NewAICliCaller(cfg, logger)

    // 初始化 CLI 路由
    router := cli.NewRouter()

    // 注册所有命令（委托给 cmd 包）
    cmd.RegisterAll(router, cfg, logger, stateMgr, gitMgr, parserFactory, cliCaller)

    // 解析并执行
    if err := router.Execute(ctx, os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "执行错误: %v\n", err)
        os.Exit(1)
    }
}
```

### 命令注册 (internal/cmd/register.go)

```go
package cmd

import (
    "github.com/morty/morty-go/internal/cli"
    "github.com/morty/morty-go/internal/config"
    "github.com/morty/morty-go/internal/git"
    "github.com/morty/morty-go/internal/logging"
    "github.com/morty/morty-go/internal/parser"
    "github.com/morty/morty-go/internal/state"
)

// RegisterAll 注册所有命令到 CLI 路由
func RegisterAll(
    router cli.Router,
    cfg config.Manager,
    logger logging.Logger,
    stateMgr state.Manager,
    gitMgr git.Manager,
    parserFactory parser.Factory,
    cliCaller callcli.AICliCaller,
) {
    // research 命令 - 研究模式（交互式）
    router.Register(cli.Command{
        Name:        "research",
        Description: "启动研究模式，交互式研究指定主题",
        Handler:     NewResearchHandler(cfg, logger, cliCaller).Execute,
        Options: []cli.Option{
            {Name: "--topic", Short: "-t", Description: "研究主题", HasValue: true},
        },
    })

    // plan 命令 - 规划模式（交互式）
    router.Register(cli.Command{
        Name:        "plan",
        Description: "启动规划模式，基于研究结果生成 Plan",
        Handler:     NewPlanHandler(cfg, logger, parserFactory, cliCaller).Execute,
        Options: []cli.Option{
            {Name: "--force", Short: "-f", Description: "强制重新生成", HasValue: false},
        },
    })

    // doing 命令 - 执行 Plan
    router.Register(cli.Command{
        Name:        "doing",
        Description: "执行开发计划",
        Handler:     NewDoingHandler(cfg, logger, stateMgr, gitMgr, parserFactory, cliCaller).Execute,
        Options: []cli.Option{
            {Name: "--restart", Short: "-r", Description: "重置后执行", HasValue: false},
            {Name: "--module", Short: "-m", Description: "指定模块", HasValue: true},
            {Name: "--job", Short: "-j", Description: "指定 Job", HasValue: true},
        },
    })

    // stat 命令 - 状态监控
    router.Register(cli.Command{
        Name:        "stat",
        Description: "显示执行状态和进度",
        Handler:     NewStatHandler(cfg, logger, stateMgr, gitMgr, parserFactory).Execute,
        Options: []cli.Option{
            {Name: "--watch", Short: "-w", Description: "监控模式", HasValue: false},
            {Name: "--json", Short: "-j", Description: "JSON格式输出", HasValue: false},
        },
    })

    // reset 命令 - 版本回滚
    router.Register(cli.Command{
        Name:        "reset",
        Description: "版本管理和回滚",
        Handler:     NewResetHandler(cfg, logger, stateMgr, gitMgr).Execute,
        Options: []cli.Option{
            {Name: "--list", Short: "-l", Description: "显示历史", HasValue: true},
            {Name: "--commit", Short: "-c", Description: "回滚到提交", HasValue: true},
        },
    })

    // version 命令 - 显示版本
    router.Register(cli.Command{
        Name:        "version",
        Description: "显示版本信息",
        Handler:     NewVersionHandler().Execute,
    })

    // help 命令 - 帮助信息
    router.Register(cli.Command{
        Name:        "help",
        Description: "显示帮助信息",
        Handler:     NewHelpHandler(router).Execute,
    })
}
```

### 命令行用法

```bash
# research - 研究模式（交互式）
morty research                        # 交互式输入主题
morty research morty-architecture     # 直接指定主题

# plan - 规划模式（交互式）
morty plan                            # 启动规划模式
morty plan --force                    # 强制重新生成

# doing - 执行 Plan
morty doing                           # 执行下一个未完成的 Job
morty doing --module cli              # 仅执行 cli 模块
morty doing --module cli --job job_1  # 仅执行指定 Job
morty doing --restart                 # 重置后执行

# stat - 状态监控
morty stat                            # 显示当前状态
morty stat -w                         # 监控模式，60s刷新
morty stat --json                     # JSON格式输出

# reset - 版本回滚
morty reset -l                        # 显示最近10次提交
morty reset -l 5                      # 显示最近5次提交
morty reset -c abc1234                # 回滚到指定提交

# version - 显示版本
morty version

# help - 帮助
morty help                            # 全局帮助
morty help doing                      # doing 命令帮助
```

---

## 数据模型

```go
// Command 表示一个 CLI 命令
type Command struct {
    Name        string
    Description string
    Handler     CommandHandler
    Options     []Option
}

type CommandHandler func(ctx context.Context, args []string, opts ParseResult) error

type Option struct {
    Name        string
    Short       string
    Description string
    HasValue    bool
    Required    bool
}

type ParseResult struct {
    Command    string
    Positional []string
    Options    map[string]string
    Flags      map[string]bool
}

// Router CLI 路由接口
type Router interface {
    Register(cmd Command) error
    Execute(ctx context.Context, args []string) error
    GetHandler(name string) (CommandHandler, bool)
    ListCommands() []Command
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: 参数解析系统

**目标**: 实现命令行参数解析，支持全局选项和命令选项

**前置条件**:
- Config 模块完成
- Logging 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/cli/parser.go` 文件
- [ ] Task 2: 实现 `Parse(args []string) (*ParseResult, error)` 方法
- [ ] Task 3: 支持长选项 (`--option`) 和短选项 (`-o`)
- [ ] Task 4: 支持带值的选项 (`--option value` 或 `--option=value`)
- [ ] Task 5: 支持布尔标志 (`--flag`)
- [ ] Task 6: 支持位置参数
- [ ] Task 7: 处理 `--` 终止选项解析
- [ ] Task 8: 编写单元测试 `parser_test.go`

**验证器**:
- [ ] 正确解析命令名称
- [ ] 正确解析长选项和短选项
- [ ] 正确解析带值的选项
- [ ] 正确解析布尔标志
- [ ] 正确处理 `--` 后的参数
- [ ] 错误选项时返回友好错误信息
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 2: 命令路由系统

**目标**: 实现命令注册和路由分发功能

**前置条件**:
- Job 1 完成 (参数解析)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/cli/router.go` 文件
- [ ] Task 2: 实现 `Router` 结构体
- [ ] Task 3: 实现 `Register(cmd Command) error` 方法
- [ ] Task 4: 实现 `Execute(ctx context.Context, args []string) error` 方法
- [ ] Task 5: 实现 `GetHandler(name string) (CommandHandler, bool)` 方法
- [ ] Task 6: 实现 `ListCommands() []Command` 方法
- [ ] Task 7: 对未知命令返回友好错误信息
- [ ] Task 8: 编写单元测试 `router_test.go`

**验证器**:
- [ ] 注册命令后能通过名称获取 Handler
- [ ] 路由到已注册命令时正确执行 Handler
- [ ] 路由到未注册命令时返回错误 "未知命令: xxx"
- [ ] `ListCommands()` 返回所有已注册命令
- [ ] 命令名称不区分大小写
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: 全局选项处理

**目标**: 实现全局选项处理机制

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 定义全局选项常量 (`--verbose`, `--debug`)
- [ ] Task 2: 实现全局选项解析和存储
- [ ] Task 3: 实现 `GetGlobalOptions() GlobalOptions`
- [ ] Task 4: 集成 `--verbose` 到日志系统
- [ ] Task 5: 集成 `--debug` 到日志系统
- [ ] Task 6: 编写单元测试

**验证器**:
- [ ] `--verbose` 开启详细日志输出
- [ ] `--debug` 开启调试模式
- [ ] 全局选项可在任何命令前使用
- [ ] 全局选项传递给 Logger
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的命令生命周期: 解析 → 路由 → 执行
- [ ] 支持命令: `morty version`, `morty help`, `morty help doing`
- [ ] 复杂参数组合正确解析和执行
- [ ] 全局选项正确传递给各模块
- [ ] 错误处理机制正常工作 (友好错误信息)
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 模块依赖关系

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Module                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   CLIParser │  │   Router    │  │  Global Options     │  │
│  │ (参数解析)   │  │ (命令路由)   │  │  (全局选项处理)      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  research_cmd │    │   plan_cmd    │    │   doing_cmd   │
│   (研究命令)   │    │   (规划命令)   │    │   (执行命令)   │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   stat_cmd    │    │   reset_cmd   │    │   version     │
│   (状态命令)   │    │   (回滚命令)   │    │    help       │
└───────────────┘    └───────────────┘    └───────────────┘
```

---

## 文件清单

- `internal/cli/parser.go` - 参数解析实现
- `internal/cli/router.go` - 命令路由实现
- `internal/cli/options.go` - 全局选项处理
- `internal/cli/command.go` - Command 结构体定义
- `internal/cmd/register.go` - 命令注册入口

---

## 使用示例

```bash
# 解析示例
$ morty doing --module cli --job job_1 --restart
ParseResult:
  Command: "doing"
  Options: map[string]string{"module": "cli", "job": "job_1"}
  Flags:   map[string]bool{"restart": true}

# 未知命令
$ morty unknown
错误: 未知命令: unknown
可用命令: research, plan, doing, stat, reset, version, help

# 帮助
$ morty help
可用命令:
  research    启动研究模式
  plan        启动规划模式
  doing       执行开发计划
  stat        显示执行状态
  reset       版本管理和回滚
  version     显示版本信息
  help        显示帮助信息
```
