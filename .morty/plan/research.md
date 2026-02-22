# Plan: research

## 模块概述

**模块职责**: 交互式代码库/文档库研究，生成结构化的调研报告

**对应 Research**:
- `morty-project-research.md` 第 3.5 节 Research 模式分析

**依赖模块**: cli

**被依赖模块**: plan

## 接口定义

### 输入接口
- `morty research [topic]`: 启动研究模式
- `claude --permission-mode plan -p "$(cat prompts/research.md)"`: 以 Plan 模式调用 Claude，传递系统提示词

### 输出接口
- `.morty/research/[主题].md`: 结构化研究报告

## 数据模型

```
.morty/research/
└── [主题].md          # 研究报告
```

## Jobs

---

### Job 1: Research 模式基础框架

**目标**: 实现 `morty_research.sh`，以 Plan 模式调用 Claude 执行研究

**前置条件**: cli, config 模块完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_research.sh` 脚本
- [ ] 读取 `prompts/research.md` 作为系统提示词
- [ ] 从 config 获取 ai_cli 命令：`AI_CLI=$(config_get "cli.command" "claude")`
- [ ] 构建 Claude 命令参数：
  ```bash
  CLAUDE_ARGS=(
      "$AI_CLI"
      "--dangerously-skip-permissions"
      "--allowedTools" "Read" "Write" "Glob" "Grep" "WebSearch" "WebFetch" "Edit" "Task"
  )
  ```
- [ ] 以 Plan 模式调用：`$AI_CLI --permission-mode plan -p "$PROMPT"`
- [ ] 创建 `.morty/research/` 目录
- [ ] 验证输出目录是否生成内容：
  ```bash
  if [[ -d "$RESEARCH_DIR" ]]; then
      RESEARCH_FILES=$(find "$RESEARCH_DIR" -name "*.md" -type f 2>/dev/null || true)
      if [[ -n "$RESEARCH_FILES" ]]; then
          log SUCCESS "研究报告已生成"
      else
          log WARN "Research 目录为空"
      fi
  fi
  ```

**验证器**:
- `morty research` 能够启动研究流程
- 脚本从 config 读取 `cli.command` 作为 ai_cli 命令
- 以 Plan 模式调用 ai_cli，传递系统提示词
- 研究报告生成到 `.morty/research/[主题].md`

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 完整的研究流程能够生成符合格式的报告
- 系统提示词 `prompts/research.md` 被正确传递
