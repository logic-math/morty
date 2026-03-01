# Status.json V2 使用指南

## 快速开始

### 1. 生成 V2 格式的 status.json

```bash
# 在项目根目录
cd /path/to/your/project

# 生成 status.json（会自动读取 .morty/plan/ 中的所有 plan 文件）
morty init-status

# 如果已存在 status.json，使用 --force 覆盖
morty init-status --force
```

### 2. 查看状态

```bash
# 查看当前状态
morty stat

# JSON 格式输出
morty stat --json

# 持续监控（每秒刷新）
morty stat --watch
```

### 3. 开始执行

```bash
# 自动按拓扑序执行所有 jobs
morty doing

# V2 格式下不需要指定模块和 job，系统会自动选择下一个
```

## V2 格式示例

### 完整结构

```json
{
  "version": "2.0",
  "global": {
    "status": "RUNNING",
    "start_time": "2026-03-01T12:00:00+08:00",
    "last_update": "2026-03-01T12:30:00+08:00",
    "current_module_index": 1,
    "current_job_index": 5,
    "total_modules": 5,
    "total_jobs": 20
  },
  "modules": [
    {
      "index": 0,
      "name": "quicksort_core",
      "display_name": "快速排序核心算法",
      "plan_file": "quicksort_core.md",
      "status": "COMPLETED",
      "dependencies": [],
      "jobs": [
        {
          "index": 0,
          "global_index": 0,
          "name": "实现分区函数",
          "status": "COMPLETED",
          "prerequisites": [],
          "tasks_total": 4,
          "tasks_completed": 4,
          "loop_count": 0,
          "retry_count": 0,
          "tasks": [...],
          "created_at": "2026-03-01T12:00:00+08:00",
          "updated_at": "2026-03-01T12:10:00+08:00"
        }
      ],
      "created_at": "2026-03-01T12:00:00+08:00",
      "updated_at": "2026-03-01T12:15:00+08:00"
    }
  ]
}
```

## 与 V1 的区别

| 特性 | V1 (Map) | V2 (Array) |
|------|----------|------------|
| 模块存储 | `modules: { "name": {...} }` | `modules: [{...}]` |
| 顺序 | 无序（map） | 有序（拓扑排序） |
| 查找 | O(1) 按名称 | O(n) 遍历，但执行时 O(1) |
| 依赖检查 | 运行时 | 生成时 |
| 执行逻辑 | 复杂（排序+检查） | 简单（顺序遍历） |
| 可读性 | 差 | 好 |

## 命令变化

### morty init-status（新命令）

生成 V2 格式的 status.json。

**用法**:
```bash
morty init-status [--force]
```

**选项**:
- `--force, -f`: 强制覆盖已存在的 status.json

**示例**:
```bash
# 首次生成
morty init-status

# 重新生成（覆盖现有文件）
morty init-status --force
```

### morty doing（简化）

V2 格式下，doing 命令更简单：

**V1 用法**:
```bash
morty doing -m module_name -j job_name  # 需要指定
morty doing                              # 需要复杂的拓扑排序
```

**V2 用法**:
```bash
morty doing  # 自动按顺序执行，无需指定模块和 job
```

**行为**:
- 自动找到第一个 PENDING 状态的 job
- 执行完成后自动进入下一个
- 不需要检查前置条件（顺序已保证）

### morty stat（增强）

V2 格式下，stat 命令显示更直观：

**输出示例**:
```
═══════════════════════════════════════════════════════════════
  Morty Status (V2)
═══════════════════════════════════════════════════════════════

Overall Status: ▶️ RUNNING
Progress: 8/20 jobs completed (40.0%)
Modules: 2/5 completed
Last Update: 2026-03-01 12:30:45

Module Progress:
───────────────────────────────────────────────────────────────
  ✅ [1] 快速排序核心算法
      Jobs: 4/4

  🔄 [2] 文件读写处理
      Jobs: 2/4 (running: 1)
      Dependencies: quicksort_core
      Jobs:
        ✅ [2.1] 实现文件读取函数
        ✅ [2.2] 实现文件写入函数
        ▶️ [2.3] 实现错误处理和格式兼容 (2/6 tasks)

  ⏳ [3] 命令行接口
      Jobs: 0/3
      Dependencies: quicksort_core, file_handler

  ⏳ [4] 测试套件
      Jobs: 0/5
      Dependencies: quicksort_core, file_handler, cli_interface

  ⏳ [5] 端到端测试
      Jobs: 0/3
      Dependencies: __ALL__

Current Execution:
───────────────────────────────────────────────────────────────
  Module: 文件读写处理
  Job: 实现错误处理和格式兼容 (job 3/4 in module)
  Progress: 2/6 tasks completed
  Loop: 0, Retry: 0

═══════════════════════════════════════════════════════════════
```

## 迁移指南

### 从 V1 迁移到 V2

#### 步骤 1: 备份现有 status.json

```bash
cp .morty/status.json .morty/status.json.v1.backup
```

#### 步骤 2: 生成 V2 格式

```bash
morty init-status --force
```

#### 步骤 3: 验证

```bash
# 检查模块顺序是否正确
morty stat

# 查看 JSON 结构
cat .morty/status.json | jq '.modules[] | {index, name, display_name, dependencies}'
```

#### 步骤 4: 测试执行

```bash
# 试运行一个 job
morty doing
```

### 如果需要回滚

```bash
# 恢复 V1 备份
cp .morty/status.json.v1.backup .morty/status.json
```

## 常见问题

### Q: V2 格式是否向后兼容？

A: 不完全兼容。V2 是新的格式，需要重新生成 status.json。但 morty 可以自动检测版本并使用相应的处理逻辑。

### Q: 如何知道当前使用的是哪个版本？

A: 查看 status.json 中的 `version` 字段：
```bash
cat .morty/status.json | jq '.version'
```

输出：
- `"2.0"` - V2 格式
- `null` 或不存在 - V1 格式

### Q: V2 格式下如何手动指定执行某个 job？

A: V2 格式设计为顺序执行，不建议跳过。如果确实需要，可以：
1. 手动修改 status.json，将前面的 jobs 标记为 COMPLETED
2. 或者使用 `--skip` 标志（如果实现）

### Q: 循环依赖如何处理？

A: `morty init-status` 会在生成时检测循环依赖，如果发现会报错并拒绝生成：

```
Error: cycle detected in module dependencies
```

需要修改 plan 文件，解除循环依赖。

### Q: 模块顺序不对怎么办？

A: 检查 plan 文件中的依赖声明：
```bash
# 查看所有模块的依赖
for f in .morty/plan/*.md; do
    echo "=== $f ==="
    grep "^\*\*依赖模块\*\*" "$f"
done
```

确保依赖关系正确后，重新生成：
```bash
morty init-status --force
```

### Q: Job 顺序不对怎么办？

A: 检查 plan 文件中 job 的前置条件：
```bash
# 查看某个模块的 job 前置条件
grep -A 3 "#### 前置条件" .morty/plan/module_name.md
```

确保前置条件格式正确：
- 同模块依赖: `job_1 - 描述`
- 跨模块依赖: `module:job_2 - 描述`

修复后重新生成。

## 最佳实践

### 1. 定期重新生成

如果修改了 plan 文件，记得重新生成 status.json：

```bash
morty init-status --force
```

### 2. 检查生成结果

生成后检查模块顺序：

```bash
cat .morty/status.json | jq -r '.modules[] | "\(.index + 1). \(.display_name) (deps: \(.dependencies | join(\", \")))"'
```

期望输出（拓扑序）：
```
1. 快速排序核心算法 (deps: )
2. 文件读写处理 (deps: quicksort_core)
3. 命令行接口 (deps: quicksort_core, file_handler)
4. 测试套件 (deps: quicksort_core, file_handler, cli_interface)
5. 端到端测试 (deps: __ALL__)
```

### 3. 版本控制

**不要**将 status.json 加入 git：

```bash
# .gitignore
.morty/status.json
.morty/logs/
```

每次 clone 后重新生成：

```bash
git clone <repo>
cd <repo>
morty init-status
```

### 4. CI/CD 集成

```yaml
# .github/workflows/morty.yml
- name: Generate status.json
  run: morty init-status

- name: Verify topological order
  run: morty stat

- name: Run jobs
  run: morty doing
```

## 技术细节

### 拓扑排序算法

使用 Kahn's Algorithm：

1. 计算每个节点的入度（依赖数）
2. 将入度为 0 的节点加入队列
3. 从队列取出节点，加入结果
4. 更新依赖该节点的其他节点的入度
5. 重复直到队列为空
6. 如果结果数量 < 节点总数，说明有循环

### __ALL__ 展开

`__ALL__` 依赖会在生成时展开为所有其他模块：

```json
// plan 文件中
{
  "dependencies": ["__ALL__"]
}

// 生成的 status.json
{
  "dependencies": ["module1", "module2", "module3", "module4"]
}
```

### Global Index

每个 job 有两个索引：
- `index`: 模块内索引（0-based）
- `global_index`: 全局索引（0-based）

全局索引用于快速定位当前执行的 job。

## 相关文档

- [Status.json V2 设计文档](status-json-v2-design.md)
- [Plan 文件格式指南](PLAN_FORMAT_GUIDE.md)
- [拓扑排序算法](https://en.wikipedia.org/wiki/Topological_sorting)
