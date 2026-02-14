# Reset 功能 - 版本管理和回滚

## 概述

Morty 现在支持完整的 Git 版本管理,每次循环自动创建 commit,并提供 reset 命令回滚到任意循环状态。

## 核心功能

### 1. Git 自动初始化

当你首次运行 `morty loop` 时,如果项目目录没有 git 仓库,Morty 会自动初始化:

```bash
# 自动执行以下操作:
git init
# 创建 .gitignore
# 创建初始提交
```

**自动创建的 .gitignore:**
```
# Morty 临时文件
.morty/logs/*.log
.morty/.session_id
.morty/status.json

# 常见临时文件
*.pyc
__pycache__/
node_modules/
.DS_Store
*.swp
*.swo
*~
```

### 2. 每次循环自动提交

每个循环结束后,Morty 会自动创建一个 commit:

**提交信息格式:**
```
morty: Loop #5 - completed

自动提交由 Morty 开发循环创建。

循环信息:
- 循环编号: #5
- 状态: completed
- 时间戳: 2024-01-15T10:30:45Z
- 父提交: abc123

变更统计:
- 文件数: 10
- 新增行: +234
- 删除行: -56

变更文件:
  - src/main.py
  - src/utils.py
  - tests/test_main.py
  - README.md
  ... 还有 6 个文件

---
此提交代表循环 #5 的完整状态。
使用 'morty reset -c <commit-id>' 可以回滚到此状态。

Co-Authored-By: Claude Code (Morty Loop)
Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

**特点:**
- 包含循环编号和状态
- 显示详细的变更统计
- 列出所有变更文件
- 即使循环出错也会创建 commit(状态为 "error")

### 3. Reset 命令

`morty reset` 提供三个主要功能:

#### 查看循环历史

```bash
morty reset -l
# 或
morty reset --list 30  # 显示最近 30 次提交
```

**输出示例:**
```
╔════════════════════════════════════════════════════════════╗
║              循环提交历史                                  ║
╚════════════════════════════════════════════════════════════╝

abc123 - 2024-01-15 10:30 - morty: Loop #5 - completed
def456 - 2024-01-15 10:25 - morty: Loop #4 - completed
ghi789 - 2024-01-15 10:20 - morty: Loop #3 - completed
jkl012 - 2024-01-15 10:15 - morty: Loop #2 - error
mno345 - 2024-01-15 10:10 - morty: Loop #1 - completed

使用 'morty reset -c <commit-hash>' 回滚到指定循环
使用 'git show <commit-hash>' 查看提交详情
```

#### 回滚到指定 commit

```bash
morty reset -c abc123
```

**执行流程:**
1. 显示目标 commit 信息
2. 显示将要丢弃的 commit 列表
3. 显示未提交的变更(如果有)
4. 要求用户确认(输入 "yes")
5. 关闭所有 morty-loop tmux 会话
6. 执行 `git reset --hard <commit-id>`
7. 清理未跟踪的文件(保留日志)

**示例输出:**
```
╔════════════════════════════════════════════════════════════╗
║              MORTY RESET - 版本回滚                        ║
╚════════════════════════════════════════════════════════════╝

目标 commit:
  abc123 - morty: Loop #5 - completed

将丢弃 3 个提交:
  def456 - morty: Loop #8 - completed
  ghi789 - morty: Loop #7 - completed
  jkl012 - morty: Loop #6 - completed

⚠️  警告: 此操作将丢弃所有指定 commit 之后的变更!

确认回滚? (yes/no): yes

开始回滚...

关闭 tmux 会话...
  - 关闭会话: morty-loop-1707900000
✓ Tmux 会话已关闭

执行 git reset --hard abc123...
✓ 代码已回滚到 commit abc123

清理未跟踪的文件...
✓ 清理完成(日志已保留)

════════════════════════════════════════════════════════════
回滚完成!

当前状态:
  abc123 - morty: Loop #5 - completed

下一步:
  1. 检查代码状态: git status
  2. 可选: 手动修改代码进行干预
  3. 继续循环: morty loop
```

#### 查看当前状态

```bash
morty reset -s
# 或
morty reset --status
```

**输出示例:**
```
╔════════════════════════════════════════════════════════════╗
║              当前状态                                      ║
╚════════════════════════════════════════════════════════════╝

当前 commit:
  abc123 - 2024-01-15 10:30 - morty: Loop #5 - completed

最近的循环: #5

工作区干净,没有未提交的变更

没有运行中的 tmux 会话

.morty/ 目录存在
  - PROMPT.md: ✓
  - fix_plan.md: ✓
  - AGENT.md: ✓
  - specs/: ✓ (3 个文件)
```

## 完整工作流程

### 场景 1: 正常开发循环

```bash
# 1. 初始化项目
morty fix prd.md

# 2. 启动循环(自动初始化 git)
morty loop

# 循环会自动:
# - 检查并初始化 git 仓库
# - 执行开发任务
# - 每次循环结束后创建 commit

# 3. 查看循环历史
morty reset -l

# 输出:
# abc123 - morty: Loop #5 - completed
# def456 - morty: Loop #4 - completed
# ghi789 - morty: Loop #3 - completed
```

### 场景 2: 发现问题需要回滚

```bash
# 1. 查看循环历史
morty reset -l

# 2. 找到出问题前的 commit
# 假设 Loop #5 出现问题,想回滚到 Loop #3

# 3. 回滚到 Loop #3
morty reset -c ghi789

# 确认后会:
# - 关闭 tmux 会话
# - 回滚代码
# - 保留日志

# 4. 检查代码状态
git status

# 5. 继续循环
morty loop
```

### 场景 3: 回滚后人工干预

```bash
# 1. 回滚到指定 commit
morty reset -c abc123

# 2. 手动修改代码
vim src/main.py
# 修复 bug 或调整逻辑

# 3. 查看变更
git status
git diff

# 4. 可选: 手动提交变更
git add src/main.py
git commit -m "fix: 人工修复 bug"

# 5. 继续循环(从当前状态)
morty loop

# Morty 会:
# - 检测到 .morty/ 目录存在
# - 从当前状态继续执行
# - 继续创建循环提交
```

### 场景 4: 查看某个循环的详细信息

```bash
# 1. 查看循环历史
morty reset -l

# 2. 查看特定循环的详细信息
git show abc123

# 输出完整的 commit 信息和 diff
```

## 技术细节

### Git 管理库 (lib/git_manager.sh)

**主要函数:**

1. `init_git_if_needed()`
   - 检查 .git 目录是否存在
   - 初始化 git 仓库
   - 创建 .gitignore
   - 创建初始提交

2. `create_loop_commit(loop_count, status)`
   - 检查是否有变更
   - 暂存所有变更 (git add -A)
   - 生成详细的提交信息
   - 创建 commit
   - 显示提交结果

3. `show_loop_history(limit)`
   - 使用 git log 查找 morty loop 提交
   - 格式化显示提交历史

4. `get_current_loop_number()`
   - 从最近的提交中提取循环编号
   - 用于继续循环时确定起始编号

5. `has_uncommitted_changes()`
   - 检查工作区和暂存区
   - 检查未跟踪的文件

6. `show_uncommitted_changes()`
   - 显示已修改的文件
   - 显示已暂存的文件
   - 显示未跟踪的文件

### 循环集成

在 `lib/loop_monitor.sh` 的循环脚本中:

```bash
# 循环开始时初始化 git
init_git_if_needed

# 每次循环结束后创建提交
create_loop_commit "$LOOP_COUNT" "completed"

# 如果出错也创建提交
if [[ $CLAUDE_EXIT_CODE -ne 0 ]]; then
    create_loop_commit "$LOOP_COUNT" "error"
fi
```

### Reset 实现

在 `morty_reset.sh` 中:

```bash
# 关闭 tmux 会话
stop_tmux_sessions() {
    # 查找所有 morty-loop 会话
    # 逐个关闭
}

# 回滚到指定 commit
reset_to_commit() {
    # 1. 验证 commit 存在
    # 2. 显示目标信息
    # 3. 显示将要丢弃的 commit
    # 4. 要求用户确认
    # 5. 关闭 tmux
    # 6. git reset --hard
    # 7. git clean -fd (保留日志)
}
```

## 安全特性

### 1. 交互式确认

回滚操作需要输入 "yes" 确认,防止误操作:

```bash
⚠️  警告: 此操作将丢弃所有指定 commit 之后的变更!

确认回滚? (yes/no): yes
```

### 2. 日志保留

回滚时保留所有日志文件:

```bash
git clean -fd -e ".morty/logs/*"
```

所有循环日志保留在 `.morty/logs/` 目录中,方便调试和分析。

### 3. 未提交变更警告

如果有未提交的变更,reset 会显示警告:

```bash
检测到未提交的变更,这些变更将被丢弃:

已修改的文件:
  M src/main.py
  M src/utils.py

未跟踪的文件:
  ? new_file.py
```

### 4. Commit 验证

Reset 会验证 commit ID 是否存在:

```bash
if ! git rev-parse --verify "$commit_id^{commit}" &>/dev/null; then
    log ERROR "无效的 commit ID: $commit_id"
    return 1
fi
```

## 常见问题

### Q: 回滚后日志会丢失吗?

不会。Reset 操作会保留所有 `.morty/logs/` 目录中的日志文件。

### Q: 可以回滚到非 morty 创建的 commit 吗?

可以。`morty reset -c` 可以回滚到任何有效的 commit,不限于 morty 创建的。

### Q: 回滚后可以再次前进吗?

可以。使用 `git reflog` 查看所有 commit 历史,然后用 `morty reset -c` 回滚到任何位置。

### Q: 循环出错时会创建 commit 吗?

会。即使循环出错,Morty 也会创建 commit,状态标记为 "error",方便追踪问题。

### Q: 人工修改代码后,循环会继续吗?

会。Morty 检测到 `.morty/` 目录存在时,会从当前状态继续执行循环。

### Q: 如何查看某个循环做了什么变更?

使用 `git show <commit-hash>` 查看完整的 commit 信息和 diff。

### Q: Reset 会影响远程仓库吗?

不会。Reset 只影响本地仓库。如果需要同步到远程,需要手动 push(可能需要 force push)。

## 最佳实践

1. **定期查看循环历史**
   ```bash
   morty reset -l
   ```

2. **在重要节点手动创建标签**
   ```bash
   git tag -a v0.1.0 -m "功能 A 完成"
   ```

3. **使用 git show 查看详细变更**
   ```bash
   git show abc123
   ```

4. **回滚前检查状态**
   ```bash
   morty reset -s
   ```

5. **人工干预后添加说明性提交**
   ```bash
   git commit -m "fix: 人工修复循环 #5 的问题"
   ```

6. **保留重要的循环日志**
   ```bash
   cp .morty/logs/loop_5_output.log ~/important_logs/
   ```

## 总结

Reset 功能提供了完整的版本管理能力:

- ✅ 自动 Git 初始化
- ✅ 每次循环自动提交
- ✅ 详细的提交信息
- ✅ 灵活的回滚机制
- ✅ 支持人工干预
- ✅ 保留完整日志
- ✅ 安全的确认机制

这使得 Morty 的开发循环更加可控和可追溯,你可以随时回滚到任何循环状态,进行人工干预,然后继续循环。
