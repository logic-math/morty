# Morty 重构总结 - Plan 模式 → Fix 模式

**日期**: 2026-02-14  
**提交**: 9753eed

## 重构概述

将 Morty 的 `plan` 模式重构为 `fix` 模式,实现迭代式 PRD 改进和模块化知识管理。

## 主要变更

### 1. 核心脚本
- ✅ **创建** `morty_fix.sh` - 新的 fix 模式主脚本
- ✅ **保留** `morty_plan.sh` - 旧脚本保留但不再使用
- ✅ **更新** `morty` - 主入口命令路由从 plan 改为 fix

### 2. 系统提示词
- ✅ **创建** `prompts/fix_mode_system.md` - 完整的中文提示词
  - 三种改进方向:问题诊断、功能迭代、架构优化
  - 模块化知识管理策略
  - 可选的项目结构生成
  - 对话框架和最佳实践

### 3. 命令变更
**之前:**
```bash
morty plan requirements.md [project-name]
```

**现在:**
```bash
morty fix prd.md
```

### 4. 工作流程变更

#### 之前 (Plan 模式)
1. 创建初始 PRD
2. `morty plan requirements.md`
3. 对话式改进
4. 自动生成项目结构
5. 进入项目目录
6. `morty start` 开始循环

#### 现在 (Fix 模式)
1. 已有 prd.md 文件
2. `morty fix prd.md`
3. 对话式改进(三个方向)
4. 生成改进版 prd.md
5. 更新 specs/*.md 模块规范
6. **询问**是否生成项目结构
7. `morty start` 开始循环(手动)

### 5. 知识组织变更

#### 之前
- 单个 `problem_description.md` 文件
- 所有内容在一个文档中

#### 现在
- `prd.md` - 产品需求文档(改进版)
- `specs/` 目录 - 模块化知识库
  - `specs/auth.md` - 认证模块
  - `specs/api.md` - API 模块
  - `specs/database.md` - 数据库模块
  - 等等...

### 6. 文档更新
- ✅ `README.md` - 更新为中文,反映 fix 模式
- ✅ `install.sh` - 更新帮助文本和命令路由
- ✅ 测试文件重命名:`test_plan_mode.sh` → `test_fix_mode.sh`

## 新特性

### 1. 迭代式改进
- 每次 fix 会话专注一个改进
- 建立在以前的知识基础上
- 模块规范是活文档

### 2. 三种改进方向
- **问题诊断与修复** - 根因分析,修复策略
- **功能迭代与增强** - 新功能,集成点
- **架构优化与重构** - 架构改进,重构策略

### 3. 模块化知识管理
- 每个功能模块独立文档
- 包含:目的、规范、实现、演进历史
- 便于维护和理解

### 4. 用户控制
- 项目结构生成变为可选
- 用户确认后才生成
- 更灵活的工作流程

## 技术细节

### 文件变更统计
```
6 files changed, 874 insertions(+), 139 deletions(-)
- create mode 100755 morty_fix.sh
- create mode 100644 prompts/fix_mode_system.md
- rename tests/{test_plan_mode.sh => test_fix_mode.sh} (85%)
```

### 系统提示词结构
- 1000+ 行中文提示词
- 完整的对话框架
- 详细的模块规范模板
- 最佳实践指南

### 向后兼容性
- ⚠️ **破坏性变更** - `morty plan` 命令不再可用
- ✅ 旧的 `morty_plan.sh` 保留在代码库中
- ✅ 开发循环(`morty start`)不受影响

## 测试更新

### 测试文件
- `test_fix_mode.sh` - 更新所有测试用例
- 验证 fix 模式组件存在
- 检查命令路由正确
- 验证系统提示词内容

### 运行测试
```bash
cd morty
./tests/run_all_tests.sh
```

## 使用示例

### 示例 1: 问题修复
```bash
# 1. 已有项目和 prd.md
cd my-project

# 2. 启动 fix 模式
morty fix prd.md

# 3. 对话中描述问题
# Claude: "你遇到了什么问题?"
# 用户: "登录功能有 bug..."

# 4. 生成改进的 prd.md 和 specs/auth.md

# 5. 开始开发
morty start
```

### 示例 2: 功能增强
```bash
# 1. 想要添加新功能
morty fix prd.md

# 2. 对话中描述新功能
# Claude: "想要添加什么功能?"
# 用户: "需要支持 OAuth 登录..."

# 3. 更新 prd.md 和 specs/auth.md

# 4. 可选:生成项目结构
# Claude: "是否需要生成项目结构?"
# 用户: "是"

# 5. 开始开发
morty start
```

### 示例 3: 架构重构
```bash
# 1. 想要改进架构
morty fix prd.md

# 2. 对话中讨论架构
# Claude: "当前架构有什么问题?"
# 用户: "想要从单体改为微服务..."

# 3. 更新 prd.md 和多个 specs/*.md

# 4. 开始重构
morty start
```

## 迁移指南

### 从 Plan 模式迁移

如果你之前使用 plan 模式:

1. **更新命令**
   ```bash
   # 之前
   morty plan requirements.md
   
   # 现在
   morty fix prd.md
   ```

2. **调整工作流程**
   - Fix 模式假设你已有 prd.md
   - 如果从零开始,先手动创建 prd.md
   - Fix 模式专注于迭代改进

3. **理解知识组织**
   - 不再是单个大文件
   - 使用 specs/ 目录组织模块
   - 每个模块独立演进

## 后续工作

### 可能的改进
- [ ] 添加 `morty init` 命令用于新项目
- [ ] 支持从 specs/ 合并生成完整文档
- [ ] 添加模块依赖关系可视化
- [ ] 支持模块版本管理

### 文档待完善
- [ ] 创建 `docs/FIX_MODE_GUIDE.md` 详细指南
- [ ] 更新 `docs/CONFIGURATION.md`
- [ ] 添加更多使用示例

## 总结

这次重构将 Morty 从"一次性项目生成"转变为"持续迭代改进"的工具:

- ✅ 更灵活的工作流程
- ✅ 模块化知识管理
- ✅ 三种明确的改进方向
- ✅ 用户控制项目结构生成
- ✅ 完整的中文支持

Fix 模式更适合真实的软件开发场景,支持持续改进和知识积累。
