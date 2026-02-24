# Plan: Git

## 模块概述

**模块职责**: 实现 Git 版本管理功能，包括仓库初始化、循环提交、版本回滚和历史查看

**对应 Research**:
- `morty-go-refactor-plan.md` 第 4.5 节 Git 模块接口定义
- `morty-project-research.md` 第 3.7 节 Git 管理分析

**现有实现参考**:
- 原 Shell 版本: `lib/git_manager.sh`, `lib/version_manager.sh`
- 提交信息格式: `morty: Loop #N - <status>`

**依赖模块**: 无 (基础模块)

**被依赖模块**: Executor

---

## 接口定义

### 输入接口
- Git 仓库目录路径
- 循环编号和状态
- 提交哈希 (用于回滚)

### 输出接口
- `Manager` 接口实现
- 提交哈希
- 变更统计
- 循环提交历史

---

## 数据模型

```go
// Manager Git 管理接口
type Manager interface {
    InitIfNeeded(dir string) error
    HasUncommittedChanges(dir string) (bool, error)
    GetRepoRoot(dir string) (string, error)
    CreateLoopCommit(loopNumber int, status string, dir string) (string, error)
    GetCurrentLoopNumber(dir string) (int, error)
    GetChangeStats(dir string) (*ChangeStats, error)
    ResetToCommit(commitHash string, dir string) error
    ShowLoopHistory(n int, dir string) ([]LoopCommit, error)
    CreateBackupBranch(dir string) (string, error)
}

// ChangeStats 变更统计
type ChangeStats struct {
    Staged    int `json:"staged"`
    Unstaged  int `json:"unstaged"`
    Untracked int `json:"untracked"`
    Added     int `json:"added"`
    Deleted   int `json:"deleted"`
}

// LoopCommit 循环提交信息
type LoopCommit struct {
    Hash        string    `json:"hash"`
    LoopNumber  int       `json:"loop_number"`
    Status      string    `json:"status"`
    Timestamp   time.Time `json:"timestamp"`
    Message     string    `json:"message"`
    Author      string    `json:"author"`
}
```

---

## Jobs (Loop 块列表)

---

### Job 1: Git 基础操作实现

**目标**: 实现 Git 基础操作，包括初始化和变更检测

**前置条件**:
- 无

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/git/git.go` 定义 Git 接口
- [ ] Task 2: 创建 `internal/git/manager.go` 实现 Manager
- [ ] Task 3: 实现 `InitIfNeeded(dir string) error` 自动初始化仓库
- [ ] Task 4: 实现 `HasUncommittedChanges(dir string) (bool, error)`
- [ ] Task 5: 实现 `GetRepoRoot(dir string) (string, error)`
- [ ] Task 6: 实现 `GetChangeStats(dir string) (*ChangeStats, error)`
- [ ] Task 7: 处理 Git 命令执行和错误处理
- [ ] Task 8: 编写单元测试 `manager_test.go`

**验证器**:
- [ ] InitIfNeeded 在无仓库目录创建 Git 仓库
- [ ] InitIfNeeded 在已有仓库目录不做任何操作
- [ ] HasUncommittedChanges 返回正确结果
- [ ] GetRepoRoot 返回正确的仓库根目录
- [ ] GetChangeStats 返回正确的变更统计
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 2: 循环提交实现

**目标**: 实现 Morty 循环提交功能，包含规范提交信息

**前置条件**:
- Job 1 完成 (Git 基础)

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/git/commit.go` 文件结构
- [x] Task 2: 实现 `CreateLoopCommit(loopNumber int, status string, dir string) (string, error)`
- [x] Task 3: 生成规范提交信息格式:
  ```
  morty: loop N - <status>

  Change Statistics:
  - Files added: N
  - Files modified: N
  - Files deleted: N
  - Lines added: N
  - Lines deleted: N
  ```
- [x] Task 4: 自动添加所有变更到暂存区
- [x] Task 5: 实现 `GetCurrentLoopNumber(dir string) (int, error)`
- [x] Task 6: 实现 `CreateBackupBranch(dir string) (string, error)`
- [x] Task 7: 编写单元测试 `commit_test.go`

**验证器**:
- [x] CreateLoopCommit 创建正确的提交
- [x] 提交信息包含循环编号和状态
- [x] 提交信息包含变更统计
- [x] GetCurrentLoopNumber 返回正确的下一个循环编号
- [x] CreateBackupBranch 创建备份分支
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: Go模块路径解析错误, 测试文件未被发现, 猜想: 1)Go缓存问题 2)工作目录映射错误, 验证: 检查go list输出, 修复: 发现实际模块路径在/home/sankuai/下, 将文件复制到正确位置, 已修复
- debug2: commit.go编译失败, 运行go test时报错unused variable, 猜想: 1)变量声明未使用 2)遗漏了_cleanup_, 验证: 检查commit.go第67行, 修复: 将output改为_忽略返回值, 已修复

---

### Job 3: 版本回滚实现

**目标**: 实现版本回滚和历史查看功能

**前置条件**:
- Job 2 完成 (循环提交)

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/git/version.go` 文件结构
- [ ] Task 2: 实现 `ResetToCommit(commitHash string, dir string) error`
- [ ] Task 3: 回滚前创建备份分支
- [ ] Task 4: 支持 hard reset 和 soft reset 选项
- [ ] Task 5: 实现 `ShowLoopHistory(n int, dir string) ([]LoopCommit, error)`
- [ ] Task 6: 解析提交信息提取循环编号和状态
- [ ] Task 7: 按时间倒序返回历史记录
- [ ] Task 8: 实现历史记录格式化输出
- [ ] Task 9: 编写单元测试 `version_test.go`

**验证器**:
- [ ] ResetToCommit 正确回滚到指定提交
- [ ] 回滚前自动创建备份分支
- [ ] ShowLoopHistory 返回最近的 N 条循环提交
- [ ] 循环提交正确解析出编号和状态
- [ ] 历史记录按时间倒序排列
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的 Git 工作流: 初始化 → 修改 → 提交 → 查看历史 → 回滚
- [ ] 循环提交序列正确 (loop:1, loop:2, ...)
- [ ] 回滚后状态正确恢复
- [ ] 备份分支正确创建
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充
