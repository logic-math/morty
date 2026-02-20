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
cp morty_reset.sh "$INSTALL_DIR/"
cp morty_research.sh "$INSTALL_DIR/"
cp morty_plan.sh "$INSTALL_DIR/"
cp morty_doing.sh "$INSTALL_DIR/"

# Copy library and prompts
cp -r lib "$INSTALL_DIR/"
cp -r prompts "$INSTALL_DIR/"

# Make library scripts executable
chmod +x "$INSTALL_DIR/lib"/*.sh

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
    research [topic]        交互式代码库/文档库研究
    plan                    基于研究结果创建 TDD 开发计划
    doing [options]         执行 Plan 的分层 TDD 开发
    fix <prd.md>            迭代式 PRD 改进(问题修复/功能增强/架构优化)
    loop [options]          启动开发循环(集成监控)
    reset [options]         版本回滚和循环管理
    version                 显示版本

示例:
    morty research                     # 启动研究模式
    morty research "api架构"           # 研究指定主题
    morty plan                         # 基于研究结果创建 TDD 计划
    morty doing                        # 执行分层 TDD 开发
    morty fix prd.md                   # 改进 PRD 并生成 .morty/ 目录
    morty loop                         # 启动带监控的开发循环(推荐)
    morty reset -l                     # 查看循环提交历史
    morty reset -c abc123              # 回滚到指定 commit

新工作流程 (research → plan → doing):
    1. morty research [topic]          # 研究代码库/文档库
    2. morty plan                      # 基于研究结果创建 TDD 计划
    3. morty doing                     # 执行分层 TDD 开发
    4. morty reset -l                  # 查看历史
    5. morty reset -c <commit>         # 回滚到指定版本

传统工作流程 (fix → loop):
    1. morty fix <prd.md>              # 迭代式 PRD 改进
    2. morty loop                      # 启动循环(自动启动 tmux 监控)
    3. morty reset -l                  # 查看循环历史
    4. morty reset -c <commit>         # 回滚到指定版本

监控功能:
    默认情况下,loop 会在 tmux 中启动三面板监控:
    - 左侧(50%): 循环实时日志(项目进度)
    - 右上(30%): Claude Code 监控(Token 使用、错误、资源)
    - 右下(70%): 交互式命令行

    便捷命令: status, progress, logs, plan, help
    使用 Ctrl+B D 可以分离会话,循环将在后台继续运行。

HELP
}

show_version() {
    echo "Morty version $VERSION"
}

# Command routing
case "${1:-}" in
    research)
        shift
        exec "$MORTY_HOME/morty_research.sh" "$@"
        ;;
    plan)
        shift
        exec "$MORTY_HOME/morty_plan.sh" "$@"
        ;;
    doing)
        shift
        exec "$MORTY_HOME/morty_doing.sh" "$@"
        ;;
    fix)
        shift
        exec "$MORTY_HOME/morty_fix.sh" "$@"
        ;;
    loop)
        shift
        exec "$MORTY_HOME/morty_loop.sh" "$@"
        ;;
    reset)
        shift
        exec "$MORTY_HOME/morty_reset.sh" "$@"
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
log INFO "  morty research    # 研究代码库/文档库"
log INFO "  morty plan        # 创建 TDD 开发计划"
log INFO "  morty doing       # 执行分层 TDD 开发"
log INFO "  morty fix prd.md  # (传统)改进 PRD 并生成 .morty/ 目录"
log INFO "  morty loop        # (传统)启动开发循环"
log INFO ""
log SUCCESS "使用 Morty 愉快编码! 🚀"
