# Plan 格式验证器修复说明

## 修复日期
2026-03-01

## 问题描述

在之前的 plan 提示词重构中，存在以下不一致问题：

1. **文件名不一致**:
   - 提示词第 39 行: 要求 `e2e_test.md`
   - 提示词第 253 行: 写成 `ete_test.md` (typo)
   - 提示词第 427 行: 写成 `[生产测试].md`

2. **Validator 不匹配**:
   - Validator 只允许 `[生产测试].md` 作为特殊文件名
   - 没有检查 `e2e_test.md` 是否存在
   - 提示词验证器第 78 行明确要求检查 `e2e_test.md` 存在

3. **命名规范冲突**:
   - 提示词强调所有文件名必须使用小写+下划线 (`^[a-z0-9_]+\.md$`)
   - 但 `[生产测试].md` 包含方括号和中文，违反规范

## 修复方案

### 1. 统一文件名为 `e2e_test.md`

**理由**:
- 符合小写+下划线命名规范
- 清晰表达端到端测试（End-to-End Test）的含义
- 与其他模块文件名保持一致性
- 便于程序处理和解析

**修改内容**:
- 提示词中所有 `ete_test.md` 改为 `e2e_test.md`
- 提示词中所有 `[生产测试].md` 改为 `e2e_test.md`
- 模块标题从 "Plan: 生产测试" 改为 "Plan: e2e_test"
- README 表格中的模块名从 "生产测试" 改为 "E2E测试"

### 2. 更新 Validator 实现

#### 2.1 文件名验证 (plan_validator.go:131-145)

**修改前**:
```go
func (v *PlanValidator) validateFilename(fileName string) error {
	// Special case: [生产测试].md
	if fileName == "[生产测试].md" {
		return nil
	}
	// Regular files: lowercase, numbers, underscores only
	matched, _ := regexp.MatchString(`^[a-z0-9_]+\.md$`, fileName)
	if !matched {
		return fmt.Errorf("invalid filename format")
	}
	return nil
}
```

**修改后**:
```go
func (v *PlanValidator) validateFilename(fileName string) error {
	// Special cases: e2e_test.md and README.md
	if fileName == "e2e_test.md" || fileName == "README.md" {
		return nil
	}
	// Regular files: lowercase, numbers, underscores only
	matched, _ := regexp.MatchString(`^[a-z0-9_]+\.md$`, fileName)
	if !matched {
		return fmt.Errorf("invalid filename format")
	}
	return nil
}
```

**说明**:
- 移除 `[生产测试].md` 特殊处理
- 添加 `e2e_test.md` 作为特殊文件名（虽然它也符合正则，但明确标注为特殊文件）
- `README.md` 也是特殊文件，明确标注

#### 2.2 添加 e2e_test.md 存在性检查 (plan_validator.go:54-72)

**修改前**:
```go
func (v *PlanValidator) ValidateAll() ([]*ValidationResult, error) {
	// Find all .md files
	files, err := filepath.Glob(filepath.Join(v.planDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to list plan files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no plan files found in %s", v.planDir)
	}

	results := make([]*ValidationResult, 0, len(files))
	for _, file := range files {
		result := v.ValidateFile(file)
		results = append(results, result)
	}

	return results, nil
}
```

**修改后**:
```go
func (v *PlanValidator) ValidateAll() ([]*ValidationResult, error) {
	// Find all .md files
	files, err := filepath.Glob(filepath.Join(v.planDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to list plan files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no plan files found in %s", v.planDir)
	}

	// Check if e2e_test.md exists
	e2eTestExists := false
	for _, file := range files {
		if filepath.Base(file) == "e2e_test.md" {
			e2eTestExists = true
			break
		}
	}

	results := make([]*ValidationResult, 0, len(files))

	// If e2e_test.md is missing, add a validation error
	if !e2eTestExists {
		results = append(results, &ValidationResult{
			File:   filepath.Join(v.planDir, "e2e_test.md"),
			Passed: false,
			Errors: []*ValidationError{
				{
					Code:     "E003",
					File:     filepath.Join(v.planDir, "e2e_test.md"),
					Message:  "缺少必需的 e2e_test.md 文件",
					Expected: "每个 plan 目录必须包含 e2e_test.md 作为端到端测试模块",
				},
			},
		})
	}

	for _, file := range files {
		result := v.ValidateFile(file)
		results = append(results, result)
	}

	return results, nil
}
```

**说明**:
- 检查 `e2e_test.md` 是否存在
- 如果不存在，添加 E003 错误
- 与提示词验证器第 78 行的要求一致

### 3. 更新文档

#### 3.1 prompts/plan.md

修改内容:
- 第 39 行: 保持 `e2e_test.md`
- 第 51 行: `[生产测试].md` → `e2e_test.md`
- 第 253 行: `ete_test.md` → `e2e_test.md`
- 第 262 行: "Plan: 生产测试" → "Plan: e2e_test"
- 第 274 行: `__ALL__` 依赖保持不变
- 第 398 行: 强调最后一个 Job 是"集成测试"
- 第 427 行: 表格中 "生产测试" → "E2E测试", 文件名 `[生产测试].md` → `e2e_test.md`
- 第 445 行: 依赖关系图中 `[生产测试]` → `e2e_test`
- 第 454 行: 执行顺序中 `[生产测试]` → `e2e_test`
- 第 460 行: 统计信息中 "生产测试模块" → "e2e_test 模块"
- 第 500 行: 设计原则中 "生产测试" → "E2E测试"
- 第 692 行: 输出信号中 `[生产测试]` → `e2e_test`
- 第 702 行: 文件清单中 `[生产测试].md` → `e2e_test.md`

#### 3.2 docs/PLAN_FORMAT_GUIDE.md

添加内容:
- 文件命名部分添加 `e2e_test.md` 作为特殊文件示例
- 添加 E003 错误代码说明（缺少 e2e_test.md）

#### 3.3 docs/plan-prompt-refactoring.md

修改内容:
- 第 36 行: 标题 "删除'生产测试' section，改为特殊模块" → "删除'生产测试' section，改为 e2e_test 模块"
- 第 50-58 行: 示例中 `[生产测试].md` → `e2e_test.md`
- 第 61-64 行: 影响描述更新
- 第 123 行: 特例从 `[生产测试].md` → `e2e_test.md`
- 第 240 行: 设计原则更新
- 第 267 行: 输出信号更新
- 第 299 行: 不兼容变更说明
- 第 340 行: 测试建议更新

## 错误代码定义

### E003: 缺少必需的 e2e_test.md 文件

**错误描述**: plan 目录中没有 `e2e_test.md` 文件

**修复方法**: 创建 `e2e_test.md` 文件，作为端到端测试模块

**文件模板**:
```markdown
# Plan: e2e_test

## 模块概述

**模块职责**: 验证整个系统的端到端功能、性能和稳定性

**对应 Research**:
- `.morty/research/deployment.md` - [部署相关调研]

**现有实现参考**: 无

**依赖模块**: __ALL__

**被依赖模块**: 无

## 接口定义

### 输入接口
- 完整的系统部署环境
- 所有功能模块已完成并通过集成测试

### 输出接口
- 端到端测试报告
- 性能测试结果

## 数据模型

无

## Jobs

---

### Job 1: 开发环境启动验证

#### 目标

确保开发环境正确启动且等价于生产环境

#### 前置条件

- 所有功能模块的集成测试已完成

#### Tasks

- [ ] Task 1: 启动开发环境
- [ ] Task 2: 验证服务健康状态
- [ ] Task 3: 验证配置加载正确

#### 验证器

- 开发环境启动后,所有服务处于健康状态
- 配置文件加载无错误

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 2: 端到端功能测试

#### 目标

验证完整业务流程正确工作

#### 前置条件

- job_1 - 开发环境启动验证通过

#### Tasks

- [ ] Task 1: 部署完整服务栈
- [ ] Task 2: 执行端到端测试套件
- [ ] Task 3: 验证关键业务指标

#### 验证器

- 用户可以完成完整的业务旅程
- 系统在预期负载下稳定运行

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 3: 集成测试

#### 目标

验证整个系统的端到端集成正确性

#### 前置条件

- job_1 - 开发环境启动验证通过
- job_2 - 端到端功能测试通过

#### Tasks

- [ ] Task 1: 验证所有模块协同工作正常
- [ ] Task 2: 验证系统在压力下的稳定性
- [ ] Task 3: 验证生产环境配置正确
- [ ] Task 4: 生成测试报告

#### 验证器

- 所有模块协同工作产生正确结果
- 系统在压力测试下保持稳定
- 生产环境配置验证通过
- 测试报告生成完整

#### 调试日志

无

#### 完成状态

⏳ 待开始
```

## 向后兼容性

### 不兼容变更

- ❌ **文件名变更**: `[生产测试].md` → `e2e_test.md`
  - 旧的 plan 文件如果使用 `[生产测试].md` 需要重命名
  - 需要更新 README.md 中的引用

### 迁移步骤

对于现有的 plan 文件:

1. **重命名文件**:
   ```bash
   cd .morty/plan/
   mv "[生产测试].md" e2e_test.md
   ```

2. **更新文件内容**:
   ```bash
   # 更新模块标题
   sed -i 's/# Plan: 生产测试/# Plan: e2e_test/' e2e_test.md
   ```

3. **更新 README.md**:
   - 模块列表表格中: `[生产测试].md` → `e2e_test.md`
   - 依赖关系图中: `[生产测试]` → `e2e_test`
   - 执行顺序中: `[生产测试]` → `e2e_test`

4. **验证格式**:
   ```bash
   morty plan validate --verbose
   ```

## 测试验证

### 1. 文件名验证测试

```bash
# 测试 e2e_test.md 被接受
echo "# Plan: e2e_test" > .morty/plan/e2e_test.md
morty plan validate e2e_test.md
# 期望: ✅ 通过

# 测试 [生产测试].md 被拒绝
echo "# Plan: 生产测试" > ".morty/plan/[生产测试].md"
morty plan validate "[生产测试].md"
# 期望: ❌ E001 文件名不符合规范
```

### 2. 存在性检查测试

```bash
# 删除 e2e_test.md
rm .morty/plan/e2e_test.md

# 运行验证
morty plan validate --verbose
# 期望: ❌ E003 缺少必需的 e2e_test.md 文件
```

### 3. 完整 plan 验证

```bash
# 生成新的 plan
morty plan

# AI 应该:
# 1. 生成 e2e_test.md（不是 [生产测试].md）
# 2. 自动运行 morty plan validate --verbose
# 3. 如果验证失败，自动修复错误
# 4. 最终输出: "格式验证: ✅ 所有文件通过检查"
```

## 相关文件

### 修改的文件
- `prompts/plan.md` - Plan 提示词（修正文件名和引用）
- `internal/validator/plan_validator.go` - Plan 验证器（添加 e2e_test.md 检查）
- `docs/PLAN_FORMAT_GUIDE.md` - Plan 格式指南（添加 E003 错误说明）
- `docs/plan-prompt-refactoring.md` - Plan 提示词重构文档（更新描述）

### 新增的文件
- `docs/plan-format-validator-fix.md` - 本文档

## 总结

本次修复的核心目标是**确保提示词、validator 和文档的一致性**:

1. **统一文件名**: 所有地方都使用 `e2e_test.md`，符合命名规范
2. **强化验证**: Validator 检查 `e2e_test.md` 必须存在（E003）
3. **保持一致**: 提示词、validator、文档三者完全对齐

这些修复确保：
- AI 生成的 plan 文件严格符合格式规范
- 所有模块文件名遵循小写+下划线规范
- 端到端测试模块必须存在且格式正确
- 验证器能够准确检测格式错误
