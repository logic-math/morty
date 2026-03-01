# Bug 修复: Job Index 9999 问题

## Bug 描述

### 现象

执行 `morty doing` 时，日志显示：

```
[2026-03-01T12:15:59.718+08:00] INFO    Found executable job module=命令行接口 job=实现命令行参数解析 job_index=9999
```

**job_index=9999 是一个异常值**，表示没有正确读取到 job 的 index。

### 影响

1. **错误的执行顺序**: 所有 job 的 index 都是 9999，导致无法按 plan 文件中定义的顺序执行
2. **依赖关系混乱**: 无法保证 Job 1 在 Job 2 之前执行
3. **拓扑排序失效**: 模块内的 job 顺序完全错误

### 实际案例

caipiao 项目中，第一次执行选择了 `命令行接口` 模块的 job，但根据依赖关系：

- `quicksort_core` (无依赖) - **应该最先执行**
- `file_handler` (依赖 quicksort_core)
- `cli_interface` (依赖 quicksort_core, file_handler) - **不应该第一个执行**

## 根本原因

在 `internal/cmd/doing.go` 的 `findExecutableJob` 函数（line 701）：

```go
// Load plan to get job indices for proper ordering
planFile := filepath.Join(h.getPlanDir(), moduleName+".md")
content, err := os.ReadFile(planFile)
```

**问题**: 使用 `moduleName + ".md"` 拼接文件名

**实际情况**:
- 模块名: `命令行接口` (中文)
- 实际文件名: `cli_interface.md` (英文)
- 拼接结果: `命令行接口.md` ❌ **文件不存在！**

### 为什么会有这个问题？

`ModuleState` 结构体有两个字段：

```go
type ModuleState struct {
    Name     string `json:"name"`      // 模块标识符（可能是中文）
    PlanFile string `json:"plan_file"` // 实际的 plan 文件名
    ...
}
```

注释（line 93-94）明确说明：

> PlanFile is the actual plan file name (e.g., "test_hello_world.md").
> This may differ from Name when the module has a Chinese name.

但 `findExecutableJob` 没有使用 `module.PlanFile`，而是错误地使用了 `moduleName`。

## 修复方案

### 修改前

```go
func (h *DoingHandler) findExecutableJob(moduleName string, module *state.ModuleState) string {
    // ...

    // Load plan to get job indices for proper ordering
    planFile := filepath.Join(h.getPlanDir(), moduleName+".md")
    content, err := os.ReadFile(planFile)
    jobIndexMap := make(map[string]int)
    // ...
}
```

### 修改后

```go
func (h *DoingHandler) findExecutableJob(moduleName string, module *state.ModuleState) string {
    // ...

    // Load plan to get job indices for proper ordering
    // Use module.PlanFile if available, otherwise fall back to moduleName+".md"
    planFileName := module.PlanFile
    if planFileName == "" {
        planFileName = moduleName + ".md"
    }
    planFile := filepath.Join(h.getPlanDir(), planFileName)
    content, err := os.ReadFile(planFile)
    jobIndexMap := make(map[string]int)
    // ...
}
```

### 修复逻辑

1. **优先使用** `module.PlanFile` 字段（实际文件名）
2. **回退机制**: 如果 `PlanFile` 为空，使用 `moduleName + ".md"`（向后兼容）

## 测试验证

### 1. 重新编译

```bash
cd /opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/morty
go build -o bin/morty cmd/morty/main.go
```

### 2. 重置状态

```bash
cd /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/caipiao
rm .morty/status.json
```

### 3. 重新执行

```bash
morty doing
```

### 4. 验证日志

期望看到：

```
Found executable job module=quicksort_core job=... job_index=1
```

或

```
Found executable job module=快速排序核心算法 job=实现分区函数 job_index=1
```

**不应该再看到 job_index=9999**

### 5. 验证执行顺序

```bash
# 查看执行日志
tail -100 .morty/logs/*.log | grep "module="
```

期望顺序：
1. `quicksort_core` 的 jobs
2. `file_handler` 的 jobs
3. `cli_interface` 的 jobs
4. `test_suite` 的 jobs
5. `e2e_test` 的 jobs

## 相关问题

### 问题 1: 为什么 status.json 中模块顺序也不对？

参见 `docs/status-json-order-fix.md`

**原因**: `SyncFromPlanDir` 使用 `os.ReadDir` 按字母序读取，没有拓扑排序

### 问题 2: 为什么执行时还能大致正确？

`selectTargetJob` 方法（line 662）会对**模块**进行拓扑排序：

```go
moduleNames, err := h.sortModulesByTopology(stateData)
```

但**模块内的 job 顺序**依赖 `findExecutableJob` 正确读取 job index。

如果 job index 全是 9999，模块内的 job 执行顺序就是随机的（取决于 map 遍历顺序）。

## 影响范围

### 受影响的场景

1. **中文模块名**: 所有使用中文模块名的项目
2. **特殊字符模块名**: 模块名与文件名不一致的情况
3. **Job 顺序依赖**: 依赖 job 按 plan 文件顺序执行的场景

### 不受影响的场景

1. **英文模块名**: 如果模块名恰好等于文件名（不含 .md），则不受影响
2. **单 job 模块**: 只有一个 job 的模块，顺序无所谓

## 代码审查建议

### 类似问题排查

搜索所有使用 `moduleName + ".md"` 的地方：

```bash
grep -rn 'moduleName.*\.md' internal/
```

检查是否应该使用 `module.PlanFile`。

### 最佳实践

1. **总是使用 `module.PlanFile`** 获取实际文件名
2. **只在必要时** 使用 `moduleName` 作为 fallback
3. **添加日志** 记录使用的文件路径，便于调试

## 总结

| 维度 | 详情 |
|------|------|
| **Bug 类型** | 文件路径拼接错误 |
| **严重程度** | 高（影响执行顺序） |
| **影响范围** | 中文模块名、特殊字符模块名 |
| **修复难度** | 低（3 行代码） |
| **测试验证** | 重新执行 morty doing，检查 job_index |

**修复后效果**:
- ✅ job_index 正确读取（1, 2, 3...）
- ✅ 模块内 job 按正确顺序执行
- ✅ 依赖关系正确处理
- ✅ 支持中文模块名
