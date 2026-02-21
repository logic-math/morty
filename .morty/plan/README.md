# Plan 索引

**生成时间**: 2026-02-21T00:00:00Z

**对应 Research**:
- `.morty/research/morty-project-research.md` - Morty 项目重构调研报告

---

## 模块列表

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| config | config.md | 3 | 无 | 规划中 |
| logging | logging.md | 4 | config | 已实现 |
| version_manager | version_manager.md | 3 | config, logging | 规划中 |
| doing | doing.md | 5 | config, logging, version_manager | 规划中 |
| cli | cli.md | 5 | 所有模块 | 规划中 |
| 生产测试 | 生产测试.md | 6 | 所有功能模块 | 规划中 |

**注意**: plan_mode 和 research_mode 由用户手动实现，不包含在 doing 执行计划中。

---

## 依赖关系图

```text
                         ┌─────────────────┐
                         │      cli        │
                         │   (命令路由)     │
                         └────────┬────────┘
                                  │
        ┌─────────────────────────┼──────────────────────────┐
        │                         │                          │
        ▼                         ▼                          ▼
┌───────────────┐        ┌─────────────────┐        ┌─────────────────┐
│     doing     │        │  version_manager │        │     stat       │
│   (执行模式)   │◄───────│   (版本管理)     │        │  (监控大盘)     │
└───────┬───────┘        └────────┬────────┘        └─────────────────┘
        │                         │
        │                         ▼
        │                ┌─────────────────┐
        └───────────────►│    logging      │
                         │   (日志系统)     │
                         └────────┬────────┘
                                  │
                                  ▼
                         ┌─────────────────┐
                         │     config      │
                         │   (基础配置)     │
                         └─────────────────┘
```

---

## 执行策略

### 断点自动恢复
`morty doing` 默认从上次中断处自动恢复执行：
- 从未完成的 Job 开始继续
- 状态通过 `.morty/status.json` 维护

### 拓扑排序执行
- **模块级**: 按依赖关系拓扑排序（被依赖模块先执行）
- **Job 级**: 按前置条件拓扑排序（0 依赖 Job 优先）

### 状态管理
所有执行状态通过 `.morty/status.json` 集中维护：

```json
{
  "state": "running|completed|error",
  "current": { "module": "config", "job": "job_2" },
  "modules": {
    "config": {
      "status": "completed",
      "jobs": {
        "job_1": {
          "status": "COMPLETED",
          "loop_count": 1,
          "tasks_total": 5,
          "tasks_completed": 5
        }
      }
    }
  },
  "summary": { "total_jobs": 25, "completed_jobs": 5, "progress_percentage": 20 }
}
```

通过 `morty stat` 命令查看实时进度：
```bash
morty stat              # 显示文本格式状态
morty stat --json       # 显示 JSON 格式
morty stat --watch      # 持续监控
```

---

## 执行顺序

基于依赖关系，模块应按以下顺序执行：

### 第一阶段：基础模块
1. **config** (无依赖)
2. **logging** (依赖 config) - 已实现

### 第二阶段：核心服务模块
3. **version_manager** (依赖 config, logging)

### 第三阶段：执行模块
4. **doing** (依赖 config, logging, version_manager)

### 第四阶段：入口整合
5. **cli** (依赖所有模块)

### 第五阶段：验证
6. **生产测试** (所有模块完成后)

---

## Jobs 统计

| 模块 | Jobs | 关键验证点 |
|------|------|------------|
| config | 3 | 配置优先级、验证 |
| logging | 4 | 日志级别、轮转、Job 级日志 - 已实现 |
| version_manager | 3 | 提交格式、回滚 |
| doing | 5 | 状态机、黑箱执行、重试、Git 提交 |
| cli | 5 | 路由、stat、reset、帮助 |

**总计**: 25 个 Jobs (logging 已实现，剩余 21 个)

---

## 执行流程示例

```
第1次执行 morty doing:
  → config/job1 (新, 0依赖) → COMPLETED → Git 提交
  → config/job2 (新, 依赖job1) → COMPLETED → Git 提交
  → config/job3 (新, 依赖job2) → COMPLETED → Git 提交
  → logging/job1 (已完成) → 跳过
  → ...

第2次执行 morty doing (自动恢复):
  → config/* (已完成) → 跳过
  → version_manager/job1 (新) → 执行 → COMPLETED
  → ...
```

---

## 关键设计决策

### 1. 分层架构
```
基础层: config, logging
服务层: version_manager
执行层: doing
入口层: cli
```

### 2. 黑箱执行
- doing 模式调用 ai_cli 以黑箱方式执行 Job
- 所有输出通过 logging 模块记录到日志
- 人类通过日志观察执行细节
- 所有修改在 plan 目录闭环

### 3. 状态管理
- 所有状态集中存储在 `.morty/status.json`
- 每个 Job 执行一次算一次循环，记录 loop_count
- 通过 `morty stat` 查看实时进度（监控大盘）

### 4. 提示词收敛
- 所有提示词放在 `prompts/` 目录
- doing 脚本不内置任何提示词
- 运行时动态组合提示词 + Plan 文件

### 5. 模块精简
- plan_mode 和 research_mode 由用户手动实现
- version_manager 替代 git_manager（便于扩展隔离性）
- 简化 doing 为单循环调度器

---

## 文件清单

### Plan 文件
- `plan/README.md` - Plan 索引（本文件）
- `plan/config.md` - 配置管理模块
- `plan/logging.md` - 日志系统模块（已实现）
- `plan/version_manager.md` - 版本管理模块
- `plan/doing.md` - 执行模式模块
- `plan/cli.md` - 命令行接口模块
- `plan/生产测试.md` - 端到端测试计划

### 提示词文件（prompts/）
- `prompts/doing.md` - Doing 模式系统提示词

---

## 下一步

运行 `morty doing` 开始分层 TDD 开发

执行顺序:
1. config → version_manager (基础层)
2. doing (核心执行)
3. cli (入口整合)
4. 生产测试 (验证发布)

**注意**: logging 模块已实现，可直接跳过。
