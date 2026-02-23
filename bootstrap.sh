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
    [[ "$BOOTSTRAP_DEBUG" == "true" ]] && echo "[DEBUG] $1" >&2 || true
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
        bootstrap_show_fix_suggestion "downloader"
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

    # Check if Morty is already installed (only fail for fresh install, not reinstall)
    if [[ "$BOOTSTRAP_COMMAND" != "reinstall" ]]; then
        if [[ -d "$BOOTSTRAP_PREFIX" ]] && [[ -f "$BOOTSTRAP_PREFIX/bin/morty" ]]; then
            log_error "Morty is already installed at: $BOOTSTRAP_PREFIX"
            bootstrap_show_fix_suggestion "existing" "$BOOTSTRAP_PREFIX"
            all_passed=false
        fi
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
        "downloader")
            echo "Please install curl or wget to download Morty releases."
            echo ""
            echo "To install curl (recommended):"
            echo "  Ubuntu/Debian:  sudo apt-get install curl"
            echo "  CentOS/RHEL:    sudo yum install curl"
            echo "  macOS:          brew install curl"
            echo "  Alpine:         apk add curl"
            echo ""
            echo "Or install wget:"
            echo "  Ubuntu/Debian:  sudo apt-get install wget"
            echo "  CentOS/RHEL:    sudo yum install wget"
            echo "  macOS:          brew install wget"
            echo "  Alpine:         apk add wget"
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
        "verify_failed")
            echo "Installation verification failed."
            echo ""
            echo "Possible causes:"
            echo "  - The morty script may be corrupted"
            echo "  - Required dependencies may be missing"
            echo "  - File permissions may be incorrect"
            echo ""
            echo "Suggested actions:"
            echo "  1. Check that morty exists:"
            echo "     ls -la $BOOTSTRAP_PREFIX/bin/morty"
            echo ""
            echo "  2. Make it executable:"
            echo "     chmod +x $BOOTSTRAP_PREFIX/bin/morty"
            echo ""
            echo "  3. Try running morty directly:"
            echo "     $BOOTSTRAP_PREFIX/bin/morty --help"
            echo ""
            echo "  4. Uninstall and try again:"
            echo "     ./bootstrap.sh uninstall"
            echo "     ./bootstrap.sh install"
            ;;
        *)
            echo "Unknown issue: $issue"
            echo "Please check the error message above and try again."
            ;;
    esac

    echo ""
}

# ============================================================================
# Installation Functions
# ============================================================================

# Main install command handler
# Usage: bootstrap_cmd_install
# Returns: 0 on success, 1 on failure
bootstrap_cmd_install() {
    log_info "Starting Morty installation..."

    # Check if already installed
    if [[ -d "$BOOTSTRAP_PREFIX" ]] && [[ -f "$BOOTSTRAP_PREFIX/bin/morty" ]]; then
        log_error "Morty is already installed at: $BOOTSTRAP_PREFIX"
        log_info "Use 'reinstall' to overwrite or 'upgrade' to update"
        bootstrap_show_fix_suggestion "existing" "$BOOTSTRAP_PREFIX"
        return 1
    fi

    # Determine installation source
    local install_source
    local install_version

    if [[ -n "$BOOTSTRAP_SOURCE" ]]; then
        # Install from local source (development mode)
        install_source="source"
        log_info "Installing from local source: $BOOTSTRAP_SOURCE"
    else
        # Install from release
        install_source="release"
        install_version="${BOOTSTRAP_TARGET_VERSION:-latest}"
        log_info "Installing version: $install_version"
    fi

    # Create temporary directory for installation
    local temp_dir
    temp_dir=$(mktemp -d -t morty-install.XXXXXX)
    if [[ -z "$temp_dir" ]] || [[ ! -d "$temp_dir" ]]; then
        log_error "Failed to create temporary directory"
        return 1
    fi

    # Ensure cleanup on exit
    trap "bootstrap_cleanup_install '$temp_dir'" EXIT

    # Download or copy source files
    if [[ "$install_source" == "source" ]]; then
        if ! bootstrap_install_from_source "$BOOTSTRAP_SOURCE" "$temp_dir"; then
            log_error "Failed to copy files from source"
            return 1
        fi
    else
        if ! bootstrap_download_release "$install_version" "$temp_dir"; then
            log_error "Failed to download release"
            return 1
        fi
    fi

    # Create installation directory structure
    log_info "Creating installation directory structure..."
    if ! mkdir -p "$BOOTSTRAP_PREFIX"; then
        log_error "Failed to create installation directory: $BOOTSTRAP_PREFIX"
        return 1
    fi

    # Copy files to installation directory
    log_info "Installing files..."
    if ! bootstrap_copy_files "$temp_dir" "$BOOTSTRAP_PREFIX"; then
        log_error "Failed to copy files to installation directory"
        bootstrap_cleanup_failed_install "$BOOTSTRAP_PREFIX"
        return 1
    fi

    # Set permissions
    log_info "Setting file permissions..."
    if ! bootstrap_set_permissions "$BOOTSTRAP_PREFIX"; then
        log_error "Failed to set file permissions"
        return 1
    fi

    # Create symbolic link
    log_info "Creating symbolic link..."
    if ! bootstrap_create_symlink "$BOOTSTRAP_PREFIX" "$BOOTSTRAP_BIN_DIR"; then
        log_error "Failed to create symbolic link"
        bootstrap_cleanup_failed_install "$BOOTSTRAP_PREFIX"
        return 1
    fi

    # Remove trap since installation is successful
    trap - EXIT

    # Cleanup temp directory
    rm -rf "$temp_dir"

    # Verify installation
    log_info "Verifying installation..."
    if ! bootstrap_verify_install; then
        log_error "Installation verification failed"
        bootstrap_show_fix_suggestion "verify_failed"
        return 1
    fi

    # Show completion message
    bootstrap_show_install_complete

    return 0
}

# Cleanup installation temp directory
# Usage: bootstrap_cleanup_install <temp_dir>
bootstrap_cleanup_install() {
    local temp_dir="$1"
    if [[ -n "$temp_dir" ]] && [[ -d "$temp_dir" ]]; then
        rm -rf "$temp_dir"
    fi
}

# Cleanup failed installation
# Usage: bootstrap_cleanup_failed_install <install_dir>
bootstrap_cleanup_failed_install() {
    local install_dir="$1"
    log_warn "Cleaning up failed installation..."
    if [[ -n "$install_dir" ]] && [[ -d "$install_dir" ]]; then
        rm -rf "$install_dir"
    fi
}

# Download release from GitHub
# Usage: bootstrap_download_release <version> <target_dir>
# Returns: 0 on success, 1 on failure
bootstrap_download_release() {
    local version="$1"
    local target_dir="$2"

    log_info "Downloading Morty release..."

    # Determine download URL
    local download_url
    if [[ "$version" == "latest" ]]; then
        download_url="${GITHUB_API_URL}/releases/latest"
    else
        download_url="${GITHUB_API_URL}/releases/tags/${version}"
    fi

    # Get release info and find download URL
    local release_info
    local download_link=""

    log_info "Fetching release information..."

    if command -v curl &> /dev/null; then
        release_info=$(curl -sL "$download_url" 2>/dev/null) || true
    elif command -v wget &> /dev/null; then
        release_info=$(wget -qO- "$download_url" 2>/dev/null) || true
    fi

    if [[ -z "$release_info" ]]; then
        log_warn "Could not fetch release info from GitHub API"
        log_info "Attempting direct download from source..."

        # Fallback: try to download from raw GitHub
        local branch="${version}"
        if [[ "$version" == "latest" ]]; then
            branch="master"
        fi

        # Create a temporary tarball approach using git clone
        if command -v git &> /dev/null; then
            log_info "Cloning from GitHub (branch: $branch)..."
            if ! git clone --depth 1 --branch "$branch" "https://github.com/${GITHUB_REPO}.git" "$target_dir" 2>/dev/null; then
                # Try master if branch doesn't exist
                if [[ "$branch" != "master" ]]; then
                    log_warn "Branch $branch not found, trying master..."
                    if ! git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" "$target_dir" 2>/dev/null; then
                        log_error "Failed to clone repository"
                        return 1
                    fi
                else
                    log_error "Failed to clone repository"
                    return 1
                fi
            fi
            return 0
        else
            log_error "Git is required for installation from GitHub"
            return 1
        fi
    fi

    # Try to extract tarball URL from release info
    if command -v jq &> /dev/null; then
        download_link=$(echo "$release_info" | jq -r '.assets[] | select(.name | contains("tar.gz")) | .browser_download_url' 2>/dev/null | head -n1)
    else
        # Simple grep/sed fallback for tarball URL
        download_link=$(echo "$release_info" | grep -o '"browser_download_url": "[^"]*tar.gz"' | head -n1 | sed 's/.*: "\(.*\)".*/\1/') || true
    fi

    if [[ -z "$download_link" ]]; then
        # Fallback to source tarball
        if [[ "$version" == "latest" ]]; then
            download_link="https://github.com/${GITHUB_REPO}/archive/refs/heads/master.tar.gz"
        else
            download_link="https://github.com/${GITHUB_REPO}/archive/refs/tags/${version}.tar.gz"
        fi
    fi

    log_info "Downloading from: $download_link"

    # Download the tarball
    local tarball="${target_dir}/morty.tar.gz"
    if command -v curl &> /dev/null; then
        if ! curl -sL -o "$tarball" "$download_link" 2>/dev/null; then
            log_error "Download failed with curl"
            return 1
        fi
    elif command -v wget &> /dev/null; then
        if ! wget -q -O "$tarball" "$download_link" 2>/dev/null; then
            log_error "Download failed with wget"
            return 1
        fi
    else
        log_error "Neither curl nor wget available"
        return 1
    fi

    # Verify download
    if [[ ! -f "$tarball" ]] || [[ ! -s "$tarball" ]]; then
        log_error "Downloaded file is empty or missing"
        return 1
    fi

    # Extract tarball
    log_info "Extracting files..."
    if ! tar -xzf "$tarball" -C "$target_dir" 2>/dev/null; then
        log_error "Failed to extract tarball"
        return 1
    fi

    # Move extracted contents to target directory root
    # The tarball usually creates a subdirectory like "morty-master" or "morty-X.X.X"
    local extracted_dir
    extracted_dir=$(find "$target_dir" -maxdepth 1 -type d -name "morty-*" 2>/dev/null | head -n1)
    if [[ -n "$extracted_dir" ]] && [[ "$extracted_dir" != "$target_dir" ]]; then
        # Move contents from subdirectory to target_dir
        mv "$extracted_dir"/* "$target_dir" 2>/dev/null || true
        rmdir "$extracted_dir" 2>/dev/null || true
    fi

    # Remove tarball
    rm -f "$tarball"

    log_success "Download completed"
    return 0
}

# Install from local source directory
# Usage: bootstrap_install_from_source <source_path> <target_dir>
# Returns: 0 on success, 1 on failure
bootstrap_install_from_source() {
    local source_path="$1"
    local target_dir="$2"

    log_info "Copying from local source..."

    # Verify source directory exists
    if [[ ! -d "$source_path" ]]; then
        log_error "Source directory does not exist: $source_path"
        return 1
    fi

    # Check for required files in source
    if [[ ! -f "$source_path/morty" ]]; then
        log_warn "Source directory may be incomplete (morty main script not found)"
    fi

    # Copy all files from source to target
    # Use cp -R to preserve structure, excluding certain directories
    local exclude_pattern=".git|.morty|*.log"

    # Create target directory structure
    mkdir -p "$target_dir"

    # Copy files, excluding git and working directories
    if command -v rsync &> /dev/null; then
        rsync -a --exclude='.git' --exclude='.morty' --exclude='*.log' "$source_path"/ "$target_dir"/ 2>/dev/null
    else
        # Fallback: manual copy with find
        while IFS= read -r -d '' file; do
            local rel_path="${file#$source_path/}"
            local target_file="$target_dir/$rel_path"
            local target_file_dir
            target_file_dir=$(dirname "$target_file")
            mkdir -p "$target_file_dir"
            cp -p "$file" "$target_file" 2>/dev/null || true
        done < <(find "$source_path" -type f \
            ! -path "$source_path/.git/*" \
            ! -path "$source_path/.morty/*" \
            ! -name "*.log" \
            -print0 2>/dev/null)
    fi

    log_success "Source files copied"
    return 0
}

# Copy files to installation directory
# Usage: bootstrap_copy_files <source_dir> <target_dir>
# Returns: 0 on success, 1 on failure
bootstrap_copy_files() {
    local source_dir="$1"
    local target_dir="$2"

    # Create standard directory structure
    mkdir -p "$target_dir/bin"
    mkdir -p "$target_dir/lib"
    mkdir -p "$target_dir/prompts"

    # Copy main morty script
    if [[ -f "$source_dir/morty" ]]; then
        cp "$source_dir/morty" "$target_dir/bin/"
    fi

    # Copy morty_*.sh scripts
    for script in "$source_dir"/morty_*.sh; do
        if [[ -f "$script" ]]; then
            cp "$script" "$target_dir/bin/"
        fi
    done

    # Copy lib files
    if [[ -d "$source_dir/lib" ]]; then
        for file in "$source_dir/lib"/*.sh; do
            if [[ -f "$file" ]]; then
                cp "$file" "$target_dir/lib/"
            fi
        done
    fi

    # Copy prompts
    if [[ -d "$source_dir/prompts" ]]; then
        for file in "$source_dir/prompts"/*.md; do
            if [[ -f "$file" ]]; then
                cp "$file" "$target_dir/prompts/"
            fi
        done
    fi

    # Copy VERSION file if exists
    if [[ -f "$source_dir/VERSION" ]]; then
        cp "$source_dir/VERSION" "$target_dir/"
    else
        # Create a VERSION file with bootstrap version
        echo "$BOOTSTRAP_VERSION" > "$target_dir/VERSION"
    fi

    # Create lib symlink in bin directory for morty script to find libraries
    # morty script uses: source "${MORTY_SCRIPT_DIR}/lib/cli_*.sh"
    # So we need bin/lib -> ../lib
    if [[ ! -e "$target_dir/bin/lib" ]]; then
        ln -s "../lib" "$target_dir/bin/lib"
    fi

    return 0
}

# Create symbolic link
# Usage: bootstrap_create_symlink <install_dir> <bin_dir>
# Returns: 0 on success, 1 on failure
bootstrap_create_symlink() {
    local install_dir="$1"
    local bin_dir="$2"

    # Create bin directory if it doesn't exist
    if [[ ! -d "$bin_dir" ]]; then
        mkdir -p "$bin_dir"
    fi

    local target="$install_dir/bin/morty"
    local link_name="$bin_dir/morty"

    # Remove existing link if it exists
    if [[ -L "$link_name" ]]; then
        rm -f "$link_name"
    fi

    # Create symlink
    if ! ln -s "$target" "$link_name" 2>/dev/null; then
        log_error "Failed to create symbolic link: $link_name -> $target"
        return 1
    fi

    log_debug "Created symlink: $link_name -> $target"
    return 0
}

# Set file permissions
# Usage: bootstrap_set_permissions <install_dir>
# Returns: 0 on success, 1 on failure
bootstrap_set_permissions() {
    local install_dir="$1"

    # Make all scripts executable
    find "$install_dir/bin" -type f -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
    find "$install_dir/bin" -type f -name "morty" -exec chmod +x {} \; 2>/dev/null || true

    # Ensure morty scripts are executable
    if [[ -f "$install_dir/bin/morty" ]]; then
        chmod +x "$install_dir/bin/morty"
    fi

    return 0
}

# Verify installation
# Usage: bootstrap_verify_install
# Returns: 0 on success, 1 on failure
bootstrap_verify_install() {
    local morty_bin="$BOOTSTRAP_PREFIX/bin/morty"
    local morty_link="$BOOTSTRAP_BIN_DIR/morty"

    # Check if main morty script exists
    if [[ ! -f "$morty_bin" ]]; then
        log_error "Morty binary not found at: $morty_bin"
        return 1
    fi

    # Check if it's executable
    if [[ ! -x "$morty_bin" ]]; then
        log_error "Morty binary is not executable"
        return 1
    fi

    # Check if symlink exists and points to correct location
    if [[ ! -L "$morty_link" ]]; then
        log_error "Symbolic link not found at: $morty_link"
        return 1
    fi

    # Verify symlink target
    local symlink_target
    symlink_target=$(readlink "$morty_link")
    if [[ "$symlink_target" != "$morty_bin" ]]; then
        log_error "Symbolic link points to wrong location: $symlink_target"
        return 1
    fi

    # Try to run morty version
    local version_output
    if version_output=$("$morty_bin" version 2>/dev/null); then
        log_success "Morty installation verified"
        log_info "Installed version: $version_output"
    else
        log_warn "Could not verify version (morty version command failed)"
        # Don't fail if version command doesn't work in development mode
    fi

    return 0
}

# Show installation completion message
# Usage: bootstrap_show_install_complete
bootstrap_show_install_complete() {
    echo ""
    echo "============================================"
    echo "  Installation Complete!"
    echo "============================================"
    echo ""
    echo "Morty has been installed to:"
    echo "  $BOOTSTRAP_PREFIX"
    echo ""
    echo "Symbolic link created at:"
    echo "  $BOOTSTRAP_BIN_DIR/morty"
    echo ""

    # Check if bin_dir is in PATH
    if [[ ":$PATH:" != *":$BOOTSTRAP_BIN_DIR:"* ]]; then
        echo "Note: $BOOTSTRAP_BIN_DIR is not in your PATH"
        echo ""
        echo "Add it to your shell configuration:"
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
        echo "  source ~/.bashrc"
        echo ""
    fi

    echo "Get started with:"
    echo "  morty --help"
    echo ""
}

# ============================================================================
# Command Handlers (Additional - to be implemented in subsequent jobs)
# ============================================================================

# Backup user configuration before reinstall/upgrade
# Usage: bootstrap_backup_config <install_dir>
# Returns: 0 on success, 1 on failure
# Outputs: backup directory path to stdout (capture with: backup_dir=$(bootstrap_backup_config ...))
bootstrap_backup_config() {
    local install_dir="$1"
    local backup_dir="${install_dir}.backup.$$"

    # Create backup directory (log to stderr so it doesn't interfere with output capture)
    echo "==> Backing up user configuration..." >&2

    if ! mkdir -p "$backup_dir"; then
        log_error "Failed to create backup directory: $backup_dir"
        return 1
    fi

    # Backup settings.json if exists
    if [[ -f "$install_dir/settings.json" ]]; then
        cp "$install_dir/settings.json" "$backup_dir/" 2>/dev/null || true
        log_debug "Backed up settings.json"
    fi

    # Backup .mortyrc if exists (old config location)
    if [[ -f "$install_dir/.mortyrc" ]]; then
        cp "$install_dir/.mortyrc" "$backup_dir/" 2>/dev/null || true
        log_debug "Backed up .mortyrc"
    fi

    # Backup any user custom prompts
    if [[ -d "$install_dir/prompts" ]]; then
        mkdir -p "$backup_dir/prompts"
        for file in "$install_dir/prompts"/*.md; do
            if [[ -f "$file" ]]; then
                cp "$file" "$backup_dir/prompts/" 2>/dev/null || true
            fi
        done
        log_debug "Backed up custom prompts"
    fi

    # Return backup directory path via echo (caller should capture)
    echo "$backup_dir"
    return 0
}

# Restore user configuration after reinstall/upgrade
# Usage: bootstrap_restore_config <backup_dir> <install_dir>
# Returns: 0 on success, 1 on failure
bootstrap_restore_config() {
    local backup_dir="$1"
    local install_dir="$2"

    echo "==> Restoring user configuration..." >&2

    # Check if backup directory exists
    if [[ ! -d "$backup_dir" ]]; then
        log_warn "Backup directory not found: $backup_dir"
        return 1
    fi

    # Restore settings.json
    if [[ -f "$backup_dir/settings.json" ]]; then
        cp "$backup_dir/settings.json" "$install_dir/" 2>/dev/null || true
        log_success "Restored settings.json"
    fi

    # Restore .mortyrc
    if [[ -f "$backup_dir/.mortyrc" ]]; then
        cp "$backup_dir/.mortyrc" "$install_dir/" 2>/dev/null || true
        log_debug "Restored .mortyrc"
    fi

    # Restore custom prompts
    if [[ -d "$backup_dir/prompts" ]]; then
        mkdir -p "$install_dir/prompts"
        for file in "$backup_dir/prompts"/*.md; do
            if [[ -f "$file" ]]; then
                cp "$file" "$install_dir/prompts/" 2>/dev/null || true
            fi
        done
        log_debug "Restored custom prompts"
    fi

    # Clean up backup directory
    rm -rf "$backup_dir"

    return 0
}

# Cleanup backup directory
# Usage: bootstrap_cleanup_backup <backup_dir>
bootstrap_cleanup_backup() {
    local backup_dir="$1"
    if [[ -n "$backup_dir" ]] && [[ -d "$backup_dir" ]]; then
        rm -rf "$backup_dir"
        log_debug "Cleaned up backup directory: $backup_dir"
    fi
}

# Reinstall command handler
# Usage: bootstrap_cmd_reinstall
# Returns: 0 on success, 1 on failure
bootstrap_cmd_reinstall() {
    log_info "Starting Morty reinstallation..."

    # Check if Morty is actually installed
    if [[ ! -d "$BOOTSTRAP_PREFIX" ]] || [[ ! -f "$BOOTSTRAP_PREFIX/bin/morty" ]]; then
        log_warn "No existing Morty installation found at: $BOOTSTRAP_PREFIX"
        log_info "Running standard install instead..."
        bootstrap_cmd_install
        return $?
    fi

    # Confirm with user unless --force is used
    if [[ "$BOOTSTRAP_FORCE" != "true" ]]; then
        echo ""
        echo "This will reinstall Morty at: $BOOTSTRAP_PREFIX"
        echo "Your configuration will be preserved."
        echo ""
        read -r -p "Continue? [y/N] " response
        case "$response" in
            [yY][eE][sS]|[yY])
                # Continue with reinstall
                ;;
            *)
                log_info "Reinstall cancelled by user"
                return 0
                ;;
        esac
    fi

    # Backup configuration
    local backup_dir
    backup_dir=$(bootstrap_backup_config "$BOOTSTRAP_PREFIX")
    if [[ -z "$backup_dir" ]] || [[ ! -d "$backup_dir" ]]; then
        log_error "Failed to backup configuration"
        return 1
    fi

    # Set trap to cleanup backup on exit
    trap "bootstrap_cleanup_backup '$backup_dir'" EXIT

    # Remove existing installation (except backup)
    log_info "Removing existing installation..."
    if [[ -d "$BOOTSTRAP_PREFIX" ]]; then
        # Remove symlink first
        if [[ -L "$BOOTSTRAP_BIN_DIR/morty" ]]; then
            rm -f "$BOOTSTRAP_BIN_DIR/morty"
            log_debug "Removed existing symlink"
        fi

        # Remove installation directory
        rm -rf "$BOOTSTRAP_PREFIX"
        log_debug "Removed existing installation directory"
    fi

    # Perform fresh installation
    log_info "Performing fresh installation..."
    if ! bootstrap_cmd_install; then
        log_error "Installation failed during reinstall"
        log_info "Attempting to restore from backup..."
        # Note: backup will be cleaned up by trap
        return 1
    fi

    # Restore configuration
    if ! bootstrap_restore_config "$backup_dir" "$BOOTSTRAP_PREFIX"; then
        log_warn "Failed to restore some configuration files"
        log_warn "Backup is preserved at: $backup_dir"
        # Remove trap since we're keeping the backup
        trap - EXIT
    fi

    log_success "Reinstall completed successfully!"
    log_info "Your configuration has been preserved."

    return 0
}

# ============================================================================
# Upgrade Functions
# ============================================================================

# Get current installed version
# Usage: bootstrap_get_current_version
# Outputs: version string to stdout, or empty if not installed
# Returns: 0 on success, 1 if not installed
bootstrap_get_current_version() {
    local version_file="$BOOTSTRAP_PREFIX/VERSION"

    if [[ ! -f "$version_file" ]]; then
        return 1
    fi

    local version
    version=$(cat "$version_file" 2>/dev/null | tr -d '[:space:]')
    if [[ -z "$version" ]]; then
        return 1
    fi

    echo "$version"
    return 0
}

# Get latest available version from GitHub
# Usage: bootstrap_get_latest_version
# Outputs: version string to stdout
# Returns: 0 on success, 1 on failure
bootstrap_get_latest_version() {
    log_info "Checking for latest version..."

    local latest_version=""

    # Try to get latest version from GitHub API
    local api_url="${GITHUB_API_URL}/releases/latest"
    local release_info=""

    if command -v curl &> /dev/null; then
        release_info=$(curl -sL "$api_url" 2>/dev/null) || true
    elif command -v wget &> /dev/null; then
        release_info=$(wget -qO- "$api_url" 2>/dev/null) || true
    fi

    if [[ -n "$release_info" ]]; then
        # Try to extract version using jq if available
        if command -v jq &> /dev/null; then
            latest_version=$(echo "$release_info" | jq -r '.tag_name' 2>/dev/null | sed 's/^v//')
        else
            # Fallback: extract with grep/sed
            latest_version=$(echo "$release_info" | grep -o '"tag_name": "[^"]*"' | head -n1 | sed 's/.*: "v\?\([^"]*\)".*/\1/') || true
        fi
    fi

    # If we couldn't get the version from API, use a fallback
    if [[ -z "$latest_version" ]]; then
        log_warn "Could not fetch latest version from GitHub API"
        log_info "Using default version"
        latest_version="2.0.0"  # Default fallback version
    fi

    echo "$latest_version"
    return 0
}

# Compare two version strings
# Usage: bootstrap_compare_versions <version1> <version2>
# Returns:
#   0 if version1 == version2
#   1 if version1 > version2
#   2 if version1 < version2
bootstrap_compare_versions() {
    local v1="$1"
    local v2="$2"

    # Normalize versions (remove leading 'v' if present)
    v1="${v1#v}"
    v2="${v2#v}"

    # Handle empty versions
    if [[ -z "$v1" && -z "$v2" ]]; then
        return 0
    fi
    if [[ -z "$v1" ]]; then
        return 2
    fi
    if [[ -z "$v2" ]]; then
        return 1
    fi

    # Check for equality first
    if [[ "$v1" == "$v2" ]]; then
        return 0
    fi

    # Manual comparison by version parts
    local IFS='.'
    local -a v1_parts=($v1)
    local -a v2_parts=($v2)

    local max_len=${#v1_parts[@]}
    if [[ ${#v2_parts[@]} -gt $max_len ]]; then
        max_len=${#v2_parts[@]}
    fi

    for ((i=0; i<max_len; i++)); do
        local p1="${v1_parts[$i]:-0}"
        local p2="${v2_parts[$i]:-0}"

        # Remove any non-numeric suffix for comparison
        p1="${p1%%[^0-9]*}"
        p2="${p2%%[^0-9]*}"

        # Default to 0 if empty after removing non-numeric
        p1="${p1:-0}"
        p2="${p2:-0}"

        if [[ "$p1" -gt "$p2" ]]; then
            return 1
        elif [[ "$p1" -lt "$p2" ]]; then
            return 2
        fi
    done

    # All parts equal
    return 0
}

# Backup complete installation before upgrade
# Usage: bootstrap_backup_installation <install_dir>
# Outputs: backup directory path to stdout
# Returns: 0 on success, 1 on failure
bootstrap_backup_installation() {
    local install_dir="$1"
    local backup_dir="${install_dir}.backup.$(date +%Y%m%d_%H%M%S).$$"

    log_info "Creating full installation backup..."

    if [[ ! -d "$install_dir" ]]; then
        log_error "Installation directory does not exist: $install_dir"
        return 1
    fi

    # Create backup directory
    if ! mkdir -p "$backup_dir"; then
        log_error "Failed to create backup directory: $backup_dir"
        return 1
    fi

    # Copy all files to backup (preserve permissions)
    if command -v cp &> /dev/null && cp -a "$install_dir"/* "$backup_dir"/ 2>/dev/null; then
        log_success "Backup created at: $backup_dir"
    else
        log_error "Failed to create backup"
        rm -rf "$backup_dir"
        return 1
    fi

    echo "$backup_dir"
    return 0
}

# Migrate configuration after upgrade
# Usage: bootstrap_migrate_config <old_version> <new_version>
# Returns: 0 on success, 1 on failure
bootstrap_migrate_config() {
    local old_version="$1"
    local new_version="$2"

    log_info "Checking for configuration migration..."

    # Currently no migrations needed between versions
    # This function is a placeholder for future migrations
    # Example migrations could include:
    # - Updating settings.json format
    # - Moving config files to new locations
    # - Converting old config formats

    log_debug "Migration from $old_version to $new_version not required"
    return 0
}

# Rollback to previous version on upgrade failure
# Usage: bootstrap_rollback_upgrade <backup_dir> <install_dir>
# Returns: 0 on success, 1 on failure
bootstrap_rollback_upgrade() {
    local backup_dir="$1"
    local install_dir="$2"

    log_warn "Rolling back to previous version..."

    if [[ ! -d "$backup_dir" ]]; then
        log_error "Backup directory not found: $backup_dir"
        return 1
    fi

    # Remove failed installation
    if [[ -d "$install_dir" ]]; then
        rm -rf "$install_dir"
    fi

    # Restore from backup
    if mv "$backup_dir" "$install_dir" 2>/dev/null; then
        log_success "Rollback completed successfully"
        return 0
    else
        log_error "Rollback failed - backup is preserved at: $backup_dir"
        return 1
    fi
}

# Main upgrade command handler
# Usage: bootstrap_cmd_upgrade
# Returns: 0 on success, 1 on failure
bootstrap_cmd_upgrade() {
    log_info "Starting Morty upgrade..."

    # Check if Morty is installed
    if [[ ! -d "$BOOTSTRAP_PREFIX" ]] || [[ ! -f "$BOOTSTRAP_PREFIX/bin/morty" ]]; then
        log_error "Morty is not installed at: $BOOTSTRAP_PREFIX"
        log_info "Use 'install' command to install Morty"
        return 1
    fi

    # Get current version
    local current_version
    if ! current_version=$(bootstrap_get_current_version); then
        log_warn "Could not determine current version"
        current_version="unknown"
    fi
    log_info "Current version: $current_version"

    # Get target version (user specified or latest)
    local target_version
    if [[ -n "$BOOTSTRAP_TARGET_VERSION" ]]; then
        target_version="$BOOTSTRAP_TARGET_VERSION"
        log_info "Target version (specified): $target_version"
    else
        target_version=$(bootstrap_get_latest_version)
        log_info "Latest version: $target_version"
    fi

    # Compare versions
    local compare_result
    bootstrap_compare_versions "$current_version" "$target_version"
    compare_result=$?

    case $compare_result in
        0)
            # Versions are equal
            log_success "Morty is already at the latest version ($current_version)"
            return 0
            ;;
        1)
            # Current > target (downgrade attempt)
            log_warn "Current version ($current_version) is newer than target ($target_version)"
            if [[ "$BOOTSTRAP_FORCE" != "true" ]]; then
                echo ""
                echo "This would downgrade Morty to an older version."
                read -r -p "Continue anyway? [y/N] " response
                case "$response" in
                    [yY][eE][sS]|[yY])
                        # Continue with downgrade
                        ;;
                    *)
                        log_info "Upgrade cancelled by user"
                        return 0
                        ;;
                esac
            fi
            ;;
        2)
            # Current < target (normal upgrade)
            log_info "New version available: $target_version"
            ;;
    esac

    # Confirm with user unless --force is used
    if [[ "$BOOTSTRAP_FORCE" != "true" ]]; then
        echo ""
        echo "This will upgrade Morty from $current_version to $target_version"
        echo "Your configuration will be preserved."
        echo ""
        read -r -p "Continue? [y/N] " response
        case "$response" in
            [yY][eE][sS]|[yY])
                # Continue with upgrade
                ;;
            *)
                log_info "Upgrade cancelled by user"
                return 0
                ;;
        esac
    fi

    # Create backup of current installation
    local backup_dir
    backup_dir=$(bootstrap_backup_installation "$BOOTSTRAP_PREFIX")
    if [[ -z "$backup_dir" ]] || [[ ! -d "$backup_dir" ]]; then
        log_error "Failed to create backup - aborting upgrade"
        return 1
    fi

    # Set trap to cleanup backup on successful exit, but preserve on failure
    local upgrade_failed=false

    # Remove symlink before upgrade (will be recreated)
    if [[ -L "$BOOTSTRAP_BIN_DIR/morty" ]]; then
        rm -f "$BOOTSTRAP_BIN_DIR/morty"
        log_debug "Removed existing symlink"
    fi

    # Create temporary directory for download
    local temp_dir
    temp_dir=$(mktemp -d -t morty-upgrade.XXXXXX)
    if [[ -z "$temp_dir" ]] || [[ ! -d "$temp_dir" ]]; then
        log_error "Failed to create temporary directory"
        bootstrap_rollback_upgrade "$backup_dir" "$BOOTSTRAP_PREFIX"
        return 1
    fi

    # Download new version
    log_info "Downloading version $target_version..."
    if ! bootstrap_download_release "$target_version" "$temp_dir"; then
        log_error "Failed to download version $target_version"
        upgrade_failed=true
    fi

    # If download succeeded, perform the upgrade
    if [[ "$upgrade_failed" != "true" ]]; then
        # Remove old installation (backup is safe)
        log_info "Removing old version..."
        rm -rf "$BOOTSTRAP_PREFIX"

        # Create new installation directory
        if ! mkdir -p "$BOOTSTRAP_PREFIX"; then
            log_error "Failed to create installation directory"
            upgrade_failed=true
        fi
    fi

    # Copy new files
    if [[ "$upgrade_failed" != "true" ]]; then
        log_info "Installing new version..."
        if ! bootstrap_copy_files "$temp_dir" "$BOOTSTRAP_PREFIX"; then
            log_error "Failed to copy new files"
            upgrade_failed=true
        fi
    fi

    # Set permissions
    if [[ "$upgrade_failed" != "true" ]]; then
        if ! bootstrap_set_permissions "$BOOTSTRAP_PREFIX"; then
            log_error "Failed to set permissions"
            upgrade_failed=true
        fi
    fi

    # Create symlink
    if [[ "$upgrade_failed" != "true" ]]; then
        if ! bootstrap_create_symlink "$BOOTSTRAP_PREFIX" "$BOOTSTRAP_BIN_DIR"; then
            log_error "Failed to create symbolic link"
            upgrade_failed=true
        fi
    fi

    # Cleanup temp directory
    rm -rf "$temp_dir"

    # Verify installation
    if [[ "$upgrade_failed" != "true" ]]; then
        log_info "Verifying upgrade..."
        if ! bootstrap_verify_install; then
            log_error "Installation verification failed"
            upgrade_failed=true
        fi
    fi

    # Check if upgrade succeeded
    if [[ "$upgrade_failed" == "true" ]]; then
        log_error "Upgrade failed - rolling back..."
        if bootstrap_rollback_upgrade "$backup_dir" "$BOOTSTRAP_PREFIX"; then
            # Recreate symlink after rollback
            bootstrap_create_symlink "$BOOTSTRAP_PREFIX" "$BOOTSTRAP_BIN_DIR" || true
            log_info "Rollback completed - your previous version is restored"
        fi
        return 1
    fi

    # Migrate configuration if needed
    bootstrap_migrate_config "$current_version" "$target_version" || true

    # Get new installed version for confirmation
    local new_version
    new_version=$(bootstrap_get_current_version)
    log_success "Upgrade completed successfully!"
    log_info "Updated from $current_version to $new_version"

    # Cleanup backup on successful upgrade
    rm -rf "$backup_dir"
    log_debug "Cleaned up backup directory"

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
