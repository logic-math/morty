# Plan: config

## 模块概述

**模块职责**: 提供统一的配置管理系统，集中管理环境变量、用户配置和项目级配置，解决原有配置分散在多个地方的问题。

**对应 Research**: [重构机会] 统一配置管理；环境变量和硬编码配置分散

**依赖模块**: 无

**被依赖模块**: logging, git_manager, research, plan, doing, cli

## 接口定义

### 输入接口
- `morty init`: 初始化用户级配置文件
- `morty config <key> [value]`: 获取/设置配置项
- 环境变量: `MORTY_HOME`, `CLAUDE_CODE_CLI`, `MAX_LOOPS`, `LOOP_DELAY`

### 输出接口
- `config_get(key)`: 获取配置值（按优先级：环境变量 > 项目配置 > 用户配置 > 默认值）
- `config_set(key, value, scope)`: 设置配置值（scope: user/project）
- `config_load()`: 加载所有配置到内存
- `config_validate()`: 验证配置完整性和有效性

## 数据模型

### 配置文件结构

```yaml
# ~/.mortyrc (用户级配置)
morty:
  version: "2.0"
  cli:
    command: "claude"  # 或自定义 ai_cli
  defaults:
    max_loops: 50
    loop_delay: 5
    log_level: "INFO"
  paths:
    work_dir: ".morty_work"
    log_dir: ".morty/logs"

# .morty/config.yaml (项目级配置)
project:
  name: "my-project"
  type: "nodejs"  # auto-detected
  commands:
    install: "npm install"
    build: "npm run build"
    test: "npm test"
  plan:
    retry_count: 3
    coverage_threshold: 80
```

### 配置优先级（从高到低）
1. 环境变量 (e.g., `MORTY_MAX_LOOPS=100`)
2. 项目级配置 (`.morty/config.yaml`)
3. 用户级配置 (`~/.mortyrc`)
4. 内置默认值

## Jobs (Loop 块列表)

---

### Job 1: 配置系统基础架构

**目标**: 建立配置系统的核心框架，支持多层配置加载和优先级管理

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] 创建 `lib/config.sh` 配置文件管理模块
- [ ] 实现 `config_get()` 函数，支持按优先级读取配置
- [ ] 实现 `config_set()` 函数，支持用户级和项目级配置写入
- [ ] 定义内置默认配置表
- [ ] 实现配置验证函数 `config_validate()`

**验证器**:
- 当环境变量 `MORTY_TEST_KEY=value` 设置时，`config_get TEST_KEY` 应返回 "value"
- 当用户级配置文件中设置 `test_key: user_value`，且无环境变量时，`config_get test_key` 应返回 "user_value"
- 当项目级配置设置 `test_key: project_value`，应覆盖用户级配置但低于环境变量
- 无效的配置键应返回空值或默认值，不抛出异常
- 配置加载时间应小于 100ms

**调试日志**:
- 无

---

### Job 2: 配置文件初始化命令

**目标**: 实现 `morty init` 和 `morty config` 命令，支持配置的交互式初始化和管理

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `morty init` 命令，创建 `~/.mortyrc` 配置文件
- [ ] 实现 `morty config <key>=<value>` 语法（k=v 方式设置）
- [ ] 实现 `morty config get <key>` 命令
- [ ] 实现 `morty config list` 命令
- [ ] 添加配置项验证（如路径是否存在、数值是否合法）
- [ ] 创建配置模板生成器
- [ ] 更新 `morty` 主命令路由，添加 config 命令

**验证器**:
- 运行 `morty init` 后，`~/.mortyrc` 文件应存在且包含有效 YAML 格式
- 运行 `morty config ai_cli='mc --code'` 应更新用户级配置的 `cli.command` 为 `"mc --code"`
- 运行 `morty config get ai_cli` 应返回当前配置的值
- 运行 `morty config list` 应显示所有配置项及其来源（环境变量/用户/项目/默认）
- 运行 `morty config max_loops=100` 后，`~/.mortyrc` 中的值应更新为 100
- 无效的配置值应被拒绝并显示错误信息
- 当 `~/.mortyrc` 不存在时运行 `morty config key=value`，应先创建默认配置再设置值

**调试日志**:
- 无

---

### Job 3: 项目类型自动检测配置

**目标**: 将原有的项目类型检测功能整合到配置系统中，支持自动检测和手动指定

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 迁移 `detect_project_type()` 到 config 模块
- [ ] 迁移 `detect_build_command()` 和 `detect_test_command()` 到 config 模块
- [ ] 实现项目类型到命令的映射表
- [ ] 支持用户覆盖自动检测的命令
- [ ] 添加项目配置缓存机制

**验证器**:
- 在 Node.js 项目目录运行 `config_detect_project_type()` 应返回 "nodejs"
- 在 Python 项目目录（存在 requirements.txt 或 setup.py）应返回 "python"
- 当 `.morty/config.yaml` 中手动指定 `project.type: rust`，应使用该值而非自动检测
- 自动检测的命令应与手动配置的命令合并，手动配置优先级更高
- 未知项目类型应返回 "generic" 并使用空命令列表

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 配置系统可以被其他模块正常导入和使用
- 环境变量、用户配置、项目配置、默认值四层优先级正确工作
- 配置变更可以实时生效（无需重启 morty）
- 配置验证可以检测并报告无效配置
- 并发读取配置不会导致竞态条件

---

## 待实现方法签名

```bash
# lib/config.sh

# 配置读取
config_get(key, default="")
config_get_int(key, default=0)
config_get_bool(key, default=false)

# 配置写入
config_set(key, value, scope="project")
config_unset(key, scope="project")

# 配置加载
config_load()
config_reload()

# 配置验证
config_validate()
config_validate_key(key, value)

# 项目类型检测
config_detect_project_type()
config_get_build_command()
config_get_test_command()

# 配置持久化
config_save_user_config()
config_save_project_config()

# CLI 命令
config_cli_init()           # morty init
config_cli_set(kv_pair)     # morty config key=value
config_cli_get(key)         # morty config get key
config_cli_list()           # morty config list
config_parse_kv_pair(string)  # 解析 key=value 格式
```

---

## 使用示例

### 初始化配置
```bash
morty init
# 创建 ~/.mortyrc 配置文件
```

### 设置配置项（k=v 语法）
```bash
# 设置 ai_cli 命令
morty config ai_cli='mc --code'

# 设置数值配置
morty config max_loops=100

# 设置字符串配置（含空格）
morty config log_level='DEBUG'
```

### 获取配置项
```bash
morty config get ai_cli
# 输出: mc --code

morty config get max_loops
# 输出: 100
```

### 列出所有配置
```bash
morty config list
# 输出:
# ai_cli = mc --code      [user]
# max_loops = 100         [user]
# log_level = DEBUG       [default]
```

### 配置优先级示例
```bash
# 1. 环境变量（最高优先级）
export MORTY_AI_CLI="claude --verbose"

# 2. 用户级配置
morty config ai_cli='mc --code'

# 3. 项目级配置（.morty/config.yaml）

# 4. 默认值（最低优先级）
# ai_cli = claude
```
