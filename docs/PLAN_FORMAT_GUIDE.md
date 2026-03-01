# Plan 文件格式指南

## 概述

本指南说明如何创建符合 Morty 规范的 Plan 文件，以及如何使用格式检查工具。

---

## 快速开始

### 1. 创建 Plan 文件

```bash
# 启动 plan 模式
morty plan

# AI 会引导你完成以下步骤：
# - 汇总 research 内容
# - 询问需求描述
# - 设计架构
# - 生成 plan 文件
# - 自动运行格式检查
```

### 2. 验证格式

```bash
# 检查所有 plan 文件
morty plan validate

# 检查单个文件
morty plan validate user_auth.md

# 显示详细错误信息
morty plan validate --verbose

# 自动修复（未来功能）
morty plan validate --fix
```

### 3. 查看示例

参考 `docs/examples/plan_format_example.md` 查看完整示例。

---

## 格式规范摘要

### 文件命名

- ✅ `user_auth.md` - 小写字母+下划线
- ✅ `data_processor_v2.md` - 可包含数字
- ✅ `e2e_test.md` - 特殊文件（端到端测试模块，必需）
- ❌ `UserAuth.md` - 禁止大写
- ❌ `user-auth.md` - 禁止连字符
- ❌ `用户认证.md` - 禁止中文

### 模块概述

```markdown
## 模块概述

**模块职责**: [一句话，不超过100字]

**对应 Research**: [列表]
- `.morty/research/file.md` - [描述]

**现有实现参考**: 无

**依赖模块**: 无

**被依赖模块**: 无
```

**依赖模块格式**:
- 无依赖: `**依赖模块**: 无`
- 单个依赖: `**依赖模块**: module_name`
- 多个依赖: `**依赖模块**: module1, module2, module3`
- 依赖所有: `**依赖模块**: __ALL__`

### Job 格式

```markdown
---

### Job 1: [Job名称]

#### 目标

[一句话描述]

#### 前置条件

- job_1 - [描述]
- module_name:job_2 - [跨模块依赖]

#### Tasks

- [ ] Task 1: [描述]
- [ ] Task 2: [描述]
- [x] Task 3: [已完成]

#### 验证器

- [验证标准1]
- [验证标准2]

#### 调试日志

无

#### 完成状态

⏳ 待开始

---
```

**关键要点**:
- Job 编号必须从 1 开始连续
- Task 必须包含 `Task N:` 前缀
- 前置条件使用 `job_N` 或 `module:job_N` 格式
- 完成状态必须使用标准标记：✅ 🚧 ⏸️ ❌ ⏳

---

## 常见错误及修复

### E001: 文件名不符合规范

**错误**: `UserAuth.md`
**修复**: 重命名为 `user_auth.md`

### E003: 缺少必需的 e2e_test.md 文件

**错误**: plan 目录中没有 `e2e_test.md`
**修复**: 创建 `e2e_test.md` 文件，作为端到端测试模块，依赖所有其他模块 (`__ALL__`)

### E004: Job 编号不连续

**错误**:
```markdown
### Job 1: 功能A
### Job 3: 功能B
```

**修复**:
```markdown
### Job 1: 功能A
### Job 2: 功能B
```

### E005: 依赖模块格式错误

**错误**: `**依赖模块**: UserAuth`
**修复**: `**依赖模块**: user_auth`

**错误**: `**依赖模块**:`
**修复**: `**依赖模块**: 无`

### E006: Task 格式错误

**错误**: `- [ ] 创建数据库表`
**修复**: `- [ ] Task 1: 创建数据库表`

**错误**: `- [X] Task 1: 完成`
**修复**: `- [x] Task 1: 完成` (小写 x)

### E007: 前置条件格式错误

**错误**: `- Job 1 完成`
**修复**: `- job_1 - 完成`

**错误**: `- UserAuth:job_1`
**修复**: `- user_auth:job_1`

### E008: 完成状态标记无效

**错误**: `#### 完成状态\n\n已完成`
**修复**: `#### 完成状态\n\n✅ 已完成`

**允许的标记**:
- ✅ 已完成
- 🚧 进行中
- ⏸️ 暂停
- ❌ 失败
- ⏳ 待开始

---

## 格式检查输出示例

### 成功

```
✅ Plan 格式验证通过

检查的文件: 3
  - user_auth.md: ✅ 通过
  - data_processor.md: ✅ 通过
  - [生产测试].md: ✅ 通过
```

### 失败

```
❌ Plan 格式验证失败

user_auth.md:
  ❌ E004: Job 编号不连续 (第 45 行)
     发现: Job 3
     期望: Job 2
  ❌ E006: Task 格式错误 (第 52 行)
     发现: - [ ] 创建测试文件
     期望: - [ ] Task N: 创建测试文件

总计: 2 个错误
```

---

## AI 自动修复流程

当 AI 在 plan 模式中生成文件后，会自动执行以下流程：

1. **运行验证**: `morty plan validate --verbose`
2. **读取错误**: 解析错误代码和位置
3. **应用修复**: 根据错误类型修复问题
4. **重新验证**: 再次运行验证
5. **循环直到通过**: 重复步骤 2-4

AI 会处理的常见修复：
- 文件重命名（E001）
- Job 重新编号（E004）
- 依赖模块格式化（E005）
- Task 添加编号（E006）
- 前置条件规范化（E007）
- 添加完成状态标记（E008）

---

## 最佳实践

### 1. 模块设计

- **单一职责**: 每个模块只负责一个明确的功能域
- **清晰接口**: 定义明确的输入输出接口
- **合理依赖**: 避免循环依赖，形成清晰的依赖层次

### 2. Job 设计

- **可验证**: 每个 Job 必须有明确的验证标准
- **独立性**: Job 之间尽量减少依赖
- **原子性**: Job 要么完全成功，要么完全失败

### 3. 验证器设计

- **自然语言**: 使用人类可读的描述
- **可测试**: 描述可以转化为测试代码
- **完整性**: 覆盖正常流程、边界情况和错误处理
- **可量化**: 包含可量化的指标（时间、内存、准确率等）

### 4. 依赖管理

- **显式声明**: 在模块概述和 Job 前置条件中明确声明依赖
- **避免循环**: 使用拓扑排序验证无循环依赖
- **最小化依赖**: 只依赖真正需要的模块/Job

---

## 工具使用

### 命令行选项

```bash
# 基本用法
morty plan validate

# 详细输出
morty plan validate --verbose
morty plan validate -v

# 单文件检查
morty plan validate user_auth.md

# 自动修复（未来）
morty plan validate --fix
morty plan validate -f
```

### 集成到 CI/CD

```yaml
# .github/workflows/validate-plan.yml
name: Validate Plan Files

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install Morty
        run: |
          curl -L https://github.com/morty/morty/releases/latest/download/morty-linux-amd64 -o morty
          chmod +x morty
          sudo mv morty /usr/local/bin/
      - name: Validate Plan Files
        run: morty plan validate --verbose
```

---

## 参考资料

- [Plan 文件格式规范 v2.0](plan_format_spec.md) - 完整的技术规范
- [Plan 文件示例](examples/plan_format_example.md) - 完整的示例文件
- [Plan 提示词](../bin/prompts/plan.md) - AI 使用的提示词模板

---

## 故障排除

### 问题: 验证总是失败

**检查**:
1. 文件是否在 `.morty/plan/` 目录下
2. 文件编码是否为 UTF-8
3. 是否包含所有必需的 sections

### 问题: AI 生成的文件格式不对

**解决**:
1. 确保使用最新版本的 morty
2. 检查 `bin/prompts/plan.md` 是否是最新版本
3. 手动运行 `morty plan validate` 并修复错误
4. 向 AI 反馈错误信息，让它修复

### 问题: 无法自动修复

**解决**:
1. 查看详细错误信息: `morty plan validate --verbose`
2. 参考本指南的"常见错误及修复"部分
3. 手动修复后重新验证
4. 如果是复杂问题，考虑重新生成 plan 文件

---

## 版本历史

- **v2.0** (2026-02-28): 引入严格格式规范和自动验证
- **v1.0** (2026-02-01): 初始版本，基本格式定义
