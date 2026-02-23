# Doing

在满足`执行意图`的约束下不断执行`循环`中的工作步骤,结合[当前Job上下文]对[任务列表]进行执行,直到满足`验证器`中的约束,才能结束循环,完成Job。

---

# 循环

loop:[验证器]

    step0: [加载上下文] 读取 `.morty/status.json` 获取当前执行状态,读取 `.morty/plan/[模块名].md` 获取当前 Job 的定义和 Tasks。

    step1: [理解Job] 理解当前 Job 的目标、前置条件、Tasks 列表和验证器要求。

    step2: [执行Task] 按顺序执行当前 Job 中未完成的 Tasks:
           - 检查每个 Task 的状态,跳过已完成的 Task
           - 执行未完成的 Task
           - 标记 Task 为完成状态
           - 记录执行过程中的问题和解决方案到 debug_log

    step3: [验证Job] 执行 Job 的验证器,检查所有验收标准是否满足:
           - 运行生成的测试
           - 检查结果是否符合预期
           - 如验证失败,记录 debug_log 并准备重试

    step4: [更新状态] 更新 `.morty/status.json`:
           - 标记已完成的 Tasks
           - 更新 Job 状态 (RUNNING/COMPLETED/FAILED)
           - 记录 loop_count
           - 添加 debug_log (如有问题)
           - 创建 Git 提交

    step5: [输出RALPH] 输出 RALPH_STATUS 块,包含本次循环的执行摘要

---

# 验证器

这是一个 Job 完成检查器

0. 如果当前 Job 的所有 Tasks 已完成且验证器通过,则检查通过,结束循环。
1. 如果当前 Job 存在未解决的 debug_log,则检查不通过,需要重试。
2. 如果验证器执行失败,则检查不通过,记录 debug_log 并准备重试。
3. 如果达到最大重试次数,则标记 Job 为 BLOCKED,结束循环。
4. 其他情况下,继续执行下一个 Task 或重试当前 Task。

---

# 执行意图

## Task 执行规范

1. **读取状态**: 首先读取 `.morty/status.json` 了解当前 Job 的执行进度

2. **跳过已完成**: 检查每个 Task 的完成状态,已完成的 Task 直接跳过

3. **顺序执行**: 按顺序执行未完成的 Tasks,一次只执行一个 Task

4. **及时标记**: 每个 Task 完成后立即更新状态文件,标记为完成

5. **问题记录**: 遇到问题时记录详细的 debug_log:
   - 现象描述
   - 复现方法
   - 猜想原因（按置信度排序）
   - 修复方法

## 验证器执行

1. 根据 Job 定义中的 `验证器` 描述生成测试
2. 执行测试并收集结果
3. 如测试通过,标记 Job 为 COMPLETED
4. 如测试失败,记录 debug_log 并标记为 FAILED (准备重试)

## Git 集成

1. 每次 Job 完成后创建 Git 提交
2. 提交信息包含 Job 名称和状态
3. 保留完整的变更历史

---

# RALPH_STATUS 格式

每个循环结束时必须输出:

```markdown
<!-- RALPH_STATUS -->
{
  "module": "[模块名]",
  "job": "[Job名]",
  "status": "[RUNNING/COMPLETED/FAILED]",
  "tasks_completed": [N],
  "tasks_total": [M],
  "loop_count": [N],
  "debug_issues": [N],
  "summary": "[执行摘要]"
}
<!-- END_RALPH_STATUS -->
```

---

# 调试日志格式

当遇到问题需要记录 debug_log 时,按以下格式记录到 status.json:

```json
{
  "id": 1,
  "timestamp": "ISO8601",
  "phenomenon": "[错误现象]",
  "reproduction": "[复现步骤]",
  "hypotheses": ["[猜想1]", "[猜想2]"],
  "verification_todo": ["[验证1]", "[验证2]"],
  "fix": "[修复方法]",
  "fix_progress": "[修复进展]",
  "resolved": false
}
```

---

# 当前 Job 上下文

**模块**: install
**Job**: job_6
**当前 Task**: #3
**Task 描述**: 编写 `INSTALL.md` 安装指南文档

## 任务列表

- [ ] 确保 `bootstrap.sh` 支持管道执行（`curl ... | bash`）\n- [ ] 测试一键安装命令：`curl -sSL https\n- [ ] 编写 `INSTALL.md` 安装指南文档\n- [ ] 在 `README.md` 中添加安装说明\n

## 验证器

- `curl -sSL <url> | bash` 应能成功安装\n- 离线环境下手动安装步骤应清晰可行\n- 安装文档应覆盖所有使用场景\n- 无\n

## 执行指令

请按照 Doing 模式的循环步骤执行：
1. 读取 .morty/status.json 了解当前状态
2. 执行当前 Task: 编写 `INSTALL.md` 安装指南文档
3. 如有问题，记录 debug_log
4. 更新状态文件
5. 输出 RALPH_STATUS

开始执行!
