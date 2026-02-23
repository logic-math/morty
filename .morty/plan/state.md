# Plan: State

## 模块概述

**模块职责**: 实现状态管理功能，包括 status.json 操作、Job 状态机和状态转换规则

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.4 节 State 模块接口定义
- `morty-go-refactor-plan.md` 第 6.2 节状态转换状态机
- `morty-project-research.md` 第 4.2 节 status.json 分析

**现有实现参考**:
- 原 Shell 版本: `morty_doing.sh` 中的状态管理逻辑

**依赖模块**: Config (获取 status.json 路径)

**被依赖模块**: Executor

---

## 接口定义

### 输入接口
- status.json 文件路径
- 状态更新请求

### 输出接口
- `Manager` 接口实现
- 状态查询结果
- 待处理 Job 列表

---

## 数据模型

```go
// Status Job 状态类型
type Status string
const (
    StatusPending   Status = "PENDING"
    StatusRunning   Status = "RUNNING"
    StatusCompleted Status = "COMPLETED"
    StatusFailed    Status = "FAILED"
    StatusBlocked   Status = "BLOCKED"
)

// Manager 状态管理接口
type Manager interface {
    Load() error
    Save() error
    GetJobStatus(module, job string) (Status, error)
    UpdateJobStatus(module, job string, status Status) error
    IsValidTransition(from, to Status) bool
    GetCurrent() (*CurrentJob, error)
    SetCurrent(module, job string, status Status) error
    GetSummary() (*Summary, error)
    GetPendingJobs() []JobRef
}

// CurrentJob 当前执行的 Job
type CurrentJob struct {
    Module string `json:"module"`
    Job    string `json:"job"`
    Status Status `json:"status"`
}

// JobRef Job 引用
type JobRef struct {
    Module string `json:"module"`
    Job    string `json:"job"`
    Status Status `json:"status"`
}

// Summary 状态摘要
type Summary struct {
    TotalJobs     int `json:"total_jobs"`
    CompletedJobs int `json:"completed_jobs"`
    FailedJobs    int `json:"failed_jobs"`
    PendingJobs   int `json:"pending_jobs"`
}

// StatusJSON status.json 完整结构
type StatusJSON struct {
    Version string `json:"version"`
    State   string `json:"state"`
    Current CurrentJob `json:"current"`
    Session struct {
        StartTime  string `json:"start_time"`
        LastUpdate string `json:"last_update"`
        TotalLoops int    `json:"total_loops"`
    } `json:"session"`
    Modules map[string]ModuleState `json:"modules"`
}

type ModuleState struct {
    Status string `json:"status"`
    Jobs   map[string]JobState `json:"jobs"`
}

type JobState struct {
    Status         Status   `json:"status"`
    LoopCount      int      `json:"loop_count"`
    RetryCount     int      `json:"retry_count"`
    TasksTotal     int      `json:"tasks_total"`
    TasksCompleted int      `json:"tasks_completed"`
    DebugLogs      []string `json:"debug_logs"`
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: 状态类型定义与 status.json 操作

**目标**: 定义状态类型，实现 status.json 的加载和保存

**前置条件**:
- Config 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/state/state.go` 定义 Status 类型和常量
- [ ] Task 2: 定义 StatusJSON, ModuleState, JobState 结构体
- [ ] Task 3: 创建 `internal/state/status_json.go` 实现文件操作
- [ ] Task 4: 实现 `Load() error` 从文件加载状态
- [ ] Task 5: 实现 `Save() error` 保存状态到文件
- [ ] Task 6: 处理文件不存在时初始化默认状态
- [ ] Task 7: 实现状态文件备份机制
- [ ] Task 8: 编写单元测试 `status_json_test.go`

**验证器**:
- [ ] Status 常量定义正确 (PENDING, RUNNING, COMPLETED, FAILED, BLOCKED)
- [ ] 加载存在的 status.json 返回正确结构
- [ ] 加载不存在的文件创建默认状态结构
- [ ] Save 后文件内容正确且格式美观
- [ ] 状态文件损坏时返回错误
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 2: 状态管理器实现

**目标**: 实现状态管理器，支持 Job 状态查询和更新

**前置条件**:
- Job 1 完成 (status.json 操作)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/state/manager.go` 文件结构
- [ ] Task 2: 实现 `GetJobStatus(module, job string) (Status, error)`
- [ ] Task 3: 实现 `UpdateJobStatus(module, job string, status Status) error`
- [ ] Task 4: 实现 `GetCurrent() (*CurrentJob, error)` 获取当前 Job
- [ ] Task 5: 实现 `SetCurrent(module, job string, status Status) error`
- [ ] Task 6: 实现 `GetSummary() (*Summary, error)` 获取统计摘要
- [ ] Task 7: 实现 `GetPendingJobs() []JobRef` 获取待处理 Job 列表
- [ ] Task 8: 编写单元测试 `manager_test.go`

**验证器**:
- [ ] 获取存在的 Job 状态返回正确值
- [ ] 获取不存在的 Job 返回错误
- [ ] 更新 Job 状态后 Save 到文件
- [ ] GetCurrent 返回当前执行的 Job
- [ ] GetSummary 返回正确的统计数据
- [ ] GetPendingJobs 只返回 PENDING 状态的 Job
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: 状态转换规则实现

**目标**: 实现 Job 状态机转换规则验证

**前置条件**:
- Job 2 完成 (状态管理器)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/state/transitions.go` 文件结构
- [ ] Task 2: 定义有效状态转换规则表
- [ ] Task 3: 实现 `IsValidTransition(from, to Status) bool`
- [ ] Task 4: 实现 PENDING → RUNNING → COMPLETED 主流程
- [ ] Task 5: 实现 FAILED → PENDING 重试流程
- [ ] Task 6: 实现 BLOCKED → PENDING 解除阻塞流程
- [ ] Task 7: 实现 RUNNING → BLOCKED 阻塞流程
- [ ] Task 8: 无效转换返回错误并记录日志
- [ ] Task 9: 编写单元测试 `transitions_test.go`

**验证器**:
- [ ] PENDING → RUNNING 是有效转换
- [ ] RUNNING → COMPLETED 是有效转换
- [ ] RUNNING → FAILED 是有效转换
- [ ] FAILED → PENDING 是有效转换 (重试)
- [ ] PENDING → COMPLETED 是无效转换
- [ ] 无效转换返回 false 和错误信息
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的状态生命周期: 初始化 → 更新 → 查询 → 保存
- [ ] 状态转换规则正确执行
- [ ] 并发状态更新安全 (如适用)
- [ ] 状态文件持久化正确
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充
