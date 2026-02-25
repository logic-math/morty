#!/bin/bash
#
# upgrade.sh - Morty Upgrade Script
# 检查版本更新并升级到最新版本
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# 默认配置
INSTALL_DIR="${HOME}/.morty"
BIN_DIR="${INSTALL_DIR}/bin"
BACKUP_DIR="${INSTALL_DIR}/backups"
CHECK_ONLY=false
TARGET_VERSION=""
OFFLINE_BINARY=""
FORCE=false

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

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# 显示帮助
show_help() {
    cat << EOF
Morty Upgrade Script

Usage: $0 [options]

Options:
    -h, --help          Show this help message
    -c, --check-only    Only check for updates, don't upgrade
    -v, --version VER   Upgrade to specific version
    -o, --offline PATH  Upgrade from local binary (offline mode)
    -f, --force         Force upgrade even if versions match
    --prefix PATH       Installation prefix (default: ~/.morty)

Examples:
    $0                          # Check and upgrade to latest
    $0 --check-only             # Only check for updates
    $0 --version 2.1.0          # Upgrade to specific version
    $0 --offline ./dist/morty   # Upgrade from local binary
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
            -c|--check-only)
                CHECK_ONLY=true
                shift
                ;;
            -v|--version)
                TARGET_VERSION="$2"
                shift 2
                ;;
            -o|--offline)
                OFFLINE_BINARY="$2"
                shift 2
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            --prefix)
                INSTALL_DIR="$2"
                BIN_DIR="${INSTALL_DIR}/bin"
                BACKUP_DIR="${INSTALL_DIR}/backups"
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

# ==================== Task 1: 检测当前版本 ====================
task1_detect_current_version() {
    print_step "Task 1: Detecting current version..."

    if [ ! -f "${BIN_DIR}/morty" ]; then
        print_error "Morty is not installed at ${BIN_DIR}/morty"
        print_info "Please run install.sh first"
        exit 1
    fi

    # Get current version info
    CURRENT_VERSION_INFO=$("${BIN_DIR}/morty" -version 2>&1)
    if [ $? -ne 0 ]; then
        print_error "Failed to get current version"
        exit 1
    fi

    # Extract version from JSON output
    CURRENT_VERSION=$(echo "$CURRENT_VERSION_INFO" | grep -o '"version": "[^"]*"' | cut -d'"' -f4)
    CURRENT_GIT_COMMIT=$(echo "$CURRENT_VERSION_INFO" | grep -o '"git_commit": "[^"]*"' | cut -d'"' -f4)
    CURRENT_BUILD_TIME=$(echo "$CURRENT_VERSION_INFO" | grep -o '"build_time": "[^"]*"' | cut -d'"' -f4)

    if [ -z "$CURRENT_VERSION" ]; then
        CURRENT_VERSION="unknown"
    fi

    print_success "Current version detected: $CURRENT_VERSION"
    print_info "  Git Commit: ${CURRENT_GIT_COMMIT:-unknown}"
    print_info "  Build Time: ${CURRENT_BUILD_TIME:-unknown}"
}

# ==================== Task 2: 获取最新版本信息 ====================
task2_get_latest_version() {
    print_step "Task 2: Getting latest version information..."

    if [ -n "$OFFLINE_BINARY" ]; then
        # Offline mode - get version from binary
        if [ ! -f "$OFFLINE_BINARY" ]; then
            print_error "Offline binary not found: $OFFLINE_BINARY"
            exit 1
        fi

        # Make temporary copy to check version (avoid modifying original)
        TMP_BINARY=$(mktemp)
        cp "$OFFLINE_BINARY" "$TMP_BINARY"
        chmod +x "$TMP_BINARY"

        OFFLINE_VERSION_INFO=$($TMP_BINARY -version 2>&1)
        rm -f "$TMP_BINARY"

        LATEST_VERSION=$(echo "$OFFLINE_VERSION_INFO" | grep -o '"version": "[^"]*"' | cut -d'"' -f4)
        LATEST_GIT_COMMIT=$(echo "$OFFLINE_VERSION_INFO" | grep -o '"git_commit": "[^"]*"' | cut -d'"' -f4)

        print_success "Offline binary version: $LATEST_VERSION"
        return
    fi

    if [ -n "$TARGET_VERSION" ]; then
        # Specific version requested
        LATEST_VERSION="$TARGET_VERSION"
        print_info "Target version specified: $LATEST_VERSION"
    else
        # Try to get latest version from git tags
        LATEST_VERSION=$(get_latest_version_from_git)

        if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "dev" ]; then
            # Try to get from local build
            if [ -f "./bin/morty" ]; then
                LOCAL_VERSION_INFO=$(./bin/morty -version 2>&1)
                LATEST_VERSION=$(echo "$LOCAL_VERSION_INFO" | grep -o '"version": "[^"]*"' | cut -d'"' -f4)
            fi
        fi

        if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "dev" ]; then
            LATEST_VERSION="latest"
        fi

        print_success "Latest version: $LATEST_VERSION"
    fi
}

# Helper: Get latest version from git tags
get_latest_version_from_git() {
    if ! command -v git &> /dev/null; then
        return
    fi

    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        return
    fi

    # Get latest tag
    local latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

    if [ -n "$latest_tag" ]; then
        # Remove 'v' prefix if present
        echo "$latest_tag" | sed 's/^v//'
    fi
}

# ==================== Task 3: 版本对比，判断是否需要升级 ====================
task3_compare_versions() {
    print_step "Task 3: Comparing versions..."

    # If force flag is set, always upgrade
    if [ "$FORCE" = true ]; then
        print_warning "Force flag set, will upgrade regardless of version"
        NEED_UPGRADE=true
        return
    fi

    # If target version is specified, always upgrade to it
    if [ -n "$TARGET_VERSION" ]; then
        print_info "Target version specified, will upgrade"
        NEED_UPGRADE=true
        return
    fi

    # If offline binary specified, check if it's different
    if [ -n "$OFFLINE_BINARY" ]; then
        if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ] && [ "$CURRENT_VERSION" != "dev" ]; then
            # Compare git commits if versions are same
            if [ "$CURRENT_GIT_COMMIT" = "$LATEST_GIT_COMMIT" ] && [ "$CURRENT_GIT_COMMIT" != "unknown" ]; then
                print_info "Current version ($CURRENT_VERSION, $CURRENT_GIT_COMMIT) is same as offline binary"
                NEED_UPGRADE=false
            else
                print_info "Same version but different commit, will upgrade"
                NEED_UPGRADE=true
            fi
        else
            NEED_UPGRADE=true
        fi
        return
    fi

    # Compare versions
    if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ] && [ "$CURRENT_VERSION" != "dev" ] && [ "$CURRENT_VERSION" != "unknown" ]; then
        print_success "Already at latest version ($CURRENT_VERSION)"
        NEED_UPGRADE=false
    else
        print_info "Upgrade needed: $CURRENT_VERSION -> $LATEST_VERSION"
        NEED_UPGRADE=true
    fi
}

# ==================== Task 4: 备份当前版本二进制和配置 ====================
task4_backup_current() {
    print_step "Task 4: Backing up current version..."

    # Create backup directory with timestamp
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_PATH="${BACKUP_DIR}/${BACKUP_TIMESTAMP}"
    mkdir -p "$BACKUP_PATH"

    # Backup binary
    if [ -f "${BIN_DIR}/morty" ]; then
        cp "${BIN_DIR}/morty" "${BACKUP_PATH}/morty"
        print_success "Binary backed up to ${BACKUP_PATH}/morty"
    fi

    # Backup config
    if [ -f "${INSTALL_DIR}/config.json" ]; then
        cp "${INSTALL_DIR}/config.json" "${BACKUP_PATH}/config.json"
        print_success "Config backed up to ${BACKUP_PATH}/config.json"
    fi

    # Save version info
    echo "$CURRENT_VERSION_INFO" > "${BACKUP_PATH}/version.json"
    print_success "Version info saved to ${BACKUP_PATH}/version.json"

    # Keep only last 5 backups
    cleanup_old_backups
}

# Helper: Clean up old backups (keep only last 5)
cleanup_old_backups() {
    if [ -d "$BACKUP_DIR" ]; then
        local backup_count=$(ls -1 "$BACKUP_DIR" | wc -l)
        if [ "$backup_count" -gt 5 ]; then
            ls -1t "$BACKUP_DIR" | tail -n +6 | while read -r old_backup; do
                rm -rf "${BACKUP_DIR}/${old_backup}"
            done
            print_info "Cleaned up old backups (keeping last 5)"
        fi
    fi
}

# ==================== Task 5: 下载/编译新版本 ====================
task5_prepare_new_version() {
    print_step "Task 5: Preparing new version..."

    # Create temp directory for new binary
    TEMP_DIR=$(mktemp -d)
    NEW_BINARY="${TEMP_DIR}/morty"

    if [ -n "$OFFLINE_BINARY" ]; then
        # Use offline binary
        print_info "Using offline binary: $OFFLINE_BINARY"
        cp "$OFFLINE_BINARY" "$NEW_BINARY"
        chmod +x "$NEW_BINARY"
    elif [ -f "./bin/morty" ] && [ -z "$TARGET_VERSION" ]; then
        # Use local build
        print_info "Using local build: ./bin/morty"
        cp "./bin/morty" "$NEW_BINARY"
        chmod +x "$NEW_BINARY"
    elif [ -f "./scripts/build.sh" ]; then
        # Build from source
        print_info "Building from source..."
        if [ -n "$TARGET_VERSION" ]; then
            ./scripts/build.sh --output "$NEW_BINARY" --version "$TARGET_VERSION"
        else
            ./scripts/build.sh --output "$NEW_BINARY"
        fi
    else
        print_error "No binary available and no build script found"
        print_info "Please provide: --offline PATH, or ensure ./bin/morty exists, or run from source directory"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    # Verify new binary
    if [ ! -f "$NEW_BINARY" ]; then
        print_error "New binary not found after preparation"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    if ! chmod +x "$NEW_BINARY" 2>/dev/null; then
        print_error "Cannot make new binary executable"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    # Verify new binary works
    if ! "$NEW_BINARY" -version > /dev/null 2>&1; then
        print_error "New binary is not working properly"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    NEW_VERSION_INFO=$($NEW_BINARY -version 2>&1)
    NEW_VERSION=$(echo "$NEW_VERSION_INFO" | grep -o '"version": "[^"]*"' | cut -d'"' -f4)

    print_success "New version prepared: ${NEW_VERSION:-unknown}"
}

# ==================== Task 6: 安装新版本 ====================
task6_install_new_version() {
    print_step "Task 6: Installing new version..."

    # Remove old binary
    if [ -f "${BIN_DIR}/morty" ]; then
        rm -f "${BIN_DIR}/morty"
    fi

    # Install new binary
    cp "$NEW_BINARY" "${BIN_DIR}/morty"
    chmod +x "${BIN_DIR}/morty"

    # Clean up temp directory
    rm -rf "$TEMP_DIR"

    print_success "New version installed to ${BIN_DIR}/morty"
}

# ==================== Task 7: 验证升级成功 ====================
task7_verify_upgrade() {
    print_step "Task 7: Verifying upgrade..."

    # Check binary exists
    if [ ! -f "${BIN_DIR}/morty" ]; then
        print_error "Binary not found after installation"
        return 1
    fi

    # Check binary is executable
    if ! "${BIN_DIR}/morty" -version > /dev/null 2>&1; then
        print_error "Binary is not executable or not working"
        return 1
    fi

    # Get new version info
    INSTALLED_VERSION_INFO=$("${BIN_DIR}/morty" -version 2>&1)
    INSTALLED_VERSION=$(echo "$INSTALLED_VERSION_INFO" | grep -o '"version": "[^"]*"' | cut -d'"' -f4)

    print_success "Upgrade verified successfully"
    print_info "Installed version: ${INSTALLED_VERSION:-unknown}"

    return 0
}

# ==================== Task 8: 支持回滚 ====================
rollback_upgrade() {
    print_step "Task 8: Rolling back to previous version..."

    if [ -z "$BACKUP_PATH" ] || [ ! -d "$BACKUP_PATH" ]; then
        print_error "No backup available for rollback"
        return 1
    fi

    # Restore binary
    if [ -f "${BACKUP_PATH}/morty" ]; then
        cp "${BACKUP_PATH}/morty" "${BIN_DIR}/morty"
        chmod +x "${BIN_DIR}/morty"
        print_success "Binary restored from backup"
    fi

    # Restore config
    if [ -f "${BACKUP_PATH}/config.json" ]; then
        cp "${BACKUP_PATH}/config.json" "${INSTALL_DIR}/config.json"
        print_success "Config restored from backup"
    fi

    # Verify rollback
    if "${BIN_DIR}/morty" -version > /dev/null 2>&1; then
        print_success "Rollback completed successfully"
        ROLLED_BACK=true
        return 0
    else
        print_error "Rollback verification failed"
        return 1
    fi
}

# 显示升级结果
show_upgrade_result() {
    echo
    echo "========================================"

    if [ "${ROLLED_BACK:-false}" = true ]; then
        print_error "Upgrade failed - Rolled back to previous version"
        echo "========================================"
        echo "Previous Version: $CURRENT_VERSION"
        echo "Backup Location: $BACKUP_PATH"
    else
        print_success "Upgrade completed successfully!"
        echo "========================================"
        echo "Previous Version: $CURRENT_VERSION"
        echo "New Version: ${INSTALLED_VERSION:-$NEW_VERSION}"
        echo
        echo "Version Details:"
        "${BIN_DIR}/morty" -version
    fi

    echo
    echo "Backup Location: $BACKUP_PATH"
    echo
    echo "To rollback manually, run:"
    echo "  cp ${BACKUP_PATH}/morty ${BIN_DIR}/morty"
    echo "========================================"
}

# 仅检查模式
run_check_only() {
    echo "========================================"
    echo "  Morty Upgrade Checker"
    echo "========================================"
    echo

    task1_detect_current_version
    task2_get_latest_version
    task3_compare_versions

    echo
    echo "========================================"
    echo "Version Check Result"
    echo "========================================"
    echo "Current Version: $CURRENT_VERSION"
    echo "Latest Version:  $LATEST_VERSION"
    echo

    if [ "$NEED_UPGRADE" = true ]; then
        print_warning "Update available!"
        echo "Run './scripts/upgrade.sh' to upgrade"
    else
        print_success "You are up to date!"
    fi
    echo "========================================"
}

# 主流程
main() {
    echo "========================================"
    echo "  Morty Upgrade Script"
    echo "========================================"
    echo

    parse_args "$@"

    # Check-only mode
    if [ "$CHECK_ONLY" = true ]; then
        run_check_only
        exit 0
    fi

    # Normal upgrade flow
    task1_detect_current_version
    task2_get_latest_version
    task3_compare_versions

    # Check if upgrade is needed
    if [ "$NEED_UPGRADE" = false ]; then
        echo
        print_success "No upgrade needed. Current version is up to date."
        exit 0
    fi

    # Confirm upgrade
    echo
    print_info "Ready to upgrade: $CURRENT_VERSION -> $LATEST_VERSION"
    if [ "$FORCE" != true ]; then
        read -p "Do you want to proceed? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Upgrade cancelled"
            exit 0
        fi
    fi

    # Execute upgrade
    ROLLED_BACK=false
    UPGRADE_FAILED=false

    task4_backup_current
    task5_prepare_new_version
    task6_install_new_version

    # Verify and rollback if needed
    if ! task7_verify_upgrade; then
        UPGRADE_FAILED=true
        print_error "Upgrade verification failed, rolling back..."
        if ! rollback_upgrade; then
            print_error "Rollback also failed! Manual intervention required."
            print_info "Backup location: $BACKUP_PATH"
            exit 1
        fi
    fi

    # Show result
    show_upgrade_result

    if [ "$UPGRADE_FAILED" = true ]; then
        exit 1
    fi

    exit 0
}

# 执行主函数
main "$@"
