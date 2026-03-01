# Morty 改进：CLI 输出控制和完成状态标记

## 改进日期
2026-02-28

## 改进概览

本次更新解决了两个关键问题：
1. **CLI 输出污染控制台** - AI CLI 的输出现在只写入日志文件，不打印到控制台
2. **AI 完成状态标记** - 强化 prompt 要求 AI 必须在 plan 文件中标记完成状态

---

## 问题 1: CLI 输出污染控制台

### 问题描述

执行 `morty doing` 时，AI CLI 的所有输出（包括大量 JavaScript 代码和调试信息）都会打印到控制台，严重影响监控质量和用户体验。

**现象**：
```
[2026-02-28T16:15:58.605+08:00] INFO    Starting job execution
[大量 Claude Code CLI 输出...]
[JavaScript 代码...]
[调试信息...]
[2026-02-28T16:20:30.123+08:00] SUCCESS Job completed
```

### 解决方案

修改 `internal/executor/engine.go` 中的输出模式，从 `OutputStream`（输出到控制台）改为 `OutputCapture`（只捕获到内存和日志文件）。

**修改位置**: `internal/executor/engine.go:402-415`

**修改前**：
```go
opts := callcli.Options{
    Timeout:    0,
    Stdin:      prompt,
    WorkingDir: e.config.WorkingDir,
    Output: callcli.OutputConfig{
        Mode: callcli.OutputStream, // Stream output to terminal
    },
}
```

**修改后**：
```go
opts := callcli.Options{
    Timeout:    0,
    Stdin:      prompt,
    WorkingDir: e.config.WorkingDir,
    Output: callcli.OutputConfig{
        Mode: callcli.OutputCapture, // Capture output to memory (don't pollute console)
    },
}
```

### 效果

**改进前**：
- CLI 输出直接打印到控制台
- 大量无用信息干扰监控
- 难以快速查看 morty 自身的日志

**改进后**：
- CLI 输出只写入 `.morty/logs/{module}_{job}_{timestamp}.log`
- 控制台只显示 morty 自身的结构化日志
- 监控质量大幅提升

**示例输出**（控制台）：
```
[2026-02-28T16:25:10.451+08:00] INFO    Starting job execution module=测试模块 job=创建测试文件结构
[2026-02-28T16:25:10.452+08:00] INFO    Job log file created log_file=.morty/logs/测试模块_创建测试文件结构_20260228_162510.log
[2026-02-28T16:27:45.123+08:00] SUCCESS Job completed successfully module=测试模块 job=创建测试文件结构
```

CLI 的完整输出保存在日志文件中，需要时可以查看：
```bash
cat .morty/logs/测试模块_创建测试文件结构_20260228_162510.log
```

---

## 问题 2: AI 完成状态标记缺失

### 问题描述

执行完成后，morty 会发出警告：
```
[2026-02-28T16:15:58.605+08:00] WARN    Job completion not marked in plan file module=测试结构重组 job=创建测试目录结构
```

这是因为 AI 没有在 plan 文件中标记完成状态，导致 morty 无法验证 job 是否真正完成。

### 解决方案

在 `bin/prompts/doing.md` 中强化完成状态标记的要求，并在多个位置提醒 AI。

### 修改内容

#### 1. 添加专门的完成标记步骤

**修改位置**: `bin/prompts/doing.md:83-95`

```markdown
step4.5: [标记Job完成状态] **重要**: 当所有 Tasks 完成且验证通过后，必须在 Plan 文件中标记完成:
       - 读取 `.morty/plan/[模块名].md`
       - 在对应 Job 的末尾添加 **完成状态** 部分
       - 格式: `**完成状态**: ✅ 已完成 (YYYY-MM-DD HH:MM:SS)`
       - 可选: 添加 **验证结果** 部分记录测试通过情况
       - 保存更新后的 Plan 文件
       - **注意**: 如果没有标记完成状态，morty 会发出警告
```

#### 2. 在验证器中强调

**修改位置**: `bin/prompts/doing.md:96-100`

```markdown
0. 如果当前 Job 的所有 Tasks 已完成且验证器通过:
   - **必须**在 Plan 文件中标记完成状态（step4.5）
   - 检查通过，结束循环
```

#### 3. 更新示例展示完成标记

**修改位置**: `bin/prompts/doing.md:327-347`

```markdown
**调试日志**:
- explore1: [探索发现] 项目使用单文件日志实现, lib/logging.sh 为核心模块, 使用文件追加模式写入, 已记录
- debug1: JSON 序列化失败, 复杂对象循环引用, 猜想: 1)缺少循环引用处理 2)未使用 JSON.stringify 的 replacer, 验证: 添加 replacer 函数测试, 修复: 使用 WeakSet 检测循环引用, 待修复

**完成状态**: ✅ 已完成 (2026-02-28 13:45:30)

**验证结果**:
- ✅ JSON 格式输出测试通过
- ✅ 配置切换功能正常
- ⚠️  循环引用处理待后续优化
```

#### 4. 在重要提醒中添加

**修改位置**: `bin/prompts/doing.md:373-380`

```markdown
4. **必须标记完成状态**: Job 完成后，**必须**在 Plan 文件中添加完成标记，否则 morty 会发出警告
   - 格式: `**完成状态**: ✅ 已完成 (YYYY-MM-DD HH:MM:SS)`
   - 这是 morty 验证 Job 真正完成的关键标记
```

### 完成标记格式

在 `.morty/plan/{module}.md` 中，每个完成的 Job 应该包含：

```markdown
### Job N: [Job名称]

**目标**: ...

**Tasks (Todo 列表)**:
- [x] Task 1: ...
- [x] Task 2: ...
- [x] Task 3: ...

**验证器**: ...

**调试日志**:
- debug1: ...

**完成状态**: ✅ 已完成 (2026-02-28 16:30:15)

**验证结果**:
- ✅ 所有测试通过
- ✅ 功能验证正常
- ⚠️  已知问题（如果有）
```

### morty 的验证逻辑

morty 在 job 执行完成后会：

1. 读取 plan 文件
2. 查找对应 job 的 `**完成状态**` 标记
3. 检查是否包含完成标识（✅、已完成、completed、done、finished）
4. 如果找到标记 → 正常完成
5. 如果未找到 → 发出 WARN 警告，但仍然继续（宽松模式）

**相关代码**: `internal/executor/engine.go:200-207`

```go
if !completionVerified {
    e.logger.Warn("Job completion not marked in plan file",
        logging.String("module", module),
        logging.String("job", job),
    )
    // For now, we'll proceed anyway, but log the warning
    // In stricter mode, we could return an error here
}
```

---

## 测试验证

### 构建和安装

```bash
# 构建
bash /opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/morty/scripts/build.sh

# 安装（如果 morty 正在运行，需要先复制）
cp /opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/morty/bin/morty ~/.morty/bin/morty.new
mv ~/.morty/bin/morty.new ~/.morty/bin/morty

# 更新 prompts
cp /opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/morty/bin/prompts/doing.md ~/.morty/prompts/

# 验证版本
morty -version
```

### 功能测试

```bash
# 进入测试目录
cd /home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/test

# 重置状态
morty reset -c

# 执行 job
morty doing

# 观察控制台输出（应该只有 morty 的日志，没有 CLI 输出）
# 检查日志文件
ls -lh .morty/logs/
cat .morty/logs/测试模块_*.log

# 检查 plan 文件中的完成标记
cat .morty/plan/测试模块.md | grep -A 5 "完成状态"
```

### 预期结果

1. **控制台输出干净**：
   ```
   [2026-02-28T16:30:10] INFO    Starting job execution
   [2026-02-28T16:30:10] INFO    Job log file created
   [2026-02-28T16:32:45] SUCCESS Job completed successfully
   ```

2. **日志文件包含完整 CLI 输出**：
   ```bash
   cat .morty/logs/测试模块_创建测试文件结构_20260228_163010.log
   # 包含所有 Claude Code 的输出
   ```

3. **Plan 文件包含完成标记**：
   ```markdown
   **完成状态**: ✅ 已完成 (2026-02-28 16:32:45)
   ```

4. **无警告日志**：
   - 不再出现 "Job completion not marked in plan file" 警告

---

## 相关文件

### 核心修改
- `internal/executor/engine.go:402-415` - CLI 输出模式从 OutputStream 改为 OutputCapture
- `bin/prompts/doing.md:83-95` - 添加 step4.5 完成标记步骤
- `bin/prompts/doing.md:96-100` - 验证器中强调必须标记
- `bin/prompts/doing.md:327-347` - 示例中展示完成标记格式
- `bin/prompts/doing.md:373-380` - 重要提醒中添加完成标记说明

### 相关组件
- `internal/callcli/output.go` - OutputHandler 实现
- `internal/callcli/interface.go` - OutputMode 定义
- `internal/parser/plan/parser.go` - 完成状态解析逻辑

---

## 总结

本次更新解决了两个影响用户体验的关键问题：

1. **控制台输出质量** ✅
   - CLI 输出不再污染控制台
   - 监控质量大幅提升
   - 日志文件完整保留 CLI 输出供审查

2. **完成状态验证** ✅
   - 强化 prompt 要求 AI 标记完成
   - 多处提醒确保 AI 不会遗忘
   - 提供清晰的格式示例
   - morty 验证逻辑完善

这些改进使 Morty 的执行过程更加清晰、可控、可追溯。
