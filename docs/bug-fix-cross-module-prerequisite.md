# Bug 修复: 跨模块前置条件检查失效

## Bug 描述

### 现象

执行 `morty doing` 时，第一个执行的模块是 `命令行接口` (cli_interface)：

```
[2026-03-01T12:15:59.718+08:00] INFO    Found executable job module=命令行接口 job=实现命令行参数解析
[2026-03-01T12:15:59.718+08:00] INFO    Target job selected module=命令行接口 job=实现命令行参数解析
```

**问题**: `命令行接口` 依赖 `quicksort_core` 和 `file_handler`，应该在它们之后执行！

### 依赖关系

根据 `cli_interface.md`:

```markdown
**依赖模块**: quicksort_core, file_handler
```

根据 Job 1 的前置条件：

```markdown
#### 前置条件

- file_handler:job_3 - 文件读写和错误处理已实现
```

**正确的执行顺序应该是**:
1. `quicksort_core` 的 jobs（无依赖）
2. `file_handler` 的 jobs（依赖 quicksort_core）
3. `cli_interface` 的 jobs（依赖 quicksort_core, file_handler）

### 影响

1. ❌ **依赖关系被忽略**: 跨模块的 job 依赖完全失效
2. ❌ **执行顺序错误**: 可能在依赖未满足时执行 job
3. ❌ **编译/运行失败**: 依赖的代码还没生成就尝试使用
4. ❌ **拓扑排序失效**: 模块级别的拓扑排序被绕过

## 根本原因

在 `internal/cmd/doing.go` 的 `checkPrerequisites` 函数（line 891-908）：

### 问题 1: 错误的分隔符

**代码使用 `/` 分隔符**:

```go
// Parse module/job format
var prereqModule, prereqJob string
if strings.Contains(prereq, "/") {
    parts := strings.SplitN(prereq, "/", 2)
    prereqModule = parts[0]
    prereqJob = parts[1]
} else {
    prereqModule = moduleName
    prereqJob = prereq
}
```

**Plan 文件中使用 `:` 分隔符**:

```markdown
- file_handler:job_3 - 文件读写和错误处理已实现
```

**Plan 提示词中明确定义**（`prompts/plan.md` line 604）:

```markdown
跨模块依赖: `模块名:job_N`
```

**结果**:
- `file_handler:job_3` 不包含 `/`
- 被当作同模块的 job 名称 `file_handler:job_3`
- 在当前模块中找不到这个 job
- 被跳过（line 916-921）
- **前置条件检查失效！**

### 问题 2: 没有解析 job_N 到实际 job 名称

即使修复了分隔符，`prereqJob` 是 `job_3`，但 status.json 中存储的是实际的 job 名称（如 "实现错误处理和格式兼容"）。

需要：
1. 解析 `job_3` 中的索引 `3`
2. 读取 `file_handler` 模块的 plan 文件
3. 找到 index=3 的 job
4. 获取实际的 job 名称
5. 在 status.json 中检查该 job 的状态

## 修复方案

### 修复 1: 使用正确的分隔符

**修改前**:
```go
if strings.Contains(prereq, "/") {
    parts := strings.SplitN(prereq, "/", 2)
    prereqModule = parts[0]
    prereqJob = parts[1]
}
```

**修改后**:
```go
if strings.Contains(prereq, ":job_") {
    // This is a cross-module dependency
    parts := strings.SplitN(prereq, ":", 2)
    prereqModule = strings.TrimSpace(parts[0])
    // Extract job_N part (may have " - description" suffix)
    jobPart := strings.TrimSpace(parts[1])
    if dashIdx := strings.Index(jobPart, " - "); dashIdx > 0 {
        jobPart = strings.TrimSpace(jobPart[:dashIdx])
    }
    prereqJob = jobPart
}
```

### 修复 2: 解析 job_N 到实际名称

**新增逻辑**:

```go
// Resolve job_N to actual job name if needed
actualJobName := prereqJob
if strings.HasPrefix(prereqJob, "job_") {
    // Need to resolve job_N to actual job name
    var jobIndex int
    fmt.Sscanf(prereqJob, "job_%d", &jobIndex)

    // Load the prerequisite module's plan file to find the job name
    var prereqPlanData *plan.Plan
    if prereqModule == moduleName {
        prereqPlanData = planData
    } else {
        // Load the other module's plan file
        otherModule, ok := stateData.Modules[prereqModule]
        if !ok {
            unmetPrereqs = append(unmetPrereqs, fmt.Sprintf("%s:job_%d (模块不存在)", prereqModule, jobIndex))
            continue
        }

        // Use otherModule.PlanFile
        otherPlanFileName := otherModule.PlanFile
        if otherPlanFileName == "" {
            otherPlanFileName = prereqModule + ".md"
        }
        otherPlanFile := filepath.Join(h.getPlanDir(), otherPlanFileName)
        otherContent, err := os.ReadFile(otherPlanFile)
        if err != nil {
            unmetPrereqs = append(unmetPrereqs, fmt.Sprintf("%s:job_%d (无法读取计划文件)", prereqModule, jobIndex))
            continue
        }

        prereqPlanData, err = plan.ParsePlan(string(otherContent))
        if err != nil {
            unmetPrereqs = append(unmetPrereqs, fmt.Sprintf("%s:job_%d (无法解析计划文件)", prereqModule, jobIndex))
            continue
        }
    }

    // Find the job with this index
    for _, j := range prereqPlanData.Jobs {
        if j.Index == jobIndex {
            actualJobName = j.Name
            break
        }
    }

    if actualJobName == prereqJob {
        // Job index not found
        unmetPrereqs = append(unmetPrereqs, fmt.Sprintf("%s:job_%d (Job索引不存在)", prereqModule, jobIndex))
        continue
    }
}

// Now use actualJobName to check status
```

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

### 3. 启用调试日志

```bash
export MORTY_DEBUG=1
morty doing
```

### 4. 验证日志

期望看到：

```
DEBUG: Job '命令行接口/实现命令行参数解析' (index: 1) has unmet prerequisites: 前置条件不满足: file_handler:job_3 - 文件读写和错误处理已实现 (实现错误处理和格式兼容未完成)
```

或者：

```
INFO    Found executable job module=快速排序核心算法 job=实现分区函数 job_index=1
```

**不应该看到 `命令行接口` 第一个执行**

### 5. 验证执行顺序

```bash
# 查看执行日志
grep "Target job selected" .morty/logs/*.log
```

期望顺序：
1. `quicksort_core` 或`快速排序核心算法`
2. `file_handler` 或 `文件读写处理`
3. `cli_interface` 或 `命令行接口`

## 相关问题

### 问题 1: Job index 9999

参见 `docs/bug-fix-job-index-9999.md`

**原因**: 使用 `moduleName + ".md"` 而不是 `module.PlanFile`

### 问题 2: Status.json 模块顺序

参见 `docs/status-json-order-fix.md`

**原因**: `SyncFromPlanDir` 使用字母序而不是拓扑序

### 三个 Bug 的关系

| Bug | 位置 | 影响范围 | 严重程度 |
|-----|------|----------|----------|
| **1. 跨模块前置条件失效** | `checkPrerequisites` | 跨模块依赖完全失效 | **严重** |
| **2. Job index 9999** | `findExecutableJob` | 模块内 job 顺序错误 | 高 |
| **3. Status.json 顺序** | `SyncFromPlanDir` | 可读性差，性能影响 | 中等 |

**Bug 1 是最严重的**，它导致依赖关系完全失效！

## 为什么这个 Bug 之前没被发现？

### 可能的原因

1. **测试用例不足**: 测试项目可能都是单模块或没有跨模块依赖
2. **手动指定模块**: 使用 `morty doing -m module_name` 手动指定执行顺序
3. **依赖关系简单**: 之前的项目依赖关系可能比较简单，碰巧能正常工作
4. **拓扑排序掩盖**: 模块级别的拓扑排序（`sortModulesByTopology`）能保证模块顺序，但无法保证 job 级别的依赖

## 代码审查建议

### 类似问题排查

1. **搜索所有分隔符使用**:
   ```bash
   grep -rn 'strings.Contains.*"/"' internal/
   grep -rn 'strings.Split.*"/"' internal/
   ```

2. **检查 plan 格式一致性**:
   - 提示词中定义的格式
   - Parser 解析的格式
   - Validator 验证的格式
   - 代码中使用的格式

3. **添加集成测试**:
   - 测试跨模块依赖
   - 测试 job 前置条件
   - 测试拓扑排序

### 最佳实践

1. **格式规范统一**: 在一个地方定义，所有地方引用
2. **添加单元测试**: 测试 prerequisite 解析逻辑
3. **添加调试日志**: 记录依赖检查过程
4. **错误信息详细**: 明确说明哪个前置条件未满足

## 总结

| 维度 | 详情 |
|------|------|
| **Bug 类型** | 跨模块依赖检查失效 |
| **严重程度** | 严重（依赖关系完全失效） |
| **影响范围** | 所有使用跨模块依赖的项目 |
| **修复难度** | 中等（需要解析 job_N） |
| **测试验证** | 重新执行 morty doing，检查执行顺序 |

**修复后效果**:
- ✅ 跨模块依赖正确检查
- ✅ job_N 正确解析到实际名称
- ✅ 执行顺序符合依赖关系
- ✅ 拓扑排序正确工作
- ✅ 支持 `module:job_N - description` 格式
