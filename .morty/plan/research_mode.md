# Plan: research_mode

## 模块概述

**模块职责**: 实现 Research 模式，支持交互式代码库/文档库研究，输出研究结果到 `.morty/research/`。

**对应 Research**: Research 模式详细设计；交互式研究流程

**依赖模块**: config, logging

**被依赖模块**: plan_mode

## 接口定义

### 输入接口
- `morty research <topic>`: 启动研究模式
- 用户交互输入：调查主题、搜索路径、追问回答
- 搜索源配置：本地文件、目录、URL

### 输出接口
- `.morty/research/[主题].md`: 研究结果文件
- 控制台输出：研究进度、关键发现、验证结果

## 数据模型

### 研究结果文件结构
```markdown
# [调查主题] 调研报告

**调查主题**: [topic]
**调研日期**: [ISO8601]
**调研目标**: [一句话描述]

---

## 1. 核心发现

### 1.1 事实 1
[详细描述]

### 1.2 事实 2
[详细描述]

## 2. 技术细节

### 2.1 架构分析
...

### 2.2 关键代码
...

## 3. 潜在问题

## 4. 改进建议

## 5. 相关资源

---

**文档版本**: 1.0
**研究完成时间**: [ISO8601]
**状态**: [已完成/进行中]
```

### 搜索路径配置
```yaml
research:
  topic: "api-design"
  sources:
    - type: "local"
      paths: ["src/api/", "docs/api/"]
      patterns: ["*.py", "*.md"]
    - type: "url"
      urls: ["https://example.com/docs"]
  keywords: ["endpoint", "handler", "router"]
```

## Jobs (Loop 块列表)

---

### Job 1: Research 模式基础架构

**目标**: 建立 Research 模式的核心框架，支持启动研究会话和初始化目录

**前置条件**: config, logging 模块核心功能完成

**Tasks (Todo 列表)**:
- [ ] 创建 `morty_research.sh` 脚本
- [ ] 实现 `research_check_prerequisites()`: 检查依赖
- [ ] 实现 `research_init_directory()`: 初始化 `.morty/research/` 目录
- [ ] 实现 `research_build_prompt()`: 构建研究提示词
- [ ] 实现 `research_start_session()`: 启动 Claude Code 会话

**验证器**:
- 运行 `morty research test-topic` 应启动 Claude Code 会话
- 当 `.morty/` 目录不存在时，应自动创建
- 研究提示词应包含主题、系统提示词和上下文
- Claude Code 会话应能正确接收用户输入
- 会话结束后应返回到原始 shell

**调试日志**:
- 无

---

### Job 2: 搜索结果记录

**目标**: 在研究过程中记录搜索到的资源和关键信息

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `research_record_finding()`: 记录研究发现
- [ ] 实现 `research_record_source()`: 记录搜索源
- [ ] 实现 `research_record_code_snippet()`: 记录代码片段
- [ ] 实现 `research_record_architecture()`: 记录架构信息
- [ ] 实现临时笔记文件管理

**验证器**:
- 研究发现应被追加到临时笔记文件
- 记录应包含时间戳和来源信息
- 代码片段应使用正确的 Markdown 代码块格式
- 架构图应使用文本格式存储（如 Mermaid）
- 临时文件应在研究完成时合并到最终结果

**调试日志**:
- 无

---

### Job 3: 研究结果生成

**目标**: 生成结构化的研究结果文档

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `research_generate_report()`: 生成研究报告
- [ ] 实现模板引擎（报告格式化）
- [ ] 实现发现分类整理
- [ ] 实现资源链接收集
- [ ] 实现报告验证（格式检查）

**验证器**:
- 生成的报告应包含所有必要章节（发现、技术细节、问题、建议）
- 报告应使用正确的 Markdown 格式
- 文件应保存为 `.morty/research/[主题].md`
- 报告应包含时间戳和版本信息
- 已有同名报告应备份后再覆盖

**调试日志**:
- 无

---

### Job 4: 研究验证器

**目标**: 验证研究结果的完整性和质量

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `research_validate()`: 验证研究结果
- [ ] 实现检查清单验证（目录存在、文件存在、内容非空）
- [ ] 实现关键发现数量检查
- [ ] 实现资源引用有效性检查
- [ ] 生成验证报告

**验证器**:
- 验证应检查 `.morty/research/` 目录存在
- 应检查 `[主题].md` 文件存在且非空
- 应检查报告包含至少一个核心发现
- 应检查所有引用的资源存在或可访问
- 验证失败时应提供改进建议

**调试日志**:
- 无

---

### Job 5: 多主题研究管理

**目标**: 支持多个研究主题的管理和切换

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `research_list_topics()`: 列出所有研究主题
- [ ] 实现 `research_show_topic(topic)`: 显示指定主题内容
- [ ] 实现 `research_resume_topic(topic)`: 继续已有研究
- [ ] 实现主题依赖关系（研究引用）
- [ ] 实现研究主题索引生成

**验证器**:
- `morty research --list` 应显示所有研究主题
- 可以继续之前未完成的研究
- 研究主题之间可以建立引用关系
- 索引文件应自动更新
- 删除主题时应清理相关文件

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 完整的 research 流程可以从启动到生成报告
- 生成的报告符合格式规范
- 研究结果可以被 plan 模式正确读取
- 多个研究主题可以独立管理
- 验证器可以检测不完整的研究结果

---

## 待实现方法签名

```bash
# morty_research.sh

# 入口
research_main(topic, options)

# 初始化和检查
research_check_prerequisites()
research_init_directory()
research_build_prompt(topic)
research_start_session(prompt)

# 记录
research_record_finding(type, content, source)
research_record_source(path, type)
research_record_code_snippet(code, language, source)
research_record_architecture(description)

# 报告生成
research_generate_report(topic)
research_classify_findings()
research_collect_resources()
research_apply_template(content)

# 验证
research_validate(topic)
research_check_completeness()
research_check_quality()
research_generate_validation_report()

# 多主题管理
research_list_topics()
research_show_topic(topic)
research_resume_topic(topic)
research_create_topic_index()
research_delete_topic(topic)

# 工具
research_sanitize_filename(name)
research_get_timestamp()
research_backup_existing(path)
```
