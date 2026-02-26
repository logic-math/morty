# Plan: Stat Command

## 模块概述

**模块职责**: 实现 `morty stat` 命令，显示项目执行状态和进度监控

**对应 Research**:
- `morty-project-research.md` 第 3.1 节主入口分析（stat 命令）

**依赖模块**: Config, Logging, State, Git, Parser

**被依赖模块**: CLI (命令注册)

---

## 命令行接口

### 用法

```bash
# 显示当前状态（默认表格格式）
morty stat

# 监控模式，每60秒刷新
morty stat --watch
morty stat -w

# JSON 格式输出
morty stat --json
```

### 选项

| 选项 | 简写 | 说明 |
|------|------|------|
| `--watch` | `-w` | 监控模式，定时刷新 |
| `--json` | `-j` | JSON 格式输出 |

---

## 显示内容

### 默认表格输出

```
┌─────────────────────────────────────────────────────────────┐
│                     Morty 监控大盘                           │
├─────────────────────────────────────────────────────────────┤
│ 当前执行                                                    │
│   模块: config                                              │
│   Job:  job_2 (配置管理器实现)                               │
│   状态: RUNNING (第2次循环)                                  │
│   累计时间: 00:32:15                                        │
├─────────────────────────────────────────────────────────────┤
│ 上一个 Job                                                  │
│   config/job_1: COMPLETED (耗时 00:15:30)                   │
│   摘要: 配置加载器实现完成                                    │
├─────────────────────────────────────────────────────────────┤
│ Debug 问题 (当前 Job)                                       │
│   • 配置文件解析失败 (loop 2)                                │
│     猜想: JSON 格式错误                                       │
│     状态: 待修复                                             │
├─────────────────────────────────────────────────────────────┤
│ 整体进度                                                    │
│   [████░░░░░░░░░░░░░░░░] 20% (5/25 Jobs)                    │
│   已完成: config (3/3)                                       │
│   进行中: logging (2/4)                                      │
│   待开始: version_manager, doing, cli                       │
└─────────────────────────────────────────────────────────────┘
```

### JSON 输出格式

```json
{
  "current": {
    "module": "config",
    "job": "job_2",
    "status": "RUNNING",
    "loop_count": 2,
    "elapsed_time": "00:32:15"
  },
  "previous": {
    "module": "config",
    "job": "job_1",
    "status": "COMPLETED",
    "duration": "00:15:30"
  },
  "progress": {
    "total_jobs": 25,
    "completed_jobs": 5,
    "percentage": 20
  },
  "modules": [
    {"name": "config", "status": "completed", "jobs_completed": 3, "jobs_total": 3},
    {"name": "logging", "status": "in_progress", "jobs_completed": 2, "jobs_total": 4}
  ],
  "debug_issues": [
    {
      "description": "配置文件解析失败",
      "loop": 2,
      "guess": "JSON 格式错误",
      "status": "pending"
    }
  ]
}
```

---

## 数据模型

```go
// StatHandler stat 命令处理器
type StatHandler struct {
    config        config.Manager
    logger        logging.Logger
    stateManager  state.Manager
    gitManager    git.Manager
    parserFactory parser.Factory
}

// StatOptions stat 命令选项
type StatOptions struct {
    Watch bool
    JSON  bool
}

// StatusInfo 状态信息
type StatusInfo struct {
    Current    CurrentJob     `json:"current"`
    Previous   PreviousJob    `json:"previous"`
    Progress   ProgressInfo   `json:"progress"`
    Modules    []ModuleStatus `json:"modules"`
    DebugIssues []DebugIssue  `json:"debug_issues"`
}

type CurrentJob struct {
    Module      string `json:"module"`
    Job         string `json:"job"`
    Description string `json:"description"`
    Status      string `json:"status"`
    LoopCount   int    `json:"loop_count"`
    ElapsedTime string `json:"elapsed_time"`
}

type ProgressInfo struct {
    TotalJobs      int `json:"total_jobs"`
    CompletedJobs  int `json:"completed_jobs"`
    FailedJobs     int `json:"failed_jobs"`
    PendingJobs    int `json:"pending_jobs"`
    Percentage     int `json:"percentage"`
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: Stat 命令框架

**目标**: 实现 stat 命令的基础框架

**前置条件**:
- Config, Logging, State 模块完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/cmd/stat.go` 文件
- [x] Task 2: 实现 `StatHandler` 结构体
- [x] Task 3: 实现参数解析 (`--watch`, `--json`)
- [x] Task 4: 检查 `.morty/status.json` 存在性
- [x] Task 5: 无状态时的友好提示
- [x] Task 6: 编写单元测试

**验证器**:
- [x] 正确解析 `--watch` 和 `--json`
- [x] 无状态文件时提示 "请先运行 morty doing"
- [x] 返回码正确
- [x] 所有单元测试通过

**调试日志**:
- explore1: [探索发现] 项目使用标准Handler模式, 参考research.go实现, 使用state.Manager访问status.json, 测试使用mockConfig/mockLogger模式, 已记录
- debug1: strings.Builder没有ReadFrom方法导致测试编译失败, 运行测试时发现, 猜想: strings.Builder API与其他Writer不同, 验证: 查阅Go文档确认, 修复: 使用io.ReadAll替代, 已修复

---

### Job 2: 状态数据收集

**目标**: 收集所有状态数据

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `collectStatus()` 收集状态
- [x] Task 2: 读取 `status.json` 获取当前 Job
- [x] Task 3: 查找上一个完成的 Job
- [x] Task 4: 统计所有模块的 Jobs
- [x] Task 5: 计算进度百分比
- [x] Task 6: 从日志中提取 debug 问题
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 正确读取当前 Job 信息
- [x] 正确找到上一个完成的 Job
- [x] 正确统计 Jobs 数量
- [x] 进度计算正确
- [x] 所有单元测试通过

**调试日志**:
- explore1: [探索发现] 项目使用标准 Handler 模式, state.Manager 提供 GetCurrent() 和 GetSummary() 方法, 状态文件格式为 version 1.0 (StatusJSON with Global and Modules), 已记录
- debug1: 需要从 state.Manager 获取原始数据来查找上一个完成的 Job 和 debug logs, 直接解析 status.json 文件, 使用匿名结构体匹配 version 1.0 格式, 已修复
- debug2: TestStatHandler_collectStatus 测试失败, 状态文件格式错误, 猜想: 使用了 version 2.0 格式而非 1.0, 验证: 检查 state/state.go 确认使用 version 1.0 格式 (GlobalState/ModuleState), 修复: 更新测试数据使用 version 1.0 格式, 已修复

---

### Job 3: 表格格式化输出

**目标**: 实现表格形式的状态显示

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `formatTable()` 格式化表格
- [x] Task 2: 实现表头格式化
- [x] Task 3: 实现各区块格式化
  - 当前执行
  - 上一个 Job
  - Debug 问题
  - 整体进度
- [x] Task 4: 实现进度条显示
- [x] Task 5: 对齐和美化
- [x] Task 6: 编写单元测试

**验证器**:
- [x] 表格对齐正确
- [x] 内容不溢出边框
- [x] 进度条显示正确
- [x] 彩色输出（支持终端）
- [x] 所有单元测试通过

**调试日志**:
- debug1: progress bar显示被截断, 测试TestStatHandler_outputEnhancedText失败, 猜想: 进度条宽度太大导致整行超过contentWidth, 验证: 检查formatContentLine发现内容超过57字符会被截断, 修复: 将barWidth从40减少到10, 已修复

---

### Job 4: JSON 格式输出

**目标**: 实现 JSON 格式输出

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `formatJSON()` JSON 格式化
- [x] Task 2: 构建 JSON 数据结构
- [x] Task 3: 处理时间格式
- [x] Task 4: 美化 JSON 输出（缩进）
- [x] Task 5: 确保字段完整
- [x] Task 6: 编写单元测试

**验证器**:
- [x] 输出有效的 JSON
- [x] 包含所有必要字段
- [x] 时间格式统一
- [x] JSON 可解析
- [x] 所有单元测试通过

**调试日志**:
- debug1: PreviousJob.Duration类型从time.Duration改为string导致编译错误, 运行测试时发现类型不匹配, 猜想: Duration字段用于JSON输出应为字符串格式, 验证: 检查formatPreviousJobSection和测试代码, 修复: 更新所有使用Duration的地方为string类型并格式化, 已修复
- debug2: TestStatHandler_findPreviousJob测试期望time.Duration但得到string, 运行测试时发现, 猜想: 测试代码需要同步更新, 验证: 检查测试文件, 修复: 将测试中的15 * time.Minute改为"15:00"字符串, 已修复
- debug3: 需要确保所有JSON字段完整包括loop_count和description, 检查代码时发现collectStatus未填充这些字段, 猜想: 需要从原始state数据中读取, 验证: 添加代码读取loop_count和description, 修复: 在collectStatus中添加逻辑从status.json读取额外字段, 已修复

---

### Job 5: 监控模式实现

**目标**: 实现 `--watch` 监控模式

**前置条件**:
- Job 3 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `watchMode()` 监控循环
- [ ] Task 2: 设置刷新间隔（默认 60s）
- [ ] Task 3: 实现屏幕清空（原地刷新）
- [ ] Task 4: 信号处理（Ctrl+C 优雅退出）
- [ ] Task 5: 刷新时重新收集数据
- [ ] Task 6: 显示刷新时间
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 每 60s 自动刷新
- [ ] 原地刷新无滚动
- [ ] Ctrl+C 能优雅退出
- [ ] 显示最后刷新时间
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的 stat 流程
- [ ] 表格和 JSON 格式都正常
- [ ] 监控模式正常工作
- [ ] 数据更新后显示正确
- [ ] 集成测试通过

**调试日志**:
- 待填充

---

## 使用示例

```bash
# 显示当前状态
$ morty stat
┌─────────────────────────────────────────────────────────────┐
│                     Morty 监控大盘                           │
├─────────────────────────────────────────────────────────────┤
│ 当前执行                                                    │
│   模块: config                                              │
│   Job:  job_2                                               │
│   状态: RUNNING                                             │
└─────────────────────────────────────────────────────────────┘

# JSON 格式
$ morty stat --json
{
  "current": {...},
  "progress": {...}
}

# 监控模式
$ morty stat -w
[每60秒自动刷新，Ctrl+C退出]
```

---

## 文件清单

- `internal/cmd/stat.go` - stat 命令实现
