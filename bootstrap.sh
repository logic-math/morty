#!/usr/bin/env bash
#
# bootstrap.sh - Morty Bootstrap Installation Script
#
# This is a self-contained installation script that can be run without
# Morty being installed. It provides the initial installation mechanism
# for users who don't have Morty yet.
#
# Usage:
#   curl -sSL https://get.morty.dev | bash
#   ./bootstrap.sh [command] [options]
#
# Commands:
#   install       First-time installation (default)
#   reinstall     Reinstall (overwrite existing)
#   upgrade       Upgrade to new version
#   uninstall     Uninstall Morty
#
# Options:
#   --prefix <path>      Installation directory (default: ~/.morty)
#   --bin-dir <path>     Binary directory for symlink (default: ~/.local/bin)
#   --version <version>  Install specific version
#   --force              Force operation without confirmation
#   --purge              Purge all data including configs (uninstall only)
#   --source <path>      Install from local source (development mode)
#   -h, --help           Show this help message

set -e

# ============================================================================
# Global Variables
# ============================================================================

# Script info
readonly BOOTSTRAP_VERSION="2.0.0"
readonly BOOTSTRAP_NAME="Morty Bootstrap"

# Default paths
readonly DEFAULT_PREFIX="${HOME}/.morty"
readonly DEFAULT_BIN_DIR="${HOME}/.local/bin"

# GitHub repository
readonly GITHUB_REPO="anthropics/morty"
readonly GITHUB_API_URL="https://api.github.com/repos/${GITHUB_REPO}"
readonly GITHUB_RAW_URL="https://raw.githubusercontent.com/${GITHUB_REPO}"

# Minimum required versions
readonly MIN_BASH_VERSION="4.0"
readonly MIN_GIT_VERSION="2.0"

# Parsed arguments (set by bootstrap_parse_args)
BOOTSTRAP_COMMAND=""
BOOTSTRAP_PREFIX=""
BOOTSTRAP_BIN_DIR=""
BOOTSTRAP_TARGET_VERSION=""
BOOTSTRAP_FORCE=false
BOOTSTRAP_PURGE=false
BOOTSTRAP_SOURCE=""
BOOTSTRAP_DEBUG=false

# ============================================================================
# Output Functions
# ============================================================================

# Colors for output
if [[ -t 1 ]]; then
    readonly COLOR_RESET='\033[0m'
    readonly COLOR_RED='\033[0;31m'
    readonly COLOR_GREEN='\033[0;32m'
    readonly COLOR_YELLOW='\033[1;33m'
    readonly COLOR_BLUE='\033[0;34m'
else
    readonly COLOR_RESET=''
    readonly COLOR_RED=''
    readonly COLOR_GREEN=''
    readonly COLOR_YELLOW=''
    readonly COLOR_BLUE=''
fi

log_info() {
    echo -e "${COLOR_BLUE}==>${COLOR_RESET} $1"
}

log_success() {
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} $1"
}

log_warn() {
    echo -e "${COLOR_YELLOW}⚠${COLOR_RESET} $1"
}

log_error() {
    echo -e "${COLOR_RED}✗${COLOR_RESET} $1"
}

log_debug() {
    [[ "$BOOTSTRAP_DEBUG" == "true" ]] && echo "[DEBUG] $1"
}

# ============================================================================
# Argument Parsing and Validation
# ============================================================================

# Display help information
# Usage: bootstrap_show_help
bootstrap_show_help() {
    cat << 'HELP'
Morty Bootstrap Installation Script

Usage:
  ./bootstrap.sh [command] [options]
  curl -sSL https://get.morty.dev | bash
  curl -sSL https://get.morty.dev | bash -s -- [command] [options]

Commands:
  install       First-time installation of Morty (default)
  reinstall     Reinstall Morty (overwrite existing installation)
  upgrade       Upgrade Morty to a new version
  uninstall     Uninstall Morty

Options:
  --prefix <path>      Installation directory (default: ~/.morty)
  --bin-dir <path>     Binary directory for symlink (default: ~/.local/bin)
  --version <version>  Install specific version (default: latest)
  --force              Force operation without confirmation
  --purge              Purge all data including configs (uninstall only)
  --source <path>      Install from local source directory (development mode)
  --debug              Enable debug output
  -h, --help           Show this help message

Examples:
  ./bootstrap.sh                          # Default installation
  ./bootstrap.sh install                  # Same as above
  ./bootstrap.sh install --prefix /opt/morty
  ./bootstrap.sh reinstall --force
  ./bootstrap.sh upgrade --version 2.1.0
  ./bootstrap.sh uninstall --purge
  ./bootstrap.sh --help

HELP
}

# Parse command line arguments
# Usage: bootstrap_parse_args "$@"
# Sets global variables: BOOTSTRAP_COMMAND, BOOTSTRAP_PREFIX, etc.
bootstrap_parse_args() {
    # Reset all variables
    BOOTSTRAP_COMMAND=""
    BOOTSTRAP_PREFIX=""
    BOOTSTRAP_BIN_DIR=""
    BOOTSTRAP_TARGET_VERSION=""
    BOOTSTRAP_FORCE=false
    BOOTSTRAP_PURGE=false
    BOOTSTRAP_SOURCE=""
    BOOTSTRAP_DEBUG=false

    # If no arguments, default to install
    if [[ $# -eq 0 ]]; then
        BOOTSTRAP_COMMAND="install"
        return 0
    fi

    # Check if first argument is a command or option
    local first_arg="$1"
    case "$first_arg" in
        install|reinstall|upgrade|uninstall)
            BOOTSTRAP_COMMAND="$first_arg"
            shift
            ;;
        --help|-h|--prefix|--bin-dir|--version|--force|--purge|--source|--debug)
            # First arg is an option, default to install command
            BOOTSTRAP_COMMAND="install"
            ;;
        *)
            # Unknown argument - will be caught in validation
            BOOTSTRAP_COMMAND="$first_arg"
            shift
            ;;
    esac

    # Parse remaining options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prefix)
                if [[ -z "${2:-}" ]] || [[ "${2:0:1}" == "-" ]]; then
                    log_error "--prefix requires a path argument"
                    return 1
                fi
                BOOTSTRAP_PREFIX="$2"
                shift 2
                ;;
            --bin-dir)
                if [[ -z "${2:-}" ]] || [[ "${2:0:1}" == "-" ]]; then
                    log_error "--bin-dir requires a path argument"
                    return 1
                fi
                BOOTSTRAP_BIN_DIR="$2"
                shift 2
                ;;
            --version)
                if [[ -z "${2:-}" ]] || [[ "${2:0:1}" == "-" ]]; then
                    log_error "--version requires a version argument"
                    return 1
                fi
                BOOTSTRAP_TARGET_VERSION="$2"
                shift 2
                ;;
            --source)
                if [[ -z "${2:-}" ]] || [[ "${2:0:1}" == "-" ]]; then
                    log_error "--source requires a path argument"
                    return 1
                fi
                BOOTSTRAP_SOURCE="$2"
                shift 2
                ;;
            --force)
                BOOTSTRAP_FORCE=true
                shift
                ;;
            --purge)
                BOOTSTRAP_PURGE=true
                shift
                ;;
            --debug)
                BOOTSTRAP_DEBUG=true
                shift
                ;;
            --help|-h)
                bootstrap_show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                return 1
                ;;
        esac
    done

    # Set defaults
    BOOTSTRAP_PREFIX="${BOOTSTRAP_PREFIX:-$DEFAULT_PREFIX}"
    BOOTSTRAP_BIN_DIR="${BOOTSTRAP_BIN_DIR:-$DEFAULT_BIN_DIR}"

    log_debug "Parsed command: $BOOTSTRAP_COMMAND"
    log_debug "Parsed prefix: $BOOTSTRAP_PREFIX"
    log_debug "Parsed bin-dir: $BOOTSTRAP_BIN_DIR"
    log_debug "Parsed version: $BOOTSTRAP_TARGET_VERSION"
    log_debug "Parsed source: $BOOTSTRAP_SOURCE"
    log_debug "Parsed force: $BOOTSTRAP_FORCE"
    log_debug "Parsed purge: $BOOTSTRAP_PURGE"

    return 0
}

# Validate argument combinations
# Usage: bootstrap_validate_args
# Returns: 0 if valid, 1 otherwise
bootstrap_validate_args() {
    local valid=true

    # Validate command
    case "$BOOTSTRAP_COMMAND" in
        install|reinstall|upgrade|uninstall)
            # Valid commands
            ;;
        *)
            log_error "Unknown command: $BOOTSTRAP_COMMAND"
            log_error "Valid commands: install, reinstall, upgrade, uninstall"
            valid=false
            ;;
    esac

    # Check for conflicting options
    if [[ "$BOOTSTRAP_PURGE" == "true" ]] && [[ "$BOOTSTRAP_COMMAND" != "uninstall" ]]; then
        log_error "--purge can only be used with 'uninstall' command"
        valid=false
    fi

    # Check for version with reinstall (version should be used with upgrade)
    if [[ -n "$BOOTSTRAP_TARGET_VERSION" ]] && [[ "$BOOTSTRAP_COMMAND" == "reinstall" ]]; then
        log_warn "--version with 'reinstall' will reinstall current version"
        log_warn "Use 'upgrade --version <version>' to install a specific version"
    fi

    # Check source option validity
    if [[ -n "$BOOTSTRAP_SOURCE" ]]; then
        if [[ "$BOOTSTRAP_COMMAND" != "install" ]] && [[ "$BOOTSTRAP_COMMAND" != "reinstall" ]]; then
            log_error "--source can only be used with 'install' or 'reinstall' commands"
            valid=false
        fi

        if [[ ! -d "$BOOTSTRAP_SOURCE" ]]; then
            log_error "Source directory does not exist: $BOOTSTRAP_SOURCE"
            valid=false
        fi
    fi

    # Check version option validity
    if [[ -n "$BOOTSTRAP_TARGET_VERSION" ]] && [[ "$BOOTSTRAP_COMMAND" == "uninstall" ]]; then
        log_error "--version cannot be used with 'uninstall' command"
        valid=false
    fi

    if [[ "$valid" == "false" ]]; then
        echo ""
        bootstrap_show_help
        return 1
    fi

    return 0
}

# ============================================================================
# Main Entry Point
# ============================================================================

# Main function - entry point for bootstrap script
# Usage: bootstrap_main "$@"
bootstrap_main() {
    # Parse arguments
    if ! bootstrap_parse_args "$@"; then
        return 1
    fi

    # Validate arguments (error message and help already shown by validate function)
    if ! bootstrap_validate_args; then
        return 1
    fi

    # Show banner
    echo ""
    echo "============================================"
    echo "  ${BOOTSTRAP_NAME} v${BOOTSTRAP_VERSION}"
    echo "============================================"
    echo ""

    log_info "Command: $BOOTSTRAP_COMMAND"
    log_info "Prefix: $BOOTSTRAP_PREFIX"
    log_info "Bin dir: $BOOTSTRAP_BIN_DIR"
    echo ""

    # Check system dependencies
    if ! bootstrap_check_system_deps; then
        log_error "System dependency check failed"
        return 1
    fi
    echo ""

    # Check installation environment (only for install/reinstall)
    if [[ "$BOOTSTRAP_COMMAND" == "install" ]] || [[ "$BOOTSTRAP_COMMAND" == "reinstall" ]]; then
        if ! bootstrap_check_install_env; then
            log_error "Installation environment check failed"
            return 1
        fi
        echo ""
    fi

    # Dispatch to command handler
    case "$BOOTSTRAP_COMMAND" in
        install)
            bootstrap_cmd_install
            ;;
        reinstall)
            bootstrap_cmd_reinstall
            ;;
        upgrade)
            bootstrap_cmd_upgrade
            ;;
        uninstall)
            bootstrap_cmd_uninstall
            ;;
        *)
            log_error "Unknown command: $BOOTSTRAP_COMMAND"
            return 1
            ;;
    esac
}

# ============================================================================
# Dependency Checking
# ============================================================================

# Check system dependencies (bash, git, curl/wget)
# Usage: bootstrap_check_system_deps
# Returns: 0 if all dependencies satisfied, 1 otherwise
bootstrap_check_system_deps() {
    local all_passed=true

    log_info "Checking system dependencies..."

    # Check Bash version >= 4.0
    local bash_version
    bash_version="${BASH_VERSION%%[^0-9.]*}"
    if ! bootstrap_check_version "$bash_version" "$MIN_BASH_VERSION"; then
        log_error "Bash version $bash_version is too old (required: >= $MIN_BASH_VERSION)"
        bootstrap_show_fix_suggestion "bash" "$MIN_BASH_VERSION"
        all_passed=false
    else
        log_debug "Bash version: $bash_version (OK)"
    fi

    # Check Git version >= 2.0
    if command -v git &> /dev/null; then
        local git_version
        git_version=$(git --version 2>/dev/null | sed -n 's/.*version \([0-9.]*\).*/\1/p')
        if [[ -n "$git_version" ]]; then
            if ! bootstrap_check_version "$git_version" "$MIN_GIT_VERSION"; then
                log_warn "Git version $git_version is old (recommended: >= $MIN_GIT_VERSION)"
                # Git is optional, so we just warn
            else
                log_debug "Git version: $git_version (OK)"
            fi
        fi
    else
        log_warn "Git not found (recommended for some features)"
    fi

    # Check for curl or wget
    if command -v curl &> /dev/null; then
        log_debug "Found curl for downloads"
    elif command -v wget &> /dev/null; then
        log_debug "Found wget for downloads"
    else
        log_error "Neither curl nor wget found"
        log_error "Please install curl or wget to download Morty releases"
        echo ""
        echo "  Ubuntu/Debian: sudo apt-get install curl"
        echo "  CentOS/RHEL:   sudo yum install curl"
        echo "  macOS:         brew install curl"
        echo "  Alpine:        apk add curl"
        echo ""
        all_passed=false
    fi

    if [[ "$all_passed" == "true" ]]; then
        log_success "System dependencies check passed"
        return 0
    else
        return 1
    fi
}

# Check installation environment
# Usage: bootstrap_check_install_env
# Returns: 0 if environment is ready, 1 otherwise
bootstrap_check_install_env() {
    local all_passed=true

    log_info "Checking installation environment..."

    # Check if target directory is writable
    local target_dir
    target_dir="$BOOTSTRAP_PREFIX"
    local parent_dir
    parent_dir=$(dirname "$target_dir")

    # Create parent directory if it doesn't exist
    if [[ ! -d "$parent_dir" ]]; then
        if ! mkdir -p "$parent_dir" 2>/dev/null; then
            log_error "Cannot create parent directory: $parent_dir"
            bootstrap_show_fix_suggestion "permission" "$parent_dir"
            all_passed=false
        fi
    fi

    # Check if we can write to the target location
    if [[ -d "$target_dir" ]]; then
        # Directory exists, check if writable
        if [[ ! -w "$target_dir" ]]; then
            log_error "Target directory exists but is not writable: $target_dir"
            bootstrap_show_fix_suggestion "permission" "$target_dir"
            all_passed=false
        fi
    else
        # Try to create the directory
        if ! mkdir -p "$target_dir" 2>/dev/null; then
            log_error "Cannot create target directory: $target_dir"
            bootstrap_show_fix_suggestion "permission" "$target_dir"
            all_passed=false
        else
            # Clean up the test directory
            rmdir "$target_dir" 2>/dev/null || true
        fi
    fi

    # Check if bin directory parent is writable
    local bin_parent
    bin_parent=$(dirname "$BOOTSTRAP_BIN_DIR")
    if [[ ! -d "$bin_parent" ]]; then
        if ! mkdir -p "$bin_parent" 2>/dev/null; then
            log_error "Cannot create bin directory parent: $bin_parent"
            bootstrap_show_fix_suggestion "permission" "$bin_parent"
            all_passed=false
        fi
    fi

    # Check disk space (at least 50MB free)
    local required_space=50
    local available_space
    available_space=$(df -m "$parent_dir" 2>/dev/null | awk 'NR==2 {print $4}')
    if [[ -n "$available_space" ]]; then
        if [[ "$available_space" -lt "$required_space" ]]; then
            log_error "Insufficient disk space: ${available_space}MB available, ${required_space}MB required"
            all_passed=false
        else
            log_debug "Available disk space: ${available_space}MB (OK)"
        fi
    else
        log_warn "Could not determine available disk space"
    fi

    # Check if Morty is already installed
    if [[ -d "$BOOTSTRAP_PREFIX" ]] && [[ -f "$BOOTSTRAP_PREFIX/bin/morty" ]]; then
        log_warn "Morty appears to already be installed at: $BOOTSTRAP_PREFIX"
        log_warn "Use './bootstrap.sh reinstall' to overwrite the existing installation"
        log_warn "Use './bootstrap.sh upgrade' to upgrade to a newer version"
        all_passed=false
    fi

    if [[ "$all_passed" == "true" ]]; then
        log_success "Installation environment check passed"
        return 0
    else
        return 1
    fi
}

# Compare two version strings
# Usage: bootstrap_check_version <current> <minimum>
# Returns: 0 if current >= minimum, 1 otherwise
bootstrap_check_version() {
    local current="$1"
    local minimum="$2"

    # Use sort -V for version comparison if available
    if command -v sort &> /dev/null && echo "test" | sort -V &> /dev/null; then
        local higher
        higher=$(printf '%s\n%s\n' "$minimum" "$current" | sort -V | tail -n1)
        [[ "$higher" == "$current" ]]
        return $?
    else
        # Fallback: simple numeric comparison
        # This handles X.Y format but not X.Y.Z with letters
        local current_major current_minor
        local minimum_major minimum_minor

        current_major="${current%%.*}"
        current_minor="${current#*.}"
        current_minor="${current_minor%%.*}"

        minimum_major="${minimum%%.*}"
        minimum_minor="${minimum#*.}"
        minimum_minor="${minimum_minor%%.*}"

        # Default to 0 if empty
        current_major="${current_major:-0}"
        current_minor="${current_minor:-0}"
        minimum_major="${minimum_major:-0}"
        minimum_minor="${minimum_minor:-0}"

        if [[ "$current_major" -gt "$minimum_major" ]]; then
            return 0
        elif [[ "$current_major" -eq "$minimum_major" ]] && [[ "$current_minor" -ge "$minimum_minor" ]]; then
            return 0
        else
            return 1
        fi
    fi
}

# Show friendly fix suggestions for common issues
# Usage: bootstrap_show_fix_suggestion <issue_type> [details]
bootstrap_show_fix_suggestion() {
    local issue="$1"
    local details="${2:-}"

    echo ""
    echo "Fix suggestion:"
    echo "==============="

    case "$issue" in
        "bash")
            local min_version="$details"
            echo "Your Bash version is too old. Morty requires Bash >= $min_version."
            echo ""
            echo "To upgrade Bash:"
            echo "  Ubuntu/Debian:  sudo apt-get update && sudo apt-get install bash"
            echo "  CentOS/RHEL 7:  sudo yum install bash"
            echo "  CentOS/RHEL 8+: sudo dnf install bash"
            echo "  macOS:          brew install bash"
            echo "  Alpine:         apk add bash"
            echo ""
            echo "After installation, restart your terminal or run:"
            echo "  exec bash"
            ;;
        "git")
            echo "Git is not installed or too old."
            echo ""
            echo "To install Git:"
            echo "  Ubuntu/Debian:  sudo apt-get install git"
            echo "  CentOS/RHEL:    sudo yum install git"
            echo "  macOS:          brew install git"
            echo "  Alpine:         apk add git"
            ;;
        "permission")
            local dir="$details"
            echo "Permission denied when accessing: $dir"
            echo ""
            echo "Possible solutions:"
            echo "  1. Change the installation prefix to a directory you own:"
            echo "     ./bootstrap.sh install --prefix ~/my-morty"
            echo ""
            echo "  2. Create the directory with correct permissions:"
            echo "     mkdir -p $dir"
            echo "     chmod u+rwx $dir"
            echo ""
            echo "  3. Use sudo (not recommended for personal installations):"
            echo "     sudo ./bootstrap.sh install --prefix $dir"
            ;;
        "existing")
            local install_dir="$details"
            echo "Morty is already installed at: $install_dir"
            echo ""
            echo "Options:"
            echo "  1. Upgrade to the latest version:"
            echo "     ./bootstrap.sh upgrade"
            echo ""
            echo "  2. Reinstall (keeps configuration):"
            echo "     ./bootstrap.sh reinstall"
            echo ""
            echo "  3. Uninstall first:"
            echo "     ./bootstrap.sh uninstall"
            ;;
        *)
            echo "Unknown issue: $issue"
            echo "Please check the error message above and try again."
            ;;
    esac

    echo ""
}

# ============================================================================
# Command Handlers (Placeholders - to be implemented in subsequent jobs)
# ============================================================================

bootstrap_cmd_install() {
    log_info "Install command placeholder"
    log_info "To be implemented in Job 3"
    return 0
}

bootstrap_cmd_reinstall() {
    log_info "Reinstall command placeholder"
    log_info "To be implemented in Job 3"
    return 0
}

bootstrap_cmd_upgrade() {
    log_info "Upgrade command placeholder"
    log_info "To be implemented in Job 4"
    return 0
}

bootstrap_cmd_uninstall() {
    log_info "Uninstall command placeholder"
    log_info "To be implemented in Job 5"
    return 0
}

# ============================================================================
# Script Execution
# ============================================================================

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    bootstrap_main "$@"
fi
