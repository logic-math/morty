# Plan: version_manager

## 模块概述

**模块职责**: 提供版本管理集成，支持自动提交、循环历史追踪和版本回滚，确保开发过程的完整可追溯性。同时提供 reset 命令的实现基础。

**对应 Research**: Git 版本管理集成；reset 模式；每次循环自动提交

**依赖模块**: config, logging

**被依赖模块**: doing, cli(reset)

## 接口定义

### 输入接口
- Git 仓库状态变更
- 循环执行完成信号
- 回滚请求（commit ID）

### 输出接口
- `version_init_if_needed()`: 初始化 Git 仓库
- `version_create_loop_commit(loop_number, status)`: 创建循环提交
- `version_show_loop_history(n)`: 显示最近 N 次循环提交
- `version_reset_to_commit(commit_id)`: 回滚到指定提交
- `version_get_current_loop_number()`: 获取当前循环编号

## 数据模型

### 提交信息格式
```
morty: Loop #N - <状态描述>

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

### Job 1: 版本管理基础

**目标**: 建立版本管理的基础功能，包括仓库初始化和状态检测

**前置条件**: config, logging 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `lib/version_manager.sh` 模块
- [ ] 实现 `version_init_if_needed()`: 检查并初始化 Git 仓库
- [ ] 实现 `version_has_uncommitted_changes()`: 检测未提交变更
- [ ] 实现 `version_get_repo_root()`: 获取仓库根目录
- [ ] 实现 `version_is_ignored(path)`: 检查路径是否被 gitignore

**验证器**:
- 在非 Git 目录调用 `version_init_if_needed()` 应初始化新的 Git 仓库
- 在已有 Git 仓库目录调用应正常返回，不重复初始化
- `version_has_uncommitted_changes()` 在有未提交文件时返回 true，否则返回 false
- `version_get_repo_root()` 应返回正确的仓库根目录绝对路径
- 当不在 Git 仓库内时，函数应返回错误码而非抛出异常

**调试日志**:
- 无

---

### Job 2: 循环提交管理

**目标**: 实现循环自动提交功能，包含详细的提交信息和元数据

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `version_create_loop_commit()`: 创建格式化的循环提交
- [ ] 实现提交信息生成器（包含循环编号、状态、变更统计）
- [ ] 实现自动暂存变更（受 gitignore 限制）
- [ ] 实现提交前的变更检查（无变更则不提交）
- [ ] 实现 `version_get_current_loop_number()`: 从提交历史解析当前循环编号

**验证器**:
- 调用 `version_create_loop_commit 5 completed` 后，应创建包含 "morty: Loop #5" 的提交
- 提交信息应包含完整的循环元数据（时间戳、变更统计、文件列表）
- 无变更时调用应跳过提交并返回相应状态
- `version_get_current_loop_number()` 应从最新提交中解析出正确的循环编号
- 提交应保留 `.morty/logs/` 目录（不应被清理）

**调试日志**:
- 无

---

### Job 3: 版本回滚 (reset 命令基础)

**目标**: 实现版本回滚功能，支持 reset 命令

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `version_reset_to_commit()`: 回滚到指定 commit
- [ ] 实现 `version_show_loop_history(n)`: 显示最近 N 次循环提交
- [ ] 实现 `version_get_loop_by_number(n)`: 获取指定循环的提交信息
- [ ] 实现回滚前自动创建备份分支
- [ ] 实现 `.morty/logs/` 目录保护（回滚时不删除）

**验证器**:
- 调用 `version_reset_to_commit abc123` 后，HEAD 应指向指定 commit
- `version_show_loop_history 10` 应显示最近 10 次循环提交
- 回滚前应自动创建备份分支
- `.morty/logs/` 目录在回滚后应仍然存在
- 回滚后 `version_get_current_loop_number()` 应返回正确编号

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 所有版本操作可以正确追踪循环历史
- 与 reset 命令的集成正常工作
- 回滚后可以正确恢复到之前状态

---

## 待实现方法签名

```bash
# lib/version_manager.sh

# 仓库管理
version_init_if_needed()
version_has_uncommitted_changes()
version_get_repo_root()
version_is_ignored(path)

# 循环提交
version_create_loop_commit(loop_number, status, message="")
version_get_current_loop_number()
version_get_last_loop_commit(loop_number)

# 历史查询
version_show_loop_history(n=10)
version_get_loop_by_number(loop_number)
version_parse_loop_commit(commit_hash)

# 版本回滚 (用于 reset 命令)
version_reset_to_commit(commit_id, backup=true)
version_reset_to_loop(loop_number, backup=true)
version_create_backup_branch()
```
