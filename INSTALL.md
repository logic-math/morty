# Morty 安装指南

本文档提供 Morty 的完整安装、升级和卸载说明。

## 目录

- [系统要求](#系统要求)
- [快速安装](#快速安装)
- [手动安装](#手动安装)
- [开发模式安装](#开发模式安装)
- [升级指南](#升级指南)
- [卸载指南](#卸载指南)
- [常见问题排查](#常见问题排查)

---

## 系统要求

### 必需依赖

| 依赖 | 最低版本 | 说明 |
|------|----------|------|
| Bash | >= 4.0 | 脚本执行环境 |
| Git | >= 2.0 | 用于版本管理和克隆 |
| curl 或 wget | 任意 | 用于下载 Release 包 |

### 可选依赖

| 依赖 | 用途 |
|------|------|
| jq | JSON 处理增强（升级时解析版本信息） |
| rsync | 本地源码安装时更快复制文件 |

### 磁盘空间

- 最低要求：50MB 可用空间
- 建议：100MB 以上

---

## 快速安装

### 一键安装（推荐）

使用以下命令一键安装最新版本的 Morty：

```bash
curl -sSL https://get.morty.dev | bash
```

如果需要自动确认（非交互模式）：

```bash
curl -sSL https://get.morty.dev | bash -s -- --force
```

### 自定义路径安装

```bash
# 指定安装路径和可执行文件路径
curl -sSL https://get.morty.dev | bash -s -- install --prefix /opt/morty --bin-dir /usr/local/bin

# 安装指定版本
curl -sSL https://get.morty.dev | bash -s -- install --version 2.1.0
```

### 验证安装

安装完成后，运行以下命令验证：

```bash
morty version
morty --help
```

如果 `morty` 命令未找到，请确保 `~/.local/bin` 在你的 PATH 中：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

可以将上述命令添加到 `~/.bashrc` 或 `~/.zshrc` 中以持久化配置。

---

## 手动安装

如果你无法使用一键安装命令，或者处于离线环境，可以手动安装 Morty。

### 步骤 1：下载源码

#### 方式 A：通过 Git 克隆

```bash
git clone https://github.com/anthropics/morty.git
cd morty
```

#### 方式 B：下载 Release 压缩包

```bash
# 下载最新版本
curl -sL -o morty.tar.gz https://github.com/anthropics/morty/archive/refs/heads/master.tar.gz

# 解压
tar -xzf morty.tar.gz
cd morty-master
```

### 步骤 2：运行安装脚本

```bash
# 默认安装（安装到 ~/.morty）
./bootstrap.sh install

# 自定义安装路径
./bootstrap.sh install --prefix /opt/morty --bin-dir /usr/local/bin
```

### 步骤 3：添加到 PATH

如果安装路径不在 PATH 中，添加以下行到你的 shell 配置文件：

```bash
# Bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### 步骤 4：验证安装

```bash
morty version
```

---

## 开发模式安装

如果你正在开发 Morty，需要从本地源码安装：

```bash
# 从当前目录安装（开发模式）
./bootstrap.sh install --source ./ --prefix ~/.morty-dev

# 强制重新安装（开发调试时使用）
./bootstrap.sh reinstall --source ./ --force
```

开发模式安装的特点：
- 直接从本地源码复制文件，不经过下载
- 适合开发和测试新功能
- 可以使用 `--prefix` 指定独立的开发环境路径

---

## 升级指南

### 升级到最新版本

```bash
./bootstrap.sh upgrade
```

### 升级到指定版本

```bash
./bootstrap.sh upgrade --version 2.1.0
```

### 强制升级（跳过确认）

```bash
./bootstrap.sh upgrade --force
```

### 升级说明

- 升级时会自动备份当前安装
- 用户配置（settings.json 和自定义 prompts）会被保留
- 升级失败时会自动回滚到之前的版本

---

## 卸载指南

### 标准卸载（保留配置）

```bash
./bootstrap.sh uninstall
```

标准卸载会：
- 删除程序文件（bin/、lib/、VERSION）
- 保留用户配置（settings.json、自定义 prompts）
- 创建备份以便误操作恢复

### 彻底卸载（删除所有数据）

```bash
./bootstrap.sh uninstall --purge
```

彻底卸载会删除所有 Morty 相关文件，包括配置和数据。

### 强制卸载（跳过确认）

```bash
./bootstrap.sh uninstall --force
```

### 指定路径卸载

如果你之前使用了自定义安装路径：

```bash
./bootstrap.sh uninstall --prefix /opt/morty --bin-dir /usr/local/bin
```

---

## 常见问题排查

### 问题 1：Bash 版本过低

**现象**：
```
✗ Bash version 3.2 is too old (required: >= 4.0)
```

**解决方案**：

```bash
# macOS
brew install bash

# Ubuntu/Debian
sudo apt-get update && sudo apt-get install bash

# CentOS/RHEL
sudo yum install bash

# 安装后重启终端或运行
exec bash
```

### 问题 2：没有 curl 或 wget

**现象**：
```
✗ Neither curl nor wget found
```

**解决方案**：

```bash
# 安装 curl（推荐）
# Ubuntu/Debian
sudo apt-get install curl

# CentOS/RHEL
sudo yum install curl

# macOS
brew install curl

# 或者安装 wget
# Ubuntu/Debian
sudo apt-get install wget

# CentOS/RHEL
sudo yum install wget
```

### 问题 3：权限不足

**现象**：
```
✗ Cannot create target directory: /opt/morty
✗ Target directory exists but is not writable: /opt/morty
```

**解决方案**：

**方案 A**：更改安装路径到用户目录
```bash
./bootstrap.sh install --prefix ~/morty --bin-dir ~/bin
```

**方案 B**：创建目录并设置权限
```bash
sudo mkdir -p /opt/morty
sudo chown $(whoami) /opt/morty
./bootstrap.sh install --prefix /opt/morty
```

**方案 C**：使用 sudo（不推荐用于个人安装）
```bash
sudo ./bootstrap.sh install --prefix /opt/morty
```

### 问题 4：已存在 Morty 安装

**现象**：
```
✗ Morty is already installed at: ~/.morty
Use 'reinstall' to overwrite or 'upgrade' to update
```

**解决方案**：

```bash
# 如果想升级到最新版本
./bootstrap.sh upgrade

# 如果想重新安装（保留配置）
./bootstrap.sh reinstall

# 如果想全新安装（先卸载再安装）
./bootstrap.sh uninstall
./bootstrap.sh install
```

### 问题 5：命令未找到（Command not found）

**现象**：安装成功但运行 `morty` 提示命令未找到。

**解决方案**：

```bash
# 检查 ~/.local/bin 是否在 PATH 中
echo $PATH

# 添加路径到 shell 配置
export PATH="$HOME/.local/bin:$PATH"

# 持久化配置
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc  # Bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc   # Zsh

# 重新加载配置
source ~/.bashrc  # 或 source ~/.zshrc
```

### 问题 6：安装验证失败

**现象**：
```
✗ Installation verification failed
```

**解决方案**：

```bash
# 1. 检查 morty 是否存在
ls -la ~/.morty/bin/morty

# 2. 确保文件可执行
chmod +x ~/.morty/bin/morty

# 3. 直接运行测试
~/.morty/bin/morty --help

# 4. 如果仍然失败，卸载后重试
./bootstrap.sh uninstall
./bootstrap.sh install
```

### 问题 7：GitHub API 限制

**现象**：升级时无法获取最新版本信息。

**解决方案**：

```bash
# 手动指定版本升级
./bootstrap.sh upgrade --version 2.1.0

# 或者手动下载安装
./bootstrap.sh reinstall --source ./
```

### 问题 8：磁盘空间不足

**现象**：
```
✗ Insufficient disk space: 20MB available, 50MB required
```

**解决方案**：

清理磁盘空间，或选择其他有足够空间的安装路径：

```bash
# 查看磁盘使用情况
df -h

# 安装到其他分区
./bootstrap.sh install --prefix /path/with/space/morty
```

---

## 安装目录结构

安装完成后，目录结构如下：

```
~/.morty/                    # 安装目录
├── bin/                     # 可执行脚本
│   ├── morty               # 主命令
│   ├── morty_doing.sh
│   ├── morty_plan.sh
│   ├── morty_research.sh
│   ├── morty_reset.sh
│   └── lib -> ../lib       # 库文件链接
├── lib/                     # 库文件
│   ├── cli_utils.sh
│   ├── config.sh
│   ├── logging.sh
│   └── version_manager.sh
├── prompts/                 # 提示词文件
│   ├── doing.md
│   └── plan.md
└── VERSION                  # 版本文件

~/.local/bin/morty -> ~/.morty/bin/morty  # 符号链接
```

---

## 获取更多帮助

- 查看帮助信息：`./bootstrap.sh --help`
- 查看 Morty 帮助：`morty --help`
- 提交 Issue：https://github.com/anthropics/morty/issues

---

**注意**：请确保使用与你 Morty 版本匹配的 bootstrap.sh 脚本。不同版本之间可能存在兼容性问题。
