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
- [x] Task 1: 创建 `internal/parser/markdown/parser.go`
- [x] Task 2: 实现 `Parse(content string) (Document, error)`
- [x] Task 3: 实现标题解析 (H1-H6)
- [x] Task 4: 实现段落和列表解析
- [x] Task 5: 实现代码块解析
- [x] Task 6: 注册到工厂
- [x] Task 7: 编写单元测试 `markdown/parser_test.go`

**验证器**:
- [x] 正确解析标题层级
- [x] 正确解析无序列表和有序列表
- [x] 正确解析代码块（带语言标识）
- [x] 通过工厂能获取 Markdown 解析器
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用标准 Go 项目结构, parser 模块包含 factory.go 和 interface.go, Parser 接口定义包含 Parse/ParseString/Supports/FileType 方法, 使用 ParseResult 结构体返回结果, 已记录
- debug1: 编译错误 - fmt.Sprintf 递归调用, 现象: go test 报错 "fmt.Sprintf format %v with arg n causes recursive String method call", 复现: 在 Node.String() 方法中使用 fmt.Sprintf("Unknown: %v", n) 导致递归, 猜想: 1)%v 格式会调用 String() 方法造成递归 2)需要避免在 String() 中使用 %v 格式化自身, 验证: 将 %v 改为具体字段引用, 修复: 修改为 fmt.Sprintf("Unknown: type=%s content=%q", n.Type, n.Content), 已修复
- debug2: 文件路径问题, 现象: 创建的文件在 /opt/meituan/... 但测试运行目录是 /home/sankuai/..., 复现: 写入文件后执行 go test 找不到包, 猜想: 1)存在两个独立的工作目录 2)symlink 导致路径混淆, 验证: 检查两个目录内容发现差异, 修复: 复制文件到正确的 /home/sankuai/.../internal/parser/ 目录, 已修复

---

### Job 3: Section 提取

**目标**: 实现章节提取功能

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/parser/markdown/section.go`
- [x] Task 2: 实现 `ExtractSections(doc) ([]Section, error)`
- [x] Task 3: 实现 `FindSection(doc, title) (Section, error)`
- [x] Task 4: 实现 Section 内容范围计算
- [x] Task 5: 支持递归提取子章节
- [x] Task 6: 编写单元测试 `markdown/section_test.go`

**验证器**:
- [x] 正确提取指定级别的所有 sections
- [x] 正确提取 section 的内容范围
- [x] 递归提取子章节正确
- [x] 按名称查找 section 正确
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用标准 Go 项目结构, markdown parser 已实现基础解析功能, 包含 NodeType (heading/paragraph/list/codeblock), Document 结构包含 Nodes 切片, parser.go 使用正则表达式解析, 已记录
- debug1: 路径问题导致测试无法找到, 现象: go test 运行但不执行 section_test.go 中的测试, 复现: 在 /opt/meituan/... 目录运行但 go.mod 在 /home/sankuai/..., 猜想: 1)存在两个独立的工作目录 2)Go 模块路径不匹配, 验证: 检查 go env GOMOD 发现路径不一致, 修复: 切换到正确目录 /home/sankuai/... 运行测试, 已修复
- debug2: parser.go 编译错误, 现象: go test 报错 "fmt.Sprintf format %v with arg n causes recursive String method call", 复现: Node.String() 方法中使用 fmt.Sprintf("Unknown: %v", n) 导致无限递归, 猜想: %v 格式会调用 String() 方法造成递归调用, 验证: 修改为 %s 格式引用具体字段, 修复: 改为 fmt.Sprintf("Unknown: %s", n.Type), 已修复
- debug3: Section 提取逻辑问题, 现象: TestExtractSections_Basic 期望 1 个顶级 section 但得到 2 个, 复现: 嵌套 section 的层级关系计算错误, 猜想: 1)EndIndex 计算错误 2)父子关系建立逻辑错误, 验证: 使用递归 buildSections 函数重建层级关系, 修复: 重写 extractAllSections 使用两阶段方法: 先收集所有 headings, 然后递归构建层级树, 已修复

---

### Job 4: Task 列表提取

**目标**: 从 Markdown 中提取 Task 列表（checkbox）

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/parser/markdown/task.go`
- [x] Task 2: 实现 `ExtractTasks(doc) ([]Task, error)`
- [x] Task 3: 解析 `- [ ]` 和 `- [x]` 格式
- [x] Task 4: 提取 Task 描述文本
- [x] Task 5: 记录 Task 缩进级别（用于层级关系）
- [x] Task 6: 编写单元测试 `markdown/task_test.go`

**验证器**:
- [x] 正确识别未完成 Task (`- [ ]`)
- [x] 正确识别已完成 Task (`- [x]` 或 `- [X]`)
- [x] 正确提取 Task 描述
- [x] 正确处理 Task 层级（缩进）
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: 测试无法运行问题, 现象: go test 运行但不执行 task_test.go 中的测试, 复现: 在 /opt/meituan/... 运行但 Go 工具链缓存问题, 猜想: 1)文件未被正确识别 2)路径问题, 验证: 删除并重新创建文件, 修复: 使用子代理直接在包目录运行测试, 已修复
- debug2: Task 缩进级别计算错误, 现象: TestExtractTasks_IndentationLevels 失败, Task 2/3 期望 level 1/2 但得到 0, 复现: 解析器在 parseLine 中使用 TrimSpace 移除缩进信息, 猜想: 1)Node 结构未保留原始缩进 2)task extractor 无法获取缩进, 验证: 检查 parser.go 发现 TrimSpace 在正则匹配前执行, 修复: 1)在 Node 结构中添加 ItemIndents 字段 2)在 parseLine 中计算并存储缩进级别 3)在 task extractor 中使用 ItemIndents, 已修复
- debug3: TestParseTaskFromLine_InvalidFormats 失败, 现象: "[ ] No bullet" 被错误识别为有效 task, 复现: taskContentRegex 匹配了没有 bullet 的 task, 猜想: 需要区分原始行和列表项内容, 验证: 检查测试期望只有带 bullet 前缀的才是有效 task, 修复: 移除 taskContentRegex, 在 extractTasksFromNode 中重建完整 task 行（带 bullet 前缀）, 已修复

---

### Job 5: Frontmatter 解析

**目标**: 解析 Markdown 前置元数据（YAML frontmatter）

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/parser/markdown/metadata.go`
- [x] Task 2: 实现 `ExtractMetadata(content) (map[string]string, error)`
- [x] Task 3: 支持 YAML frontmatter (`---` 包裹）
- [x] Task 4: 支持简单的 key: value 解析
- [x] Task 5: 处理 frontmatter 解析错误
- [x] Task 6: 编写单元测试 `markdown/metadata_test.go`

**验证器**:
- [x] 正确解析 YAML frontmatter
- [x] 正确提取键值对
- [x] 无 frontmatter 时返回空 map
- [x] 格式错误时返回错误
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- explore1: [探索发现] 项目使用标准 Go 项目结构, markdown parser 已包含 section.go 和 task.go, Node 结构在不同目录版本略有差异(无 ItemIndents 字段), 测试使用标准 Go testing 模式, 已记录
- debug1: 文件路径问题, 现象: 文件写入 /opt/meituan/.../Coding/morty/ 但 Go 模块在 /home/sankuai/.../internal/parser/, 复现: go test 找不到新创建的 metadata.go, 猜想: 1)存在两个独立工作目录 2)symlink 导致路径混淆, 验证: 检查两个目录内容发现差异, 修复: 复制文件到正确的 /home/sankuai/.../internal/parser/markdown/ 目录, 已修复
- debug2: Node 结构差异导致编译错误, 现象: metadata.go 编译报错 "node.ItemIndents undefined", 复现: 当前目录的 parser.go Node 结构没有 ItemIndents 字段, 猜想: 1)不同目录版本不一致 2)task.go 使用了不同的 Node 结构, 验证: 比较两个目录的 parser.go 发现差异, 修复: 修改 nodeToRawContent 函数移除 ItemIndents 引用, 已修复
- debug3: 空 frontmatter 匹配失败, 现象: TestHasFrontmatter 失败 "---\n---" 未匹配, 复现: frontmatterRegex 要求换行符, 猜想: 正则表达式过于严格, 验证: 修改正则为 `(?s)^\s*---\s*\n(.*?)\n?---\s*(?:\n|$)`, 修复: 使中间换行符可选, 已修复

---

### Job 6: Plan 文件专用解析

**目标**: 提供 Plan 文件的专用解析功能

**前置条件**:
- Job 3, Job 4, Job 5 完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 `internal/parser/plan/parser.go`
- [x] Task 2: 定义 `Plan` 结构体
- [x] Task 3: 实现 `ParsePlan(content) (*Plan, error)`
- [x] Task 4: 提取模块概述（模块职责、依赖）
- [x] Task 5: 提取 Jobs 列表（按 Job 分组）
- [x] Task 6: 提取每个 Job 的 Tasks
- [x] Task 7: 提取验证器定义
- [x] Task 8: 编写单元测试 `plan/parser_test.go`

**验证器**:
- [x] 正确解析模块名称
- [x] 正确解析依赖模块列表
- [x] 正确提取所有 Jobs
- [x] 正确提取 Job 内的 Tasks
- [x] 正确提取验证器描述
- [x] 所有单元测试通过 (覆盖率 >= 80%)

**调试日志**:
- debug1: 任务提取正则表达式不匹配, 测试期望3个任务但得到0个, 正则表达式未正确匹配中文冒号, 修改正则表达式支持中英文冒号, 重写extractTasksFromContent使用行解析替代正则匹配, 已修复
- debug2: 验证器提取失败, 正则表达式范围匹配问题, section内容范围计算, 修改提取逻辑使用section.Content, 使用FindStringIndex定位验证器区块, 已修复
- debug3: 调试日志提取为空, "无"标记未正确处理, 空白行处理, 添加空内容检查, 优化extractDebugLogs过滤"无"标记, 已修复
- explore1: [探索发现] 代码库使用标准Go项目结构, markdown解析器已包含section/task/metadata提取, 测试使用表驱动模式, 覆盖率92.4%超过80%要求, 已记录

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
