# Plan: config

## 模块概述

**模块职责**: 提供统一的配置管理系统，通过单一的 `settings.json` 文件管理所有配置。工作目录固定为 `.morty`，配置文件存放在 `MORTY_HOME` 指定的全局位置。

**对应 Research**: 统一配置管理；简化配置层级

**依赖模块**: 无

**被依赖模块**: logging, version_manager, doing, cli

## 接口定义

### 输入接口
- `settings.json`: 全局配置文件（位于 `$MORTY_HOME/settings.json`）

### 输出接口
- `config_get(key)`: 从 settings.json 读取配置值
- `config_set(key, value)`: 设置配置值并保存到 settings.json
- `config_load()`: 加载配置文件
- `config_get_morty_home()`: 获取 MORTY_HOME 路径

## 数据模型

### 配置文件路径
```
$MORTY_HOME/settings.json
```

### settings.json 结构
```json
{
  "version": "2.0",
  "cli": {
    "command": "claude"
  },
  "defaults": {
    "max_loops": 50,
    "loop_delay": 5,
    "log_level": "INFO",
    "stat_refresh_interval": 60
  },
  "paths": {
    "work_dir": ".morty",
    "log_dir": ".morty/logs",
    "research_dir": ".morty/research",
    "plan_dir": ".morty/plan",
    "status_file": ".morty/status.json"
  }
}
```

### 工作目录结构
```
.morty/                      # 工作目录（固定）
├── logs/                    # 日志目录
├── research/                # 研究结果
│   └── [主题].md
├── plan/                    # 计划文件
│   ├── README.md
│   ├── [模块].md
│   └── [生产测试].md
└── status.json              # 执行状态
```

## Jobs (Loop 块列表)

---

### Job 1: 配置系统基础架构

**目标**: 建立配置系统的核心框架，支持从单一 JSON 文件读取配置

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] 创建 `lib/config.sh` 配置文件管理模块
- [ ] 实现 `config_get_morty_home()`: 读取 MORTY_HOME 环境变量，验证路径存在
- [ ] 实现 `config_load()`: 加载 settings.json
- [ ] 实现 `config_get(key)`: 按 key 读取配置值
- [ ] 实现 `config_set(key, value)`: 设置配置值并保存
- [ ] 实现配置默认值机制

**验证器**:
- 当 `MORTY_HOME` 未设置时，应提示用户设置该环境变量
- 当 `settings.json` 不存在时，应自动创建默认配置文件
- `config_get("cli.command")` 应返回配置的值或默认值 "claude"
- `config_set("log_level", "DEBUG")` 应更新配置文件
- 配置加载时间应小于 100ms

**调试日志**:
- 无

---

### Job 2: 工作目录管理

**目标**: 实现工作目录 `.morty` 的自动创建和管理

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `config_check_work_dir()`: 检查当前目录是否有 `.morty`
- [ ] 实现 `config_init_work_dir()`: 初始化工作目录结构
- [ ] 实现 `config_ensure_work_dir()`: 确保工作目录存在（不存在则创建）
- [ ] 实现子目录自动创建（logs, research, plan）

**验证器**:
- 调用 `config_ensure_work_dir()` 时，如 `.morty` 不存在应自动创建
- 应同时创建 `.morty/logs/`, `.morty/research/`, `.morty/plan/` 子目录
- 如目录已存在，应正常返回不报错
- 应检查目录是否可写

**调试日志**:
- 无

---

### Job 3: 前置条件检查

**目标**: 实现 plan → doing 的前置条件检查；research 作为可选输入

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `config_check_research_exists()`: 检查是否存在 research 文件
- [ ] 实现 `config_check_plan_done()`: 检查是否已完成 plan
- [ ] 实现 `config_require_plan()`: 要求必须先 plan
- [ ] 实现 `config_load_research_facts()`: 加载 research 事实信息
- [ ] 定义前置条件检查的错误提示信息

**验证器**:
- 当 `.morty/research/` 存在且有文件时，`config_load_research_facts()` 应返回文件列表和内容
- 当 `.morty/research/` 不存在或为空时，plan 模式应提示 "未找到 research，将通过对话理解需求"
- 当 `.morty/plan/` 为空时，`config_require_plan()` 应返回错误 "请先运行 morty plan"
- doing 模式运行时如未完成 plan 应报错 "请先运行 morty plan"
- research 不是 plan 的强制前置条件

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 配置系统可以被其他模块正常导入和使用
- settings.json 可以正确读取和写入
- 工作目录可以自动创建
- plan 可以不依赖 research 直接运行
- research 存在时 plan 可以正确加载事实信息
- doing 强制依赖 plan，无 plan 时报错
- 在任意模式下，缺少 `.morty` 目录时都会自动创建

---

## 待实现方法签名

```bash
# lib/config.sh

# 路径和初始化
config_get_morty_home()
config_load()
config_init_settings()

# 配置读写
config_get(key, default="")
config_get_int(key, default=0)
config_get_bool(key, default=false)
config_set(key, value)

# 工作目录
config_check_work_dir()
config_init_work_dir()
config_ensure_work_dir()
config_get_work_dir()

# 前置条件检查
config_check_research_exists()
config_check_plan_done()
config_require_plan()
config_load_research_facts()

# 路径获取
config_get_log_dir()
config_get_research_dir()
config_get_plan_dir()
config_get_status_file()
```

---

## 配置加载流程

```
1. 读取 MORTY_HOME 环境变量
2. 加载 $MORTY_HOME/settings.json
3. 检查当前目录是否有 .morty 工作目录
4. 如无，自动创建工作目录结构
```

---

## 前置条件检查流程

```
plan 模式启动:
  └─> 检查 .morty/research/ 是否有文件
      ├─> 有 → 加载事实信息，继续执行
      └─> 无 → 提示 "未找到 research，将通过对话理解需求"

doing 模式启动:
  └─> 检查 .morty/plan/ 是否有文件
      ├─> 有 → 继续执行
      └─> 无 → 报错 "请先运行 morty plan"
```

---

## 使用示例

### 初始化配置
```bash
export MORTY_HOME=$HOME/.morty
morty research test-topic
# 自动创建 .morty 工作目录和 $MORTY_HOME/settings.json
```

### 读取配置
```bash
# 在脚本中
source lib/config.sh
config_load
AI_CLI=$(config_get "cli.command" "claude")
LOG_LEVEL=$(config_get "defaults.log_level" "INFO")
```

### 设置配置
```bash
# 在脚本中
config_set "cli.command" "mc --code"
config_set "defaults.log_level" "DEBUG"
```

---

## 重要说明

1. **单一配置源**: 只有 `$MORTY_HOME/settings.json`，无项目级/用户级配置
2. **环境变量**: 只使用 `MORTY_HOME` 指定配置目录，其他配置不从环境变量读取
3. **自动创建**: 任意模式下如缺少 `.morty` 目录都会自动创建
4. **前置条件**: plan 可选依赖 research（有则读取，无则对话），doing 必须依赖 plan
5. **工作目录固定**: 统一使用 `.morty`，不可配置
