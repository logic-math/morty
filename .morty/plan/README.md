# Plan 索引

**生成时间**: 2026-02-27T12:00:00Z

**对应 Research**:
- `.morty/research/morty-bdd-testing-strategy.md` - BDD 测试理念和实现策略
- `.morty/research/morty-go-project-analysis.md` - Morty 项目深度分析
- `.morty/research/morty-testing-strategy.md` - 生产级测试策略

**现有实现探索**: 否（独立测试模块，不需要探索现有实现）

---

## 模块列表

| 模块名称 | 文件 | Jobs 数量 | 依赖模块 | 状态 |
|----------|------|-----------|----------|------|
| BDD 用户旅程测试 | bdd.md | 7 | 无 | 规划中 |

---

## 依赖关系图

```text
[BDD 测试模块]
  ├── Job 1: Mock AI CLI 实现
  ├── Job 2: 测试辅助函数库
  ├── Job 5: Mock 响应内容定义
  │   ├──→ Job 3: Calculator 场景测试
  │   ├──→ Job 4: Hello World 场景测试
  │   └──→ Job 6: 测试运行器
  │           └──→ Job 7: 文档和使用指南
```

---

## 执行顺序

### 第 1 轮（并行）:
- Job 1: Mock AI CLI 实现
- Job 2: 测试辅助函数库
- Job 5: Mock 响应内容定义

### 第 2 轮（并行）:
- Job 3: Calculator 场景测试脚本
- Job 4: Hello World 场景测试脚本

### 第 3 轮（串行）:
- Job 6: 测试运行器

### 第 4 轮（串行）:
- Job 7: 文档和使用指南

---

## 统计信息

- **总模块数**: 1
- **总 Jobs 数**: 7
- **预计执行轮次**: 4 轮
- **探索子代理使用**: 否
- **预计完成时间**: 1-2 天

---

## 模块详情

### BDD 用户旅程测试模块

**目标**: 实现基于真实用户场景的端到端测试，验证 Morty 完整工作流

**核心功能**:
1. Mock Claude CLI - 返回预定义响应（research.md, plan.md, Python 代码）
2. 测试辅助函数 - 提供断言、环境管理、结果验证
3. 用户场景测试 - Calculator（加法器）和 Hello World 两个完整旅程
4. 自动化运行器 - 一键运行所有测试并生成报告

**测试覆盖**:
- ✅ Research 阶段：命令执行、文档生成、内容验证
- ✅ Plan 阶段：命令执行、Plan 解析、Jobs 定义
- ✅ Doing 阶段：Job 执行、代码生成、文件验证
- ✅ Git 集成：自动提交、提交消息、历史验证
- ✅ 状态管理：status.json 生成、状态查询

**验证方式**:
- 真实环境执行（临时 Git 仓库）
- 真实 morty 二进制调用
- 只 Mock AI CLI（隔离外部依赖）
- 验证文件、Git、命令输出

---

## 使用方式

### 构建 Morty
```bash
./scripts/build.sh
```

### 运行所有 BDD 测试
```bash
cd tests/bdd
./run_all.sh
```

### 运行单个场景
```bash
cd tests/bdd
./scenarios/test_hello_world.sh
```

---

## 成功标准

### 功能完整性
- ✅ 所有 7 个 Jobs 完成
- ✅ Mock CLI 正确返回预定义响应
- ✅ 测试辅助函数覆盖所有断言类型
- ✅ Calculator 和 Hello World 场景完整实现

### 测试有效性
- ✅ 验证 Research → Plan → Doing 完整流程
- ✅ 验证文件生成（.md, .py）
- ✅ 验证 Git 自动提交
- ✅ 验证生成的代码可执行

### 易用性
- ✅ 一条命令运行所有测试
- ✅ 清晰的彩色输出（✓/✗）
- ✅ 详细的错误信息
- ✅ 完善的文档和示例

### 性能
- ✅ 所有测试 < 1 分钟
- ✅ 单个场景 < 30 秒
- ✅ Hello World 场景 < 10 秒

---

## 下一步

完成 Plan 后，运行以下命令开始开发：

```bash
morty doing
```

系统将按照依赖关系自动执行 Jobs，每个 Job 完成后自动提交到 Git。

---

**Plan 版本**: 1.0
**创建日期**: 2026-02-27
**状态**: ✅ 已完成
