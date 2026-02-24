# Plan: Config

## 模块概述

**模块职责**: 实现配置管理功能，包括配置结构定义、settings.json 的加载/读取/保存、配置层级合并和环境变量支持。

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.2 节 Config 模块接口定义
- `morty-project-research.md` 第 4.1 节项目配置文件分析

**现有实现参考**:
- 原 Shell 版本: `lib/config.sh`，支持 dot notation 访问嵌套配置

**依赖模块**: 无 (基础模块)

**被依赖模块**: Logging, State, Executor, Call CLI

---

## 配置层级

```
┌─────────────────────────────────────────────────────────────┐
│                      配置层级结构                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Level 1: 代码内置默认值                             │   │
│  │  • 编译时嵌入二进制                                  │   │
│  │  • 保证基本功能可用                                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Level 2: 用户全局配置 (~/.morty/config.json)        │   │
│  │  • 用户级默认值覆盖                                  │   │
│  │  • 所有项目共享                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Level 3: 项目配置 (.morty/settings.json)            │   │
│  │  • 项目特定配置                                      │   │
│  │  • 随版本控制（可选）                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Level 4: 环境变量覆盖                               │   │
│  │  • MORTY_* 前缀                                      │   │
│  │  • CLAUDE_CODE_CLI 等                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Level 5: 命令行参数                                 │   │
│  │  • --verbose, --debug 等                             │   │
│  │  • 最高优先级                                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 接口定义

### 输入接口
- `configs/settings.json` 默认配置文件
- `~/.morty/config.json` 用户配置文件
- `.morty/settings.json` 项目配置文件
- 环境变量（MORTY_* 前缀）
- 命令行参数

### 输出接口
- `Manager` 接口实现
- 类型安全的配置值获取方法
- 配置结构定义

---

## 数据模型

### Config 结构定义

```go
// Config 完整配置结构
type Config struct {
    Version    string         `json:"version"`
    AICli      AICliConfig    `json:"ai_cli"`
    Execution  ExecutionConfig `json:"execution"`
    Logging    LoggingConfig  `json:"logging"`
    State      StateConfig    `json:"state"`
    Git        GitConfig      `json:"git"`
    Plan       PlanConfig     `json:"plan"`
    Prompts    PromptsConfig  `json:"prompts"`
}

// AICliConfig AI CLI 配置
type AICliConfig struct {
    Command               string   `json:"command"`
    EnvVar                string   `json:"env_var"`
    DefaultTimeout        string   `json:"default_timeout"`
    MaxTimeout            string   `json:"max_timeout"`
    EnableSkipPermissions bool     `json:"enable_skip_permissions"`
    DefaultArgs           []string `json:"default_args"`
    OutputFormat          string   `json:"output_format"`
}

// ExecutionConfig 执行配置
type ExecutionConfig struct {
    MaxRetryCount    int  `json:"max_retry_count"`
    AutoGitCommit    bool `json:"auto_git_commit"`
    ContinueOnError  bool `json:"continue_on_error"`
    ParallelJobs     int  `json:"parallel_jobs"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
    Level    string       `json:"level"`
    Format   string       `json:"format"`
    Output   string       `json:"output"`
    File     FileConfig   `json:"file"`
}

// FileConfig 日志文件配置
type FileConfig struct {
    Enabled     bool   `json:"enabled"`
    Path        string `json:"path"`
    MaxSize     string `json:"max_size"`
    MaxBackups  int    `json:"max_backups"`
    MaxAge      int    `json:"max_age"`
}

// StateConfig 状态配置
type StateConfig struct {
    File          string `json:"file"`
    AutoSave      bool   `json:"auto_save"`
    SaveInterval  string `json:"save_interval"`
}

// GitConfig Git 配置
type GitConfig struct {
    CommitPrefix          string `json:"commit_prefix"`
    AutoCommit            bool   `json:"auto_commit"`
    RequireCleanWorktree  bool   `json:"require_clean_worktree"`
}

// PlanConfig Plan 配置
type PlanConfig struct {
    Dir             string `json:"dir"`
    FileExtension   string `json:"file_extension"`
    AutoValidate    bool   `json:"auto_validate"`
}

// PromptsConfig 提示词配置
type PromptsConfig struct {
    Dir       string `json:"dir"`
    Research  string `json:"research"`
    Plan      string `json:"plan"`
    Doing     string `json:"doing"`
}
```

### Manager 接口

```go
// Manager 配置管理接口
type Manager interface {
    // 加载配置
    Load(path string) error
    LoadWithMerge(userConfigPath string) error

    // 读取配置（支持 dot notation，如 "ai_cli.command"）
    Get(key string, defaultValue ...interface{}) (interface{}, error)
    GetString(key string, defaultValue ...string) string
    GetInt(key string, defaultValue ...int) int
    GetBool(key string, defaultValue ...bool) bool
    GetDuration(key string, defaultValue ...time.Duration) time.Duration

    // 设置配置
    Set(key string, value interface{}) error

    // 保存配置
    Save() error
    SaveTo(path string) error

    // 路径助手
    GetWorkDir() string
    GetLogDir() string
    GetResearchDir() string
    GetPlanDir() string
    GetStatusFile() string
    GetConfigFile() string
}
```

---

## 默认配置

### 默认配置结构 (configs/settings.json)

```json
{
  "version": "2.0",
  "ai_cli": {
    "command": "ai_cli",
    "env_var": "CLAUDE_CODE_CLI",
    "default_timeout": "10m",
    "max_timeout": "30m",
    "enable_skip_permissions": true,
    "default_args": ["--verbose", "--debug"],
    "output_format": "json"
  },
  "execution": {
    "max_retry_count": 3,
    "auto_git_commit": true,
    "continue_on_error": false,
    "parallel_jobs": 1
  },
  "logging": {
    "level": "info",
    "format": "json",
    "output": "stdout",
    "file": {
      "enabled": true,
      "path": ".morty/doing/logs/morty.log",
      "max_size": "10MB",
      "max_backups": 5,
      "max_age": 7
    }
  },
  "state": {
    "file": ".morty/status.json",
    "auto_save": true,
    "save_interval": "30s"
  },
  "git": {
    "commit_prefix": "morty",
    "auto_commit": true,
    "require_clean_worktree": false
  },
  "plan": {
    "dir": ".morty/plan",
    "file_extension": ".md",
    "auto_validate": true
  },
  "prompts": {
    "dir": "prompts",
    "research": "prompts/research.md",
    "plan": "prompts/plan.md",
    "doing": "prompts/doing.md"
  }
}
```

### 用户配置示例 (~/.morty/config.json)

```json
{
  "version": "2.0",
  "ai_cli": {
    "command": "claude",
    "default_timeout": "15m"
  },
  "logging": {
    "level": "debug"
  },
  "execution": {
    "max_retry_count": 5
  }
}
```

---

## 配置字段说明

### ai_cli 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `command` | string | "ai_cli" | AI CLI 命令名称 |
| `env_var` | string | "CLAUDE_CODE_CLI" | 环境变量名，优先级高于 command |
| `default_timeout` | string | "10m" | 默认超时时间 |
| `max_timeout` | string | "30m" | 最大允许超时 |
| `enable_skip_permissions` | bool | true | 是否启用 --dangerously-skip-permissions |
| `default_args` | []string | ["--verbose", "--debug"] | 默认参数 |
| `output_format` | string | "json" | 输出格式（json/text） |

### execution 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `max_retry_count` | int | 3 | 最大重试次数 |
| `auto_git_commit` | bool | true | Job 完成后自动 Git 提交 |
| `continue_on_error` | bool | false | 错误时是否继续执行 |
| `parallel_jobs` | int | 1 | 并行 Job 数（预留） |

### logging 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `level` | string | "info" | 日志级别（debug/info/warn/error） |
| `format` | string | "json" | 日志格式（json/text） |
| `output` | string | "stdout" | 输出目标（stdout/file/both） |
| `file.enabled` | bool | true | 是否启用文件日志 |
| `file.path` | string | ".morty/doing/logs/morty.log" | 日志文件路径 |
| `file.max_size` | string | "10MB" | 单个日志文件最大大小 |
| `file.max_backups` | int | 5 | 保留的备份文件数 |
| `file.max_age` | int | 7 | 日志文件保留天数 |

### state 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `file` | string | ".morty/status.json" | 状态文件路径 |
| `auto_save` | bool | true | 自动保存状态 |
| `save_interval` | string | "30s" | 自动保存间隔 |

### git 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `commit_prefix` | string | "morty" | 提交信息前缀 |
| `auto_commit` | bool | true | 自动创建提交 |
| `require_clean_worktree` | bool | false | 是否要求干净的工作区 |

### plan 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `dir` | string | ".morty/plan" | Plan 文件目录 |
| `file_extension` | string | ".md" | Plan 文件扩展名 |
| `auto_validate` | bool | true | 自动验证 Plan 格式 |

### prompts 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `dir` | string | "prompts" | 提示词目录 |
| `research` | string | "prompts/research.md" | Research 提示词路径 |
| `plan` | string | "prompts/plan.md" | Plan 提示词路径 |
| `doing` | string | "prompts/doing.md" | Doing 提示词路径 |

---

## 环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `CLAUDE_CODE_CLI` | AI CLI 命令路径 | `export CLAUDE_CODE_CLI=/usr/local/bin/claude` |
| `MORTY_HOME` | Morty 安装目录 | `export MORTY_HOME=$HOME/.morty` |
| `MORTY_CONFIG` | 配置文件路径 | `export MORTY_CONFIG=$HOME/.morty/config.json` |
| `MORTY_LOG_LEVEL` | 日志级别覆盖 | `export MORTY_LOG_LEVEL=debug` |
| `MORTY_DEBUG` | 调试模式 | `export MORTY_DEBUG=1` |

---

## Jobs (Loop 块列表)

---

### Job 1: 配置结构定义

**目标**: 定义完整的配置结构和默认配置

**前置条件**: 无

**Tasks (Todo 列表)**:
- [x] Task 1: 定义 Config 结构体及所有子结构
- [x] Task 2: 定义 Manager 接口
- [x] Task 3: 创建 `configs/settings.json` 默认配置文件
- [x] Task 4: 实现配置默认值常量
- [x] Task 5: 定义配置验证规则
- [x] Task 6: 编写单元测试验证配置结构

**验证器**:
- [x] `configs/settings.json` 是有效的 JSON
- [x] 所有配置字段都有默认值
- [x] 配置结构自文档化（清晰易理解）
- [x] 默认值合理（适合大多数场景）
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: 文件创建后无法立即通过相对路径访问, 使用绝对路径复制文件解决, 猜想: 1)分布式文件系统缓存延迟 2)挂载点差异, 验证: 检查inode发现是同一目录, 修复: 使用绝对路径复制文件到工作目录, 已修复
- debug2: 单元测试 TestDefaultConfigJSONFile 失败, 无法找到 settings.json 文件, 猜想: 1)测试工作目录错误 2)configs目录未复制, 验证: 检查configs目录为空, 修复: 复制settings.json到configs目录, 已修复

---

### Job 2: 配置加载器实现

**目标**: 实现配置的加载和层级合并

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/config/loader.go` 文件结构
- [ ] Task 2: 实现 `Load(path string) error` 加载单个配置
- [ ] Task 3: 实现 `LoadWithMerge(userConfigPath string) error` 合并加载
- [ ] Task 4: 支持 JSON 格式解析
- [ ] Task 5: 实现配置层级合并（默认 → 用户 → 环境变量）
- [ ] Task 6: 处理文件不存在时使用默认配置
- [ ] Task 7: 实现配置验证（检查必需字段、类型、范围）
- [ ] Task 8: 编写单元测试 `loader_test.go`

**验证器**:
- [ ] 加载存在的配置文件返回正确结构
- [ ] 加载不存在的配置文件使用默认配置
- [ ] 用户配置正确覆盖默认配置
- [ ] 环境变量正确覆盖配置文件
- [ ] 加载无效 JSON 返回错误
- [ ] 配置验证失败时返回具体错误信息
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: 配置管理器实现

**目标**: 实现配置值的读取和设置，支持 dot notation

**前置条件**:
- Job 2 完成 (配置加载)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/config/manager.go` 文件结构
- [ ] Task 2: 实现 Manager 接口的所有方法
- [ ] Task 3: 实现 `Get(key string, ...)` 方法，支持 dot notation
- [ ] Task 4: 实现类型安全的 getter: `GetString`, `GetInt`, `GetBool`, `GetDuration`
- [ ] Task 5: 实现 `Set(key string, value interface{}) error`，支持嵌套设置
- [ ] Task 6: 实现 `Save() error` 和 `SaveTo(path string) error` 方法
- [ ] Task 7: 处理默认值逻辑
- [ ] Task 8: 编写单元测试 `manager_test.go`

**验证器**:
- [ ] `Get("ai_cli.command")` 返回正确值
- [ ] `Get("execution.max_retry_count")` 返回正确值
- [ ] `GetString("nonexistent", "default")` 返回默认值
- [ ] `Set("execution.max_retry_count", 100)` 正确更新嵌套值
- [ ] `Save()` 后文件内容正确更新
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 4: 路径管理实现

**目标**: 实现 Morty 工作目录路径管理

**前置条件**:
- Job 3 完成 (配置管理)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/config/paths.go` 文件结构
- [ ] Task 2: 实现 `GetWorkDir() string` 返回 `.morty` 路径
- [ ] Task 3: 实现 `GetLogDir() string` 返回 `.morty/doing/logs` 路径
- [ ] Task 4: 实现 `GetResearchDir() string` 返回 `.morty/research` 路径
- [ ] Task 5: 实现 `GetPlanDir() string` 返回 `.morty/plan` 路径
- [ ] Task 6: 实现 `GetStatusFile() string` 返回 `.morty/status.json` 路径
- [ ] Task 7: 实现 `GetConfigFile() string` 返回配置文件路径
- [ ] Task 8: 实现路径自动创建 (如果不存在)
- [ ] Task 9: 编写单元测试 `paths_test.go`

**验证器**:
- [ ] `GetWorkDir()` 返回正确的绝对路径
- [ ] `GetLogDir()` 返回 `.morty/doing/logs`
- [ ] 路径不存在时自动创建目录
- [ ] 路径包含特殊字符时正确处理
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的配置生命周期: 定义 → 加载 → 读取 → 设置 → 保存
- [ ] 配置层级合并正确（默认 → 用户 → 环境变量 → 命令行）
- [ ] 配置变更后重新加载正确
- [ ] 路径管理器与配置管理器协同工作
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 使用示例

### 基本使用

```go
// 创建配置管理器
cfg, err := config.NewManager()
if err != nil {
    log.Fatal(err)
}

// 加载配置（自动合并默认、用户、环境变量配置）
err = cfg.LoadWithMerge("~/.morty/config.json")

// 读取配置
aiCmd := cfg.GetString("ai_cli.command", "ai_cli")
timeout := cfg.GetDuration("ai_cli.default_timeout", 10*time.Minute)
maxRetry := cfg.GetInt("execution.max_retry_count", 3)
enableDebug := cfg.GetBool("logging.debug", false)

// 修改配置
cfg.Set("execution.max_retry_count", 5)
cfg.Save()
```

### 创建用户配置

```bash
# 创建 ~/.morty/config.json
mkdir -p ~/.morty
cat > ~/.morty/config.json << 'EOF'
{
  "version": "2.0",
  "ai_cli": {
    "command": "claude",
    "default_timeout": "15m"
  },
  "logging": {
    "level": "debug"
  }
}
EOF
```

### 使用环境变量覆盖

```bash
# 临时使用不同的 AI CLI
export CLAUDE_CODE_CLI="/path/to/claude"
morty doing

# 启用调试模式
export MORTY_DEBUG=1
morty doing --verbose
```

---

## 文件清单

- `internal/config/config.go` - 配置结构定义
- `internal/config/manager.go` - 配置管理器实现
- `internal/config/loader.go` - 配置加载器实现
- `internal/config/paths.go` - 路径管理实现
- `configs/settings.json` - 默认配置模板
- `configs/example.json` - 用户配置示例
- `plan/config.md` - 本文件
