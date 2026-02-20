# Plan: plan_mode

## 模块概述

**模块职责**: 实现 Plan 模式，将 Research 的研究结果转化为可执行的开发计划，支持架构设计、模块划分、Job 规划和验证器定义。

**对应 Research**: Plan 模式详细设计文档；分层 TDD 开发范式

**依赖模块**: config, logging, git_manager

**被依赖模块**: doing

## 接口定义

### 输入接口
- `morty plan [topic]`: 启动 Plan 模式
- `.morty/research/*.md`: Research 模式输出的研究结果
- 用户交互输入：架构确认、模块调整、Job 设计确认

### 输出接口
- `.morty/plan/README.md`: Plan 索引文件
- `.morty/plan/[模块名].md`: 功能模块计划文件
- `.morty/plan/[生产测试].md`: 端到端测试计划
- `plan_validate()`: 验证 Plan 完整性

## 数据模型

### Plan 文件结构
```
.morty/plan/
├── README.md              # Plan 总览
├── [模块A].md             # 功能模块 A
├── [模块B].md             # 功能模块 B
└── [生产测试].md          # 端到端测试
```

### Job 结构
```yaml
job:
  name: "job_name"
  target: "一句话描述目标"
  prerequisites:
    - "依赖的 Job"
  tasks:
    - "Task 1: 具体任务"
    - "Task 2: 具体任务"
  validator:
    description: "自然语言描述的验收标准"
    criteria:
      - "当输入 X 时，应输出 Y"
      - "边界情况 Z 不应导致异常"
  rollback:
    max_retries: 3
    on_failure: "skip"  # skip/terminate/fix
  status: "pending"  # pending/running/completed/failed/blocked
```

### 模块依赖图
```yaml
modules:
  config:
    dependencies: []
    dependents: [logging, git_manager, research, plan_mode, doing, monitor]
  logging:
    dependencies: [config]
    dependents: [git_manager, research, plan_mode, doing, monitor]
  git_manager:
    dependencies: [config, logging]
    dependents: [plan_mode, doing]
  plan_mode:
    dependencies: [config, logging, git_manager]
    dependents: [doing]
```

## Jobs (Loop 块列表)

---

### Job 1: Plan 模式基础架构

**目标**: 建立 Plan 模式的核心框架，支持读取 Research 结果和生成 Plan 文件

**前置条件**: config, logging, git_manager 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_plan.sh` 脚本
- [ ] 实现 `plan_check_prerequisites()`: 检查 .morty/research/ 目录存在
- [ ] 实现 `plan_load_research()`: 读取所有 research 文件
- [ ] 实现 `plan_init_directory()`: 初始化 .morty/plan/ 目录
- [ ] 实现 `plan_validate_structure()`: 验证 Plan 目录结构

**验证器**:
- 当 `.morty/research/` 不存在时，应提示用户先运行 `morty research`
- 当 research 目录存在但为空时，应显示警告但允许继续
- `plan_load_research()` 应正确读取并合并所有 `.md` 文件内容
- 运行 `morty plan` 后，`.morty/plan/` 目录应被创建（如果不存在）
- 已有的 Plan 文件应被保留（增量更新而非覆盖）

**调试日志**:
- 无

---

### Job 2: 架构设计与模块划分

**目标**: 基于 Research 结果自动识别功能模块，生成模块依赖关系

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `plan_analyze_research()`: 分析 Research 内容提取关键信息
- [ ] 实现 `plan_identify_modules()`: 识别功能模块
- [ ] 实现 `plan_detect_dependencies()`: 检测模块间依赖关系
- [ ] 实现依赖图生成器（文本格式）
- [ ] 检测循环依赖并警告

**验证器**:
- 从 Research 文档中应能识别出 config, logging, git_manager 等模块
- 模块依赖关系应正确反映（如 plan_mode 依赖 git_manager）
- 检测到循环依赖时应输出警告信息和建议
- 生成的模块列表应覆盖 Research 中提到的所有功能点
- 每个模块应有明确的职责描述

**调试日志**:
- 无

---

### Job 3: Plan 文件生成器

**目标**: 根据模块划分自动生成 Plan 文件模板

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `plan_generate_module_file(module)`: 生成模块 Plan 文件
- [ ] 实现 `plan_generate_production_test_file()`: 生成生产测试文件
- [ ] 实现 `plan_generate_readme(modules)`: 生成 Plan 索引
- [ ] 实现模板引擎（支持变量替换）
- [ ] 实现文件写入（带备份）

**验证器**:
- 生成的 `[模块名].md` 文件应包含：模块概述、接口定义、数据模型、Jobs 列表、集成测试
- 生成的 `[生产测试].md` 应包含：部署架构、环境同构策略、Jobs、回滚策略
- 生成的 `README.md` 应包含：模块列表、依赖关系图、执行顺序
- 文件写入前应备份原有文件（如果存在）
- 模板变量（如 `[模块名]`）应被正确替换

**调试日志**:
- 无

---

### Job 4: 交互式确认流程

**目标**: 实现交互式确认，让用户审查和调整生成的 Plan

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `plan_show_summary()`: 显示 Plan 概要
- [ ] 实现 `plan_confirm_modules()`: 确认模块划分
- [ ] 实现 `plan_edit_interactive()`: 交互式编辑（可选）
- [ ] 实现 `plan_accept_and_save()`: 确认并保存
- [ ] 实现 `plan_discard_and_retry()`: 放弃并重试

**验证器**:
- 应显示清晰的 Plan 概要，包括模块数量和 Jobs 总数
- 用户应能选择接受、编辑或放弃当前 Plan
- 编辑功能应允许修改模块名称、依赖关系和 Job 列表
- 确认后 Plan 文件应被最终保存
- 放弃后应清理临时文件并允许重新开始

**调试日志**:
- 无

---

### Job 5: Job 设计与验证器生成

**目标**: 为每个模块设计 Jobs，生成验证器描述

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `plan_design_jobs(module)`: 基于模块职责设计 Jobs
- [ ] 实现 `plan_generate_validator(job)`: 生成验证器描述
- [ ] 实现 `plan_estimate_effort()`: 估算每个 Job 的工作量
- [ ] 实现 Job 前置条件分析
- [ ] 生成 Job 执行顺序

**验证器**:
- 每个 Job 应有明确的单一职责
- 每个 Job 应包含：目标、前置条件、Tasks、验证器、回滚策略
- 验证器应使用自然语言描述，清晰可测试
- Job 之间应形成有向无环图，无循环依赖
- 复杂 Job 应被拆分为多个小 Job

**调试日志**:
- 无

---

### Job 6: Plan 验证与完整性检查

**目标**: 验证生成的 Plan 完整性和可执行性

**前置条件**: Job 5 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `plan_validate()`: 验证 Plan 完整性
- [ ] 实现 `plan_check_jobs()`: 检查每个 Job 的必填字段
- [ ] 实现 `plan_check_dependencies()`: 验证依赖关系有效性
- [ ] 实现 `plan_check_validators()`: 检查验证器描述
- [ ] 生成验证报告

**验证器**:
- 所有模块文件应存在且格式正确
- 所有 Job 应有目标、Tasks 和验证器
- 引用的前置条件 Job 应实际存在
- 依赖关系不应形成循环
- `[生产测试].md` 文件应存在

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 从 Research 到完整 Plan 的完整流程可以正常运行
- 生成的 Plan 文件可以被 doing 模式正确解析
- 用户可以成功审查、编辑和确认 Plan
- 多次运行 `morty plan` 可以增量更新现有 Plan
- Plan 验证可以检测并报告问题

---

## 待实现方法签名

```bash
# morty_plan.sh

# 入口
plan_main(topic="")

# 前置检查
plan_check_prerequisites()
plan_load_research()
plan_init_directory()

# 分析
plan_analyze_research(content)
plan_identify_modules()
plan_detect_dependencies(modules)
plan_check_circular_dependencies(graph)

# 生成
plan_generate_module_file(module)
plan_generate_production_test_file()
plan_generate_readme(modules)
plan_generate_validator(job)
plan_design_jobs(module)

# 交互
plan_show_summary()
plan_confirm_modules()
plan_edit_interactive()
plan_accept_and_save()
plan_discard_and_retry()

# 验证
plan_validate()
plan_check_jobs()
plan_check_dependencies()
plan_check_validators()
plan_validate_structure()
```
