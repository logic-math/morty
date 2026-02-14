#!/bin/bash
# Morty 安装脚本

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    local level=$1
    shift
    local message="$*"
    local color=""

    case $level in
        INFO)  color=$BLUE ;;
        WARN)  color=$YELLOW ;;
        ERROR) color=$RED ;;
        SUCCESS) color=$GREEN ;;
    esac

    echo -e "${color}[$level] $message${NC}"
}

# Installation paths
INSTALL_DIR="$HOME/.morty"
BIN_DIR="$HOME/.local/bin"

log INFO "安装 Morty..."
log INFO "安装目录: $INSTALL_DIR"
log INFO "命令目录: $BIN_DIR"

# Create directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$BIN_DIR"

# Copy files
log INFO "复制文件..."

# Copy main scripts
cp morty_fix.sh "$INSTALL_DIR/"
cp morty_loop.sh "$INSTALL_DIR/"

# Copy library and prompts
cp -r lib "$INSTALL_DIR/"
cp -r prompts "$INSTALL_DIR/"

# Make scripts executable
chmod +x "$INSTALL_DIR"/*.sh

# Create main morty command
log INFO "创建 morty 命令..."

cat > "$BIN_DIR/morty" << 'EOF'
#!/bin/bash
# Morty - 简化的 AI 开发循环

VERSION="0.3.0"
MORTY_HOME="${MORTY_HOME:-$HOME/.morty}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_help() {
    cat << 'HELP'
Morty - 简化的 AI 开发循环

用法: morty <command> [options]

命令:
    fix <prd.md>            迭代式 PRD 改进(问题修复/功能增强/架构优化)
    loop                    启动开发循环
    version                 显示版本

示例:
    morty fix prd.md                   # 改进 PRD 并生成 .morty/ 目录
    morty fix docs/requirements.md     # 指定 PRD 文件路径
    morty loop                         # 启动开发循环
    morty loop --max-loops 100         # 自定义最大循环次数

工作流程:
    1. morty fix <prd.md>              # 迭代式 PRD 改进
    2. 查看生成的 .morty/specs/*.md    # 检查模块规范
    3. morty loop                      # 启动开发循环

HELP
}

show_version() {
    echo "Morty version $VERSION"
}

# Command routing
case "${1:-}" in
    fix)
        shift
        exec "$MORTY_HOME/morty_fix.sh" "$@"
        ;;
    loop)
        shift
        exec "$MORTY_HOME/morty_loop.sh" "$@"
        ;;
    version|--version|-v)
        show_version
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo -e "${RED}错误: 未知命令 '$1'${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
EOF

chmod +x "$BIN_DIR/morty"

log SUCCESS "安装完成!"
log INFO ""
log INFO "Morty 已安装到: $INSTALL_DIR"
log INFO "命令已安装到: $BIN_DIR/morty"
log INFO ""

# Check if BIN_DIR is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    log WARN "$BIN_DIR 不在你的 PATH 中"
    log INFO "添加这一行到你的 ~/.bashrc 或 ~/.zshrc:"
    log INFO "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    log INFO ""
fi

log INFO "快速开始:"
log INFO "  morty fix prd.md  # 改进 PRD 并生成 .morty/ 目录"
log INFO "  morty loop        # 启动开发循环"
log INFO ""
log SUCCESS "使用 Morty 愉快编码! 🚀"
