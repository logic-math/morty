# Plan: install

## 模块概述

**模块职责**: 提供 Morty 的安装、升级和卸载功能。**统一使用 `bootstrap.sh` 脚本作为唯一入口**，通过参数控制不同操作。无需 Morty 即可运行，避免"先有鸡还是先有蛋"的问题。

**设计原则**: **单一入口，参数控制**
> 不实现 `morty install/uninstall/upgrade` 命令，所有安装相关操作统一通过 `bootstrap.sh` 完成。

**使用方式**:
```bash
# 首次安装
curl -sSL https://get.morty.dev | bash

# 或本地执行
./bootstrap.sh install

# 重新安装（覆盖现有安装）
./bootstrap.sh reinstall

# 升级到新版本
./bootstrap.sh upgrade

# 卸载
./bootstrap.sh uninstall

# 指定版本
./bootstrap.sh install --version 2.1.0

# 自定义路径
./bootstrap.sh install --prefix /opt/morty --bin-dir /usr/local/bin
```

**对应 Research**: 生产测试.md - 安装和升级测试

**依赖模块**: 无（bootstrap.sh 是自包含脚本，不依赖 Morty 任何模块）

**被依赖模块**: 无（安装完成后才有 Morty 命令）

---

## 数据模型

### 安装路径结构

```
$MORTY_HOME/                    # 默认: ~/.morty
├── bin/                        # 可执行脚本
│   ├── morty                   # 主命令
│   ├── morty_doing.sh
│   ├── morty_fix.sh
│   ├── morty_plan.sh
│   ├── morty_research.sh
│   ├── morty_reset.sh
│   └── morty_stat.sh
├── lib/                        # 库文件
│   ├── common.sh
│   ├── config.sh
│   ├── logging.sh
│   └── version_manager.sh
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
    - name: curl 或 wget
      purpose: "下载 release 包"
  optional:
    - name: jq
      check_cmd: "jq --version"
      purpose: "JSON 处理增强"
```

---

## Jobs (Loop 块列表)

---

### Job 1: Bootstrap 脚本框架

**目标**: 创建 `bootstrap.sh` 脚本框架和参数解析

**前置条件**: 无

**Tasks (Todo 列表)**:
- [ ] 创建 `bootstrap.sh` 脚本，添加 shebang 和基础配置
- [ ] 实现 `bootstrap_parse_args()`: 解析命令行参数
  - [ ] 支持子命令: `install`, `reinstall`, `upgrade`, `uninstall`
  - [ ] 支持选项: `--prefix`, `--bin-dir`, `--version`, `--force`, `--purge`
  - [ ] 支持帮助: `--help`, `-h`
- [ ] 实现 `bootstrap_show_help()`: 显示使用帮助
- [ ] 实现 `bootstrap_validate_args()`: 验证参数组合合法性
- [ ] 实现 `bootstrap_main()`: 主入口函数，根据子命令分发

**命令设计**:
```bash
bootstrap.sh [命令] [选项]

命令:
  install      首次安装 Morty（默认命令）
  reinstall    重新安装（覆盖现有安装）
  upgrade      升级到新版本
  uninstall    卸载 Morty

选项:
  --prefix <path>       安装路径（默认: $HOME/.morty）
  --bin-dir <path>      命令链接路径（默认: $HOME/.local/bin）
  --version <version>   指定版本（默认: latest）
  --force               强制操作（安装/卸载时跳过确认）
  --purge               彻底卸载（删除配置和数据）
  --source <path>       从本地源码安装（开发模式）
  -h, --help            显示帮助信息
```

**验证器**:
- `./bootstrap.sh --help` 应显示完整的帮助信息
- `./bootstrap.sh install --prefix /tmp/test` 应正确解析参数
- 不支持的子命令应报错并显示帮助
- 冲突参数组合应报错（如 `install --purge`）

**调试日志**:
- 无

---

### Job 2: 依赖检查和环境验证

**目标**: 实现系统依赖检查和安装环境验证

**前置条件**: Job 1 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `bootstrap_check_system_deps()`: 检查系统依赖
  - [ ] 检查 Bash 版本 >= 4.0
  - [ ] 检查 Git 版本 >= 2.0
  - [ ] 检查 curl 或 wget 存在
- [ ] 实现 `bootstrap_check_install_env()`: 检查安装环境
  - [ ] 检查目标目录是否可写
  - [ ] 检查磁盘空间是否充足（至少 50MB）
  - [ ] 检查是否已存在 Morty 安装
- [ ] 实现友好的错误提示和修复建议

**验证器**:
- Bash < 4.0 时应提示升级 Bash
- 无 curl/wget 时应提示安装其中之一
- 目录无写权限时应提示权限不足
- 已存在安装时应提示使用 `reinstall` 或 `upgrade`

**调试日志**:
- 无

---

### Job 3: 安装功能

**目标**: 实现 `bootstrap.sh install` 和 `bootstrap.sh reinstall` 命令

**前置条件**: Job 2 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `bootstrap_cmd_install()`: 处理 install 命令
  - [ ] 检查是否已存在安装，存在则提示使用 reinstall
  - [ ] 确定安装版本（latest 或指定版本）
  - [ ] 下载 release 包或复制本地源码
  - [ ] 创建安装目录结构
  - [ ] 复制文件到安装目录
  - [ ] 创建符号链接
  - [ ] 设置文件权限
  - [ ] 验证安装（运行 `morty version`）
- [ ] 实现 `bootstrap_cmd_reinstall()`: 处理 reinstall 命令
  - [ ] 备份现有配置（settings.json）
  - [ ] 执行全新安装
  - [ ] 恢复用户配置
- [ ] 实现 `bootstrap_download_release()`: 从 GitHub 下载 release
- [ ] 实现 `bootstrap_install_from_source()`: 从本地源码安装（开发模式）

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 测试安装功能时，必须使用 `--prefix /tmp/morty_test_xxx` 指向临时目录。

- 安装后 `$MORTY_HOME/bin/morty` 应存在且可执行
- 安装后 `$BIN_DIR/morty` 符号链接应正确指向
- 安装后 `morty version` 应返回正确版本
- reinstall 应保留用户配置（settings.json）
- 安装失败时应清理已创建的文件，不遗留垃圾

**调试日志**:
- 无

---

### Job 4: 升级功能

**目标**: 实现 `bootstrap.sh upgrade` 命令

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `bootstrap_cmd_upgrade()`: 处理 upgrade 命令
  - [ ] 获取当前安装版本
  - [ ] 获取最新可用版本（GitHub API）
  - [ ] 比较版本，如果已是最新则提示退出
  - [ ] 备份当前完整安装
  - [ ] 下载并安装新版本
  - [ ] 迁移配置（如有必要）
  - [ ] 验证升级成功
  - [ ] 失败时回滚到备份
- [ ] 实现 `bootstrap_get_current_version()`: 获取当前版本
- [ ] 实现 `bootstrap_get_latest_version()`: 获取最新版本
- [ ] 实现 `bootstrap_compare_versions()`: 版本号比较

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 测试升级功能时，必须使用临时目录和测试版本。

- 已是最新版本时应提示并退出
- 升级后用户配置应被保留
- 升级失败时应能回滚到原版本
- 升级后 `morty version` 应显示新版本

**调试日志**:
- 无

---

### Job 5: 卸载功能

**目标**: 实现 `bootstrap.sh uninstall` 命令

**前置条件**: Job 3 完成

**Tasks (Todo 列表)**:
- [ ] 实现 `bootstrap_cmd_uninstall()`: 处理 uninstall 命令
  - [ ] 检查是否存在 Morty 安装
  - [ ] 显示将要删除的内容预览
  - [ ] 要求用户确认（除非 `--force`）
  - [ ] 删除安装目录
  - [ ] 删除符号链接
  - [ ] 清理空目录
  - [ ] 显示卸载完成信息
- [ ] 实现 `--purge` 模式：同时删除配置文件和数据
- [ ] 实现 `bootstrap_backup_before_uninstall()`: 卸载前备份（用于误操作恢复）

**验证器**（测试时遵循隔离执行原则）:

> **测试原则**: 测试卸载功能时，必须在临时目录的安装上测试。

- 无 Morty 安装时应提示并退出
- 默认卸载应保留 `~/.mortyrc` 配置
- `--purge` 应删除所有相关文件
- 卸载后 `morty` 命令应不可用
- 卸载前应要求确认（有提示信息）

**调试日志**:
- 无

---

### Job 6: 远程安装和文档

**目标**: 实现 curl | bash 一键安装和编写安装文档

**前置条件**: Job 3, 4, 5 完成

**Tasks (Todo 列表)**:
- [ ] 确保 `bootstrap.sh` 支持管道执行（`curl ... | bash`）
- [ ] 测试一键安装命令：`curl -sSL https://get.morty.dev | bash`
- [ ] 编写 `INSTALL.md` 安装指南文档
  - [ ] 快速安装（一键安装）
  - [ ] 手动安装步骤
  - [ ] 升级指南
  - [ ] 卸载指南
  - [ ] 常见问题排查
- [ ] 在 `README.md` 中添加安装说明

**验证器**:
- `curl -sSL <url> | bash` 应能成功安装
- 离线环境下手动安装步骤应清晰可行
- 安装文档应覆盖所有使用场景

**调试日志**:
- 无

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**测试原则 - 隔离执行**:
> **所有安装/升级/卸载测试必须在临时目录中进行，禁止操作当前工作目录。**

**测试场景**:

| 场景 | 命令 | 验证点 |
|------|------|--------|
| 首次安装 | `./bootstrap.sh install --prefix /tmp/t1` | 安装成功，`morty version` 可用 |
| 重复安装 | `./bootstrap.sh install --prefix /tmp/t1` | 提示已安装，建议使用 reinstall |
| 重新安装 | `./bootstrap.sh reinstall --prefix /tmp/t1` | 覆盖安装，配置保留 |
| 升级 | `./bootstrap.sh upgrade --prefix /tmp/t1` | 版本更新，配置保留 |
| 卸载 | `./bootstrap.sh uninstall --prefix /tmp/t1` | 完全清理，命令不可用 |
| 彻底卸载 | `./bootstrap.sh uninstall --purge --prefix /tmp/t1` | 包括配置全部删除 |

**测试执行要求**:
1. 使用 `--prefix /tmp/morty_test_<timestamp>/` 指向临时目录
2. 测试完成后清理临时目录
3. 验证当前工作目录未被修改

---

### Job 7: 清理过时代码

**目标**: 删除不再使用的旧版脚本和代码，保持代码库整洁

**前置条件**: Job 6 完成（新的 bootstrap.sh 已可用）

**待清理的过时代码清单**:

| 文件/目录 | 说明 | 删除原因 |
|-----------|------|----------|
| `morty_fix.sh` | 旧版修复脚本 | 被 `morty doing` + `morty reset` 替代 |
| `morty_loop.sh` | 旧版循环执行脚本 | 被 `morty doing` 替代 |
| `lib/loop_monitor.sh` | 循环监控模块 | 随 loop 模式一起废弃 |
| `install.sh` | 旧版安装脚本 | 被 `bootstrap.sh` 替代 |
| `morty` (旧版主脚本) | 如果存在旧版入口 | 被新版 `morty` 主命令替代 |

**Tasks (Todo 列表)**:
- [ ] **识别过时代码**
  - [ ] 列出所有待删除的文件清单
  - [ ] 确认这些文件不再被任何代码引用
  - [ ] 确认这些功能已被新实现替代
- [ ] **备份（可选）**
  - [ ] 创建过时代码的备份分支或标签（如需要保留历史）
- [ ] **执行删除**
  - [ ] 删除 `morty_fix.sh`
  - [ ] 删除 `morty_loop.sh`
  - [ ] 删除 `lib/loop_monitor.sh`
  - [ ] 删除 `install.sh`
  - [ ] 清理 `bin/` 目录中过时的脚本链接
- [ ] **验证清理**
  - [ ] 确保 `morty doing` 仍能正常运行
  - [ ] 确保 `morty plan` 仍能正常运行
  - [ ] 确保 `morty research` 仍能正常运行
  - [ ] 确保 `morty reset` 仍能正常运行
  - [ ] 运行测试套件，无失败
- [ ] **更新文档**
  - [ ] 从 README.md 中移除过时命令的引用
  - [ ] 从文档中删除 `fix`、`loop` 等过时模式的说明
  - [ ] 添加变更日志记录删除操作

**验证器**:
- 删除后 `morty_fix` 命令应不存在
- 删除后 `morty_loop` 命令应不存在
- 删除后 `morty doing/plan/research/reset` 应正常工作
- 代码库中不应再有对 loop_monitor 的引用
- git 历史中仍保留这些文件的记录（可回溯）

**安全准则**:
- 只删除已确认被替代的功能
- 确保有 git 历史可回溯
- 删除前在团队中沟通
- 删除后进行完整测试

**调试日志**:
- 无

---

## 待实现方法签名

```bash
# bootstrap.sh

# ============================================================================
# 参数解析和主入口
# ============================================================================
bootstrap_parse_args()
bootstrap_validate_args()
bootstrap_show_help()
bootstrap_main()

# ============================================================================
# 命令处理函数
# ============================================================================
bootstrap_cmd_install()
bootstrap_cmd_reinstall()
bootstrap_cmd_upgrade()
bootstrap_cmd_uninstall()

# ============================================================================
# 依赖检查
# ============================================================================
bootstrap_check_system_deps()
bootstrap_check_install_env()

# ============================================================================
# 安装操作
# ============================================================================
bootstrap_download_release(version)
bootstrap_install_from_source(source_path)
bootstrap_copy_files(target_dir)
bootstrap_create_symlink(target, link_name)
bootstrap_set_permissions(dir)
bootstrap_verify_install()

# ============================================================================
# 升级操作
# ============================================================================
bootstrap_get_current_version()
bootstrap_get_latest_version()
bootstrap_compare_versions(v1, v2)
bootstrap_backup_installation()
bootstrap_migrate_config()

# ============================================================================
# 卸载操作
# ============================================================================
bootstrap_backup_before_uninstall()
bootstrap_remove_files(dir)
bootstrap_remove_symlink(link)
```

---

## 命令示例

### 安装

```bash
# 一键安装（推荐）
curl -sSL https://get.morty.dev | bash

# 本地执行安装
./bootstrap.sh install

# 自定义路径安装
./bootstrap.sh install --prefix /opt/morty --bin-dir /usr/local/bin

# 安装指定版本
./bootstrap.sh install --version 2.1.0

# 开发模式（从本地源码安装）
./bootstrap.sh install --source ./ --prefix ~/.morty-dev
```

### 重新安装

```bash
# 重新安装（覆盖现有安装，保留配置）
./bootstrap.sh reinstall

# 强制重新安装（不提示确认）
./bootstrap.sh reinstall --force
```

### 升级

```bash
# 升级到最新版本
./bootstrap.sh upgrade

# 升级到指定版本
./bootstrap.sh upgrade --version 2.1.0
```

### 卸载

```bash
# 标准卸载（保留配置）
./bootstrap.sh uninstall

# 彻底卸载（删除所有数据）
./bootstrap.sh uninstall --purge

# 强制卸载（不提示确认）
./bootstrap.sh uninstall --force
```

---

## 安装流程图

### install 流程

```
开始安装
    │
    ▼
解析参数
    │
    ▼
检查系统依赖 ──失败──→ 显示错误和修复建议 ──→ 退出
    │
    通过
    ▼
检查安装环境
    │
    ├──已存在安装──→ 提示使用 reinstall ──→ 退出
    │
    └──未安装──→ 继续
    │
    ▼
确定版本（latest 或指定）
    │
    ▼
下载 release / 复制源码
    │
    ▼
创建安装目录
    │
    ▼
复制文件
    │
    ▼
创建符号链接
    │
    ▼
设置权限
    │
    ▼
验证安装（运行 morty version）
    │
    ├──失败──→ 清理 ──→ 报错退出
    │
    └──成功
    │
    ▼
显示安装完成信息和下一步
```

### upgrade 流程

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
备份当前安装
    │
    ▼
下载并安装新版本
    │
    ▼
迁移配置
    │
    ▼
验证升级
    │
    ├──失败──→ 回滚 ──→ 报错退出
    │
    └──成功
    │
    ▼
升级完成
```

---

## 与现有命令的关系

**Morty 核心命令**（安装后可用）:
- `morty research` - 研究模式
- `morty plan` - 规划模式
- `morty doing` - 执行模式
- `morty fix` - 修复模式
- `morty reset` - 重置/回滚
- `morty version` - 版本信息

**安装管理**（独立脚本，无需 Morty）:
- `bootstrap.sh install` - 安装
- `bootstrap.sh reinstall` - 重新安装
- `bootstrap.sh upgrade` - 升级
- `bootstrap.sh uninstall` - 卸载

**设计优势**:
1. **单一入口**: 所有安装管理通过一个脚本完成
2. **无依赖**: bootstrap.sh 不依赖 Morty，解决初始安装问题
3. **维护简单**: 不需要维护两套安装逻辑
4. **用户友好**: 命令直观，学习成本低
