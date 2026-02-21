# Plan: plan

## 模块概述

**模块职责**: 将 Research 的事实性信息转化为可执行的开发计划，生成结构化 Plan 文件

**对应 Research**:
- `plan-mode-design.md` 完整设计文档

**依赖模块**: research

**被依赖模块**: doing

## 接口定义

### 输入接口
- `morty plan`: 启动 Plan 模式
- `claude --permission-mode plan -p "$(cat prompts/plan.md)"`: 以 Plan 模式调用 Claude，传递系统提示词

### 输出接口
- `.morty/plan/README.md`: Plan 总览索引
- `.morty/plan/[模块名].md`: 功能模块计划
- `.morty/plan/[生产测试].md`: 端到端测试计划

## 数据模型

```
.morty/plan/
├── README.md              # Plan 索引
├── [模块A].md             # 功能模块 A 计划
├── [模块B].md             # 功能模块 B 计划
└── [生产测试].md          # 端到端测试计划
```

## Jobs

---

### Job 1: Plan 模式基础框架

**目标**: 实现 `morty_plan.sh`，以 Plan 模式调用 Claude 执行规划

**前置条件**: research, config 模块完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_plan.sh` 脚本
- [ ] 读取 `prompts/plan.md` 作为系统提示词
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
- [ ] 创建 `.morty/plan/` 目录
- [ ] 验证输出目录是否生成内容：
  ```bash
  if [[ -d "$PLAN_DIR" ]]; then
      PLAN_FILES=$(find "$PLAN_DIR" -name "*.md" -type f 2>/dev/null || true)
      if [[ -n "$PLAN_FILES" ]]; then
          PLAN_COUNT=$(echo "$PLAN_FILES" | wc -l)
          log SUCCESS "Plan 文件已生成: $PLAN_COUNT 个"
          echo "$PLAN_FILES" | while read -r file; do
              log INFO "  - $(basename "$file")"
          done
          log INFO ""
          log INFO "下一步:"
          log INFO "  运行 'morty doing' 开始分层 TDD 开发"
      else
          log WARN "Plan 目录为空，可能没有生成文件"
      fi
  else
      log WARN "Plan 目录未创建"
  fi
  ```

**验证器**:
- `morty plan` 能够启动规划流程
- 脚本从 config 读取 `cli.command` 作为 ai_cli 命令
- 以 Plan 模式调用 ai_cli，传递系统提示词
- Plan 文件生成到 `.morty/plan/` 目录

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 完整的研究→Plan 流程能够生成所有必要的 Plan 文件
- 系统提示词 `prompts/plan.md` 被正确传递
- Plan 文件格式能够被 `morty doing` 正确解析
