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
