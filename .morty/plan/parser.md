# Plan: Parser

## 模块概述

**模块职责**: 提供通用的文件解析框架，支持多种文件格式的解析。当前实现 Markdown 解析，未来可扩展 JSON、YAML、TOML 等格式。

**对应 Research**:
- `plan-mode-design.md` 第 3 节 Plan 文件规范

**依赖模块**: 无

**被依赖模块**: plan_cmd, research_cmd, doing_cmd (通过 ParserFactory 获取具体解析器)

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      Parser Framework                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │  ParserFactory  │  │  Parser Interface│  │ FileType    │ │
│  │  (解析器工厂)    │  │  (通用接口)      │  │ (文件类型)   │ │
│  └────────┬────────┘  └────────┬────────┘  └──────┬──────┘ │
└───────────┼────────────────────┼──────────────────┼────────┘
            │                    │                  │
            ▼                    ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    具体解析器实现                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │ MarkdownParser  │  │  JsonParser     │  │ YamlParser  │ │
│  │ (Markdown解析)   │  │  (预留)         │  │ (预留)      │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │ PlanParser      │  │ PromptParser    │                   │
│  │ (Plan专用)       │  │ (Prompt专用)    │                   │
│  └─────────────────┘  └─────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 接口定义

### 核心接口

```go
// Parser 通用解析器接口
type Parser interface {
    // Parse 解析内容字符串
    Parse(content string) (Document, error)
    // ParseFile 解析文件
    ParseFile(filepath string) (Document, error)
    // SupportedExtensions 返回支持的文件扩展名
    SupportedExtensions() []string
}

// Document 通用文档接口
type Document interface {
    // GetType 返回文档类型
    GetType() FileType
    // GetMetadata 获取元数据
    GetMetadata() map[string]string
    // GetRawContent 获取原始内容
    GetRawContent() string
}

// SectionExtractor 章节提取接口（可选实现）
type SectionExtractor interface {
    ExtractSections(doc Document) ([]Section, error)
    FindSection(doc Document, title string) (Section, error)
}

// TaskExtractor 任务提取接口（可选实现）
type TaskExtractor interface {
    ExtractTasks(doc Document) ([]Task, error)
}

// MetadataExtractor 元数据提取接口（可选实现）
type MetadataExtractor interface {
    ExtractMetadata(content string) (map[string]string, error)
}

// FileType 文件类型
type FileType string

const (
    FileTypeMarkdown FileType = "markdown"
    FileTypeJSON     FileType = "json"
    FileTypeYAML     FileType = "yaml"
    FileTypeTOML     FileType = "toml"
    FileTypeUnknown  FileType = "unknown"
)
```

### Markdown 专用接口

```go
// MarkdownDocument Markdown 文档
type MarkdownDocument struct {
    Type        FileType
    Title       string
    Sections    []MarkdownSection
    Metadata    map[string]string
    RawContent  string
}

func (d MarkdownDocument) GetType() FileType { return d.Type }
func (d MarkdownDocument) GetMetadata() map[string]string { return d.Metadata }
func (d MarkdownDocument) GetRawContent() string { return d.RawContent }

// MarkdownSection Markdown 章节
type MarkdownSection struct {
    Level       int
    Title       string
    Content     string
    SubSections []MarkdownSection
}

// Task 任务项 (从 checkbox 提取)
type Task struct {
    Index       int
    Description string
    Completed   bool
    Level       int
}

// CodeBlock 代码块
type CodeBlock struct {
    Language    string
    Content     string
}

// MarkdownParser Markdown 解析器接口
type MarkdownParser interface {
    Parser
    SectionExtractor
    TaskExtractor
    MetadataExtractor
    // ExtractCodeBlocks 提取代码块
    ExtractCodeBlocks(section MarkdownSection) []CodeBlock
}
```

### 解析器工厂

```go
// Factory 解析器工厂
type Factory interface {
    // Register 注册解析器
    Register(p Parser) error
    // Get 根据文件类型获取解析器
    Get(fileType FileType) (Parser, error)
    // GetByExtension 根据文件扩展名获取解析器
    GetByExtension(ext string) (Parser, error)
    // DetectFileType 检测文件类型
    DetectFileType(filepath string) FileType
}

// NewFactory 创建解析器工厂
func NewFactory() Factory
```

---

## 输入输出接口

### 输入接口
- 文件路径或内容字符串
- 解析选项（如提取特定 section）

### 输出接口
- 结构化的 Document 数据
- 提取的 sections、tasks、validators 等

---

## Jobs (Loop 块列表)

---

### Job 1: 解析器框架核心

**目标**: 实现 Parser 框架核心接口和工厂

**前置条件**:
- 无

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/parser/interface.go` 定义核心接口
- [x] Task 2: 创建 `internal/parser/factory.go` 实现解析器工厂
- [x] Task 3: 实现 `DetectFileType()` 文件类型检测
- [x] Task 4: 实现 `Register()` 解析器注册
- [x] Task 5: 实现 `Get()` 和 `GetByExtension()` 解析器获取
- [x] Task 6: 实现错误处理（未知文件类型等）
- [x] Task 7: 编写单元测试 `factory_test.go`

**验证器**:
- [x] 能根据扩展名正确检测文件类型
- [x] 能注册和获取解析器
- [x] 未知文件类型返回错误
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用标准 Go 项目结构, internal/ 包含 config/ 和 git/ 模块, 测试使用标准 Go testing 模式, 已记录
- debug1: 初始文件创建路径错误, 文件写入到 /opt/meituan/... 但 shell 工作目录是 /home/sankuai/..., 验证: 检查目录结构发现差异, 修复: 在正确位置 /home/sankuai/.../internal/parser/ 重新创建文件, 已修复

---

### Job 2: Markdown 解析器基础

**目标**: 实现 Markdown 解析器

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/parser/markdown/parser.go`
- [ ] Task 2: 实现 `Parse(content string) (Document, error)`
- [ ] Task 3: 实现标题解析 (H1-H6)
- [ ] Task 4: 实现段落和列表解析
- [ ] Task 5: 实现代码块解析
- [ ] Task 6: 注册到工厂
- [ ] Task 7: 编写单元测试 `markdown/parser_test.go`

**验证器**:
- [ ] 正确解析标题层级
- [ ] 正确解析无序列表和有序列表
- [ ] 正确解析代码块（带语言标识）
- [ ] 通过工厂能获取 Markdown 解析器
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 3: Section 提取

**目标**: 实现章节提取功能

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/parser/markdown/section.go`
- [ ] Task 2: 实现 `ExtractSections(doc) ([]Section, error)`
- [ ] Task 3: 实现 `FindSection(doc, title) (Section, error)`
- [ ] Task 4: 实现 Section 内容范围计算
- [ ] Task 5: 支持递归提取子章节
- [ ] Task 6: 编写单元测试 `markdown/section_test.go`

**验证器**:
- [ ] 正确提取指定级别的所有 sections
- [ ] 正确提取 section 的内容范围
- [ ] 递归提取子章节正确
- [ ] 按名称查找 section 正确
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 4: Task 列表提取

**目标**: 从 Markdown 中提取 Task 列表（checkbox）

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/parser/markdown/task.go`
- [ ] Task 2: 实现 `ExtractTasks(doc) ([]Task, error)`
- [ ] Task 3: 解析 `- [ ]` 和 `- [x]` 格式
- [ ] Task 4: 提取 Task 描述文本
- [ ] Task 5: 记录 Task 缩进级别（用于层级关系）
- [ ] Task 6: 编写单元测试 `markdown/task_test.go`

**验证器**:
- [ ] 正确识别未完成 Task (`- [ ]`)
- [ ] 正确识别已完成 Task (`- [x]` 或 `- [X]`)
- [ ] 正确提取 Task 描述
- [ ] 正确处理 Task 层级（缩进）
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 5: Frontmatter 解析

**目标**: 解析 Markdown 前置元数据（YAML frontmatter）

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/parser/markdown/metadata.go`
- [ ] Task 2: 实现 `ExtractMetadata(content) (map[string]string, error)`
- [ ] Task 3: 支持 YAML frontmatter (`---` 包裹）
- [ ] Task 4: 支持简单的 key: value 解析
- [ ] Task 5: 处理 frontmatter 解析错误
- [ ] Task 6: 编写单元测试 `markdown/metadata_test.go`

**验证器**:
- [ ] 正确解析 YAML frontmatter
- [ ] 正确提取键值对
- [ ] 无 frontmatter 时返回空 map
- [ ] 格式错误时返回错误
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 6: Plan 文件专用解析

**目标**: 提供 Plan 文件的专用解析功能

**前置条件**:
- Job 3, Job 4, Job 5 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/parser/plan/parser.go`
- [ ] Task 2: 定义 `Plan` 结构体
- [ ] Task 3: 实现 `ParsePlan(content) (*Plan, error)`
- [ ] Task 4: 提取模块概述（模块职责、依赖）
- [ ] Task 5: 提取 Jobs 列表（按 Job 分组）
- [ ] Task 6: 提取每个 Job 的 Tasks
- [ ] Task 7: 提取验证器定义
- [ ] Task 8: 编写单元测试 `plan/parser_test.go`

**验证器**:
- [ ] 正确解析模块名称
- [ ] 正确解析依赖模块列表
- [ ] 正确提取所有 Jobs
- [ ] 正确提取 Job 内的 Tasks
- [ ] 正确提取验证器描述
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

### Job 7: Prompt 文件解析

**目标**: 提供 Prompt 文件的专用解析

**前置条件**:
- Job 5 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 `internal/parser/prompt/parser.go`
- [ ] Task 2: 定义 `Prompt` 结构体
- [ ] Task 3: 实现 `ParsePrompt(filepath) (*Prompt, error)`
- [ ] Task 4: 解析 frontmatter 元数据
- [ ] Task 5: 提取模板变量（如 `{{variable}}`）
- [ ] Task 6: 实现变量替换功能
- [ ] Task 7: 编写单元测试 `prompt/parser_test.go`

**验证器**:
- [ ] 正确解析 prompt 文件
- [ ] 正确提取 frontmatter
- [ ] 正确识别模板变量
- [ ] 变量替换功能正常
- [ ] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 扩展设计

### 未来支持的文件类型

```go
// JSON 解析器（预留）
type JSONParser struct {}
func (p *JSONParser) Parse(content string) (Document, error) { ... }

// YAML 解析器（预留）
type YAMLParser struct {}
func (p *YAMLParser) Parse(content string) (Document, error) { ... }

// TOML 解析器（预留）
type TOMLParser struct {}
func (p *TOMLParser) Parse(content string) (Document, error) { ... }
```

### 添加新解析器的步骤

1. 实现 `Parser` 接口
2. 实现对应的提取接口（可选）
3. 注册到工厂：`factory.Register(newParser)`

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 解析器工厂能正确管理所有解析器
- [ ] Markdown 解析流程完整
- [ ] Plan 文件解析正确
- [ ] Prompt 文件解析正确
- [ ] 性能测试：大文件（>100KB）解析 < 100ms
- [ ] 集成测试通过 (覆盖率 >= 80%)

**调试日志**:
- 待填充

---

## 使用示例

```go
// 创建解析器工厂
factory := parser.NewFactory()

// 注册 Markdown 解析器（框架自动注册）
// factory.Register(markdown.NewParser())

// 根据文件类型获取解析器
p, err := factory.Get(parser.FileTypeMarkdown)
doc, err := p.ParseFile("plan/cli.md")

// 使用 Markdown 专用功能
if md, ok := p.(parser.MarkdownParser); ok {
    sections, _ := md.ExtractSections(doc)
    tasks, _ := md.ExtractTasks(doc)
}

// 解析 Plan 文件
planParser := parser.NewPlanParser(factory)
plan, err := planParser.ParsePlanFile("plan/config.md")
fmt.Printf("模块: %s, Jobs 数量: %d\n", plan.Name, len(plan.Jobs))

// 解析 Prompt 并替换变量
promptParser := parser.NewPromptParser(factory)
prompt, err := promptParser.ParsePrompt("prompts/doing.md")
content := prompt.ReplaceVars(map[string]string{
    "MODULE": "config",
    "JOB": "job_1",
})
```

---

## 文件清单

- `internal/parser/interface.go` - 核心接口定义
- `internal/parser/factory.go` - 解析器工厂
- `internal/parser/markdown/parser.go` - Markdown 基础解析
- `internal/parser/markdown/section.go` - Section 提取
- `internal/parser/markdown/task.go` - Task 提取
- `internal/parser/markdown/metadata.go` - Frontmatter 解析
- `internal/parser/plan/parser.go` - Plan 专用解析
- `internal/parser/prompt/parser.go` - Prompt 专用解析
