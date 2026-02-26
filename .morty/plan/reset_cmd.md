# Plan: Reset Command

## 模块概述

**模块职责**: 实现 `morty reset` 命令，提供版本回滚和循环历史查看功能

**对应 Research**:
- `morty-project-research.md` 第 3.6 节 Reset 模式分析

**依赖模块**: Config, Logging, State, Git

**被依赖模块**: CLI (命令注册)

---

## 命令行接口

### 用法

```bash
# 显示最近10次循环提交
morty reset -l

# 显示最近 N 次循环提交
morty reset -l 5

# 回滚到指定提交
morty reset -c abc1234
```

### 选项

| 选项 | 简写 | 说明 | 参数 |
|------|------|------|------|
| `--list` | `-l` | 显示循环历史 | 可选数量（默认10） |
| `--commit` | `-c` | 回滚到指定提交 | 提交哈希 |

---

## 显示格式

### 循环历史输出

```
$ morty reset -l

┌──────────┬───────────────────────────────────────────┬─────────────────────┐
│ CommitID │ Message                                   │ Time                │
├──────────┼───────────────────────────────────────────┼─────────────────────┤
│ abc1234  │ morty[loop:5]: [config/job_2: COMPLETED]  │ 2026-02-23 14:30:00 │
│ def5678  │ morty[loop:4]: [config/job_1: COMPLETED]  │ 2026-02-23 14:15:00 │
│ ghi9012  │ morty[loop:3]: [logging/job_2: FAILED]    │ 2026-02-23 14:00:00 │
│ jkl3456  │ morty[loop:2]: [logging/job_1: COMPLETED] │ 2026-02-23 13:45:00 │
│ mno7890  │ morty[loop:1]: [cli/job_1: COMPLETED]     │ 2026-02-23 13:30:00 │
└──────────┴───────────────────────────────────────────┴─────────────────────┘
```

### 回滚确认

```
$ morty reset -c abc1234

确认回滚到 commit abc1234?
提交信息: morty[loop:2]: [logging/job_1: COMPLETED]
时间: 2026-02-23 13:45:00

这将重置工作目录到该提交状态，未提交的变更将丢失。
[Y/n]: y

正在回滚...
✓ 已回滚到 commit abc1234

当前状态: logging/job_2 PENDING
```

---

## 数据模型

```go
// ResetHandler reset 命令处理器
type ResetHandler struct {
    config       config.Manager
    logger       logging.Logger
    stateManager state.Manager
    gitManager   git.Manager
}

// ResetOptions reset 命令选项
type ResetOptions struct {
    List   bool
    Count  int    // -l 的参数，默认10
    Commit string // -c 的参数
}

// LoopCommit 循环提交信息
type LoopCommit struct {
    Hash       string    `json:"hash"`
    LoopNumber int       `json:"loop_number"`
    Module     string    `json:"module"`
    Job        string    `json:"job"`
    Status     string    `json:"status"`
    Timestamp  time.Time `json:"timestamp"`
    Message    string    `json:"message"`
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: Reset 命令框架

**目标**: 实现 reset 命令的基础框架

**前置条件**:
- Config, Logging, Git 模块完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/cmd/reset.go` 文件
- [x] Task 2: 实现 `ResetHandler` 结构体
- [x] Task 3: 实现参数解析 (`-l`, `-c`)
- [x] Task 4: 检查 Git 仓库存在性
- [x] Task 5: 检查互斥选项（-l 和 -c 不能同时用）
- [x] Task 6: 无选项时的友好提示
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 正确解析 `-l` 和 `-c`
- [x] `-l` 和 `-c` 同时使用时报错
- [x] 无选项时提示使用 `-l` 或 `-c`
- [x] 非 Git 仓库时友好报错
- [x] 所有单元测试通过

**调试日志**:
- explore1: [探索发现] 项目使用标准 handler 模式, 参考 stat.go/doing.go 实现, 配置使用 config.Manager 接口, 日志使用 logging.Logger 接口, 已记录
- debug1: git 包缺少 GetRepoRoot 包级别函数, 在 reset.go 中引用时发现未定义, 修复: 在 git/manager.go 中添加 GetRepoRoot 函数, 已修复

---

### Job 2: 循环历史查询

**目标**: 实现 `-l` 循环历史查看

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `showLoopHistory(count)`
- [ ] Task 2: 调用 Git 获取提交历史
- [ ] Task 3: 过滤 morty 循环提交（按提交信息格式）
- [ ] Task 4: 解析提交信息提取 loop 编号、模块、Job、状态
- [ ] Task 5: 限制返回数量（默认10）
- [ ] Task 6: 处理无循环提交的情况
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 正确获取 Git 提交历史
- [ ] 只显示 morty 循环提交
- [ ] 正确解析 loop 编号
- [ ] 正确解析模块/Job/状态
- [ ] 数量限制正确
- [ ] 无提交时友好提示
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

### Job 3: 表格格式化输出

**目标**: 实现表格形式的历史显示

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现 `formatHistoryTable()`
- [x] Task 2: 计算列宽（自适应）
- [x] Task 3: 格式化 CommitID（短哈希）
- [x] Task 4: 格式化时间
- [x] Task 5: 对齐输出
- [x] Task 6: 彩色输出（状态颜色区分）
- [x] Task 7: 编写单元测试

**验证器**:
- [x] 表格对齐正确
- [x] CommitID 显示短哈希（7位）
- [x] 时间格式统一
- [x] 状态颜色区分（COMPLETED=绿, FAILED=红）
- [x] 所有单元测试通过

**调试日志**:
- debug1: stat_test.go 中 TestTableFormatter_topBorder 测试失败, 检查 border 长度时出错, 猜想: Unicode 字符使用多字节编码，len() 返回字节数而非字符数, 验证: 测试中断言 len(border) == tableWidth，但 border 包含 3 字节 Unicode 字符, 修复: 该测试在 stat_test.go 中，与当前 Job 无关，是预存问题, 已记录
- explore1: [探索发现] 参考 stat.go 中的 TableFormatter 实现，实现了 HistoryTableFormatter，支持 Unicode 表格边框、自适应列宽、状态颜色区分, 已记录

---

### Job 4: 回滚功能实现

**目标**: 实现 `-c` 回滚功能

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `resetToCommit(hash)`
- [ ] Task 2: 验证提交哈希有效性
- [ ] Task 3: 获取提交信息（用于确认提示）
- [ ] Task 4: 交互式确认（Y/n）
- [ ] Task 5: 创建备份分支（可选）
- [ ] Task 6: 执行 `git reset --hard`
- [ ] Task 7: 恢复对应的状态文件
- [ ] Task 8: 编写单元测试

**验证器**:
- [ ] 无效哈希时报错
- [ ] 确认提示显示提交信息
- [ ] 用户取消时无操作
- [ ] 回滚成功
- [ ] 状态文件同步恢复
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

### Job 5: 状态同步

**目标**: 回滚后同步状态文件

**前置条件**:
- Job 4 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 实现 `syncStatusAfterReset(commit)`
- [ ] Task 2: 从提交信息解析回滚到的位置
- [ ] Task 3: 重置该位置之后的所有 Job 为 PENDING
- [ ] Task 4: 更新 `status.json`
- [ ] Task 5: 保持回滚位置之前的完成状态
- [ ] Task 6: 输出当前状态提示
- [ ] Task 7: 编写单元测试

**验证器**:
- [ ] 正确解析回滚位置
- [ ] 之后 Jobs 重置为 PENDING
- [ ] 之前 Jobs 保持 COMPLETED
- [ ] status.json 正确更新
- [ ] 所有单元测试通过

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的 reset 流程
- [ ] 历史查询正确
- [ ] 回滚功能正常
- [ ] 状态同步正确
- [ ] 集成测试通过

**调试日志**:
- 待填充

---

## 使用示例

```bash
# 查看历史
$ morty reset -l
┌──────────┬───────────────────────────────────────────┬─────────────────────┐
│ CommitID │ Message                                   │ Time                │
├──────────┼───────────────────────────────────────────┼─────────────────────┤
│ abc1234  │ morty[loop:5]: [config/job_2: COMPLETED]  │ 2026-02-23 14:30:00 │
│ def5678  │ morty[loop:4]: [config/job_1: COMPLETED]  │ 2026-02-23 14:15:00 │
└──────────┴───────────────────────────────────────────┴─────────────────────┘

# 回滚
$ morty reset -c def5678
确认回滚到 commit def5678?
提交信息: morty[loop:4]: [config/job_1: COMPLETED]
[Y/n]: y
✓ 已回滚到 commit def5678
当前状态: config/job_2 PENDING
```

---

## 文件清单

- `internal/cmd/reset.go` - reset 命令实现
