#!/usr/bin/env bash
# Morty Installation Script
# Main entry point for installing Morty

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the install module
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/logging.sh"
source "$SCRIPT_DIR/lib/install.sh"

# Parse command line arguments
PREFIX=""
BIN_DIR=""
FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --prefix)
            PREFIX="$2"
            shift 2
            ;;
        --bin-dir)
            BIN_DIR="$2"
            shift 2
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help|-h)
            cat << 'HELP'
Morty Installer

Usage: ./install.sh [options]

Options:
    --prefix <path>     Installation directory (default: ~/.morty)
    --bin-dir <path>    Binary directory for symlink (default: ~/.local/bin)
    --force             Force overwrite existing installation
    --help, -h          Show this help message

Examples:
    ./install.sh                          # Default installation
    ./install.sh --prefix /opt/morty      # Custom installation path
    ./install.sh --force                  # Reinstall with backup

HELP
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Use defaults if not specified
PREFIX="${PREFIX:-$(install_get_default_prefix)}"
BIN_DIR="${BIN_DIR:-$(install_get_default_bin_dir)}"

# Run installation
if ! install_do_install "$PREFIX" "$BIN_DIR" "$FORCE"; then
    echo ""
    echo "Installation failed!"
    exit 1
fi

echo ""
echo "Installation completed successfully!"
echo ""
echo "Next steps:"
echo "  1. Run 'morty --help' to see available commands"
echo "  2. Run 'morty research' to start researching your codebase"
echo ""
