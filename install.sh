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
cp morty_init.sh "$INSTALL_DIR/"
cp morty_import.sh "$INSTALL_DIR/"
cp morty_enable.sh "$INSTALL_DIR/"
cp morty_loop.sh "$INSTALL_DIR/"
cp morty_monitor.sh "$INSTALL_DIR/"

# Copy library
cp -r lib "$INSTALL_DIR/"

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
Morty - Simplified AI Development Loop

Usage: morty <command> [options]

Commands:
    init <project>          Create new project from scratch
    import <prd.md> [name]  Import PRD and create project
    enable                  Enable Morty in existing project
    start                   Start the development loop
    monitor                 Start with tmux monitoring
    status                  Show current status
    version                 Show version

Examples:
    morty init my-project              # Create new project
    morty import requirements.md       # Import from PRD
    morty enable                       # Enable in current project
    morty start                        # Start loop
    morty monitor                      # Start with monitoring

HELP
}

show_version() {
    echo "Morty version $VERSION"
}

# Command routing
case "${1:-}" in
    init)
        shift
        exec "$MORTY_HOME/morty_init.sh" "$@"
        ;;
    import)
        shift
        exec "$MORTY_HOME/morty_import.sh" "$@"
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

log INFO "Quick start:"
log INFO "  morty init my-project      # Create new project"
log INFO "  morty import prd.md        # Import from PRD"
log INFO "  morty enable               # Enable in existing project"
log INFO ""
log SUCCESS "Happy coding with Morty! 🚀"
