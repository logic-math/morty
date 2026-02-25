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
- [x] Task 1: 创建 `internal/state/state.go` 定义 Status 类型和常量
- [x] Task 2: 定义 StatusJSON, ModuleState, JobState 结构体
- [x] Task 3: 创建 `internal/state/status_json.go` 实现文件操作
- [x] Task 4: 实现 `Load() error` 从文件加载状态
- [x] Task 5: 实现 `Save() error` 保存状态到文件
- [x] Task 6: 处理文件不存在时初始化默认状态
- [x] Task 7: 实现状态文件备份机制
- [x] Task 8: 编写单元测试 `status_json_test.go`

**验证器**:
- [x] Status 常量定义正确 (PENDING, RUNNING, COMPLETED, FAILED, BLOCKED)
- [x] 加载存在的 status.json 返回正确结构
- [x] 加载不存在的文件创建默认状态结构
- [x] Save 后文件内容正确且格式美观
- [x] 状态文件损坏时返回错误
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用 Go modules 结构，模块名为 github.com/morty/morty，已记录
- debug1: 初始导入路径错误使用 morty/pkg/errors，猜想: go.mod 模块名不匹配，验证: 检查 go.mod 确认模块名，修复: 改为 github.com/morty/morty/pkg/errors，已修复
- debug2: state.go 中未使用的 imports 导致编译失败，猜想: imports 仅在 status_json.go 使用，验证: 移除 state.go 中未使用的 imports，修复: 清理 imports，已修复
- debug3: TestListBackups 测试失败，备份文件数量不正确，猜想: 相同时间戳导致备份文件名冲突，验证: 添加时间戳检查，修复: Backup 函数添加序号处理文件名冲突，已修复

---

### Job 2: 状态管理器实现

**目标**: 实现状态管理器，支持 Job 状态查询和更新

**前置条件**:
- Job 1 完成 (status.json 操作)

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/state/manager.go` 文件结构
- [x] Task 2: 实现 `GetJobStatus(module, job string) (Status, error)`
- [x] Task 3: 实现 `UpdateJobStatus(module, job string, status Status) error`
- [x] Task 4: 实现 `GetCurrent() (*CurrentJob, error)` 获取当前 Job
- [x] Task 5: 实现 `SetCurrent(module, job string, status Status) error`
- [x] Task 6: 实现 `GetSummary() (*Summary, error)` 获取统计摘要
- [x] Task 7: 实现 `GetPendingJobs() []JobRef` 获取待处理 Job 列表
- [x] Task 8: 编写单元测试 `manager_test.go`

**验证器**:
- [x] 获取存在的 Job 状态返回正确值
- [x] 获取不存在的 Job 返回错误
- [x] 更新 Job 状态后 Save 到文件
- [x] GetCurrent 返回当前执行的 Job
- [x] GetSummary 返回正确的统计数据
- [x] GetPendingJobs 只返回 PENDING 状态的 Job
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用 Go modules 结构，state 模块位于 internal/state/，已包含 state.go, status_json.go 和相关测试，已记录
- debug1: 编译时发现 Summary 结构体定义不完整，猜想: Modules 字段在结构体定义中但类型未定义，验证: 检查 manager.go 发现 ModuleSummary 类型定义重复，修复: 合并 ModuleSummary 到 Summary 前定义，已修复
- debug2: 工作目录不一致问题，文件创建在 /opt/meituan/... 但 shell 在 /home/sankuai/...，猜想: 两个不同路径指向不同目录，验证: stat 检查 inode 确认不同，修复: 在当前目录重新创建 internal/state/ 并复制文件，已修复
- debug3: 测试覆盖率 91.1% 超过 80% 要求，所有验证器检查通过，任务完成

---

### Job 3: 状态转换规则实现

**目标**: 实现 Job 状态机转换规则验证

**前置条件**:
- Job 2 完成 (状态管理器)

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/state/transitions.go` 文件结构
- [x] Task 2: 定义有效状态转换规则表
- [x] Task 3: 实现 `IsValidTransition(from, to Status) bool`
- [x] Task 4: 实现 PENDING → RUNNING → COMPLETED 主流程
- [x] Task 5: 实现 FAILED → PENDING 重试流程
- [x] Task 6: 实现 BLOCKED → PENDING 解除阻塞流程
- [x] Task 7: 实现 RUNNING → BLOCKED 阻塞流程
- [x] Task 8: 无效转换返回错误并记录日志
- [x] Task 9: 编写单元测试 `transitions_test.go`

**验证器**:
- [x] PENDING → RUNNING 是有效转换
- [x] RUNNING → COMPLETED 是有效转换
- [x] RUNNING → FAILED 是有效转换
- [x] FAILED → PENDING 是有效转换 (重试)
- [x] PENDING → COMPLETED 是无效转换
- [x] 无效转换返回 false 和错误信息
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 已存在 state 模块包含 state.go, status_json.go, manager.go，需要在这些基础上添加 transitions.go，已记录
- debug1: logging 包导入路径错误，猜想: 使用了不存在的 pkg/logging 路径，验证: 检查发现 logging 在 internal/logging，修复: 修改导入路径为 github.com/morty/morty/internal/logging，已修复
- debug2: Logger 接口方法签名不匹配，猜想: transitions.go 使用了错误的 logger 调用方式，验证: 检查 logging/logger.go 发现使用 logging.Attr 类型，修复: 更新为使用 logging.String() 等 Attr 构造函数，已修复
- debug3: 工作目录不一致导致测试不被识别，猜想: Go 从 /home/sankuai/... 读取但文件写在 /opt/meituan/...，验证: stat 检查 inode 确认不同，修复: 同步文件到 /home/sankuai/... 路径，已修复
- debug4: internal/logging 包编译失败，猜想: /home/sankuai/... 路径的 logging 文件版本过旧，验证: 对比两个路径文件内容确认，修复: 同步所有 logging 文件到 /home/sankuai/... 路径，已修复
- debug5: 测试覆盖率 92.2% 超过 80% 要求，所有验证器检查通过，任务完成

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
