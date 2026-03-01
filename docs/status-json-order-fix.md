# Status.json 模块顺序问题修复

## 问题描述

### 当前问题

在 `internal/state/plan_sync.go` 的 `SyncFromPlanDir` 函数中，使用 `os.ReadDir` 扫描 plan 目录并初始化模块状态。

**问题**: `os.ReadDir` 返回的是文件系统顺序（通常是字母序），不是拓扑依赖顺序。

### 实际案例

caipiao 项目的 status.json 中模块顺序：

```json
{
  "modules": {
    "": {...},                    // caipiao.md
    "命令行接口": {...},          // cli_interface.md (依赖 quicksort_core, file_handler)
    "快速排序核心算法": {...},    // quicksort_core.md (无依赖)
    "文件读写处理": {...},        // file_handler.md (依赖 quicksort_core)
    "测试套件": {...},            // test_suite.md (依赖 quicksort_core, file_handler, cli_interface)
    "端到端测试": {...}           // e2e_test.md (依赖 __ALL__)
  }
}
```

**问题**: 顺序是字母序，不是拓扑序。

### 正确的拓扑序

根据依赖关系：

```
quicksort_core (无依赖)
  ↓
file_handler (依赖 quicksort_core)
  ↓
cli_interface (依赖 quicksort_core, file_handler)
  ↓
test_suite (依赖 quicksort_core, file_handler, cli_interface)
  ↓
e2e_test (依赖 __ALL__)
```

正确顺序应该是：
1. `quicksort_core`
2. `file_handler`
3. `cli_interface`
4. `test_suite`
5. `e2e_test`

## 影响分析

### 当前影响

虽然 status.json 中顺序不对，但**执行时没有问题**，因为：

1. `doing.go` 的 `selectTargetJob` 方法（line 662）会重新进行拓扑排序：
   ```go
   moduleNames, err := h.sortModulesByTopology(stateData)
   ```

2. 执行时会按拓扑序选择下一个可执行的 job

### 潜在问题

1. **可读性差**: status.json 中模块顺序混乱，不直观
2. **调试困难**: 查看 status.json 时无法快速理解执行顺序
3. **维护困难**: 依赖关系不明显
4. **性能开销**: 每次 `selectTargetJob` 都要重新排序
5. **一致性问题**: status.json 的顺序与实际执行顺序不一致

## 修复方案

### 方案 1: 在 SyncFromPlanDir 中进行拓扑排序（推荐）

**优点**:
- status.json 中模块顺序正确
- 只需排序一次（初始化时）
- 提高可读性和可维护性
- 减少运行时开销

**实现**:
1. 在 `SyncFromPlanDir` 中，先扫描所有 plan 文件
2. 提取依赖关系
3. 进行拓扑排序
4. 按拓扑序插入到 status.json

### 方案 2: 保持现状，文档说明

**优点**:
- 不修改代码
- 执行时已经有拓扑排序

**缺点**:
- status.json 顺序混乱
- 每次执行都要重新排序

## 推荐修复

采用方案 1，修改 `internal/state/plan_sync.go`：

### 修改后的 SyncFromPlanDir 函数

```go
func (m *Manager) SyncFromPlanDir(planDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure state is initialized
	if m.state == nil {
		m.state = m.createDefaultState()
	}

	// Ensure Modules map is initialized
	if m.state.Modules == nil {
		m.state.Modules = make(map[string]*ModuleState)
	}

	// Step 1: Scan plan directory and parse all plan files
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return fmt.Errorf("failed to read plan directory: %w", err)
	}

	// Store parsed plans with their file names
	type planInfo struct {
		name         string
		fileName     string
		parsedPlan   *plan.Plan
	}
	var allPlans []planInfo
	moduleDeps := make(map[string][]string) // module -> dependencies

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "README") {
			continue
		}

		// Read plan file
		planPath := filepath.Join(planDir, name)
		content, err := os.ReadFile(planPath)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Parse plan
		parsedPlan, err := plan.ParsePlan(string(content))
		if err != nil {
			continue // Skip files that can't be parsed
		}

		// Check if module already exists
		if _, exists := m.state.Modules[parsedPlan.Name]; exists {
			// Module already exists, skip
			continue
		}

		allPlans = append(allPlans, planInfo{
			name:       parsedPlan.Name,
			fileName:   name,
			parsedPlan: parsedPlan,
		})

		// Extract dependencies
		deps := parsedPlan.Dependencies
		// Filter out "无" (none)
		filteredDeps := make([]string, 0)
		for _, dep := range deps {
			if dep != "无" && dep != "" {
				filteredDeps = append(filteredDeps, dep)
			}
		}
		moduleDeps[parsedPlan.Name] = filteredDeps
	}

	// Step 2: Expand __ALL__ dependencies
	allModuleNames := make([]string, 0, len(allPlans))
	for _, p := range allPlans {
		allModuleNames = append(allModuleNames, p.name)
	}

	for moduleName, deps := range moduleDeps {
		if len(deps) == 1 && deps[0] == "__ALL__" {
			// Depend on all other modules
			expanded := make([]string, 0)
			for _, otherModule := range allModuleNames {
				if otherModule != moduleName {
					expanded = append(expanded, otherModule)
				}
			}
			moduleDeps[moduleName] = expanded
		}
	}

	// Step 3: Topological sort using Kahn's algorithm
	sortedModules, err := m.topologicalSort(allModuleNames, moduleDeps)
	if err != nil {
		// If cycle detected, fall back to original order
		sortedModules = allModuleNames
	}

	// Step 4: Create module states in topological order
	now := time.Now()
	modulesAdded := 0

	// Create a map for quick lookup
	planMap := make(map[string]planInfo)
	for _, p := range allPlans {
		planMap[p.name] = p
	}

	for _, moduleName := range sortedModules {
		p, ok := planMap[moduleName]
		if !ok {
			continue
		}

		// Create module state
		moduleState := &ModuleState{
			Name:      p.name,
			PlanFile:  p.fileName,
			Status:    StatusPending,
			Jobs:      make(map[string]*JobState),
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Add jobs from plan
		for _, job := range p.parsedPlan.Jobs {
			// Create task states
			tasks := make([]TaskState, 0, len(job.Tasks))
			for _, task := range job.Tasks {
				taskState := TaskState{
					Index:       task.Index,
					Status:      StatusPending,
					Description: task.Description,
					UpdatedAt:   now,
				}
				tasks = append(tasks, taskState)
			}

			jobState := &JobState{
				Name:           job.Name,
				Status:         StatusPending,
				LoopCount:      0,
				RetryCount:     0,
				TasksTotal:     len(job.Tasks),
				TasksCompleted: 0,
				Tasks:          tasks,
				DebugLogs:      []DebugLogEntry{},
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			moduleState.Jobs[job.Name] = jobState
		}

		m.state.Modules[p.name] = moduleState
		modulesAdded++
	}

	if modulesAdded > 0 {
		// Save the updated state
		m.mu.Unlock()
		err := m.Save()
		m.mu.Lock()
		return err
	}

	return nil
}

// topologicalSort performs topological sort using Kahn's algorithm.
// Returns sorted module names or error if cycle detected.
func (m *Manager) topologicalSort(modules []string, deps map[string][]string) ([]string, error) {
	// Build in-degree map
	inDegree := make(map[string]int)
	for _, module := range modules {
		inDegree[module] = 0
	}

	// Calculate in-degrees
	for _, dependencies := range deps {
		for _, dep := range dependencies {
			// dep is depended on, so modules depending on it have higher in-degree
			// Actually, in-degree should be: how many modules this module depends on
		}
	}

	// Correct in-degree calculation:
	// in-degree[A] = number of modules A depends on
	for module, dependencies := range deps {
		inDegree[module] = len(dependencies)
	}

	// Queue of modules with no dependencies (in-degree = 0)
	queue := make([]string, 0)
	for _, module := range modules {
		if inDegree[module] == 0 {
			queue = append(queue, module)
		}
	}

	// Sort queue for consistent ordering
	sort.Strings(queue)

	result := make([]string, 0, len(modules))

	// Process modules
	for len(queue) > 0 {
		// Take first module
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each module that depends on current, decrease in-degree
		for module, dependencies := range deps {
			for _, dep := range dependencies {
				if dep == current {
					inDegree[module]--
					if inDegree[module] == 0 {
						queue = append(queue, module)
						sort.Strings(queue)
					}
				}
			}
		}
	}

	// Check if all modules are processed (no cycle)
	if len(result) != len(modules) {
		return nil, fmt.Errorf("cycle detected in module dependencies")
	}

	return result, nil
}
```

## 测试验证

### 1. 删除现有 status.json

```bash
rm /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/caipiao/.morty/status.json
```

### 2. 重新初始化

```bash
cd /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/caipiao
morty doing
```

### 3. 验证顺序

```bash
cat .morty/status.json | jq -r '.modules | keys[]'
```

期望输出（按拓扑序）：
```
quicksort_core
file_handler
cli_interface
test_suite
e2e_test
```

或使用 Python 验证：

```python
import json

with open('.morty/status.json') as f:
    data = json.load(f)

modules = list(data['modules'].keys())
print("Current order:")
for i, m in enumerate(modules, 1):
    print(f"{i}. {m}")
```

## 相关文件

- `internal/state/plan_sync.go` - 需要修改的文件
- `internal/cmd/doing.go` - 已有拓扑排序逻辑（line 1528-1680）
- `/home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/caipiao/.morty/status.json` - 问题示例

## 总结

**问题**: status.json 中模块顺序是字母序，不是拓扑依赖序

**根因**: `SyncFromPlanDir` 使用 `os.ReadDir` 直接遍历，没有排序

**影响**:
- ✅ 执行正确（doing.go 有重新排序）
- ❌ 可读性差
- ❌ 每次执行都要重新排序

**修复**: 在 `SyncFromPlanDir` 中添加拓扑排序逻辑

**优先级**: 中等（功能正常，但影响可维护性）
