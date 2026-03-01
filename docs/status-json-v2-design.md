# Status.json V2 设计文档

## 设计目标

1. **拓扑排序预计算**: 在生成 status.json 时完成所有拓扑排序
2. **简化执行逻辑**: doing 命令只需顺序遍历数组，找第一个未完成的 job
3. **消除依赖检查**: 不需要运行时检查前置条件，因为顺序已经保证正确
4. **提高可读性**: 数组结构直观展示执行顺序

## 新格式设计

### 整体结构

```json
{
  "version": "2.0",
  "global": {
    "status": "PENDING",
    "start_time": "2026-03-01T12:00:00+08:00",
    "last_update": "2026-03-01T12:00:00+08:00",
    "current_module_index": 0,
    "current_job_index": 0,
    "total_modules": 5,
    "total_jobs": 20
  },
  "modules": [
    {
      "index": 0,
      "name": "quicksort_core",
      "display_name": "快速排序核心算法",
      "plan_file": "quicksort_core.md",
      "status": "PENDING",
      "dependencies": [],
      "jobs": [
        {
          "index": 0,
          "name": "实现分区函数",
          "status": "PENDING",
          "prerequisites": [],
          "tasks_total": 4,
          "tasks_completed": 0,
          "loop_count": 0,
          "retry_count": 0,
          "tasks": [
            {
              "index": 1,
              "description": "创建 quicksort.py 文件",
              "status": "PENDING",
              "updated_at": "2026-03-01T12:00:00+08:00"
            }
          ],
          "created_at": "2026-03-01T12:00:00+08:00",
          "updated_at": "2026-03-01T12:00:00+08:00"
        }
      ],
      "created_at": "2026-03-01T12:00:00+08:00",
      "updated_at": "2026-03-01T12:00:00+08:00"
    }
  ]
}
```

### 字段说明

#### Global 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `version` | string | 格式版本号，"2.0" |
| `status` | string | 全局状态 (PENDING/RUNNING/COMPLETED/FAILED) |
| `start_time` | string | 开始时间 (ISO8601) |
| `last_update` | string | 最后更新时间 |
| `current_module_index` | int | 当前执行的模块索引 |
| `current_job_index` | int | 当前执行的 job 在全局的索引 |
| `total_modules` | int | 总模块数 |
| `total_jobs` | int | 总 job 数 |

#### Module 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `index` | int | 模块在数组中的索引（拓扑序） |
| `name` | string | 模块名（文件名不含.md） |
| `display_name` | string | 显示名称（可能是中文） |
| `plan_file` | string | plan 文件名 |
| `status` | string | 模块状态 |
| `dependencies` | array | 依赖的模块名列表 |
| `jobs` | array | job 数组（按拓扑序） |
| `created_at` | string | 创建时间 |
| `updated_at` | string | 更新时间 |

#### Job 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `index` | int | Job 在模块内的索引（拓扑序） |
| `global_index` | int | Job 在全局的索引（可选） |
| `name` | string | Job 名称 |
| `status` | string | Job 状态 |
| `prerequisites` | array | 前置条件（原始格式，仅用于显示） |
| `tasks_total` | int | 总任务数 |
| `tasks_completed` | int | 已完成任务数 |
| `loop_count` | int | 循环次数 |
| `retry_count` | int | 重试次数 |
| `tasks` | array | 任务列表 |
| `debug_logs` | array | 调试日志 |
| `created_at` | string | 创建时间 |
| `updated_at` | string | 更新时间 |

## 生成算法

### 步骤 1: 扫描并解析所有 plan 文件

```go
type PlanInfo struct {
    Name         string   // 模块名（文件名不含.md）
    DisplayName  string   // 显示名称（从 plan 中提取）
    FileName     string   // 文件名
    Dependencies []string // 依赖模块列表
    Jobs         []JobInfo
}

type JobInfo struct {
    Index         int
    Name          string
    Prerequisites []string // 原始前置条件
    Tasks         []TaskInfo
}
```

### 步骤 2: 模块级拓扑排序

使用 Kahn's Algorithm：

```go
func TopologicalSortModules(plans []PlanInfo) ([]PlanInfo, error) {
    // 1. 构建依赖图
    deps := make(map[string][]string)
    for _, p := range plans {
        deps[p.Name] = filterDependencies(p.Dependencies)
    }

    // 2. 展开 __ALL__
    for name, d := range deps {
        if len(d) == 1 && d[0] == "__ALL__" {
            deps[name] = getAllOtherModules(plans, name)
        }
    }

    // 3. 计算入度
    inDegree := make(map[string]int)
    for _, p := range plans {
        inDegree[p.Name] = len(deps[p.Name])
    }

    // 4. Kahn's Algorithm
    queue := []string{}
    for _, p := range plans {
        if inDegree[p.Name] == 0 {
            queue = append(queue, p.Name)
        }
    }
    sort.Strings(queue) // 稳定排序

    result := []PlanInfo{}
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        // 添加到结果
        for _, p := range plans {
            if p.Name == current {
                result = append(result, p)
                break
            }
        }

        // 更新入度
        for name, d := range deps {
            for _, dep := range d {
                if dep == current {
                    inDegree[name]--
                    if inDegree[name] == 0 {
                        queue = append(queue, name)
                        sort.Strings(queue)
                    }
                }
            }
        }
    }

    // 检查循环依赖
    if len(result) != len(plans) {
        return nil, fmt.Errorf("cycle detected in module dependencies")
    }

    return result, nil
}
```

### 步骤 3: Job 级拓扑排序

对每个模块内的 jobs 进行拓扑排序：

```go
func TopologicalSortJobs(module PlanInfo, allModules map[string]PlanInfo) ([]JobInfo, error) {
    // 1. 解析前置条件，构建依赖图
    deps := make(map[int][]int) // job index -> prerequisite job indices

    for _, job := range module.Jobs {
        prereqIndices := []int{}

        for _, prereq := range job.Prerequisites {
            // 解析 job_N 格式
            if strings.HasPrefix(prereq, "job_") {
                var idx int
                fmt.Sscanf(prereq, "job_%d", &idx)
                prereqIndices = append(prereqIndices, idx)
            }

            // 解析 module:job_N 格式
            if strings.Contains(prereq, ":job_") {
                // 跨模块依赖：确保该模块已经在前面
                // 这里只需记录，不影响模块内排序
                continue
            }
        }

        deps[job.Index] = prereqIndices
    }

    // 2. Kahn's Algorithm
    inDegree := make(map[int]int)
    for _, job := range module.Jobs {
        inDegree[job.Index] = len(deps[job.Index])
    }

    queue := []int{}
    for _, job := range module.Jobs {
        if inDegree[job.Index] == 0 {
            queue = append(queue, job.Index)
        }
    }
    sort.Ints(queue)

    result := []JobInfo{}
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        // 找到对应的 job
        for _, job := range module.Jobs {
            if job.Index == current {
                result = append(result, job)
                break
            }
        }

        // 更新入度
        for idx, d := range deps {
            for _, dep := range d {
                if dep == current {
                    inDegree[idx]--
                    if inDegree[idx] == 0 {
                        queue = append(queue, idx)
                        sort.Ints(queue)
                    }
                }
            }
        }
    }

    // 检查循环依赖
    if len(result) != len(module.Jobs) {
        return nil, fmt.Errorf("cycle detected in job dependencies for module %s", module.Name)
    }

    return result, nil
}
```

### 步骤 4: 生成 status.json

```go
func GenerateStatusJSON(planDir string) (*StatusV2, error) {
    // 1. 扫描 plan 文件
    plans, err := scanPlanFiles(planDir)
    if err != nil {
        return nil, err
    }

    // 2. 模块拓扑排序
    sortedModules, err := TopologicalSortModules(plans)
    if err != nil {
        return nil, err
    }

    // 3. 为每个模块的 jobs 拓扑排序
    allModules := make(map[string]PlanInfo)
    for _, p := range sortedModules {
        allModules[p.Name] = p
    }

    modules := []ModuleStateV2{}
    globalJobIndex := 0

    for moduleIndex, plan := range sortedModules {
        sortedJobs, err := TopologicalSortJobs(plan, allModules)
        if err != nil {
            return nil, err
        }

        jobs := []JobStateV2{}
        for jobIndex, jobInfo := range sortedJobs {
            job := JobStateV2{
                Index:           jobIndex,
                GlobalIndex:     globalJobIndex,
                Name:            jobInfo.Name,
                Status:          "PENDING",
                Prerequisites:   jobInfo.Prerequisites,
                TasksTotal:      len(jobInfo.Tasks),
                TasksCompleted:  0,
                Tasks:           convertTasks(jobInfo.Tasks),
                CreatedAt:       time.Now(),
                UpdatedAt:       time.Now(),
            }
            jobs = append(jobs, job)
            globalJobIndex++
        }

        module := ModuleStateV2{
            Index:        moduleIndex,
            Name:         plan.Name,
            DisplayName:  plan.DisplayName,
            PlanFile:     plan.FileName,
            Status:       "PENDING",
            Dependencies: plan.Dependencies,
            Jobs:         jobs,
            CreatedAt:    time.Now(),
            UpdatedAt:    time.Now(),
        }
        modules = append(modules, module)
    }

    // 4. 构建 status
    status := &StatusV2{
        Version: "2.0",
        Global: GlobalStateV2{
            Status:             "PENDING",
            StartTime:          time.Now(),
            LastUpdate:         time.Now(),
            CurrentModuleIndex: 0,
            CurrentJobIndex:    0,
            TotalModules:       len(modules),
            TotalJobs:          globalJobIndex,
        },
        Modules: modules,
    }

    return status, nil
}
```

## Doing 命令简化

### 选择下一个 Job

```go
func (h *DoingHandler) selectNextJob() (moduleIndex, jobIndex int, err error) {
    status := h.stateManager.GetStatusV2()

    // 简单遍历数组，找第一个 PENDING 的 job
    for mi, module := range status.Modules {
        for ji, job := range module.Jobs {
            if job.Status == "PENDING" {
                return mi, ji, nil
            }
        }
    }

    return -1, -1, fmt.Errorf("no pending jobs found")
}
```

### 不需要前置条件检查

因为顺序已经保证正确，前面的 job 一定已经完成。

### 更新状态

```go
func (h *DoingHandler) updateJobStatus(moduleIndex, jobIndex int, status string) error {
    v2 := h.stateManager.GetStatusV2()

    v2.Modules[moduleIndex].Jobs[jobIndex].Status = status
    v2.Modules[moduleIndex].Jobs[jobIndex].UpdatedAt = time.Now()
    v2.Modules[moduleIndex].UpdatedAt = time.Now()
    v2.Global.LastUpdate = time.Now()

    // 更新 current indices
    if status == "RUNNING" {
        v2.Global.CurrentModuleIndex = moduleIndex
        v2.Global.CurrentJobIndex = v2.Modules[moduleIndex].Jobs[jobIndex].GlobalIndex
    }

    return h.stateManager.SaveV2(v2)
}
```

## Stat 命令适配

### 显示格式

```
执行进度: 5/20 jobs completed (25%)

模块进度:
  [✓] quicksort_core (4/4 jobs)
  [▶] file_handler (1/3 jobs)
  [ ] cli_interface (0/3 jobs)
  [ ] test_suite (0/5 jobs)
  [ ] e2e_test (0/3 jobs)

当前执行:
  模块: file_handler
  Job: 实现错误处理和格式兼容 (job 2/3)
  进度: 2/6 tasks
```

### 实现

```go
func (h *StatHandler) ShowStatus() error {
    status := h.stateManager.GetStatusV2()

    // 计算总进度
    completedJobs := 0
    for _, module := range status.Modules {
        for _, job := range module.Jobs {
            if job.Status == "COMPLETED" {
                completedJobs++
            }
        }
    }

    fmt.Printf("执行进度: %d/%d jobs completed (%.0f%%)\n\n",
        completedJobs, status.Global.TotalJobs,
        float64(completedJobs)/float64(status.Global.TotalJobs)*100)

    // 显示模块进度
    fmt.Println("模块进度:")
    for _, module := range status.Modules {
        completed := 0
        running := 0
        for _, job := range module.Jobs {
            if job.Status == "COMPLETED" {
                completed++
            } else if job.Status == "RUNNING" {
                running++
            }
        }

        icon := "[ ]"
        if completed == len(module.Jobs) {
            icon = "[✓]"
        } else if running > 0 || completed > 0 {
            icon = "[▶]"
        }

        fmt.Printf("  %s %s (%d/%d jobs)\n",
            icon, module.DisplayName, completed, len(module.Jobs))
    }

    // 显示当前执行
    if status.Global.Status == "RUNNING" {
        currentModule := status.Modules[status.Global.CurrentModuleIndex]
        for _, job := range currentModule.Jobs {
            if job.GlobalIndex == status.Global.CurrentJobIndex {
                fmt.Printf("\n当前执行:\n")
                fmt.Printf("  模块: %s\n", currentModule.DisplayName)
                fmt.Printf("  Job: %s (job %d/%d)\n",
                    job.Name, job.Index+1, len(currentModule.Jobs))
                fmt.Printf("  进度: %d/%d tasks\n",
                    job.TasksCompleted, job.TasksTotal)
                break
            }
        }
    }

    return nil
}
```

## 向后兼容

### 迁移工具

提供工具将 V1 格式转换为 V2：

```bash
morty migrate-status
```

### 自动检测

```go
func (m *Manager) Load() error {
    content, err := os.ReadFile(m.statusFile)
    if err != nil {
        return err
    }

    // 检测版本
    var versionCheck struct {
        Version string `json:"version"`
    }
    json.Unmarshal(content, &versionCheck)

    if versionCheck.Version == "2.0" {
        // Load V2
        return m.loadV2(content)
    } else {
        // Load V1 (legacy)
        return m.loadV1(content)
    }
}
```

## 优势总结

| 方面 | V1 (Map) | V2 (Array) |
|------|----------|------------|
| **拓扑排序** | 运行时计算 | 生成时预计算 |
| **依赖检查** | 每次执行都检查 | 不需要 |
| **执行逻辑** | 复杂（排序+检查） | 简单（顺序遍历） |
| **可读性** | 差（无序 map） | 好（有序数组） |
| **性能** | 每次 O(n log n) | O(1) |
| **调试** | 困难 | 容易 |
| **错误处理** | 运行时发现 | 生成时发现 |

## 实现计划

### Phase 1: 新数据结构

1. 定义 `StatusV2` 结构体
2. 实现拓扑排序算法
3. 实现生成逻辑

### Phase 2: 适配命令

1. 修改 `doing` 命令
2. 修改 `stat` 命令
3. 添加 `migrate-status` 命令

### Phase 3: 测试

1. 单元测试（拓扑排序）
2. 集成测试（caipiao 项目）
3. 循环依赖检测测试

### Phase 4: 文档

1. 更新用户文档
2. 更新 API 文档
3. 添加迁移指南
