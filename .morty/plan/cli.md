# Plan: cli

## 模块概述

**模块职责**: 提供统一的命令行接口，实现命令路由与分发，支持所有 Morty 模式（research/plan/doing/stat/reset）。

**对应 Research**: 主入口 morty；命令路由与分发

**依赖模块**: config, logging, git_manager, research, plan_mode, doing

**被依赖模块**: 无（顶层模块）

## 接口定义

### 输入接口
- 命令行参数: `morty <command> [options]`
- 标准输入: 交互式输入
- 环境变量: `MORTY_HOME`, `CLAUDE_CODE_CLI` 等

### 输出接口
- 标准输出: 命令执行结果
- 标准错误: 错误信息
- 退出码: 0=成功, 1=一般错误, 2=无效参数, 3=前置条件不满足

### 支持命令
```bash
morty research <topic>      # 研究模式
morty plan [topic]          # 规划模式
morty doing [options]       # 执行模式（默认从断点自动恢复）
  --module <name>           # 指定模块
  --job <name>              # 指定 Job
  --restart                 # 强制从头开始执行
morty stat                  # 显示执行状态和进度
  --json                    # JSON 格式输出
  --watch                   # 持续监控
morty reset [options]       # 版本回滚
  -l [N]                    # 显示历史
  -c <id>                   # 回滚到提交
  -s                        # 显示状态
morty init                  # 初始化配置
morty config <key>=<value>   # 设置配置项
morty config get <key>       # 获取配置项
morty config list            # 列出所有配置
morty version               # 显示版本
morty help [command]        # 帮助信息
```

## 数据模型

### 命令路由表
```yaml
commands:
  research:
    handler: "morty_research.sh"
    description: "交互式代码库/文档库研究"
    args: ["topic"]
  plan:
    handler: "morty_plan.sh"
    description: "基于研究结果生成开发计划"
    args: ["topic?"]
  doing:
    handler: "morty_doing.sh"
    description: "执行开发计划（自动断点恢复）"
    options: ["--module", "--job", "--restart"]
  stat:
    handler: "morty_stat.sh"
    description: "显示开发执行状态和进度"
    options: ["--json", "--watch"]
  reset:
    handler: "morty_reset.sh"
    description: "版本管理和回滚"
    options: ["-l", "-c", "-s"]
  init:
    handler: "config_init"
    description: "初始化用户配置"
  config:
    handler: "config_cli"
    description: "配置管理 (k=v 语法设置)"
    examples:
      - "morty config ai_cli='mc --code'"
      - "morty config max_loops=100"
      - "morty config get ai_cli"
      - "morty config list"
  version:
    handler: "show_version"
    description: "显示版本信息"
  help:
    handler: "show_help"
    description: "显示帮助信息"
```

### 版本信息
```
Morty 2.0.0
AI 驱动的开发循环管理系统

命令:
  research    交互式代码库/文档库研究
  plan        基于研究结果生成开发计划
  doing       执行开发计划
  stat        显示执行状态和进度
  reset       版本管理和回滚
  init        初始化用户配置
  config      配置管理
  version     显示版本信息
  help        显示帮助信息

使用 'morty help <command>' 查看详细帮助
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
- 输入 `morty research test` 应调用 `morty_research.sh test`
- 输入 `morty plan` 应调用 `morty_plan.sh`
- 输入 `morty stat` 应调用 `morty_stat.sh`
- 输入未知命令时应显示错误和帮助信息
- 参数解析应正确处理选项和位置参数
- 命令执行失败时应返回适当的退出码

**调试日志**:
- 无

---

### Job 2: stat 命令实现

**目标**: 实现 `morty stat` 命令，读取并显示 `.morty/status.json`

**前置条件**: doing 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_stat.sh` 脚本
- [ ] 实现 `stat_load_status()`: 读取 status.json
- [ ] 实现 `stat_format_output()`: 格式化状态输出（表格/文本）
- [ ] 实现 `stat_show_summary()`: 显示整体进度摘要
- [ ] 实现 `stat_show_modules()`: 显示模块进度列表
- [ ] 实现 `stat_show_current()`: 显示当前执行状态
- [ ] 实现 `--json` 选项输出原始 JSON
- [ ] 实现 `--watch` 选项持续刷新（每 2 秒）

**验证器**:
- `morty stat` 应显示整体进度、当前状态、模块列表
- 当 `.morty/status.json` 不存在时，应提示 "请先运行 morty doing"
- `--json` 选项应输出有效的 JSON 格式
- `--watch` 选项应每 2 秒刷新一次，Ctrl+C 退出
- 应正确计算并显示进度百分比
- 应显示最近的 debug_log 条目（最多 5 条）

**调试日志**:
- 无

---

### Job 3: 全局选项和参数解析

**目标**: 实现全局选项（--help, --version）和子命令参数解析

**前置条件**: Job 1, Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现全局选项处理（-h/--help, -v/--version）
- [ ] 实现子命令参数解析器
- [ ] 实现参数验证（必填参数检查）
- [ ] 实现参数类型转换（字符串/整数/布尔值）
- [ ] 实现参数补全提示（可选）

**验证器**:
- `morty --help` 应显示全局帮助信息
- `morty research --help` 应显示 research 命令的详细帮助
- `morty doing --module config` 应正确解析 `--module` 选项
- `morty stat --watch` 应正确解析 `--watch` 选项
- 缺少必填参数时应显示错误并提示用法
- 无效参数值时应显示错误信息

**调试日志**:
- 无

---

### Job 4: 帮助系统

**目标**: 实现完善的帮助系统，支持多级帮助信息

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `cli_show_global_help()`: 全局帮助
- [ ] 实现 `cli_show_command_help(cmd)`: 命令帮助
- [ ] 实现帮助文档自动生成
- [ ] 实现使用示例显示
- [ ] 实现环境变量说明

**验证器**:
- `morty help` 应显示所有可用命令列表（包含 stat）
- `morty help stat` 应显示 stat 命令的详细用法
- 每个命令的帮助应包含描述、用法、选项、示例
- 帮助信息应格式清晰，易于阅读
- 帮助系统应支持中文显示

**调试日志**:
- 无

---

### Job 5: 版本和初始化

**目标**: 实现版本显示和初始化命令

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `cli_show_version()`: 显示版本信息
- [ ] 实现 `cli_init()`: 初始化命令
- [ ] 集成 config 模块的初始化功能
- [ ] 实现版本号管理（统一存储）
- [ ] 实现更新检查（可选）

**验证器**:
- `morty version` 应显示版本号、构建信息
- `morty init` 应创建用户配置文件
- 版本号应在代码中统一存储，避免多处定义
- 初始化应检查依赖（如 Git、Claude CLI）
- 重复初始化应安全处理（不覆盖已有配置）

**调试日志**:
- 无

---

### Job 6: 错误处理和诊断

**目标**: 实现统一的错误处理和诊断信息

**前置条件**: Job 5 完成

**Tasks (Todo 列表)**:
- [ ] 实现错误码定义和管理
- [ ] 实现友好的错误信息格式化
- [ ] 实现诊断模式（--verbose, --debug）
- [ ] 实现前置条件检查（依赖检查）
- [ ] 实现错误恢复建议

**验证器**:
- 命令未找到时应返回退出码 2
- 前置条件不满足时应返回退出码 3
- 使用 `--verbose` 时应显示详细的执行过程
- 错误信息应包含问题描述和解决方案建议
- 诊断信息应记录到日志文件

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 所有命令可以正确路由和执行
- `morty stat` 可以正确显示状态
- 帮助信息完整且准确
- 错误处理一致且友好
- 参数解析正确，边界情况处理得当

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
stat_format_output(format)  # text/json
stat_show_summary()
stat_show_modules()
stat_show_current()
stat_show_debug_logs(limit=5)
stat_watch()  # 持续监控

# 帮助
cli_show_global_help()
cli_show_command_help(command)
cli_generate_help_text(command)

# 版本和初始化
cli_show_version()
cli_init()
cli_check_dependencies()

# 错误处理
cli_error(message, code=1)
cli_die(message, code=1)
cli_verbose(message)
cli_debug(message)

# 参数解析
cli_getopt(args, optname)
cli_getarg(args, index)
cli_validate_args(spec, args)
```

---

## stat 命令输出示例

### 默认输出
```
$ morty stat

Morty 开发状态
==============

整体进度: 12% (5/42 Jobs)
当前状态: running
运行时间: 4小时30分钟
总循环数: 15

模块进度:
  [✓] config       100% (3/3 Jobs)  耗时: 1h30m
  [>] logging       50% (2/4 Jobs)  耗时: 2h00m  (运行中)
  [ ] git_manager    0% (0/5 Jobs)
  [ ] plan_mode      0% (0/6 Jobs)
  [ ] doing          0% (0/7 Jobs)
  [ ] cli            0% (0/6 Jobs)

当前执行:
  模块: logging
  Job: job_2
  名称: 日志轮转和归档
  状态: RUNNING (第2次循环)
  任务: Task 3/4 - 实现日志轮转
  已运行: 30分钟

最近调试记录:
  [logging/job_2] 2026-02-20 13:30:00
    现象: 日志轮转时丢失消息
    复现: 高频写入时触发轮转
    猜想: 文件句柄未正确同步, 并发写入竞争
    修复: 待修复
    状态: 未解决
```

### JSON 输出
```bash
$ morty stat --json
{
  "state": "running",
  "progress": {
    "percentage": 12,
    "completed_jobs": 5,
    "total_jobs": 42
  },
  "current": {
    "module": "logging",
    "job": "job_2",
    "status": "RUNNING",
    "loop_count": 2
  },
  "modules": [...]
}
```

### 持续监控
```bash
$ morty stat --watch
[每2秒自动刷新，显示简化状态]
```
