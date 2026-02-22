# Plan: cli

## 模块概述

**模块职责**: 提供统一的命令行接口，实现命令路由与分发，支持所有 Morty 模式（doing/stat/reset/version）。stat 命令作为项目监控大盘，基于 logs 和 status.json 展示状态。

**对应 Research**: 主入口 morty；命令路由与分发

**依赖模块**: config, logging, version_manager, doing

**被依赖模块**: 无（顶层模块）

## 接口定义

### 输入接口
- 命令行参数: `morty <command> [options]`
- 环境变量: `MORTY_HOME`, `CLAUDE_CODE_CLI` 等

### 输出接口
- 标准输出: 命令执行结果
- 标准错误: 错误信息
- 退出码: 0=成功, 1=一般错误, 2=无效参数, 3=前置条件不满足

### 支持命令

#### morty doing - 执行模式
```bash
morty doing [options]
  --restart                 # 重置模式（见下方详细说明）
  --module <name>           # 指定模块
  --job <name>              # 指定 Job（必须与 --module 一起使用）
```

**执行模式详解**:

| 命令 | 行为 |
|------|------|
| `morty doing` | 执行下一个未完成的 Job |
| `morty doing --module config` | 仅执行 config 模块中的下一个未完成 Job |
| `morty doing --module config --job job_1` | 仅执行指定的单个 Job |
| `morty doing --restart` | 完全重置：重置所有 Job 状态，从第一个开始 |
| `morty doing --restart --module config` | 重置 config 模块，从该模块第一个 Job 开始 |
| `morty doing --restart --module config --job job_1` | 重置并重新执行指定 Job |

**重要说明**:
- `--restart` 仅重置 `status.json` 状态，**不**进行 git 重置，保留工作变更
- 每次 `doing` 执行一个 Job 后退出（单循环模式）
- Job 是 AI coding agent 执行的最小任务单元
- Task 是 agent 内部 todo list 的最小执行单元

#### morty stat - 监控大盘
```bash
morty stat                  # 默认输出，自动每60s刷新
  -t                        # 表格形式输出（默认）
  -w                        # 监控模式，原地刷新（默认60s）
```

**展示信息**:
1. **当前执行**: 正在执行的模块和 Job
2. **上一个 Job**: 完成摘要
3. **Debug 问题**: 当前 Job loop 循环中需要解决的问题
4. **整体进度**: doing 完成百分比
5. **累计时间**: 当前 Job 运行时间

#### morty reset - 版本回滚
```bash
morty reset -l [N]          # 显示最近 N 次循环提交（默认10，表格形式）
morty reset -c <commit_id>  # 回滚到指定提交
```

**输出格式**:
```
┌──────────┬──────────────────────────────────────┬─────────────┐
│ CommitID │ Message                              │ Time        │
├──────────┼──────────────────────────────────────┼─────────────┤
│ abc1234  │ morty: Loop #15 - [doing/job_3: COMPLETED]  │ 2026-02-21  │
│ def5678  │ morty: Loop #14 - [doing/job_2: COMPLETED]  │ 2026-02-21  │
│ ghi9012  │ morty: Loop #13 - [doing/job_1: FAILED]     │ 2026-02-20  │
└──────────┴──────────────────────────────────────┴─────────────┘
```

#### 其他命令
```bash
morty version               # 显示版本
morty help [command]        # 帮助信息
```

## 数据模型

### 命令路由表
```yaml
commands:
  doing:
    handler: "morty_doing.sh"
    description: "执行开发计划"
    options: ["--restart", "--module", "--job"]

  stat:
    handler: "morty_stat.sh"
    description: "显示执行状态和进度（监控大盘）"
    options: ["-t", "-w"]

  reset:
    handler: "morty_reset.sh"
    description: "版本管理和回滚"
    options: ["-l", "-c"]

  version:
    handler: "show_version"
    description: "显示版本信息"

  help:
    handler: "show_help"
    description: "显示帮助信息"
```

### 版本信息
```
mort 2.0.0
上下文优先的AI Coding Agent 编排框架
```

## Jobs (Loop 块列表)

---

### Job 1: 命令路由系统

**目标**: 建立命令路由系统，支持命令注册、解析和分发

**前置条件**: config, logging 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty` 主命令脚本
- [ ] 实现 `cli_register_command()`: 命令注册
- [ ] 实现 `cli_parse_args()`: 参数解析
- [ ] 实现 `cli_route()`: 命令路由
- [ ] 实现 `cli_execute()`: 命令执行

**验证器**:
- 输入 `morty doing` 应调用 `morty_doing.sh`
- 输入 `morty doing --restart --module config` 应传递所有参数
- 输入 `morty stat` 应调用 `morty_stat.sh`
- 输入 `morty stat -w` 应进入监控模式
- 输入 `morty reset` 应调用 `morty_reset.sh`
- 输入未知命令时应显示错误和帮助信息
- 参数解析应正确处理选项和位置参数

**调试日志**:
- 无

---

### Job 2: stat 监控大盘

**目标**: 实现 `morty stat` 命令，基于 logs 和 status.json 展示项目监控大盘

**前置条件**: doing 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_stat.sh` 脚本
- [ ] 实现 `stat_load_status()`: 读取 status.json
- [ ] 实现 `stat_load_logs()`: 读取最新日志摘要
- [ ] 实现 `stat_get_current_job()`: 获取当前执行的模块和 Job
- [ ] 实现 `stat_get_previous_job()`: 获取上一个完成的 Job 摘要
- [ ] 实现 `stat_get_debug_issues()`: 获取当前 Job 的 debug 问题
- [ ] 实现 `stat_get_progress()`: 计算整体完成进度
- [ ] 实现 `stat_get_elapsed_time()`: 计算当前 Job 累计运行时间
- [ ] 实现 `stat_format_table()`: 表格形式格式化输出
- [ ] 实现 `stat_watch_mode()`: 监控模式，原地刷新（默认60s）

**验证器**:
- `morty stat` 应显示：当前执行、上一个 Job 摘要、Debug 问题、整体进度、累计时间
- 当 `.morty/status.json` 不存在时，应提示 "请先运行 morty doing"
- `-t` 选项应以表格形式输出（默认行为）
- `-w` 选项应进入监控模式，每 60s 原地刷新一次
- 应正确计算并显示整体进度百分比
- 应显示当前 Job 的累计运行时间（格式：HH:MM:SS）
- 应显示上一个完成 Job 的摘要（模块/Job/状态/耗时）
- 应显示当前 Job 需要 debug 的问题列表
- Ctrl+C 应能优雅退出监控模式

**调试日志**:
- 无

---

### Job 3: reset 命令

**目标**: 实现 `morty reset` 命令，表格形式输出循环历史，简洁无多余日志

**前置条件**: version_manager 模块完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_reset.sh` 脚本
- [ ] 实现 `reset_show_history()`: 显示最近 N 次循环提交（表格形式）
- [ ] 实现 `reset_to_commit()`: 回滚到指定提交
- [ ] 集成 `version_show_loop_history()`
- [ ] 集成 `version_reset_to_commit()`
- [ ] 实现回滚确认提示
- [ ] 实现表格格式化输出（CommitID | Message | Time）

**验证器**:
- `morty reset -l` 应表格形式显示最近 10 次循环提交
- `morty reset -l 5` 应表格形式显示最近 5 次循环提交
- 输出应只包含：CommitID、提交信息、时间（无其他日志）
- 表格应规范对齐，易于阅读
- `morty reset -c abc123` 应回滚到指定 commit
- 回滚前应提示用户确认
- 回滚后应显示新的当前状态

**调试日志**:
- 无

---

### Job 4: 帮助系统

**目标**: 实现完善的帮助系统

**前置条件**: Job 1, 2, 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `cli_show_global_help()`: 全局帮助
- [ ] 实现 `cli_show_command_help(cmd)`: 命令帮助
- [ ] 实现帮助文档自动生成

**验证器**:
- `morty help` 应显示所有可用命令列表
- `morty help doing` 应显示 doing 命令的详细用法（包含所有选项组合）
- `morty help stat` 应显示 stat 命令的详细用法
- 每个命令的帮助应包含描述、用法、选项、示例
- 帮助信息应格式清晰，易于阅读

**调试日志**:
- 无

---

### Job 5: 版本和诊断

**目标**: 实现版本显示和诊断模式

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `cli_show_version()`: 显示版本信息
- [ ] 实现诊断模式（--verbose, --debug）
- [ ] 实现前置条件检查（依赖检查）

**验证器**:
- `morty version` 应显示版本号
- 使用 `--verbose` 时应显示详细的执行过程
- 缺少依赖时应显示友好的安装指导

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 所有命令可以正确路由和执行
- `morty doing` 的各种选项组合工作正确
- `morty stat` 可以正确显示监控大盘信息（当前执行、上一个 Job、Debug 问题、进度、时间）
- `morty stat -w` 监控模式每 60s 刷新正确
- `morty reset -l` 以表格形式输出，无多余日志
- `morty reset -c` 可以正确回滚版本
- 帮助信息完整且准确

---

## 待实现方法签名

```bash
# morty (主命令)

# 命令路由
cli_register_command(name, handler, description)
cli_parse_args(args)
cli_route(command, args)
cli_execute(handler, args)

# stat 命令
stat_load_status()
stat_load_logs(limit=10)
stat_get_current_job()
stat_get_previous_job()
stat_get_debug_issues()
stat_get_progress()
stat_get_elapsed_time()
stat_format_table()
stat_watch_mode(interval=60)

# reset 命令
reset_show_history(n=10)
reset_format_table(commits)
reset_to_commit(commit_id)

# 帮助
cli_show_global_help()
cli_show_command_help(command)

# 版本和诊断
cli_show_version()
cli_check_dependencies()

# 错误处理
cli_error(message, code=1)
cli_verbose(message)
```

---

## stat 命令输出示例

### 默认/表格输出
```
$ morty stat

┌─────────────────────────────────────────────────────────────┐
│                     Morty 监控大盘                           │
├─────────────────────────────────────────────────────────────┤
│ 当前执行                                                    │
│   模块: logging                                             │
│   Job:  job_2 (日志轮转和归档)                               │
│   状态: RUNNING (第2次循环)                                  │
│   累计时间: 00:32:15                                        │
├─────────────────────────────────────────────────────────────┤
│ 上一个 Job                                                  │
│   config/job_1: COMPLETED (耗时 00:15:30)                   │
│   摘要: 日志系统核心框架实现完成                              │
├─────────────────────────────────────────────────────────────┤
│ Debug 问题 (当前 Job)                                       │
│   • 日志轮转时丢失消息 (loop 2)                              │
│     猜想: 文件句柄未正确同步, 并发写入竞争                    │
│     状态: 待修复                                             │
├─────────────────────────────────────────────────────────────┤
│ 整体进度                                                    │
│   [████░░░░░░░░░░░░░░░░] 20% (5/25 Jobs)                    │
│   已完成: config (3/3)                                       │
│   进行中: logging (2/4)                                      │
│   待开始: version_manager, doing, cli                       │
└─────────────────────────────────────────────────────────────┘

自动刷新: 60s (按 Ctrl+C 退出)
```

### 监控模式 (-w)
```bash
$ morty stat -w
[同上界面，每60s原地刷新]
```

---

## reset 命令输出示例

### 显示历史 (-l)
```bash
$ morty reset -l

┌──────────┬───────────────────────────────────────────┬─────────────────────┐
│ CommitID │ Message                                   │ Time                │
├──────────┼───────────────────────────────────────────┼─────────────────────┤
│ abc1234  │ morty: Loop #15 - [doing/job_3: COMPLETED]│ 2026-02-21 14:30:00 │
│ def5678  │ morty: Loop #14 - [doing/job_2: COMPLETED]│ 2026-02-21 14:15:00 │
│ ghi9012  │ morty: Loop #13 - [doing/job_1: FAILED]   │ 2026-02-21 14:00:00 │
│ jkl3456  │ morty: Loop #12 - [cli/job_5: COMPLETED]  │ 2026-02-21 13:45:00 │
│ mno7890  │ morty: Loop #11 - [cli/job_4: COMPLETED]  │ 2026-02-21 13:30:00 │
└──────────┴───────────────────────────────────────────┴─────────────────────┘

$ morty reset -l 3

┌──────────┬───────────────────────────────────────────┬─────────────────────┐
│ CommitID │ Message                                   │ Time                │
├──────────┼───────────────────────────────────────────┼─────────────────────┤
│ abc1234  │ morty: Loop #15 - [doing/job_3: COMPLETED]│ 2026-02-21 14:30:00 │
│ def5678  │ morty: Loop #14 - [doing/job_2: COMPLETED]│ 2026-02-21 14:15:00 │
│ ghi9012  │ morty: Loop #13 - [doing/job_1: FAILED]   │ 2026-02-21 14:00:00 │
└──────────┴───────────────────────────────────────────┴─────────────────────┘
```

### 回滚到指定提交 (-c)
```bash
$ morty reset -c abc1234

确认回滚到 commit abc1234?
这将重置工作目录到该提交状态。
[Y/n]: y

已回滚到 commit abc1234
当前状态: doing/job_2 PENDING
```

---

## doing 命令帮助示例

```
$ morty help doing

morty doing - 执行开发计划

用法:
  morty doing [options]

选项:
  --restart                 重置模式，重置 Job 状态但不重置 git
  --module <name>           指定模块
  --job <name>              指定 Job（必须与 --module 一起使用）

示例:
  # 执行下一个未完成的 Job
  morty doing

  # 仅执行 config 模块的下一个未完成 Job
  morty doing --module config

  # 仅执行指定的单个 Job
  morty doing --module config --job job_1

  # 完全重置，从第一个 Job 开始
  morty doing --restart

  # 重置并重新执行 config 模块
  morty doing --restart --module config

  # 重置并重新执行指定 Job
  morty doing --restart --module config --job job_1

说明:
  - Job 是 AI coding agent 执行的最小任务单元
  - Task 是 agent 内部 todo list 的最小执行单元
  - 每次 doing 执行一个 Job 后退出（单循环模式）
  - --restart 仅重置 status.json，保留 git 历史和工作变更
```

---

## stat 命令帮助示例

```
$ morty help stat

morty stat - 显示项目监控大盘

用法:
  morty stat [options]

选项:
  -t                        表格形式输出（默认）
  -w                        监控模式，原地刷新（默认60s）

说明:
  显示信息包括：
  - 当前正在执行的模块和 Job
  - 上一个 Job 的完成摘要
  - 当前 Job loop 循环中需要 debug 的问题
  - 整体 doing 的完成进度
  - 当前 Job 运行的累计时间

  默认每60秒自动刷新一次。

示例:
  # 显示监控大盘
  morty stat

  # 表格形式输出
  morty stat -t

  # 监控模式，持续刷新
  morty stat -w
```

---

## reset 命令帮助示例

```
$ morty help reset

morty reset - 版本管理和回滚

用法:
  morty reset -l [N]        显示最近 N 次循环提交（默认10）
  morty reset -c <commit_id> 回滚到指定提交

选项:
  -l [N]                    显示循环历史，表格形式输出
  -c <commit_id>            回滚到指定 commit

示例:
  # 显示最近10次循环提交
  morty reset -l

  # 显示最近5次循环提交
  morty reset -l 5

  # 回滚到指定 commit
  morty reset -c abc1234

说明:
  -l 选项以表格形式输出，包含 CommitID、Message、Time
  -c 选项会提示确认，回滚后当前 Job 状态将重置
```
