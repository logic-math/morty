# Plan: install

## 模块概述

**模块职责**: 提供 Morty 的安装、升级和卸载功能，支持一键安装、配置迁移、版本检查和清理卸载。确保开发环境和生产环境的同构性。

**初始安装问题（Bootstrap）**:
> 当环境中没有 Morty 时，用户无法运行 `morty install`。因此需要提供 **bootstrap（自举）安装** 方案。

**Bootstrap 安装方式**:
1. **`bootstrap.sh` 脚本**: 独立的初始安装脚本，无需 Morty 即可运行
2. **`curl | bash` 一键安装**: 从远程下载并执行安装
3. **手动安装**: 下载 release 包手动解压安装

**对应 Research**: 生产测试.md - 安装和升级测试；install.sh 现有实现

**依赖模块**: config, logging

**被依赖模块**: cli (install, upgrade, uninstall 命令)

## 接口定义

### 输入接口

```bash
# 安装命令
morty install [options]
  --prefix <path>           # 自定义安装路径（默认 $HOME/.morty）
  --bin-dir <path>          # 自定义命令路径（默认 $HOME/.local/bin）
  --force                   # 强制重新安装

# 升级命令
morty upgrade [options]
  --check                   # 仅检查更新，不执行升级
  --version <version>       # 升级到指定版本

# 卸载命令
morty uninstall [options]
  --purge                   # 彻底删除，包括配置文件和数据
```

### 输出接口

- `install_check_deps()`: 检查系统依赖
- `install_do_install()`: 执行安装
- `install_do_upgrade()`: 执行升级
- `install_do_uninstall()`: 执行卸载
- `install_get_version()`: 获取当前版本
- `install_check_update()`: 检查是否有更新

## 数据模型

### 安装路径结构

```
$MORTY_HOME/                    # 默认: ~/.morty
├── bin/                        # 可执行脚本
│   ├── morty                   # 主命令
│   ├── morty_doing.sh
│   ├── morty_fix.sh
│   ├── morty_loop.sh
│   ├── morty_plan.sh
│   ├── morty_research.sh
│   ├── morty_reset.sh
│   └── morty_stat.sh
├── lib/                        # 库文件
│   ├── common.sh
│   ├── config.sh
│   ├── logging.sh
│   ├── version_manager.sh
│   ├── cli_parse_args.sh
│   ├── cli_register_command.sh
│   ├── cli_route.sh
│   └── cli_execute.sh
├── prompts/                    # 提示词文件
│   └── doing.md
└── VERSION                     # 版本文件

$BIN_DIR/                       # 默认: ~/.local/bin
└── morty -> $MORTY_HOME/bin/morty  # 符号链接
```

### 版本信息格式

```
2.0.0
```

### 依赖检查清单

```yaml
dependencies:
  required:
    - name: bash
      version: ">= 4.0"
      check_cmd: "bash --version"
    - name: git
      version: ">= 2.0"
      check_cmd: "git --version"
    - name: claude  # 或配置的 ai_cli
      check_cmd: "claude --version"
  optional:
    - name: jq
      check_cmd: "jq --version"
      purpose: "JSON 处理增强"
    - name: tmux
      check_cmd: "tmux -V"
      purpose: "loop 监控模式"
```

## Jobs (Loop 块列表)

---

### Job 0: Bootstrap 自举安装（解决初始安装问题）

**目标**: 提供独立的初始安装方案，无需 Morty 即可安装 Morty

**前置条件**: 无（这是第一个 Job，用户环境中还没有 Morty）

**Tasks (Todo 列表)**:

- [ ] **创建 `bootstrap.sh` 独立安装脚本**
  - [ ] 实现系统依赖检查（bash >= 4.0, git >= 2.0）
  - [ ] 实现 `MORTY_HOME` 环境变量检查和设置提示
  - [ ] 实现安装目录创建（`$HOME/.morty/`）
  - [ ] 实现从 GitHub Release 下载最新版本
  - [ ] 实现解压和文件复制到安装目录
  - [ ] 实现符号链接创建（`$HOME/.local/bin/morty`）
  - [ ] 实现安装后验证（`morty version` 可执行）
  - [ ] 提供 PATH 配置提示

- [ ] **支持多种安装来源**
  - [ ] GitHub Release（默认）
  - [ ] 本地源码目录（开发模式）
  - [ ] 指定版本号安装

- [ ] **提供 `curl | bash` 一键安装方式**
  - [ ] 托管 `bootstrap.sh` 到可访问的 URL
  - [ ] 支持命令：`curl -sSL https://get.morty.dev | bash`

- [ ] **提供手动安装指南文档**
  - [ ] 下载 release 包步骤
  - [ ] 解压到指定目录步骤
  - [ ] 创建符号链接步骤

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 测试 bootstrap 安装时，必须在临时目录中进行，不得污染当前环境。

- **功能验证**:
  - 在干净环境（无 Morty）中运行 `bootstrap.sh` 应成功安装
  - 安装后 `morty version` 应能正常运行
  - `morty` 命令应能通过 `$HOME/.local/bin/morty` 访问
  - 如果 `$HOME/.local/bin` 不在 PATH 中，应给出明确的配置提示

- **隔离执行原则（测试时强制执行）**:
  - 测试必须使用 `bootstrap.sh --prefix /tmp/morty_test_xxx` 指向临时目录
  - 测试完成后必须清理临时目录
  - 当前 shell 环境不应被修改（除了可选的 PATH 提示）

**Bootstrap 脚本使用方式**:

```bash
# 方式 1: curl 一键安装（推荐）
curl -sSL https://raw.githubusercontent.com/user/morty/main/bootstrap.sh | bash

# 方式 2: 下载后执行
wget https://raw.githubusercontent.com/user/morty/main/bootstrap.sh
chmod +x bootstrap.sh
./bootstrap.sh

# 方式 3: 指定版本安装
curl -sSL https://get.morty.dev | bash -s -- --version 2.1.0

# 方式 4: 开发模式（从本地源码安装）
./bootstrap.sh --source ./ --prefix ~/.morty-dev
```

**与 `morty install` 的关系**:
- `bootstrap.sh`: 首次安装，系统中没有 Morty 时使用
- `morty install`: 重新安装或修复安装，已有 Morty 时使用

**调试日志**:
- 无

---

### Job 1: 依赖检查系统

**目标**: 实现系统依赖检查和版本验证，确保安装环境满足要求

**前置条件**: config 模块 Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 创建 `lib/install.sh` 模块
- [ ] 实现 `install_check_deps()`: 检查所有必需依赖
- [ ] 实现 `install_check_bash_version()`: 验证 Bash >= 4.0
- [ ] 实现 `install_check_git_version()`: 验证 Git >= 2.0
- [ ] 实现 `install_check_ai_cli()`: 检查 Claude Code CLI
- [ ] 实现 `install_check_optional_deps()`: 检查可选依赖
- [ ] 实现友好的缺失依赖提示和安装指导

**验证器**:
- 当 Bash 版本 < 4.0 时，应提示错误并建议升级
- 当 Git 未安装时，应提示安装 Git 的命令
- 当 Claude CLI 未找到时，应提示安装 Claude Code
- 可选依赖缺失时，应显示警告但不阻止安装
- 所有检查应返回结构化结果（通过/失败/警告）
- 检查耗时应小于 1 秒
- **必须在隔离环境检查通过后，才能执行后续操作**

**调试日志**:
- 无

---

### Job 2: 安装路径管理

**目标**: 实现安装路径的管理，支持自定义路径和冲突检测

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `install_get_default_prefix()`: 获取默认安装路径
- [ ] 实现 `install_validate_prefix()`: 验证安装路径有效
- [ ] 实现 `install_check_existing()`: 检查现有安装
- [ ] 实现 `install_ensure_dirs()`: 创建必要的目录结构
- [ ] 实现 PATH 检查，提示用户添加 PATH
- [ ] 处理已存在安装时的冲突（备份、覆盖、取消）

**验证器**（测试时遵循隔离执行原则）:
- **测试原则**: 测试时使用 `--prefix /tmp/morty_test_xxx` 指向临时目录，禁止在当前目录测试
- 默认安装路径应为 `$HOME/.morty`
- 路径包含空格时应正确处理
- 无写入权限时应提示权限不足
- 已存在安装时 `--force` 应备份并覆盖
- 应正确创建 `bin/`, `lib/`, `prompts/` 子目录
- 目录创建失败时应清理已创建的部分目录

**调试日志**:
- 无
- 路径包含空格时应正确处理
- 无写入权限时应提示权限不足
- 已存在安装时 `--force` 应备份并覆盖
- 应正确创建 `bin/`, `lib/`, `prompts/` 子目录
- 目录创建失败时应清理已创建的部分目录

**调试日志**:
- 无

---

### Job 3: 核心安装流程

**目标**: 实现从源码到安装目录的完整复制和配置

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `install_copy_files()`: 复制所有必要文件到目标路径
- [ ] 实现 `install_create_symlink()`: 创建命令符号链接
- [ ] 实现 `install_set_permissions()`: 设置文件权限
- [ ] 实现 `install_write_version()`: 写入版本文件
- [ ] 实现 `install_init_config()`: 初始化默认配置
- [ ] 更新 install.sh 调用 install 模块

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 验证 install 模块时，禁止在当前工作目录直接操作。必须创建临时隔离目录进行测试，确保测试副作用可回滚。

- **功能验证**:
  - 所有 `.sh` 脚本应被复制到目标 `$MORTY_HOME/bin/`
  - 所有 `lib/*.sh` 应被复制到 `$MORTY_HOME/lib/`
  - 所有 `prompts/*.md` 应被复制到 `$MORTY_HOME/prompts/`
  - 主命令应可通过 `$BIN_DIR/morty` 访问
  - 所有脚本应具有可执行权限（755）
  - 安装完成后 `morty version` 应返回正确版本

- **隔离执行原则（测试时强制执行）**:
  - 测试必须使用 `--prefix /tmp/morty_test_<timestamp>/` 指向临时目录
  - 测试完成后必须清理临时目录
  - 当前工作目录（项目源码目录）不得被修改

**调试日志**:
- 无

---

### Job 4: 配置迁移与升级

**目标**: 实现版本升级时的配置保留和迁移

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `install_backup_config()`: 备份现有配置
- [ ] 实现 `install_restore_config()`: 恢复用户配置
- [ ] 实现 `install_migrate_config()`: 配置版本迁移
- [ ] 实现 `install_check_update()`: 检查远程更新
- [ ] 实现 `install_get_latest_version()`: 获取最新版本
- [ ] 实现 `install_compare_versions()`: 版本号比较

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 验证升级功能时，禁止在当前工作目录直接操作。必须创建临时隔离目录进行测试。

- **功能验证**:
  - 升级时应备份 `settings.json` 到 `settings.json.backup`
  - 升级后用户配置应被保留（如 cli.command）
  - 新增配置项应使用默认值填充
  - 旧版配置项应被清理或迁移
  - 版本比较应正确识别 `2.0.0` > `1.9.9`
  - 配置迁移失败时应回滚到备份

- **隔离执行原则（测试时强制执行）**:
  - 测试必须使用 `--prefix /tmp/morty_test_<timestamp>/` 指向临时目录
  - 测试完成后必须清理临时目录
  - 当前工作目录（项目源码目录）不得被修改

**调试日志**:
- 无

---

### Job 5: 卸载功能

**目标**: 实现完整的卸载功能，支持保留数据和彻底清除

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `install_uninstall()`: 执行卸载
- [ ] 实现 `install_remove_files()`: 删除安装文件
- [ ] 实现 `install_remove_symlink()`: 删除符号链接
- [ ] 实现 `--purge` 模式，删除配置和数据
- [ ] 实现卸载确认提示
- [ ] 实现卸载后的清理检查

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 验证卸载功能时，禁止在当前工作目录直接操作。必须创建临时隔离目录进行测试。

- **功能验证**:
  - 默认卸载应删除 `$MORTY_HOME/` 和符号链接
  - 默认卸载应保留 `~/.mortyrc` 配置
  - `--purge` 应删除所有相关文件和配置
  - 卸载前应要求用户确认
  - 卸载后 `morty` 命令应不可用
  - 部分删除失败时应报告但继续

- **隔离执行原则（测试时强制执行）**:
  - 测试必须先安装到临时目录，然后在临时目录上测试卸载
  - 测试必须使用 `--prefix /tmp/morty_test_<timestamp>/` 指向临时目录
  - 测试完成后必须清理临时目录
  - 当前工作目录（项目源码目录）不得被修改

**安全措施**:
- 禁止删除当前工作目录内的文件
- 禁止删除非 Morty 相关的文件
- 必须验证待删除路径都在预期的安装目录内

**调试日志**:
- 无

---

### Job 6: 版本信息和管理

**目标**: 实现版本查询和更新检查功能

**前置条件**: Job 4 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `install_get_version()`: 获取当前安装版本
- [ ] 实现 `install_get_full_version()`: 获取详细版本信息（包含隔离环境状态）
- [ ] 实现 `install_check_update()`: 检查是否有新版本
- [ ] 实现 `install_get_sandbox_status()`: 获取当前隔离环境状态
- [ ] 集成到 `morty version` 命令
- [ ] 集成到 `morty upgrade --check` 命令
- [ ] 添加版本兼容性检查
- [ ] 添加隔离环境健康检查

**验证器**:
- `morty version` 应显示版本号
- `morty version --verbose` 应显示安装路径、配置路径、隔离环境状态等
- 检查更新应能访问 GitHub API 获取最新版本
- 有新版本时应提示用户升级
- 离线状态下应优雅降级，不报错
- 隔离环境状态检查应报告当前是否有待完成的操作

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**测试原则 - 隔离执行**:
> **执行集成测试时，禁止在当前工作目录直接操作。必须使用临时隔离目录进行测试，确保测试副作用可回滚。**

**测试执行要求**:
1. 测试前创建临时目录：`mkdir -p /tmp/morty_test_<timestamp>/`
2. 使用 `--prefix /tmp/morty_test_<timestamp>/` 执行安装
3. 在临时目录上验证所有功能
4. 测试完成后清理：`rm -rf /tmp/morty_test_<timestamp>/`
5. 验证当前工作目录未被修改

**功能验证**:
- 在新环境可以成功安装
- 安装后可以运行 `morty version` 和所有命令
- 升级后用户配置被保留
- 卸载后系统恢复干净状态
- 依赖检查正确识别缺失组件
- 安装路径冲突正确处理
- 回滚功能在失败时能正确恢复系统状态

---

## 待实现方法签名

```bash
# lib/install.sh

# ============================================================================
# Bootstrap 自举安装（独立脚本，无需 Morty 即可运行）
# ============================================================================
# bootstrap.sh 中的函数
bootstrap_check_deps()              # 检查系统依赖
bootstrap_download_release()        # 从 GitHub Release 下载
bootstrap_install_from_source()     # 从本地源码安装
bootstrap_create_symlink()          # 创建命令链接
bootstrap_verify_install()          # 验证安装结果
bootstrap_print_next_steps()        # 打印后续步骤

# ============================================================================
# 依赖检查
# ============================================================================
install_check_deps()
install_check_bash_version()
install_check_git_version()
install_check_ai_cli()
install_check_optional_deps()

# ============================================================================
# 路径管理
# ============================================================================
install_get_default_prefix()
install_validate_prefix(path)
install_check_existing(prefix)
install_ensure_dirs(prefix)

# ============================================================================
# 安装
# ============================================================================
install_do_install(prefix, bin_dir, force=false)
install_copy_files(source_dir, target_dir)
install_create_symlink(target, link_name)
install_set_permissions(dir)
install_write_version(dir, version)
install_init_config()

# ============================================================================
# 升级
# ============================================================================
install_do_upgrade(version="")
install_backup_config()
install_restore_config()
install_migrate_config(old_version, new_version)
install_check_update()
install_get_latest_version()
install_compare_versions(v1, v2)

# ============================================================================
# 卸载
# ============================================================================
install_do_uninstall(purge=false)
install_remove_files(dir)
install_remove_symlink(link)

# ============================================================================
# 版本信息
# ============================================================================
install_get_version()
install_get_full_version()
```

---

## 命令示例

### 安装命令

```bash
# 默认安装
morty install

# 自定义安装路径
morty install --prefix /opt/morty --bin-dir /usr/local/bin

# 强制重新安装（覆盖现有安装）
morty install --force
```

### 升级命令

```bash
# 检查更新
morty upgrade --check

# 升级到最新版本
morty upgrade

# 升级到指定版本
morty upgrade --version 2.1.0
```

### 卸载命令

```bash
# 标准卸载（保留配置）
morty uninstall

# 彻底卸载（删除所有数据）
morty uninstall --purge
```

### 版本命令

```bash
# 显示版本
morty version

# 详细版本信息
morty version --verbose
```

---

## 安装流程图

```
开始安装
    │
    ▼
检查依赖 ──失败──→ 显示安装指导 ──→ 退出
    │
    通过
    ▼
检查现有安装
    │
    ├──存在且无 --force──→ 提示已安装 ──→ 退出
    │
    ├──存在且有 --force──→ 备份现有安装 ──→ 继续
    │
    └──不存在──→ 继续
    │
    ▼
创建安装目录结构
    │
    ▼
复制文件到安装目录
    │
    ▼
创建符号链接
    │
    ▼
设置文件权限
    │
    ▼
写入版本文件
    │
    ▼
初始化配置
    │
    ▼
验证安装
    │
    ├──验证失败──→ 回滚并清理 ──→ 报错退出
    │
    └──验证通过
    │
    ▼
安装完成
```

---

## 升级流程图

```
开始升级
    │
    ▼
获取当前版本
    │
    ▼
获取最新版本
    │
    ▼
比较版本
    │
    ├──已是最新──→ 提示已是最新 ──→ 退出
    │
    └──有新版本──→ 继续
    │
    ▼
备份当前配置
    │
    ▼
执行安装（覆盖）
    │
    ▼
迁移配置
    │
    ▼
升级完成
```
