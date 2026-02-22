# Morty 安装指南

## 系统要求

### 必需依赖

- **Bash** >= 4.0
- **Git** >= 2.0
- **Claude Code CLI** (或其他 AI CLI)

### 可选依赖

- **jq**: JSON 处理增强
- **tmux**: Loop 监控模式

## 安装方式

### 方式 1: 一键安装 (推荐)

```bash
curl -sSL https://raw.githubusercontent.com/anthropics/morty/main/bootstrap.sh | bash
```

或者使用 wget:

```bash
wget -qO- https://raw.githubusercontent.com/anthropics/morty/main/bootstrap.sh | bash
```

### 方式 2: 下载后执行

```bash
# 下载安装脚本
wget https://raw.githubusercontent.com/anthropics/morty/main/bootstrap.sh
chmod +x bootstrap.sh

# 执行安装
./bootstrap.sh
```

### 方式 3: 从本地源码安装 (开发模式)

```bash
git clone https://github.com/anthropics/morty.git
cd morty
./bootstrap.sh --source ./
```

### 方式 4: 安装指定版本

```bash
curl -sSL https://raw.githubusercontent.com/anthropics/morty/main/bootstrap.sh | bash -s -- --version 2.1.0
```

### 方式 5: 自定义安装路径

```bash
./bootstrap.sh --prefix /opt/morty --bin-dir /usr/local/bin
```

## 手动安装

如果你想要完全手动控制安装过程，可以按以下步骤操作:

### 1. 下载 Release 包

```bash
# 获取最新版本
VERSION=$(curl -s https://api.github.com/repos/anthropics/morty/releases/latest | grep tag_name | cut -d '"' -f 4)

# 下载 release 包
wget "https://github.com/anthropics/morty/archive/refs/tags/${VERSION}.tar.gz"

# 解压
tar -xzf "${VERSION}.tar.gz"
cd "morty-${VERSION#v}"
```

### 2. 创建目录结构

```bash
MORTY_HOME="${HOME}/.morty"
mkdir -p "$MORTY_HOME"/{bin,lib,prompts}
```

### 3. 复制文件

```bash
# 复制可执行脚本
cp morty_*.sh "$MORTY_HOME/bin/"

# 复制库文件
cp lib/*.sh "$MORTY_HOME/lib/"

# 复制提示词文件
cp prompts/*.md "$MORTY_HOME/prompts/"

# 复制版本文件
cp VERSION "$MORTY_HOME/"
```

### 4. 设置权限

```bash
chmod +x "$MORTY_HOME/bin"/*.sh
```

### 5. 创建主命令

创建 `$MORTY_HOME/bin/morty` 文件，内容参考源码中的 `morty` 命令脚本。

### 6. 创建符号链接

```bash
BIN_DIR="${HOME}/.local/bin"
mkdir -p "$BIN_DIR"
ln -s "$MORTY_HOME/bin/morty" "$BIN_DIR/morty"
```

### 7. 配置 PATH

确保 `$HOME/.local/bin` 在你的 PATH 中:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### 8. 初始化配置

```bash
morty version
```

## 升级

### 检查更新

```bash
morty upgrade --check
```

### 升级到最新版本

```bash
morty upgrade
```

### 从源码重新安装

```bash
./bootstrap.sh --force
```

## 卸载

### 标准卸载 (保留配置)

```bash
morty uninstall
```

### 彻底卸载 (删除所有数据)

```bash
morty uninstall --purge
```

### 手动卸载

```bash
# 删除安装目录
rm -rf ~/.morty

# 删除符号链接
rm -f ~/.local/bin/morty

# 删除配置文件 (可选)
rm -f ~/.mortyrc
```

## 故障排除

### 依赖检查失败

```bash
# 检查 Bash 版本
bash --version

# 检查 Git 版本
git --version

# 检查 Claude Code
claude --version
```

### 权限问题

如果安装目录需要 root 权限:

```bash
sudo ./bootstrap.sh --prefix /usr/local/morty --bin-dir /usr/local/bin
```

### PATH 问题

如果运行 `morty` 提示命令未找到:

```bash
# 检查符号链接是否存在
ls -la ~/.local/bin/morty

# 手动添加 PATH
export PATH="$HOME/.local/bin:$PATH"

# 永久添加到 shell 配置
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

## 安装路径

默认安装路径:

- **安装目录**: `~/.morty/`
  - `bin/`: 可执行脚本
  - `lib/`: 库文件
  - `prompts/`: 提示词文件
  - `VERSION`: 版本文件
  - `.morty/`: 运行时数据

- **命令路径**: `~/.local/bin/morty` (符号链接)

## 环境变量

- `MORTY_HOME`: Morty 安装目录 (默认: `~/.morty`)
- `MORTY_AI_CLI`: AI CLI 命令 (默认: `claude`)

## 验证安装

```bash
# 检查版本
morty version

# 显示详细信息
morty version --verbose

# 查看帮助
morty --help
```
