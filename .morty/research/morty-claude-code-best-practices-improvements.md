# Morty 基于 Claude Code 最佳实践的改进方案

## 研究概述

**研究日期**: 2026-02-21
**研究主题**: 基于 Claude Code 官方最佳实践优化 Morty 框架
**参考文档**: `/home/sankuai/dolphinfs_sunquan20/ai_coding/Coding/morty/.morty/research/best-practices.md`

---

## 一、当前差距分析

### 1.1 上下文窗口管理

| 最佳实践要求 | Morty 现状 | 差距等级 |
|-------------|-----------|---------|
| 保持 context window 精简 | 传递完整 status.json + Plan | 🔴 高 |
| 自动压缩历史 | 无压缩机制 | 🔴 高 |
| 分离研究与执行 | Research 直接执行 | 🟡 中 |

**问题**: 随着项目规模增大，每个 Job 的提示词会越来越长，导致 Claude "遗忘"早期指令。

### 1.2 验证机制

| 最佳实践要求 | Morty 现状 | 差距等级 |
|-------------|-----------|---------|
| 给 Claude 验证方式 | 验证器定义在 Plan 中，但非强制 | 🔴 高 |
| 测试驱动 | 无强制测试要求 | 🟡 中 |
| 验证结果反馈 | 无结构化验证结果 | 🟡 中 |

**问题**: Agent 可能标记 Job 完成，但未经充分验证。

### 1.3 提示词管理

| 最佳实践要求 | Morty 现状 | 差距等级 |
|-------------|-----------|---------|
| CLAUDE.md 简洁 | prompts/doing.md 可能过长 | 🟡 中 |
| 分离持久规则与动态指令 | 混合在一起 | 🟡 中 |
| 按需加载 Skills | 无 Skills 系统 | 🟢 低 |

### 1.4 架构模式

| 最佳实践要求 | Morty 现状 | 差距等级 |
|-------------|-----------|---------|
| 使用 Subagents | 无 Subagents | 🔴 高 |
| 使用 Hooks | 无 Hooks | 🟡 中 |
| 结构化输出 (--output-format json) | 文本解析 | 🔴 高 |
| 检查点 / Rewind | Git 提交，但无快速回滚 | 🟡 中 |

---

## 二、改进方案

### 优先级 1: 立即实施 (高价值，低成本)

#### 2.1.1 精简上下文传递

**当前问题**:
```json
// 当前 status.json 传递全部历史
{
  "modules": {
    "logging": {
      "jobs": {
        "job_1": { "status": "COMPLETED", "tasks_completed": 5, ... },
        "job_2": { "status": "COMPLETED", "tasks_completed": 5, ... },
        // ... 所有历史 jobs
      }
    }
  }
}
```

**改进方案**:
```json
// 精简版上下文 - 只包含必要信息
{
  "current": {
    "module": "logging",
    "job": "job_3",
    "status": "RUNNING"
  },
  "context": {
    "completed_jobs_summary": [
      "logging/job_1: 实现日志核心 (5 tasks)",
      "logging/job_2: 日志轮转 (5 tasks)"
    ],
    "current_job": {
      "name": "job_3",
      "description": "实现结构化 JSON 日志",
      "tasks": ["Task 1", "Task 2", "Task 3"],
      "dependencies": ["logging/job_2"]
    }
  }
}
```

**实施位置**: `morty_doing.sh:build_prompt()`

**工作量**: 2-3 小时

---

#### 2.1.2 强制验证机制

**当前问题**:
```markdown
### Job 1: 实现功能
**验证器**:
1. 检查日志文件是否正确创建
2. 验证日志级别过滤功能
# ← 这些是描述，不是强制执行的命令
```

**改进方案**:
```markdown
### Job 1: 实现功能
**必须执行的验证命令**:
```bash
# 以下命令必须全部返回 0，Job 才算完成
run_tests() {
  npm test -- --grep "logging"
  npm run lint
  npm run typecheck
}
```

**验证通过标准**:
- [ ] `run_tests` 返回 0
- [ ] 代码覆盖率 >= 80%
- [ ] 无 TypeScript 错误
```

**实施方式**:
1. 在 Plan 文件格式中增加 `required_verification` 字段
2. `morty_doing.sh` 执行后自动运行验证命令
3. 验证失败标记 Job 为 FAILED

**工作量**: 4-6 小时

---

#### 2.1.3 结构化输出替代文本解析

**当前问题**:
```bash
# morty_doing.sh 当前做法
cat "$PROMPT_FILE" | claude -p --verbose --debug 2>&1 | tee "$OUTPUT_LOG"

# 解析状态 - 容易出错
if grep -q "EXIT_SIGNAL: true" "$OUTPUT_LOG"; then
    # 完成
fi
```

**改进方案**:
```bash
# 定义 JSON Schema
read -r -d '' SCHEMA << 'EOF'
{
  "type": "object",
  "properties": {
    "job_status": {
      "type": "string",
      "enum": ["COMPLETED", "FAILED", "PARTIAL", "NEEDS_RETRY"]
    },
    "tasks_completed": { "type": "integer" },
    "tasks_total": { "type": "integer" },
    "summary": { "type": "string" },
    "next_actions": { "type": "array", "items": { "type": "string" } },
    "git_commit_message": { "type": "string" }
  },
  "required": ["job_status", "tasks_completed", "summary"]
}
EOF

# 执行并获取结构化输出
claude -p \
  --output-format json \
  --json-schema "$SCHEMA" \
  < "$PROMPT_FILE" > "$OUTPUT_JSON"

# 解析 JSON - 更可靠
job_status=$(jq -r '.job_status' "$OUTPUT_JSON")
tasks_completed=$(jq -r '.tasks_completed' "$OUTPUT_JSON")
```

**工作量**: 6-8 小时

---

### 优先级 2: 中期实施 (需要开发)

#### 2.2.1 创建 Subagents

**用途**: 隔离 Research 探索，保护主上下文

**创建 `.claude/agents/morty-explorer.md`**:
```markdown
---
name: morty-explorer
description: Explore codebase structure for morty research
tools: Read, Grep, Glob, Bash
model: haiku  # 轻量级模型节省成本
---

你是一个代码库探索专家。任务是探索项目结构并生成结构化报告。

## 工作步骤
1. 使用 `Glob` 获取项目顶层文件结构
2. 识别核心配置文件 (package.json, Cargo.toml, go.mod 等)
3. 分析源代码目录结构
4. 总结项目架构模式

## 输出格式 (必须遵循)
```json
{
  "project_type": "Node.js|Python|Go|Rust|其他",
  "top_level_dirs": ["src", "lib", "tests", ...],
  "key_files": ["package.json", "README.md", ...],
  "architecture_pattern": "monolithic|microservices|layered|其他",
  "tech_stack": ["React", "TypeScript", "Express", ...],
  "summary": "项目一句话描述",
  "recommendations": ["改进建议1", "改进建议2"]
}
```

## 约束
- 最多读取 20 个文件，避免浪费上下文
- 不要修改任何文件
- 保持报告简洁，不超过 500 字
```

**在 Research 中使用**:
```bash
# 替代直接执行
claude -p "使用 morty-explorer subagent 探索当前代码库结构"
```

**工作量**: 1-2 天

---

#### 2.2.2 实现检查点机制

**用途**: 支持快速回滚到任意执行点

**设计**:
```
.morty/
├── checkpoints/
│   ├── checkpoint_001_logging_job1/  # Git stash + 状态快照
│   ├── checkpoint_002_logging_job2/
│   └── checkpoint_003_config_job1/
└── status.json
```

**实施**:
```bash
# lib/checkpoint.sh

create_checkpoint() {
    local checkpoint_name="checkpoint_$(date +%Y%m%d_%H%M%S)_${CURRENT_MODULE}_${CURRENT_JOB}"
    local checkpoint_dir="$MORTY_DIR/checkpoints/$checkpoint_name"

    mkdir -p "$checkpoint_dir"

    # 保存代码状态
    git stash push -m "morty-checkpoint: $checkpoint_name"
    echo "$checkpoint_name" > "$checkpoint_dir/git_stash_ref"

    # 保存状态文件
    cp "$MORTY_DIR/status.json" "$checkpoint_dir/status.json.backup"

    # 保存元数据
    cat > "$checkpoint_dir/meta.json" << EOF
{
  "checkpoint_name": "$checkpoint_name",
  "module": "$CURRENT_MODULE",
  "job": "$CURRENT_JOB",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "git_commit": "$(git rev-parse HEAD)"
}
EOF

    log_info "检查点已创建: $checkpoint_name"
}

restore_checkpoint() {
    local checkpoint_name=$1
    local checkpoint_dir="$MORTY_DIR/checkpoints/$checkpoint_name"

    # 恢复代码
    git stash pop $(cat "$checkpoint_dir/git_stash_ref")

    # 恢复状态
    cp "$checkpoint_dir/status.json.backup" "$MORTY_DIR/status.json"

    log_info "已回滚到检查点: $checkpoint_name"
}

list_checkpoints() {
    ls -1t "$MORTY_DIR/checkpoints/" | head -20
}
```

**CLI 集成**:
```bash
morty checkpoint -c              # 创建检查点
morty checkpoint -l              # 列出检查点
morty checkpoint -r <name>       # 恢复到检查点
```

**工作量**: 2-3 天

---

#### 2.2.3 分离持久规则与动态指令

**当前问题**: `prompts/doing.md` 既包含持久规则，又包含动态指令

**改进方案**:

**1. 创建 `CLAUDE.md` (持久规则)**:
```markdown
# Morty 项目持久规则

## 必须遵循的规则
- 执行任何 Task 前，先读取 `.morty/status.json`
- 每个 Task 完成后，立即更新状态文件
- 使用结构化 JSON 格式输出关键事件
- 遇到错误时，先记录 debug_log 再退出

## 工作流规则
1. **状态检查**: 读取 status.json，确认当前 Job 和状态
2. **跳过已完成**: 检查 Task 是否已标记完成
3. **顺序执行**: 按顺序执行未完成的 Tasks
4. **及时标记**: 每个 Task 完成后立即更新状态

## Bash 命令快捷方式
- 测试: `npm test`
- 构建: `npm run build`
- 日志: `tail -f .morty/logs/morty.log`
- 状态: `cat .morty/status.json | jq '.current'`

## 代码风格
- 使用 Bash 严格模式: `set -euo pipefail`
- 函数使用 snake_case 命名
- 日志使用 `log_info`, `log_error` 等函数
```

**2. 精简 `prompts/doing.md` (动态指令模板)**:
```markdown
# Morty Doing 模式

{{CLAUDE_MD_CONTENT}}

---

# 当前执行上下文 (动态生成)

**当前模块**: {{MODULE_NAME}}
**当前 Job**: {{JOB_NAME}}
**Job 描述**: {{JOB_DESCRIPTION}}
**前置条件**: {{PREREQUISITES}}

## 待执行 Tasks
{{TASKS_LIST}}

## 验证要求
{{VERIFICATION_REQUIREMENTS}}

---

# 执行指令

1. 读取 `.morty/status.json` 确认当前状态
2. 按顺序执行以上 Tasks
3. 每个 Task 完成后立即更新状态
4. 执行验证命令
5. 输出结构化结果

## 输出格式

执行完成后，输出以下 JSON:
```json
{
  "job_status": "COMPLETED|FAILED|PARTIAL",
  "tasks_completed": number,
  "tasks_total": number,
  "summary": "执行摘要",
  "git_commit_message": "建议的提交信息"
}
```
```

**3. `morty_doing.sh` 动态填充**:
```bash
build_prompt() {
    local module=$1
    local job=$2

    # 读取持久规则
    local claude_md=$(cat "$MORTY_HOME/CLAUDE.md")

    # 读取 Plan
    local plan_content=$(cat ".morty/plan/${module}.md")

    # 读取当前 Job 信息
    local job_info=$(jq -r ".modules.${module}.jobs.${job}" ".morty/status.json")

    # 填充模板
    cat "$MORTY_HOME/prompts/doing.md" | \
        sed "s|{{CLAUDE_MD_CONTENT}}|$claude_md|g" | \
        sed "s|{{MODULE_NAME}}|$module|g" | \
        sed "s|{{JOB_NAME}}|$job|g" | \
        sed "s|{{JOB_DESCRIPTION}}|$(echo "$job_info" | jq -r '.description')|g"
}
```

**工作量**: 1-2 天

---

### 优先级 3: 长期优化 (架构调整)

#### 2.3.1 Skills 系统

**设计**: 将可复用工作流封装为 Skills

**目录结构**:
```
.claude/skills/
├── morty-research/
│   └── SKILL.md
├── morty-plan/
│   └── SKILL.md
├── morty-doing/
│   └── SKILL.md
└── morty-commit/
    └── SKILL.md
```

**示例: `morty-commit/SKILL.md`**:
```markdown
---
name: morty-commit
description: Create structured git commit for morty job
disable-model-invocation: true
---

根据 Job 执行结果创建规范提交。

## 输入
- 模块名: $MODULE
- Job 名: $JOB
- Job 状态: $STATUS

## 步骤
1. 读取 `.morty/status.json` 获取 Job 详情
2. 生成提交信息格式: `feat(module): 描述 (Job X)`
3. 执行 `git add` 和 `git commit`
4. 推送分支 (如果配置了远程)

## 示例提交信息
```
feat(logging): 实现日志轮转和压缩功能 (Job 3)

- 实现基于文件大小的日志轮转
- 添加 gzip 压缩支持
- 配置最大保留文件数
```
```

**使用方式**:
```bash
/morty-commit --module logging --job job_3
```

**工作量**: 3-5 天

---

#### 2.3.2 MCP 集成

**用途**: 连接外部工具 (项目管理、文档、监控)

**潜在集成**:
| MCP Server | 用途 | 优先级 |
|-----------|------|-------|
| GitHub | 自动创建 PR、关联 Issue | 高 |
| Notion | 文档同步 | 中 |
| Slack | 进度通知 | 中 |
| Figma | UI 设计参考 | 低 |

**示例: GitHub MCP**:
```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@anthropic-ai/mcp-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

**在 Morty 中使用**:
```markdown
### Job 5: 完成并提交

**执行步骤**:
1. 运行最终测试验证
2. 使用 GitHub MCP 创建 PR
   - 标题: `feat(logging): 完整日志系统实现`
   - 描述: 包含所有 Jobs 的摘要
3. 关联到相关 Issue
```

**工作量**: 2-3 天

---

#### 2.3.3 Hooks 系统

**用途**: 在关键节点自动执行脚本

**设计**:
```
.claude/hooks/
├── pre-job.sh      # Job 执行前
├── post-job.sh     # Job 执行后
├── pre-commit.sh   # Git 提交前
└── on-error.sh     # 发生错误时
```

**示例 Hooks**:

**`post-job.sh`**:
```bash
#!/bin/bash
# Job 完成后自动执行

# 1. 运行测试
if [ -f "package.json" ]; then
  npm test -- --bail || exit 1
fi

# 2. 更新状态文件时间戳
touch ".morty/last_job_completed"

# 3. 发送通知 (如果配置了)
if [ -n "$SLACK_WEBHOOK" ]; then
  curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"Job completed: '"$CLAUDE_MODULE/$CLAUDE_JOB"'"}' \
    "$SLACK_WEBHOOK"
fi
```

**`on-error.sh`**:
```bash
#!/bin/bash
# 发生错误时执行

# 1. 记录错误日志
echo "[$(date)] Error in $CLAUDE_MODULE/$CLAUDE_JOB: $CLAUDE_ERROR" >> ".morty/error.log"

# 2. 自动重试 (如果配置了)
if [ "$CLAUDE_RETRY_COUNT" -lt 3 ]; then
  echo "Auto-retrying..."
  morty doing --module "$CLAUDE_MODULE" --job "$CLAUDE_JOB"
fi
```

**工作量**: 2-3 天

---

## 三、实施路线图

### Phase 1: 基础优化 (2 周)

**目标**: 解决高优先级问题，提升可靠性

| 任务 | 负责人 | 预计时间 | 依赖 |
|-----|-------|---------|-----|
| 精简上下文传递 | TBD | 3 小时 | 无 |
| 强制验证机制 | TBD | 6 小时 | 无 |
| 结构化输出 | TBD | 1 天 | 无 |
| 分离 CLAUDE.md | TBD | 1 天 | 无 |
| 测试与验证 | TBD | 2 天 | 以上全部 |

### Phase 2: 架构增强 (3 周)

**目标**: 引入 Subagents 和检查点机制

| 任务 | 负责人 | 预计时间 | 依赖 |
|-----|-------|---------|-----|
| Subagents 系统 | TBD | 2 天 | Phase 1 |
| 检查点机制 | TBD | 3 天 | Phase 1 |
| Research 模式重构 | TBD | 2 天 | Subagents |
| Reset 模式增强 | TBD | 1 天 | 检查点 |
| 测试与验证 | TBD | 3 天 | 以上全部 |

### Phase 3: 高级功能 (4 周)

**目标**: 集成 Skills、MCP、Hooks

| 任务 | 负责人 | 预计时间 | 依赖 |
|-----|-------|---------|-----|
| Skills 框架 | TBD | 5 天 | Phase 2 |
| MCP 集成 | TBD | 3 天 | Skills |
| Hooks 系统 | TBD | 3 天 | Skills |
| 生产测试 | TBD | 5 天 | 以上全部 |
| 文档更新 | TBD | 2 天 | 以上全部 |

---

## 四、预期收益

### 4.1 性能提升

| 指标 | 当前 | 预期 | 提升 |
|-----|-----|-----|-----|
| 平均 Job 执行时间 | 5-10 min | 3-5 min | 40% ↓ |
| 上下文超限错误 | 偶尔 | 极少 | 80% ↓ |
| 误报完成率 | 10-15% | <5% | 60% ↓ |

### 4.2 可靠性提升

| 指标 | 当前 | 预期 | 提升 |
|-----|-----|-----|-----|
| 失败恢复时间 | 手动 | 自动 | 100% ↓ |
| 回滚粒度 | Commit 级 | Job 级 | 更精细 |
| 验证覆盖率 | 依赖自觉 | 100% | 强制 |

### 4.3 可维护性提升

- **提示词管理**: 从单一文件 → 分层结构
- **功能扩展**: 从无系统 → Skills/Hooks 框架
- **外部集成**: 从无 → MCP 生态

---

## 五、风险评估

### 5.1 技术风险

| 风险 | 概率 | 影响 | 缓解措施 |
|-----|-----|-----|---------|
| 结构化输出不稳定 | 中 | 高 | 保留文本解析作为 fallback |
| Subagents 增加复杂度 | 中 | 中 | 渐进式引入，非强制 |
| 向后兼容性破坏 | 低 | 高 | 版本控制，逐步迁移 |

### 5.2 实施风险

| 风险 | 概率 | 影响 | 缓解措施 |
|-----|-----|-----|---------|
| 开发时间超期 | 中 | 中 | 分阶段交付，MVP 优先 |
| 测试覆盖不足 | 中 | 高 | 每个 Phase 预留测试时间 |
| 文档滞后 | 高 | 中 | 强制要求文档同步更新 |

---

## 六、参考资源

### 6.1 Claude Code 官方文档

- [Best Practices](/zh-CN/best-practices)
- [How Claude Code Works](/zh-CN/how-claude-code-works)
- [Features Overview](/zh-CN/features-overview)
- [Subagents](/zh-CN/sub-agents)
- [Skills](/zh-CN/skills)
- [Hooks](/zh-CN/hooks-guide)
- [MCP](/zh-CN/mcp)

### 6.2 Morty 相关文件

- `prompts/doing.md` - 当前系统提示词
- `morty_doing.sh` - Doing 模式实现
- `.morty/status.json` - 状态文件格式
- `.morty/research/best-practices.md` - 完整最佳实践文档

---

## 七、附录

### 附录 A: 改进前后对比示例

#### A.1 提示词长度对比

**改进前**:
```
系统提示词: 500 tokens
Plan 文件: 2000 tokens
完整 status.json: 3000 tokens
总计: ~5500 tokens
```

**改进后**:
```
CLAUDE.md: 200 tokens
动态上下文: 800 tokens
总计: ~1000 tokens
节省: 80%
```

#### A.2 验证机制对比

**改进前**:
```markdown
**验证器**:
1. 检查日志文件是否正确创建
2. 验证日志级别过滤功能
# ← 描述性文字，不强制执行
```

**改进后**:
```markdown
**必须执行的验证命令**:
```bash
#!/bin/bash
set -e

# 命令 1: 测试
echo "Running tests..."
npm test -- --grep "logging"

# 命令 2: Lint
echo "Running linter..."
npm run lint

# 命令 3: 类型检查
echo "Running type check..."
npm run typecheck

echo "All verifications passed!"
```

**验证通过标准**: 以上脚本返回 0
```

#### A.3 状态解析对比

**改进前**:
```bash
# 文本解析 - 容易出错
if grep -q "EXIT_SIGNAL: true" "$OUTPUT_LOG"; then
    status="COMPLETED"
fi
```

**改进后**:
```bash
# JSON 解析 - 可靠
status=$(jq -r '.job_status' "$OUTPUT_JSON")
tasks_completed=$(jq -r '.tasks_completed' "$OUTPUT_JSON")
```

---

**文档版本**: 1.0
**最后更新**: 2026-02-21
**作者**: Claude Code Research Agent
