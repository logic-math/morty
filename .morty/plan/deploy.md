# Plan: Deploy 模块

## 模块概述

**模块职责**: 实现 Morty 的构建、安装、卸载和升级功能。提供 shell 脚本完成整个部署生命周期管理。

**对应 Research**:
- `morty-project-research.md` 第 3.4 节 版本管理分析

**依赖模块**: 无（独立脚本）

**被依赖模块**: 无（顶层部署脚本）

---

## 安装目录设计

### 用户级安装目录 `~/.morty/`

```
~/.morty/
├── bin/
│   └── morty                    # 可执行文件
└── config.json                  # 用户全局配置
```

### 配置说明

- **bin/**: 存放编译后的 `morty` 可执行文件
- **config.json**: 用户全局配置文件（可选，默认配置内嵌）
- 安装路径**: 固定在用户主目录下的 `.morty`，便于管理和卸载
- **PATH 配置**: 安装时自动将 `~/.morty/bin` 添加到 shell 配置文件

---

## 脚本清单

| 脚本 | 功能 | 使用场景 |
|------|------|----------|
| `build.sh` | 编译 Go 源码生成可执行文件 | 开发时构建 |
| `install.sh` | 安装到 `~/.morty/` 并配置环境 | 首次安装 |
| `uninstall.sh` | 从 `~/.morty/` 卸载并清理环境 | 不再需要时 |
| `upgrade.sh` | 检查更新并升级到新版本 | 版本更新时 |

---

## Jobs (Loop 块列表)

---

### Job 1: build.sh - 编译脚本

**目标**: 实现 Go 源码编译，生成可执行文件

**前置条件**:
- Go 1.21+ 环境
- 源代码完整

**Tasks (Todo 列表)**:
- [x] Task 1: 检测 Go 环境（版本检查）
- [x] Task 2: 解析构建参数（--output, --version, --os, --arch）
- [x] Task 3: 执行 `go mod tidy` 整理依赖
- [x] Task 4: 执行 `go build` 编译
- [x] Task 5: 注入版本信息（git commit, build time）
- [x] Task 6: 验证编译结果（文件存在、可执行）
- [x] Task 7: 输出构建信息

**验证器**:
- [x] Go 版本 >= 1.21 (实际: 1.21.6)
- [x] 编译成功生成二进制文件 (./bin/morty, 2.3MB)
- [x] 二进制文件可执行 (ELF 64-bit LSB executable)
- [x] 版本信息正确注入 (version: 2.0.0, git_commit: dfb4958, build_time: 2026-02-25)
- [x] 支持交叉编译（Linux/Mac）(成功构建 Darwin/amd64 Mach-O 可执行文件)

**使用示例**:
```bash
./scripts/build.sh                      # 默认构建
./scripts/build.sh --output ./bin/morty # 指定输出路径
./scripts/build.sh --version 2.0.0      # 指定版本号
```

**调试日志**:
- explore1: [探索发现] 项目使用 Go 1.21.6, 核心代码在 internal/ 目录, 现有 cmd/test_task/main.go 作为示例入口, 需要创建正式的 cmd/morty/main.go, 已记录
- debug1: 无重大问题, 构建脚本创建成功, 支持参数解析(--output, --version, --os, --arch), 版本信息注入(ldflags), 交叉编译(Linux/Darwin), 已修复

---

### Job 2: install.sh - 安装脚本

**目标**: 安装 Morty 到 `~/.morty/` 目录并配置环境

**前置条件**:
- 已编译的二进制文件 或 源码（可现场编译）

**Tasks (Todo 列表)**:
- [ ] Task 1: 检查是否已安装（存在 `~/.morty/bin/morty`）
- [ ] Task 2: 创建 `~/.morty/` 目录结构
- [ ] Task 3: 复制/编译二进制文件到 `~/.morty/bin/morty`
- [ ] Task 4: 创建默认配置文件 `~/.morty/config.json`
- [ ] Task 5: 检测并配置 PATH（`~/.bashrc`, `~/.zshrc`）
- [ ] Task 6: 验证安装（`morty version` 能执行）
- [ ] Task 7: 输出安装成功信息和使用说明

**验证器**:
- [ ] `~/.morty/` 目录创建成功
- [ ] `~/.morty/bin/morty` 可执行文件存在且可运行
- [ ] `~/.morty/config.json` 配置文件创建
- [ ] PATH 配置正确（当前 shell 或下次登录生效）
- [ ] `morty version` 能正常输出

**使用示例**:
```bash
./scripts/install.sh                    # 从源码安装
./scripts/install.sh --from-dist ./dist/morty  # 从预编译包安装
./scripts/install.sh --force            # 强制重新安装（覆盖）
```

**调试日志**:
- 待填充

---

### Job 3: uninstall.sh - 卸载脚本

**目标**: 从系统中卸载 Morty，清理安装目录和环境配置

**前置条件**:
- 无（即使未安装也能安全执行）

**Tasks (Todo 列表)**:
- [ ] Task 1: 检查安装状态（`~/.morty/` 是否存在）
- [ ] Task 2: 可选备份配置（询问用户）
- [ ] Task 3: 从 PATH 中移除 `~/.morty/bin`
- [ ] Task 4: 删除 `~/.morty/` 目录
- [ ] Task 5: 清理 shell 配置文件中的 Morty 相关配置
- [ ] Task 6: 验证卸载（`morty` 命令不再可用）
- [ ] Task 7: 输出卸载完成信息

**验证器**:
- [ ] `~/.morty/` 目录已删除
- [ ] shell 配置文件中 PATH 已清理
- [ ] `morty` 命令不再可用
- [ ] 用户项目目录 `.morty/` 保留（询问是否删除）

**使用示例**:
```bash
./scripts/uninstall.sh                  # 交互式卸载
./scripts/uninstall.sh --force          # 强制卸载（不询问）
./scripts/uninstall.sh --keep-config    # 保留配置文件
```

**调试日志**:
- 待填充

---

### Job 4: upgrade.sh - 升级脚本

**目标**: 检查版本更新并升级到最新版本

**前置条件**:
- 已安装 Morty
- 网络连接（可选，支持离线升级）

**Tasks (Todo 列表)**:
- [ ] Task 1: 检测当前版本（`morty version`）
- [ ] Task 2: 获取最新版本信息（本地检查或远程查询）
- [ ] Task 3: 版本对比，判断是否需要升级
- [ ] Task 4: 备份当前版本二进制和配置
- [ ] Task 5: 下载/编译新版本
- [ ] Task 6: 安装新版本（调用 install.sh 逻辑）
- [ ] Task 7: 验证升级成功
- [ ] Task 8: 支持回滚（升级失败时恢复旧版本）

**验证器**:
- [ ] 正确检测当前版本
- [ ] 正确判断是否需要升级
- [ ] 升级前备份旧版本
- [ ] 新版本安装成功
- [ ] 升级失败能回滚到旧版本
- [ ] 升级后配置保留

**使用示例**:
```bash
./scripts/upgrade.sh                    # 检查并升级
./scripts/upgrade.sh --check-only       # 仅检查，不升级
./scripts/upgrade.sh --version 2.1.0    # 升级到指定版本
./scripts/upgrade.sh --offline ./dist/morty  # 离线升级
```

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] build.sh 能正确编译生成二进制
- [ ] install.sh 能正确安装并配置环境
- [ ] uninstall.sh 能完全卸载并清理
- [ ] upgrade.sh 能检测升级并完成更新
- [ ] 升级失败能正确回滚
- [ ] 集成测试通过

**调试日志**:
- 待填充

---

## 使用示例

### 完整安装流程

```bash
# 1. 克隆仓库
git clone https://github.com/morty/morty.git
cd morty

# 2. 编译
./scripts/build.sh

# 3. 安装
./scripts/install.sh

# 4. 验证
morty version
morty help
```

### 日常升级

```bash
# 检查更新
./scripts/upgrade.sh --check-only

# 执行升级
cd morty
git pull
./scripts/upgrade.sh

# 或离线升级
./scripts/upgrade.sh --offline ./dist/morty-v2.1.0
```

### 完全卸载

```bash
# 卸载（保留项目配置）
./scripts/uninstall.sh

# 或强制卸载（不询问）
./scripts/uninstall.sh --force
```

---

## 配置文件示例

### `~/.morty/config.json` (用户全局配置)

```json
{
  "version": "2.0",
  "ai_cli": {
    "command": "ai_cli",
    "default_timeout": "10m",
    "enable_skip_permissions": true
  },
  "logging": {
    "level": "info",
    "format": "json"
  },
  "defaults": {
    "max_retry_count": 3,
    "auto_git_commit": true
  }
}
```

---

## 文件清单

- `scripts/build.sh` - 编译脚本
- `scripts/install.sh` - 安装脚本
- `scripts/uninstall.sh` - 卸载脚本
- `scripts/upgrade.sh` - 升级脚本
- `scripts/lib/common.sh` - 脚本公共库（可选）
