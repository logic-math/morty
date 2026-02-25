#!/bin/bash
#
# Morty Uninstall Script
# 从系统中卸载 Morty
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
BACKUP_DIR=""
SKIP_BACKUP=false

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
Morty Uninstall Script

Usage: $0 [options]

Options:
    -h, --help          Show this help message
    --skip-backup       Skip backup step
    --backup-dir PATH   Custom backup directory

Examples:
    $0                          # Uninstall with backup prompt
    $0 --skip-backup            # Uninstall without backup
    $0 --backup-dir /tmp/bak    # Backup to custom directory
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
            --skip-backup)
                SKIP_BACKUP=true
                shift
                ;;
            --backup-dir)
                BACKUP_DIR="$2"
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

# 检查安装状态
check_installation() {
    if [ ! -d "${INSTALL_DIR}" ]; then
        print_warning "Morty is not installed at ${INSTALL_DIR}"
        exit 0
    fi
    print_info "Found Morty installation at ${INSTALL_DIR}"
}

# 备份配置
backup_config() {
    if [ "$SKIP_BACKUP" = true ]; then
        print_info "Skipping backup as requested"
        return
    fi

    if [ -z "$BACKUP_DIR" ]; then
        BACKUP_DIR="/tmp/morty_backup_$(date +%Y%m%d_%H%M%S)"
    fi

    print_info "Backing up configuration to ${BACKUP_DIR}..."
    mkdir -p "${BACKUP_DIR}"
    cp -r "${INSTALL_DIR}"/* "${BACKUP_DIR}/" 2>/dev/null || true
    print_success "Configuration backed up to ${BACKUP_DIR}"
}

# 清理 shell 配置文件
clean_shell_configs() {
    print_info "Cleaning shell configuration files..."

    local shell_configs=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.profile")

    for config in "${shell_configs[@]}"; do
        if [ -f "$config" ]; then
            # 删除 Morty PATH 配置（包括注释）
            if grep -q "morty" "$config" 2>/dev/null; then
                # 创建临时文件
                local tmp_file="${config}.tmp"

                # 删除 Morty 相关行（包括空行和注释）
                awk '
                    /^# Morty CLI$/ { skip=1; next }
                    skip && /^export PATH="\${HOME}\/\.morty\/bin:\${PATH}"$/ { skip=0; next }
                    skip { next }
                    { print }
                ' "$config" > "$tmp_file"

                # 如果文件有变化则替换
                if [ -s "$tmp_file" ]; then
                    mv "$tmp_file" "$config"
                    print_success "Cleaned ${config}"
                else
                    rm -f "$tmp_file"
                fi
            fi
        fi
    done
}

# 删除安装目录
remove_installation() {
    print_info "Removing Morty installation directory..."
    if [ -d "${INSTALL_DIR}" ]; then
        rm -rf "${INSTALL_DIR}"
        print_success "Removed ${INSTALL_DIR}"
    fi
}

# 验证卸载
verify_uninstall() {
    print_info "Verifying uninstallation..."

    local verify_passed=true

    # 检查目录是否删除
    if [ -d "${INSTALL_DIR}" ]; then
        print_error "Directory ${INSTALL_DIR} still exists"
        verify_passed=false
    else
        print_success "Installation directory removed"
    fi

    # 检查 shell 配置
    local shell_configs=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile")
    local configs_cleaned=true
    for config in "${shell_configs[@]}"; do
        if [ -f "$config" ] && grep -q "morty" "$config" 2>/dev/null; then
            print_error "Morty references still in ${config}"
            configs_cleaned=false
            verify_passed=false
        fi
    done

    if [ "$configs_cleaned" = true ]; then
        print_success "Shell configurations cleaned"
    fi

    if [ "$verify_passed" = true ]; then
        print_success "Uninstallation verified successfully"
        return 0
    else
        return 1
    fi
}

# 显示卸载成功信息
show_success_message() {
    echo
    echo "========================================"
    print_success "Morty uninstalled successfully!"
    echo "========================================"
    echo
    if [ "$SKIP_BACKUP" = false ] && [ -n "$BACKUP_DIR" ]; then
        echo "Backup location: ${BACKUP_DIR}"
    fi
    echo
    echo "Note: Please restart your shell or run:"
    echo "  source ~/.bashrc"
    echo "  # or"
    echo "  source ~/.zshrc"
    echo
    echo "To reinstall Morty, run:"
    echo "  ./scripts/install.sh"
    echo
}

# 主函数
main() {
    echo "========================================"
    echo "  Morty Uninstaller"
    echo "========================================"
    echo

    parse_args "$@"
    check_installation
    backup_config
    clean_shell_configs
    remove_installation
    verify_uninstall
    show_success_message
}

# 执行主函数
main "$@"
