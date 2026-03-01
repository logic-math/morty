# Morty V2 快速上手指南

## 立即开始

### 1. 准备 Plan 文件

确保你的 `.morty/plan/` 目录中有正确格式的 plan 文件。

检查格式：
```bash
morty plan validate --verbose
```

### 2. 生成 Status.json V2

```bash
# 删除旧的 status.json（如果存在）
rm .morty/status.json

# 生成 V2 格式
morty init-status
```

输出示例：
```
✅ Status file generated successfully

File: .morty/status.json
Version: 2.0
Modules: 5
Jobs: 20

Module execution order:
  1. 快速排序核心算法 (4 jobs, deps: none)
  2. 文件读写处理 (4 jobs, deps: [quicksort_core])
  3. 命令行接口 (3 jobs, deps: [quicksort_core file_handler])
  4. 测试套件 (5 jobs, deps: [quicksort_core file_handler cli_interface])
  5. 端到端测试 (3 jobs, deps: [__ALL__])

Next step: Run 'morty doing' to start execution
```

### 3. 验证顺序

```bash
morty stat
```

你应该看到模块按正确的拓扑顺序排列。

### 4. 开始执行

```bash
morty doing
```

就这么简单！系统会自动按顺序执行所有 jobs。

## 关键改进

### 之前（V1）

```bash
# 需要手动指定模块和 job
morty doing -m quicksort_core -j "实现分区函数"

# 或者让系统自动选择（但每次都要重新排序）
morty doing  # 复杂的拓扑排序逻辑
```

**问题**:
- ❌ 可能选错顺序（Bug #1）
- ❌ job_index 可能是 9999（Bug #2）
- ❌ status.json 顺序混乱（Bug #3）

### 现在（V2）

```bash
# 一次生成，永久有序
morty init-status

# 简单执行，自动按序
morty doing
```

**优势**:
- ✅ 顺序保证正确（生成时拓扑排序）
- ✅ 执行逻辑简单（顺序遍历）
- ✅ 可读性好（数组有序）
- ✅ 性能更好（O(1) vs O(n log n)）

## 常用命令

### 查看状态

```bash
# 简洁显示
morty stat

# JSON 格式
morty stat --json

# 持续监控
morty stat --watch
```

### 重新生成

如果修改了 plan 文件：

```bash
morty init-status --force
```

### 验证 Plan 格式

```bash
morty plan validate --verbose
```

## 故障排除

### 问题：模块顺序不对

**检查依赖声明**:
```bash
for f in .morty/plan/*.md; do
    echo "=== $(basename $f) ==="
    grep "^\*\*依赖模块\*\*" "$f"
done
```

**修复后重新生成**:
```bash
morty init-status --force
```

### 问题：循环依赖

**错误信息**:
```
Error: cycle detected in module dependencies
```

**解决**:
1. 检查依赖关系图
2. 找到循环
3. 修改 plan 文件打破循环
4. 重新生成

### 问题：Job 顺序不对

**检查前置条件**:
```bash
grep -A 3 "#### 前置条件" .morty/plan/module_name.md
```

**确保格式正确**:
- 同模块: `job_1 - 描述`
- 跨模块: `module:job_2 - 描述`

## 完整示例

### Caipiao 项目

```bash
# 1. 进入项目
cd /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/caipiao

# 2. 清理旧状态
rm .morty/status.json

# 3. 生成 V2 格式
morty init-status

# 4. 查看顺序
morty stat

# 5. 开始执行
morty doing
```

**期望的模块执行顺序**:
1. 快速排序核心算法 (quicksort_core)
2. 文件读写处理 (file_handler)
3. 命令行接口 (cli_interface)
4. 测试套件 (test_suite)
5. 端到端测试 (e2e_test)

## 下一步

- 阅读 [Status.json V2 设计文档](status-json-v2-design.md)
- 阅读 [Status.json V2 使用指南](status-v2-usage-guide.md)
- 查看 [重构总结](REFACTORING_SUMMARY.md)

## 需要帮助？

- GitHub Issues: https://github.com/morty/morty/issues
- 文档: `docs/` 目录
- 示例: `examples/` 目录
