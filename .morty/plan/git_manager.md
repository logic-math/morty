# Plan: git_manager

## 模块概述

**模块职责**: 提供 Git 版本管理集成，支持自动提交、循环历史追踪和版本回滚，确保开发过程的完整可追溯性。

**对应 Research**: Git 版本管理集成；reset 模式；每次循环自动提交

**依赖模块**: config, logging

**被依赖模块**: plan, doing

## 接口定义

### 输入接口
- Git 仓库状态变更
- 循环执行完成信号
- 回滚请求（commit ID）

### 输出接口
- `git_init_if_needed()`: 初始化 Git 仓库
- `git_create_loop_commit(loop_number, status)`: 创建循环提交
- `git_show_loop_history(n)`: 显示最近 N 次循环提交
- `git_reset_to_commit(commit_id)`: 回滚到指定提交
- `git_get_current_loop_number()`: 获取当前循环编号

## 数据模型

### 提交信息格式
```
变更总结性描述
详细变更描述
- 在[模块 A]中变更了 xxx 文件, 实现了 xxx 功能
- 在[模块 A]中变更了 xxx 文件, 解决了 xxx 问题
```

### 循环历史数据结构
```json
{
  "loops": [
    {
      "number": 5,
      "commit": "abc123",
      "status": "completed",
      "timestamp": "2026-02-20T14:30:00Z",
      "message": "morty: Loop #5 - completed",
      "files_changed": 5,
      "insertions": 100,
      "deletions": 20
    }
  ],
  "current_loop": 5
}
```

## Jobs (Loop 块列表)

---

### Job 1: Git 仓库管理基础

**目标**: 建立 Git 管理的基础功能，包括仓库初始化和状态检测

**前置条件**: config, logging 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `lib/git_manager.sh` 模块
- [ ] 实现 `git_init_if_needed()`: 检查并初始化 Git 仓库
- [ ] 实现 `git_has_uncommitted_changes()`: 检测未提交变更
- [ ] 实现 `git_get_repo_root()`: 获取仓库根目录
- [ ] 实现 `git_is_ignored(path)`: 检查路径是否被 gitignore

**验证器**:
- 在非 Git 目录调用 `git_init_if_needed()` 应初始化新的 Git 仓库
- 在已有 Git 仓库目录调用应正常返回，不重复初始化
- `git_has_uncommitted_changes()` 在有未提交文件时返回 true，否则返回 false
- `git_get_repo_root()` 应返回正确的仓库根目录绝对路径
- 当不在 Git 仓库内时，函数应返回错误码而非抛出异常

**调试日志**:
- 无

---

### Job 2: 循环提交管理

**目标**: 实现循环自动提交功能，包含详细的提交信息和元数据

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `git_create_loop_commit()`: 创建格式化的循环提交
- [ ] 实现提交信息生成器（包含循环编号、状态、变更统计）
- [ ] 实现自动暂存变更（受 gitignore 限制）
- [ ] 实现提交前的变更检查（无变更则不提交）
- [ ] 实现 `git_get_current_loop_number()`: 从提交历史解析当前循环编号

**验证器**:
- 调用 `git_create_loop_commit 5 completed` 后，应创建包含 "morty: Loop #5" 的提交
- 提交信息应包含完整的循环元数据（时间戳、变更统计、文件列表）
- 无变更时调用应跳过提交并返回相应状态
- `git_get_current_loop_number()` 应从最新提交中解析出正确的循环编号
- 提交应保留 `.morty/logs/` 目录（不应被清理）

**调试日志**:
- 无

---

### Job 3: 循环历史查询

**目标**: 实现循环历史的查询和展示功能

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `git_show_loop_history(n)`: 显示最近 N 次循环提交
- [ ] 实现提交信息解析器（从提交消息提取元数据）
- [ ] 实现格式化输出（表格或列表形式）
- [ ] 支持按状态过滤（只显示 completed/failed）
- [ ] 实现 `git_get_loop_by_number(n)`: 获取指定循环的提交信息

**验证器**:
- `git_show_loop_history 10` 应显示最近 10 次循环提交的信息
- 输出应包含循环编号、状态、时间戳、变更统计
- 无循环提交历史时应显示友好提示
- `git_get_loop_by_number 5` 应准确找到 Loop #5 的提交
- 查询不存在的循环编号应返回空值而非错误

**调试日志**:
- 无

---
## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 所有 Git 操作可以正确追踪循环历史
- 与 reset 模式的集成正常工作

---

## 待实现方法签名

```bash
# lib/git_manager.sh

# 仓库管理
git_init_if_needed()
git_has_uncommitted_changes()
git_get_repo_root()
git_is_ignored(path)

# 循环提交
git_create_loop_commit(loop_number, status, message="")
git_get_current_loop_number()
git_get_last_loop_commit(loop_number)

# 历史查询
git_show_loop_history(n=10)
git_get_loop_by_number(loop_number)
git_parse_loop_commit(commit_hash)

# 版本回滚
git_reset_to_commit(commit_id, backup=true)
git_reset_to_loop(loop_number, backup=true)
git_create_backup_branch()
git_restore_from_backup(branch_name)

# 里程碑管理
git_create_milestone(name, description="")
git_list_milestones()
git_checkout_milestone(name)
git_delete_milestone(name)

# 实验分支
git_create_experiment(name)
git_list_experiments()
git_merge_experiment(name)
git_delete_experiment(name)
```
