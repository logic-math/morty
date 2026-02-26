# Plan: 端到端功能测试 (E2E Test)

## 模块概述

**模块职责**: 通过完整的用户旅程测试验证 Morty CLI 所有功能是否正常工作

**测试范围**: 覆盖安装、research、plan、doing、stat、reset 完整工作流程

**依赖模块**: 所有模块（CLI, Config, Logging, State, Git, Plan Parser, Executor）

**被依赖模块**: 无

---

## 测试目标

验证 Morty 完整用户旅程：
1. 安装 → 2. 创建项目 → 3. Research → 4. Plan → 5. Doing → 6. Stat → 7. Reset

同时验证生产环境部署流程和发布检查清单。

---

## 部署架构验证

### 目标环境

#### 开发环境
- **操作系统**: Linux/macOS
- **Go 版本**: 1.21+
- **依赖**: Git 2.0+, Claude Code CLI
- **工具**: build.sh, go test

#### 生产环境
- **发布方式**: GitHub Releases / 源码安装
- **安装路径**: `$HOME/.morty/bin/morty`
- **配置路径**: `$HOME/.morty/config.json`

### 环境同构策略

#### 策略 1: Go 模块管理
- 使用 Go Modules 管理依赖
- go.mod 锁定依赖版本
- vendor 目录可选

#### 策略 2: 依赖版本声明
- Go: >= 1.21
- Git: >= 2.0
- Claude Code CLI: 最新版

#### 策略 3: 构建脚本标准化
```bash
./scripts/build.sh          # 构建可执行文件
./scripts/install.sh        # 安装到 ~/.morty/
go test ./...               # 运行单元测试
go test -cover ./...        # 运行覆盖率测试
./scripts/uninstall.sh      # 卸载
```

#### 策略 4: 配置管理
- 默认配置内嵌在二进制中
- 用户配置: `~/.morty/config.json`
- 项目配置: `./.morty/status.json`
- 环境变量覆盖配置

### 部署流程

```yaml
deployment:
  steps:
    - name: "构建"
      command: "./scripts/build.sh"
      outputs: ["dist/morty"]

    - name: "单元测试"
      command: "go test ./..."
      requires: ["构建"]

    - name: "覆盖率测试"
      command: "go test -cover ./..."
      requires: ["单元测试"]

    - name: "安装验证"
      command: "./scripts/install.sh && morty version"
      requires: ["覆盖率测试"]

    - name: "E2E 测试"
      command: "python3 tests/e2e/test_user_journey.py"
      requires: ["安装验证"]
```

---

## 测试环境

### 前置条件
- 干净的 Linux/macOS 环境
- 网络连接（下载 Go、Git 等）
- Bash 4.0+
- 无现有 Morty 安装

### 测试项目
- **项目类型**: Python CLI 数独游戏
- **项目目录**: `/tmp/test-sudoku-project`
- **功能需求**: 生成数独、显示数独、检查答案

---

## Jobs (Loop 块列表)

---

### Job 1: 安装 Morty

**目标**: 运行 install.sh 完成 Morty 安装

**前置条件**:
- 系统依赖已满足（bash, git, curl/wget）

**Tasks (Todo 列表)**:
- [x] Task 1: 下载 install.sh 脚本
  - 脚本位于项目目录 scripts/install.sh，无需下载
- [x] Task 2: 执行安装脚本
  - 使用构建的二进制文件执行 /opt/meituan/dolphinfs_sunquan20/ai_coding/Coding/morty/scripts/install.sh --force --from-dist /tmp/morty
- [x] Task 3: 验证安装目录创建
  - `$HOME/.morty/` 存在 ✓
  - `$HOME/.morty/bin/morty` 存在 ✓
- [x] Task 4: 验证命令可用
  - `which morty` 返回 /home/hadoop-recsys-gpu/.morty/bin/morty ✓
  - `morty version` 返回 2.0.0 ✓
- [x] Task 5: 检查默认配置生成
  - `$HOME/.morty/config.json` 存在且格式正确 ✓

**验证器**:
- [x] `morty version` 返回正确版本号（如 `morty 2.0.0`）
- [x] `which morty` 返回 `$HOME/.morty/bin/morty`
- [x] `$HOME/.morty/config.json` 存在且包含默认配置
- [x] 无安装错误信息

**调试日志**:
- debug1: [探索发现] 项目使用 Go 构建系统, install.sh 位于 scripts/install.sh, 支持 --force 和 --from-dist 参数, 安装到 ~/.morty/bin/morty, 已记录
- debug2: 验证器描述与实际安装行为存在差异, Plan 文件描述安装到 ~/.local/bin/morty 但实际安装到 ~/.morty/bin/morty, Plan 文件期望 settings.json 但实际生成 config.json, 不影响功能仅文档差异, 已记录

---

### Job 2: 创建测试项目

**目标**: 创建 Python 数独 CLI 项目作为测试目标

**前置条件**:
- Morty 已安装

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建项目目录
  ```bash
  mkdir -p /tmp/test-sudoku-project
  cd /tmp/test-sudoku-project
  ```
- [ ] Task 2: 初始化 Git 仓库
  ```bash
  git init
  git config user.email "test@test.com"
  git config user.name "Test User"
  ```
- [ ] Task 3: 创建基础项目结构
  ```
  /tmp/test-sudoku-project/
  ├── README.md              # 项目说明
  ├── requirements.txt       # 依赖
  └── sudoku/                # 源码目录
      └── __init__.py
  ```
- [ ] Task 4: 编写 README.md 描述需求
  ```markdown
  # 数独 CLI 游戏

  ## 功能需求
  - 生成有效的数独谜题
  - 显示数独网格（美观的 ASCII 格式）
  - 允许用户输入答案
  - 检查答案是否正确
  - 提供提示功能

  ## 使用方法
  python -m sudoku
  ```
- [ ] Task 5: 创建基础文件占位
  ```bash
  touch sudoku/__init__.py
  echo "" > requirements.txt
  ```
- [ ] Task 6: 初始提交
  ```bash
  git add .
  git commit -m "Initial commit: project setup"
  ```

**验证器**:
- [ ] 项目目录 `/tmp/test-sudoku-project` 存在
- [ ] `git status` 显示干净的工作区
- [ ] README.md 包含功能需求描述
- [ ] 目录结构正确

**调试日志**:
- 待填充

---

### Job 3: Research 模式测试

**目标**: 运行 `morty research` 进入研究状态，验证输出日志正常

**前置条件**:
- 在项目目录 `/tmp/test-sudoku-project`
- 项目已初始化

**Tasks (Todo 列表)**:
- [ ] Task 1: 启动 research 模式
  ```bash
  cd /tmp/test-sudoku-project
  morty research sudoku-game
  ```
- [ ] Task 2: 验证交互式提示
  - 检查是否提示 "请输入研究主题"（如无参数）
  - 或直接开始研究（如有参数）
- [ ] Task 3: 验证目录创建
  - `.morty/` 目录已创建
  - `.morty/research/` 子目录已创建
- [ ] Task 4: 观察 Claude Code Plan 模式启动
  - 检查日志输出：`cat .morty/logs/morty.log`
  - 确认加载 `prompts/research.md`
  - 确认传递研究主题
- [ ] Task 5: 模拟研究完成（测试脚本中使用超时或 mock）
- [ ] Task 6: 验证研究文件生成
  - `.morty/research/sudoku-game.md` 存在
  - 文件内容非空，包含结构化研究内容

**验证器**:
- [ ] `.morty/research/sudoku-game.md` 存在且非空
- [ ] `.morty/logs/morty.log` 包含研究模式启动日志
- [ ] 日志中包含研究主题信息
- [ ] 无错误日志
- [ ] 返回码为 0

**调试日志**:
- 待填充

---

### Job 4: Plan 模式测试

**目标**: 运行 `morty plan`，验证生成 Plan 文件

**前置条件**:
- Research 已完成，研究文件存在
- 在项目目录 `/tmp/test-sudoku-project`

**Tasks (Todo 列表)**:
- [ ] Task 1: 启动 plan 模式
  ```bash
  cd /tmp/test-sudoku-project
  morty plan
  ```
- [ ] Task 2: 验证研究文件加载
  - 检查日志是否显示 "找到研究文件: sudoku-game.md"
- [ ] Task 3: 验证交互式规划流程
  - 检查 Claude Code Plan 模式启动
  - 确认加载 `prompts/plan.md`
- [ ] Task 4: 验证 Plan 目录创建
  - `.morty/plan/` 目录存在
- [ ] Task 5: 验证 Plan 文件生成
  - `.morty/plan/README.md` 存在
  - 至少一个模块 Plan 文件（如 `.morty/plan/sudoku.md`）
- [ ] Task 6: 验证 Plan 文件格式
  - README.md 包含模块列表
  - 模块文件包含 Jobs 定义
  - 包含验证器定义

**验证器**:
- [ ] `.morty/plan/README.md` 存在且格式正确
- [ ] 至少有一个 `[模块名].md` 文件
- [ ] 每个 Plan 文件包含至少 1 个 Job
- [ ] 日志显示规划成功完成
- [ ] 返回码为 0

**调试日志**:
- 待填充

---

### Job 5: Doing 模式测试

**目标**: 运行 `morty doing` 执行真实开发

**前置条件**:
- Plan 文件已生成
- 在项目目录 `/tmp/test-sudoku-project`

**Tasks (Todo 列表)**:
- [ ] Task 1: 执行第一个 Job
  ```bash
  cd /tmp/test-sudoku-project
  morty doing
  ```
- [ ] Task 2: 验证状态检查
  - 检查日志是否读取 `status.json`
  - 确认找到第一个 PENDING Job
- [ ] Task 3: 验证 AI CLI 调用
  - 检查日志显示调用 Claude Code
  - 确认传递正确的提示词
- [ ] Task 4: 验证 Job 执行
  - 观察开发过程（测试脚本可设置超时）
  - 检查代码文件是否被修改/创建
- [ ] Task 5: 验证状态更新
  - `.morty/status.json` 创建并更新
  - Job 状态标记为 COMPLETED
- [ ] Task 6: 验证 Git 提交
  - `git log` 显示新的提交
  - 提交信息格式：`morty[loop:1]: [模块/job: COMPLETED]`
- [ ] Task 7: 连续执行多个 Jobs（可选）
  - 运行多次 `morty doing` 直到所有 Jobs 完成
  - 或使用循环执行

**验证器**:
- [ ] `.morty/status.json` 存在且状态正确
- [ ] Git 提交历史包含 morty 循环提交
- [ ] 源代码文件被修改/创建
- [ ] 每个 Job 完成后有对应的 Git 提交
- [ ] 日志记录完整的执行过程
- [ ] 返回码为 0

**调试日志**:
- 待填充

---

### Job 6: Stat 模式测试

**目标**: 运行 `morty stat` 验证监控功能正常

**前置条件**:
- 已有 Jobs 执行完成
- `status.json` 存在

**Tasks (Todo 列表)**:
- [ ] Task 1: 运行 stat 命令
  ```bash
  cd /tmp/test-sudoku-project
  morty stat
  ```
- [ ] Task 2: 验证默认输出格式
  - 表格形式输出
  - 包含当前执行模块/Job
  - 包含整体进度百分比
- [ ] Task 3: 验证监控模式
  ```bash
  morty stat -w &
  STAT_PID=$!
  sleep 5
  kill $STAT_PID
  ```
  - 检查是否每 60s 刷新
  - 检查原地刷新（无滚动）
- [ ] Task 4: 验证状态信息完整性
  - 当前执行模块和 Job
  - 上一个完成的 Job 摘要
  - 整体进度（完成百分比）
  - 累计时间
- [ ] Task 5: 验证 JSON 输出（如支持）
  ```bash
  morty stat --json
  ```
  - 输出有效的 JSON 格式

**验证器**:
- [ ] 输出包含当前模块/Job 信息
- [ ] 输出包含进度百分比
- [ ] 表格格式整齐易读
- [ ] 监控模式正常工作
- [ ] JSON 输出格式正确（如支持）
- [ ] 返回码为 0

**调试日志**:
- 待填充

---

### Job 7: Reset 模式测试

**目标**: 测试 `morty reset -l` 和 `morty reset -c` 功能

**前置条件**:
- 已有多个 Jobs 执行完成
- Git 提交历史存在

**Tasks (Todo 列表)**:
- [ ] Task 1: 查看循环历史
  ```bash
  cd /tmp/test-sudoku-project
  morty reset -l
  ```
- [ ] Task 2: 验证历史输出格式
  - 表格形式
  - 包含 CommitID
  - 包含 Message（如 `morty[loop:1]: ...`）
  - 包含时间
- [ ] Task 3: 验证指定数量
  ```bash
  morty reset -l 3
  ```
  - 只显示最近 3 条提交
- [ ] Task 4: 获取一个提交哈希
  ```bash
  COMMIT=$(morty reset -l 1 | tail -1 | awk '{print $1}')
  ```
- [ ] Task 5: 测试回滚功能
  ```bash
  morty reset -c $COMMIT
  ```
- [ ] Task 6: 验证回滚结果
  - 提示用户确认（Y/n）
  - `git log` 显示已回滚到指定提交
  - `git status` 显示工作区干净
  - `.morty/status.json` 状态更新

**验证器**:
- [ ] `reset -l` 表格格式正确
- [ ] 提交信息包含循环编号
- [ ] `reset -c` 成功回滚
- [ ] 回滚后代码状态正确
- [ ] 状态文件同步更新
- [ ] 返回码为 0

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 所有 E2E Jobs 完成

**验证器**:
- [ ] 完整的用户旅程通过
- [ ] 所有命令返回码为 0
- [ ] 日志文件无错误信息
- [ ] 生成的代码可运行
- [ ] Git 历史完整可追溯
- [ ] 状态文件准确反映进度

**调试日志**:
- 待填充

---

## 测试脚本模板

```python
#!/usr/bin/env python3
"""
Morty E2E 测试脚本
测试完整的用户旅程：安装 → Research → Plan → Doing → Stat → Reset
"""

import subprocess
import os
import tempfile
import shutil
import sys

def run_cmd(cmd, cwd=None, timeout=60):
    """运行命令并返回结果"""
    print(f"\n[执行] {cmd}")
    result = subprocess.run(
        cmd, shell=True, cwd=cwd, capture_output=True, text=True, timeout=timeout
    )
    print(result.stdout)
    if result.stderr:
        print(f"[stderr] {result.stderr}")
    return result.returncode == 0

def test_install():
    """测试安装"""
    print("\n" + "="*50)
    print("步骤 1: 安装 Morty")
    print("="*50)

    # 下载并执行安装脚本
    assert run_cmd("curl -sSL .../install.sh -o /tmp/install.sh")
    assert run_cmd("bash /tmp/install.sh")

    # 验证安装
    assert run_cmd("which morty")
    assert run_cmd("morty version")
    print("✓ 安装成功")

def test_create_project():
    """测试创建项目"""
    print("\n" + "="*50)
    print("步骤 2: 创建 Python 数独项目")
    print("="*50)

    project_dir = "/tmp/test-sudoku-project"
    os.makedirs(project_dir, exist_ok=True)

    # 初始化 Git
    assert run_cmd("git init", cwd=project_dir)

    # 创建文件
    with open(f"{project_dir}/README.md", "w") as f:
        f.write("# 数独 CLI 游戏\n\n## 功能需求\n...")

    os.makedirs(f"{project_dir}/sudoku", exist_ok=True)
    open(f"{project_dir}/sudoku/__init__.py", "w").close()

    # 初始提交
    assert run_cmd("git add .", cwd=project_dir)
    assert run_cmd('git commit -m "Initial commit"', cwd=project_dir)
    print("✓ 项目创建成功")

def test_research():
    """测试 Research 模式"""
    print("\n" + "="*50)
    print("步骤 3: Research 模式")
    print("="*50)

    project_dir = "/tmp/test-sudoku-project"

    # 启动 research（这里需要 mock 或超时处理）
    # 实际测试中使用 subprocess.Popen 和超时
    print("模拟 research 完成...")

    # 验证结果
    assert os.path.exists(f"{project_dir}/.morty/research/sudoku-game.md")
    print("✓ Research 成功")

def test_plan():
    """测试 Plan 模式"""
    print("\n" + "="*50)
    print("步骤 4: Plan 模式")
    print("="*50)

    project_dir = "/tmp/test-sudoku-project"

    # 启动 plan
    print("模拟 plan 完成...")

    # 验证结果
    assert os.path.exists(f"{project_dir}/.morty/plan/README.md")
    print("✓ Plan 成功")

def test_doing():
    """测试 Doing 模式"""
    print("\n" + "="*50)
    print("步骤 5: Doing 模式")
    print("="*50)

    project_dir = "/tmp/test-sudoku-project"

    # 执行一个 Job
    print("模拟 doing 执行...")

    # 验证结果
    assert os.path.exists(f"{project_dir}/.morty/status.json")
    print("✓ Doing 成功")

def test_stat():
    """测试 Stat 模式"""
    print("\n" + "="*50)
    print("步骤 6: Stat 模式")
    print("="*50)

    project_dir = "/tmp/test-sudoku-project"

    # 运行 stat
    assert run_cmd("morty stat", cwd=project_dir)
    print("✓ Stat 成功")

def test_reset():
    """测试 Reset 模式"""
    print("\n" + "="*50)
    print("步骤 7: Reset 模式")
    print("="*50)

    project_dir = "/tmp/test-sudoku-project"

    # 查看历史
    assert run_cmd("morty reset -l", cwd=project_dir)

    # 获取提交哈希并回滚（简化版）
    print("模拟 reset -c 回滚...")
    print("✓ Reset 成功")

def main():
    """主测试流程"""
    print("\n" + "="*60)
    print("Morty 端到端功能测试")
    print("="*60)

    try:
        test_install()
        test_create_project()
        test_research()
        test_plan()
        test_doing()
        test_stat()
        test_reset()

        print("\n" + "="*60)
        print("✓ 所有测试通过！")
        print("="*60)
        return 0

    except AssertionError as e:
        print(f"\n✗ 测试失败: {e}")
        return 1
    except Exception as e:
        print(f"\n✗ 测试异常: {e}")
        return 1

if __name__ == "__main__":
    sys.exit(main())
```

---

## 文件清单

- `plan/e2e_test.md` - 本文件
- `tests/e2e/test_user_journey.py` - E2E 测试脚本（待生成）

---

## 附录

### Go 项目验证命令

```bash
# 1. 运行单元测试
go test ./...

# 2. 运行覆盖率测试
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 3. 构建可执行文件
./scripts/build.sh

# 4. 功能验证
./dist/morty version           # 显示版本
./dist/morty help              # 显示帮助
./dist/morty doing --help      # 显示 doing 命令帮助
./dist/morty doing             # 执行 Plan
./dist/morty doing --restart   # 重置并执行

# 5. 性能测试
go test -bench=. ./...

# 6. 静态检查
go vet ./...
go fmt ./...
```

### 发布检查清单

- [ ] 所有单元测试通过
- [ ] 代码覆盖率 >= 80%
- [ ] 集成测试通过
- [ ] 端到端测试通过 (本文件所有 Jobs)
- [ ] 部署脚本测试通过 (build/install/uninstall/upgrade)
- [ ] 文档已更新
- [ ] CHANGELOG 已更新
- [ ] 版本号已更新
- [ ] Git tag 已创建
- [ ] GitHub Release 已发布（如适用）

---

**注意**: 此 E2E 测试必须在所有功能模块开发完成后执行，用于验证完整用户旅程和生产部署流程。