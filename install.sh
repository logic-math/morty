#!/bin/bash
# Morty Installation Script

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

log INFO "Installing Morty..."
log INFO "Installation directory: $INSTALL_DIR"
log INFO "Binary directory: $BIN_DIR"

# Create directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$BIN_DIR"

# Copy files
log INFO "Copying files..."

# Copy main scripts
cp morty_fix.sh "$INSTALL_DIR/"
cp morty_enable.sh "$INSTALL_DIR/"
cp morty_loop.sh "$INSTALL_DIR/"
cp morty_monitor.sh "$INSTALL_DIR/"

# Copy library and prompts
cp -r lib "$INSTALL_DIR/"
cp -r prompts "$INSTALL_DIR/"

# Make scripts executable
chmod +x "$INSTALL_DIR"/*.sh

# Create main morty command
log INFO "Creating morty command..."

cat > "$BIN_DIR/morty" << 'EOF'
#!/bin/bash
# Morty - Simplified AI Development Loop

VERSION="0.1.0"
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
    enable                  在现有项目中启用 Morty
    start                   启动开发循环
    monitor                 启动并带 tmux 监控
    status                  显示当前状态
    rollback <loop-number>  回滚到特定循环迭代
    history                 显示 git 提交中的循环历史
    version                 显示版本

示例:
    morty fix prd.md                   # 改进 PRD 并更新规范
    morty fix docs/requirements.md     # 指定 PRD 文件路径
    morty enable                       # 在现有项目中启用
    morty start                        # 启动开发循环
    morty monitor                      # 启动并监控
    morty rollback 5                   # 回滚到循环 #5
    morty history                      # 显示循环提交历史

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
    enable)
        shift
        exec "$MORTY_HOME/morty_enable.sh" "$@"
        ;;
    start)
        shift
        exec "$MORTY_HOME/morty_loop.sh" "$@"
        ;;
    monitor)
        shift
        exec "$MORTY_HOME/morty_loop.sh" --monitor "$@"
        ;;
    status)
        shift
        exec "$MORTY_HOME/morty_loop.sh" --status "$@"
        ;;
    rollback)
        shift
        # Source common.sh for git functions
        source "$MORTY_HOME/lib/common.sh"
        if [[ -z "${1:-}" ]]; then
            echo -e "${RED}Error: Loop number required${NC}"
            echo "Usage: morty rollback <loop-number>"
            exit 1
        fi
        git_rollback "$1"
        ;;
    history)
        shift
        # Source common.sh for git functions
        source "$MORTY_HOME/lib/common.sh"
        git_loop_history
        ;;
    version|--version|-v)
        show_version
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo -e "${RED}Error: Unknown command '$1'${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
EOF

chmod +x "$BIN_DIR/morty"

log SUCCESS "Installation complete!"
log INFO ""
log INFO "Morty has been installed to: $INSTALL_DIR"
log INFO "Command installed to: $BIN_DIR/morty"
log INFO ""

# Check if BIN_DIR is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    log WARN "$BIN_DIR is not in your PATH"
    log INFO "Add this line to your ~/.bashrc or ~/.zshrc:"
    log INFO "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    log INFO ""
fi

log INFO "快速开始:"
log INFO "  morty fix prd.md  # 改进 PRD 并更新规范"
log INFO "  morty enable      # 在现有项目中启用"
log INFO ""
log SUCCESS "使用 Morty 愉快编码! 🚀"
