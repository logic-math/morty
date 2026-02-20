# Plan 索引

**生成时间**: 2026-02-20T19:10:00Z

**对应 Research**:
- `.morty/research/morty-project-research.md` - Morty 项目重构调研报告
- `.morty/research/plan-mode-design.md` - Plan 模式详细设计文档

---

## 模块列表

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| config | config.md | 3 | 无 | 规划中 |
| logging | logging.md | 4 | config | 规划中 |
| git_manager | git_manager.md | 5 | config, logging | 规划中 |
| research_mode | research_mode.md | 5 | config, logging | 规划中 |
| plan_mode | plan_mode.md | 6 | config, logging, git_manager | 规划中 |
| doing | doing.md | 7 | config, logging, git_manager, plan_mode | 规划中 |
| cli | cli.md | 6 | 所有模块 | 规划中 |
| 生产测试 | 生产测试.md | 6 | 所有功能模块 | 规划中 |

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
│     doing     │        │    plan_mode    │        │  research_mode  │
│   (执行模式)   │◄───────│   (规划模式)     │        │   (研究模式)     │
└───────┬───────┘        └────────┬────────┘        └─────────────────┘
        │                         │
        │                         ▼
        │                ┌─────────────────┐
        └───────────────►│   git_manager   │
                         │   (Git 管理)     │
                         └────────┬────────┘
                                  │
                                  ▼
                         ┌─────────────────┐
                         │    logging      │
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
- 在 Job 内从未完成的 Task 继续
- 有未解决 debug_log 的 Job 自动重试

### 拓扑排序执行
- **模块级**: 按依赖关系拓扑排序（被依赖模块先执行）
- **Job 级**: 按前置条件拓扑排序（0 依赖 Job 优先）

### 状态管理
所有执行状态通过 `.morty/status.json` 集中维护：

```json
{
  "state": "running|completed|error|blocked|paused",
  "current": { "module": "config", "job": "job_2" },
  "modules": {
    "config": {
      "status": "completed",
      "jobs": {
        "job_1": {
          "status": "COMPLETED",
          "loop_count": 1,
          "tasks_completed": 5,
          "debug_logs": []
        }
      }
    }
  },
  "summary": { "total_jobs": 36, "completed_jobs": 5, "progress_percentage": 14 }
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
2. **logging** (依赖 config)

### 第二阶段：核心服务模块
3. **git_manager** (依赖 config, logging)

### 第三阶段：模式模块
4. **research_mode** (依赖 config, logging)
5. **plan_mode** (依赖 config, logging, git_manager)
6. **doing** (依赖 config, logging, git_manager, plan_mode)

### 第四阶段：入口整合
7. **cli** (依赖所有模块)

### 第五阶段：验证
8. **生产测试** (所有模块完成后)

---

## Jobs 统计

| 模块 | Jobs | 平均前置条件数 | 关键验证点 |
|------|------|----------------|------------|
| config | 3 | 0.3 | 配置优先级、验证 |
| logging | 4 | 0.5 | 日志级别、轮转、Job 级日志 |
| git_manager | 5 | 0.6 | 提交格式、回滚、里程碑 |
| research_mode | 5 | 0.4 | 报告生成、验证 |
| plan_mode | 6 | 0.7 | 模块识别、验证器生成 |
| doing | 7 | 0.9 | 状态机、测试生成、重试、拓扑调度、断点恢复、stat |
| cli | 6 | 0.5 | 路由、stat、帮助、断点恢复 |

**总计**: 36 个 Jobs

---

## 执行流程示例

```
第1次执行 morty doing:
  → config/job1 (新, 0依赖) → Task1-5 → COMPLETED
  → config/job2 (新, 依赖job1) → Task1-2 → Task3 FAILED
  → 状态保存，记录 debug_log

第2次执行 morty doing (自动恢复):
  → config/job1 (已完成) → 跳过
  → config/job2 (FAILED) → Task1-2 (已完成,跳过) → Task3 (重试) → Task4-5 → COMPLETED
  → config/job3 (新, 依赖job2) → 执行 → COMPLETED
  → logging/job1 (新, 0依赖, config已完成) → 执行...
```

---

## 关键设计决策

### 1. 分层架构
```
基础层: config, logging
服务层: git_manager
模式层: research_mode, plan_mode, doing
入口层: cli
```

### 2. 分层 TDD 验证
```
Layer 1: Job 级单元测试（每个 Job 执行前生成）
Layer 2: 模块级集成测试（模块 Jobs 完成后）
Layer 3: 端到端生产测试（所有模块完成后）
```

### 3. 状态管理
- 所有状态集中存储在 `.morty/status.json`
- 每个 Job 执行一次算一次循环，记录 loop_count
- Task 级完成状态记录，支持 Task 级断点恢复
- 调试日志记录在 Job 的 debug_logs 数组中
- 通过 `morty stat` 查看实时进度
- `morty doing --restart` 强制从头执行

### 4. 验证器设计
- 使用自然语言描述验收标准
- 由 doing 模式解析并生成测试
- 支持重试（最多 3 次）和跳过策略

### 5. 环境同构
- Bash 4.0+ 作为统一脚本语言
- 依赖版本声明（Git 2.0+, Bash 4.0+）
- 配置模板化

---

## 文件清单

### Plan 文件
- `plan/README.md` - Plan 索引（本文件）
- `plan/config.md` - 配置管理模块
- `plan/logging.md` - 日志系统模块
- `plan/git_manager.md` - Git 管理模块
- `plan/research_mode.md` - 研究模式模块
- `plan/plan_mode.md` - 规划模式模块
- `plan/doing.md` - 执行模式模块（含 status.json 设计）
- `plan/cli.md` - 命令行接口模块（含 stat 命令）
- `plan/生产测试.md` - 端到端测试计划

---

## 下一步

运行 `morty doing` 开始分层 TDD 开发

执行顺序:
1. config → logging → git_manager (基础层)
2. research_mode → plan_mode (模式层基础)
3. doing (核心执行)
4. cli (入口整合)
5. 生产测试 (验证发布)
