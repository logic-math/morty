#!/bin/bash
#
# Morty Install Script
# 安装 Morty 到 ~/.morty/ 目录并配置环境
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
INSTALL_DIR="${HOME}/.morty"
BIN_DIR="${INSTALL_DIR}/bin"
FORCE=false
FROM_DIST=""

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助
show_help() {
    cat << EOF
Morty Install Script

Usage: $0 [options]

Options:
    -h, --help          Show this help message
    -f, --force         Force reinstall (overwrite existing)
    --from-dist PATH    Install from pre-compiled binary
    --prefix PATH       Installation prefix (default: ~/.morty)

Examples:
    $0                          # Install from source
    $0 --force                  # Force reinstall
    $0 --from-dist ./dist/morty # Install from pre-compiled binary
EOF
}

# 解析参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            --from-dist)
                FROM_DIST="$2"
                shift 2
                ;;
            --prefix)
                INSTALL_DIR="$2"
                BIN_DIR="${INSTALL_DIR}/bin"
                shift 2
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 检查是否已安装
check_existing() {
    if [ -f "${BIN_DIR}/morty" ] && [ "$FORCE" = false ]; then
        print_warning "Morty is already installed at ${BIN_DIR}/morty"
        print_info "Use --force to reinstall"

        # 显示当前版本
        if "${BIN_DIR}/morty" -version 2>/dev/null; then
            : # 版本信息已输出
        fi

        read -p "Do you want to reinstall? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Installation cancelled"
            exit 0
        fi
        FORCE=true
    fi
}

# 创建目录结构
create_directories() {
    print_info "Creating directory structure..."
    mkdir -p "${BIN_DIR}"
    print_success "Directories created at ${INSTALL_DIR}"
}

# 复制/编译二进制文件
install_binary() {
    print_info "Installing Morty binary..."

    if [ -n "$FROM_DIST" ]; then
        # 从预编译包安装
        if [ ! -f "$FROM_DIST" ]; then
            print_error "Binary not found: $FROM_DIST"
            exit 1
        fi
        cp "$FROM_DIST" "${BIN_DIR}/morty"
        print_success "Binary copied from ${FROM_DIST}"
    elif [ -f "./bin/morty" ]; then
        # 从项目 bin/ 目录复制
        cp "./bin/morty" "${BIN_DIR}/morty"
        print_success "Binary copied from ./bin/morty"
    elif [ -f "./scripts/build.sh" ]; then
        # 现场编译
        print_info "Compiling from source..."
        ./scripts/build.sh --output "${BIN_DIR}/morty"
        print_success "Binary compiled successfully"
    else
        print_error "No binary found and no build script available"
        exit 1
    fi

    chmod +x "${BIN_DIR}/morty"
}

# 创建默认配置
create_config() {
    print_info "Creating default configuration..."

    cat > "${INSTALL_DIR}/config.json" << 'EOF'
{
  "version": "2.0",
  "ai_cli": {
    "command": "claude",
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
EOF

    print_success "Configuration created at ${INSTALL_DIR}/config.json"
}

# 配置 PATH
configure_path() {
    print_info "Configuring PATH..."

    local path_line='export PATH="${HOME}/.morty/bin:${PATH}"'
    local shell_configs=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile")

    for config in "${shell_configs[@]}"; do
        if [ -f "$config" ]; then
            if ! grep -q "\.morty/bin" "$config"; then
                echo "" >> "$config"
                echo "# Morty CLI" >> "$config"
                echo "$path_line" >> "$config"
                print_success "PATH configured in ${config}"
            else
                print_info "PATH already configured in ${config}"
            fi
        fi
    done

    # 如果没有找到配置文件，创建 .bashrc
    if [ ! -f "$HOME/.bashrc" ] && [ ! -f "$HOME/.zshrc" ]; then
        echo "# Morty CLI" > "$HOME/.bashrc"
        echo "$path_line" >> "$HOME/.bashrc"
        print_success "Created ~/.bashrc with PATH configuration"
    fi
}

# 验证安装
verify_installation() {
    print_info "Verifying installation..."

    if [ ! -f "${BIN_DIR}/morty" ]; then
        print_error "Binary not found at ${BIN_DIR}/morty"
        exit 1
    fi

    if ! "${BIN_DIR}/morty" -version > /dev/null 2>&1; then
        print_error "Binary is not executable"
        exit 1
    fi

    print_success "Installation verified"
    echo
    print_info "Version information:"
    "${BIN_DIR}/morty" -version
}

# 显示安装成功信息
show_success_message() {
    echo
    echo "========================================"
    print_success "Morty installed successfully!"
    echo "========================================"
    echo
    echo "Installation directory: ${INSTALL_DIR}"
    echo "Binary location: ${BIN_DIR}/morty"
    echo "Configuration: ${INSTALL_DIR}/config.json"
    echo
    echo "Usage:"
    echo "  morty -version     Show version"
    echo "  morty -help        Show help"
    echo "  morty research     Research mode"
    echo "  morty plan         Plan mode"
    echo "  morty doing        Doing mode"
    echo
    echo "Note: Please restart your shell or run:"
    echo "  source ~/.bashrc"
    echo "  # or"
    echo "  source ~/.zshrc"
    echo
}

# 主函数
main() {
    echo "========================================"
    echo "  Morty Installer"
    echo "========================================"
    echo

    parse_args "$@"
    check_existing
    create_directories
    install_binary
    create_config
    configure_path
    verify_installation
    show_success_message
}

# 执行主函数
main "$@"
