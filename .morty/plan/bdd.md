# Plan: BDD 用户旅程测试

## 模块概述

**模块职责**: 实现基于真实用户场景的 BDD 测试，验证 Morty 完整工作流 (Research → Plan → Doing) 的正确性

**对应 Research**:
- `.morty/research/morty-bdd-testing-strategy.md` - BDD 测试理念和 Mock 策略
- `.morty/research/morty-go-project-analysis.md` - Morty 架构和执行流程

**现有实现参考**:
- `internal/executor/engine.go` - Job 执行引擎
- `internal/cmd/doing.go` - Doing 命令处理
- `scripts/build.sh` - 构建流程

**依赖模块**: 无

**被依赖模块**: 无（独立测试模块）

## 接口定义

### 输入接口
- **测试场景定义**: 用户希望完成的开发任务（如"实现加法器"）
- **Mock AI 响应**: 预定义的 research.md, plan.md, 代码文件内容
- **验证规则**: 期望的文件、Git 提交、命令输出

### 输出接口
- **测试报告**: 每个场景的通过/失败状态
- **执行日志**: 命令输入输出、文件变化、错误信息
- **产物验证**: 生成的代码文件、文档、Git 历史

## 数据模型

```go
// 测试场景
type BDDScenario struct {
    Name        string
    Description string
    Steps       []TestStep
    Assertions  []Assertion
}

// 测试步骤
type TestStep struct {
    Command     string   // 执行的 morty 命令
    Args        []string
    Input       string   // stdin 输入
    MockFiles   map[string]string // Mock 生成的文件
}

// 断言
type Assertion struct {
    Type     string // file_exists, file_contains, git_commit, command_output
    Target   string
    Expected string
}
```

## Jobs (Loop 块列表)

---

### Job 1: Mock AI CLI 实现

**目标**: 创建 Mock Claude CLI 脚本，返回预定义的响应内容

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/mock_claude.sh` 脚本
- [ ] Task 2: 实现 Research 场景的 Mock 响应（生成 research.md）
- [ ] Task 3: 实现 Plan 场景的 Mock 响应（生成 plan.md）
- [ ] Task 4: 实现 Doing 场景的 Mock 响应（生成 Python 代码）
- [ ] Task 5: 添加日志记录功能，记录所有调用

**验证器**:
```
当 Mock CLI 被调用时，应该：
1. 根据输入内容识别场景（research/plan/doing）
2. 返回对应的预定义内容
3. 记录调用日志到 /tmp/mock_claude.log
4. 退出码为 0 表示成功

Mock CLI 应该能够：
- 接收 stdin 输入
- 根据输入关键词匹配场景
- 输出格式化的 Markdown 或代码
- 模拟 0.5 秒延迟（可配置）
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

### Job 2: 测试辅助函数库

**目标**: 创建 Shell 测试辅助函数，提供断言、环境管理等功能

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/test_helpers.sh` 文件
- [ ] Task 2: 实现 `create_test_project()` - 创建临时测试项目
- [ ] Task 3: 实现 `cleanup_test_project()` - 清理测试环境
- [ ] Task 4: 实现 `assert_success()` - 断言命令成功
- [ ] Task 5: 实现 `assert_file_exists()` - 断言文件存在
- [ ] Task 6: 实现 `assert_file_contains()` - 断言文件包含内容
- [ ] Task 7: 实现 `assert_git_commit_exists()` - 断言 Git 提交存在
- [ ] Task 8: 实现 `print_test_summary()` - 打印测试总结

**验证器**:
```
辅助函数应该：
1. create_test_project 创建独立的临时目录，初始化 Git 仓库
2. cleanup_test_project 完全删除测试目录
3. assert_* 函数在断言失败时返回非 0 退出码
4. assert_* 函数输出彩色的 ✓ PASSED 或 ✗ FAILED
5. print_test_summary 显示总测试数、通过数、失败数

测试计数器应该正确累加：
- TESTS_TOTAL 包含所有执行的断言
- TESTS_PASSED 包含通过的断言
- TESTS_FAILED 包含失败的断言
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

### Job 3: 加法器场景测试脚本

**目标**: 实现"开发 Python 加法器"的完整用户旅程测试

**前置条件**:
- Job 1 完成（Mock CLI 可用）
- Job 2 完成（测试辅助函数可用）

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/scenarios/test_calculator.sh` 文件
- [ ] Task 2: 实现测试环境初始化（创建临时项目）
- [ ] Task 3: 实现 Step 1 - 执行 `morty research "implement calculator"`
- [ ] Task 4: 实现 Step 2 - 验证 `.morty/research/implement-calculator.md` 存在
- [ ] Task 5: 实现 Step 3 - 执行 `morty plan`
- [ ] Task 6: 实现 Step 4 - 验证 `.morty/plan/*.md` 存在且包含 Jobs
- [ ] Task 7: 实现 Step 5 - 执行 `morty doing` (使用 Mock CLI)
- [ ] Task 8: 实现 Step 6 - 验证 `calculator.py` 文件生成
- [ ] Task 9: 实现 Step 7 - 验证 Python 代码可执行且输出正确
- [ ] Task 10: 实现 Step 8 - 验证 Git 自动提交存在
- [ ] Task 11: 实现测试清理和结果报告

**验证器**:
```
完整的用户旅程应该验证：

1. Research 阶段：
   - morty research 命令执行成功（退出码 0）
   - .morty/research/implement-calculator.md 文件存在
   - 文件包含 "calculator" 或 "addition" 关键词
   - 文件格式为有效的 Markdown

2. Plan 阶段：
   - morty plan 命令执行成功
   - .morty/plan/ 目录存在
   - 至少有一个 .md 文件
   - Plan 文件包含 "Job" 关键词
   - Plan 文件包含 "Tasks" 关键词

3. Doing 阶段：
   - morty doing 命令执行成功
   - calculator.py 文件被创建
   - Python 代码包含 "def add" 函数定义
   - 执行 python calculator.py 输出 "Hello World"

4. Git 集成：
   - 至少有一个 Git 提交包含 "morty:" 前缀
   - Git 提交消息包含 module/job 信息
   - Git 历史可以正常查看

5. 状态管理：
   - .morty/status.json 文件存在
   - status.json 包含 Job 状态信息
   - morty stat 命令可以正常显示状态

所有步骤应该在 30 秒内完成（使用 Mock CLI）
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

### Job 4: Hello World 场景测试脚本

**目标**: 实现"输出 Hello World"的最简单用户旅程测试（作为冒烟测试）

**前置条件**:
- Job 1 完成（Mock CLI 可用）
- Job 2 完成（测试辅助函数可用）

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/scenarios/test_hello_world.sh` 文件
- [ ] Task 2: 实现最简化的测试流程（只测试核心功能）
- [ ] Task 3: 执行 morty research "hello world"
- [ ] Task 4: 执行 morty plan
- [ ] Task 5: 执行 morty doing
- [ ] Task 6: 验证 hello.py 文件生成且包含 print("Hello World")
- [ ] Task 7: 验证 Python 代码可执行
- [ ] Task 8: 验证 Git 提交存在

**验证器**:
```
Hello World 场景应该：
1. 在 10 秒内完成所有步骤
2. 生成的 hello.py 文件包含 print("Hello World")
3. 执行 python hello.py 输出 "Hello World"
4. Git 历史包含至少一个 morty 提交
5. 无任何错误输出

这是最基础的冒烟测试，如果失败说明核心功能有问题。
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

### Job 5: Mock 响应内容定义

**目标**: 定义所有 Mock 响应的具体内容（research.md, plan.md, 代码）

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/mock_responses.sh` 文件
- [ ] Task 2: 定义 Research 响应模板 - Calculator 场景
- [ ] Task 3: 定义 Plan 响应模板 - Calculator 场景（包含完整 Jobs）
- [ ] Task 4: 定义 Doing 响应模板 - Python 加法器代码
- [ ] Task 5: 定义 Research 响应模板 - Hello World 场景
- [ ] Task 6: 定义 Plan 响应模板 - Hello World 场景
- [ ] Task 7: 定义 Doing 响应模板 - Hello World Python 代码
- [ ] Task 8: 实现响应选择逻辑（根据输入关键词匹配）

**验证器**:
```
Mock 响应应该：

1. Research 响应格式：
   - 有效的 Markdown 格式
   - 包含 # 标题
   - 包含 ## 章节（Overview, Requirements, Implementation）
   - 长度 > 100 字符

2. Plan 响应格式：
   - 有效的 Markdown 格式
   - 包含 "# Plan:" 标题
   - 包含 "## Module:" 章节
   - 包含 "### Job N:" 定义（至少 1 个）
   - 每个 Job 包含 "**Tasks**:" 列表
   - 符合 morty plan 解析器的格式要求

3. Doing 响应（Python 代码）：
   - 有效的 Python 语法
   - Calculator: 包含 def add(a, b) 函数
   - Hello World: 包含 print("Hello World")
   - 可以被 Python 解释器执行

4. 响应匹配逻辑：
   - 输入包含 "calculator" → 返回 Calculator 响应
   - 输入包含 "hello" → 返回 Hello World 响应
   - 输入包含 "research" → 返回 Research 格式
   - 输入包含 "plan" → 返回 Plan 格式
   - 输入包含 "task" 或 "doing" → 返回代码
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

### Job 6: 测试运行器

**目标**: 创建统一的测试运行器，执行所有 BDD 场景并生成报告

**前置条件**:
- Job 1-5 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/run_all.sh` 文件
- [ ] Task 2: 实现 Morty 二进制检查（确保已构建）
- [ ] Task 3: 实现 Mock CLI 路径配置（设置环境变量）
- [ ] Task 4: 实现场景发现（自动查找 scenarios/*.sh）
- [ ] Task 5: 实现场景执行循环（串行执行所有场景）
- [ ] Task 6: 实现结果收集（记录每个场景的通过/失败）
- [ ] Task 7: 实现最终报告生成（汇总所有场景结果）
- [ ] Task 8: 实现退出码设置（任何失败则返回 1）

**验证器**:
```
测试运行器应该：

1. 环境检查：
   - 验证 bin/morty 二进制存在
   - 验证 tests/bdd/mock_claude.sh 存在且可执行
   - 验证 tests/bdd/test_helpers.sh 存在

2. 执行流程：
   - 自动发现 tests/bdd/scenarios/*.sh 所有场景
   - 按字母顺序执行场景
   - 为每个场景设置独立环境（CLAUDE_CODE_CLI 环境变量）
   - 捕获每个场景的退出码

3. 结果报告：
   - 显示每个场景的名称和结果（PASSED/FAILED）
   - 显示总场景数、通过数、失败数
   - 使用彩色输出（绿色 ✓ / 红色 ✗）
   - 失败场景显示错误摘要

4. 退出码：
   - 所有场景通过 → 退出码 0
   - 任何场景失败 → 退出码 1

5. 性能：
   - 所有场景执行时间 < 1 分钟（使用 Mock CLI）
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

### Job 7: 文档和使用指南

**目标**: 编写 BDD 测试的使用文档和故障排查指南

**前置条件**:
- Job 1-6 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `tests/bdd/README.md` 文件
- [ ] Task 2: 编写快速开始指南（如何运行测试）
- [ ] Task 3: 编写 Mock CLI 工作原理说明
- [ ] Task 4: 编写如何添加新场景的指南
- [ ] Task 5: 编写故障排查 FAQ
- [ ] Task 6: 编写 CI/CD 集成说明
- [ ] Task 7: 添加测试架构图和流程图

**验证器**:
```
文档应该包含：

1. 快速开始（Quick Start）：
   - 构建 Morty: ./scripts/build.sh
   - 运行测试: cd tests/bdd && ./run_all.sh
   - 运行单个场景: ./scenarios/test_hello_world.sh
   - 预期输出示例

2. Mock CLI 说明：
   - Mock CLI 的工作原理
   - 如何自定义响应
   - 环境变量配置（CLAUDE_CODE_CLI）
   - 日志位置和格式

3. 添加新场景：
   - 场景文件命名规范
   - 必需的 source 语句
   - 测试步骤模板
   - 断言使用示例

4. 故障排查：
   - "Morty binary not found" → 运行 build.sh
   - "Mock CLI not executable" → chmod +x
   - "Test timeout" → 检查 Mock 延迟配置
   - "Git commit not found" → 检查 Git 配置

5. CI/CD 集成：
   - GitHub Actions 配置示例
   - Docker 容器运行方式
   - 测试报告上传

文档应该清晰易懂，新用户 5 分钟内能运行测试。
```

**调试日志**:
- 如果验证失败，记录 debug 日志到此处

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
```
完整的 BDD 测试模块应该：

1. 功能完整性：
   - 可以独立运行测试（不依赖真实 Claude CLI）
   - 覆盖 Research → Plan → Doing 完整流程
   - 验证文件生成、Git 提交、命令输出
   - 支持多个测试场景（Calculator, Hello World）

2. 易用性：
   - 一条命令运行所有测试: ./run_all.sh
   - 清晰的测试输出（彩色、格式化）
   - 失败时提供详细错误信息
   - 文档完善，易于理解

3. 可维护性：
   - Mock 响应集中管理
   - 测试辅助函数复用
   - 新场景易于添加（复制模板即可）
   - 代码结构清晰

4. 性能：
   - 所有测试 < 1 分钟完成
   - 每个场景 < 30 秒
   - Mock CLI 延迟可配置

5. 真实性：
   - 在真实文件系统中执行
   - 使用真实的 morty 二进制
   - 验证真实的 Git 操作
   - 只 Mock AI CLI 调用

端到端验证流程：
1. 运行 ./scripts/build.sh 构建 Morty
2. 运行 ./tests/bdd/run_all.sh
3. 所有场景应该通过（绿色 ✓）
4. 验证生成的代码文件可执行
5. 验证 Git 历史包含 morty 提交
6. 清理后无残留文件
```

---

## 文件清单

完成后将生成以下文件：

```
tests/bdd/
├── README.md                          # 使用文档
├── mock_claude.sh                     # Mock AI CLI 脚本
├── mock_responses.sh                  # Mock 响应内容定义
├── test_helpers.sh                    # 测试辅助函数
├── run_all.sh                         # 测试运行器
└── scenarios/
    ├── test_hello_world.sh            # Hello World 场景
    └── test_calculator.sh             # Calculator 场景
```

---

## 依赖关系

```
Job 1 (Mock CLI) ─────┐
                      ├──→ Job 3 (Calculator 场景)
Job 2 (Test Helpers) ─┤
                      ├──→ Job 4 (Hello World 场景)
                      │
Job 5 (Mock Responses)┘
                      │
                      ├──→ Job 6 (测试运行器)
                      │
                      └──→ Job 7 (文档)
```

---

## 执行顺序

1. **并行执行**: Job 1, Job 2, Job 5（无依赖）
2. **并行执行**: Job 3, Job 4（依赖 Job 1, 2, 5）
3. **串行执行**: Job 6（依赖 Job 1-5）
4. **串行执行**: Job 7（依赖 Job 1-6）

---

## 成功标准

✅ 所有 7 个 Jobs 完成
✅ 运行 `./tests/bdd/run_all.sh` 所有场景通过
✅ Calculator 场景生成可执行的 Python 代码
✅ Hello World 场景在 10 秒内完成
✅ Git 提交历史包含 morty 前缀
✅ 文档完善，新用户可快速上手

---

**预计完成时间**: 1-2 天
**测试覆盖**: Research → Plan → Doing 完整流程
**Mock 策略**: 只 Mock AI CLI，其他都是真实执行
