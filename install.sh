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
cp morty_plan.sh "$INSTALL_DIR/"
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
Morty - Simplified AI Development Loop

Usage: morty <command> [options]

Commands:
    plan <prd.md> [name]    Interactive PRD refinement (generates project)
    enable                  Enable Morty in existing project
    start                   Start the development loop
    monitor                 Start with tmux monitoring
    status                  Show current status
    rollback <loop-number>  Rollback to specific loop iteration
    history                 Show loop history from git commits
    version                 Show version

Examples:
    morty plan requirements.md         # Refine PRD and generate project
    morty plan docs/prd.md my-app      # With custom project name
    morty enable                       # Enable in existing project
    morty start                        # Start development loop
    morty monitor                      # Start with monitoring
    morty rollback 5                   # Rollback to loop #5
    morty history                      # Show loop commit history

HELP
}

show_version() {
    echo "Morty version $VERSION"
}

# Command routing
case "${1:-}" in
    plan)
        shift
        exec "$MORTY_HOME/morty_plan.sh" "$@"
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

log INFO "Quick start:"
log INFO "  morty plan requirements.md # Refine PRD and generate project"
log INFO "  morty enable               # Enable in existing project"
log INFO ""
log SUCCESS "Happy coding with Morty! 🚀"
