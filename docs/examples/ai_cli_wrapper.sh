#!/bin/bash
# Example Enterprise Claude Code CLI Wrapper
#
# This is an example of how to create a custom CLI wrapper for enterprise use.
# Copy and modify this script for your organization's needs.

set -e

# Script information
SCRIPT_NAME="ai_cli"
VERSION="1.0.0"

# Configuration
ENTERPRISE_CONFIG_DIR="${ENTERPRISE_CONFIG_DIR:-/opt/company/config}"
ENTERPRISE_API_ENDPOINT="${ENTERPRISE_API_ENDPOINT:-https://api.company.com/claude}"
ENTERPRISE_PROXY="${ENTERPRISE_PROXY:-http://proxy.company.com:8080}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Show help
show_help() {
    cat << EOF
$SCRIPT_NAME - Enterprise Claude Code CLI Wrapper

Usage: $SCRIPT_NAME [options] [claude-args...]

Options:
    --config <env>      Use specific environment config (dev, staging, prod)
    --auth <method>     Authentication method (sso, token, cert)
    --team <name>       Team identifier for billing/tracking
    --debug             Enable debug logging
    --version           Show version
    --help              Show this help

Examples:
    $SCRIPT_NAME --config prod --auth sso
    $SCRIPT_NAME --team data-science -p "Hello, Claude"
    $SCRIPT_NAME --debug --auth token

Environment Variables:
    ENTERPRISE_CONFIG_DIR    Configuration directory (default: /opt/company/config)
    ENTERPRISE_API_ENDPOINT  API endpoint (default: https://api.company.com/claude)
    ENTERPRISE_PROXY         HTTP proxy (default: http://proxy.company.com:8080)
    ENTERPRISE_API_KEY       API key (if using token auth)

EOF
}

# Parse arguments
CONFIG_ENV="prod"
AUTH_METHOD="sso"
TEAM_ID=""
DEBUG=false
CLAUDE_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --config)
            CONFIG_ENV="$2"
            shift 2
            ;;
        --auth)
            AUTH_METHOD="$2"
            shift 2
            ;;
        --team)
            TEAM_ID="$2"
            shift 2
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        --version)
            echo "$SCRIPT_NAME version $VERSION"
            exit 0
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            # Pass remaining args to Claude Code
            CLAUDE_ARGS+=("$1")
            shift
            ;;
    esac
done

# Debug logging
if [[ "$DEBUG" == "true" ]]; then
    set -x
    log_info "Debug mode enabled"
fi

# Load configuration
CONFIG_FILE="$ENTERPRISE_CONFIG_DIR/claude_${CONFIG_ENV}.conf"
if [[ -f "$CONFIG_FILE" ]]; then
    log_info "Loading config: $CONFIG_FILE"
    source "$CONFIG_FILE"
else
    log_warn "Config file not found: $CONFIG_FILE (using defaults)"
fi

# Authentication
case $AUTH_METHOD in
    sso)
        log_info "Authenticating via SSO..."
        # Example: Get SSO token
        if command -v company-sso-auth &> /dev/null; then
            ENTERPRISE_API_KEY=$(company-sso-auth get-token --service claude 2>/dev/null)
            if [[ -z "$ENTERPRISE_API_KEY" ]]; then
                log_error "SSO authentication failed"
                log_info "Run: company-sso-auth login"
                exit 1
            fi
            log_success "SSO authentication successful"
        else
            log_error "SSO auth command not found: company-sso-auth"
            exit 1
        fi
        ;;
    token)
        log_info "Using token authentication..."
        if [[ -z "$ENTERPRISE_API_KEY" ]]; then
            log_error "ENTERPRISE_API_KEY not set"
            log_info "Set: export ENTERPRISE_API_KEY='your-key'"
            exit 1
        fi
        ;;
    cert)
        log_info "Using certificate authentication..."
        # Example: Certificate-based auth
        CERT_FILE="$ENTERPRISE_CONFIG_DIR/certs/client.crt"
        KEY_FILE="$ENTERPRISE_CONFIG_DIR/certs/client.key"
        if [[ ! -f "$CERT_FILE" ]] || [[ ! -f "$KEY_FILE" ]]; then
            log_error "Certificate files not found"
            exit 1
        fi
        ;;
    *)
        log_error "Unknown auth method: $AUTH_METHOD"
        exit 1
        ;;
esac

# Set up proxy
if [[ -n "$ENTERPRISE_PROXY" ]]; then
    export HTTP_PROXY="$ENTERPRISE_PROXY"
    export HTTPS_PROXY="$ENTERPRISE_PROXY"
    log_info "Using proxy: $ENTERPRISE_PROXY"
fi

# Set up API endpoint
export CLAUDE_API_ENDPOINT="$ENTERPRISE_API_ENDPOINT"
log_info "API endpoint: $ENTERPRISE_API_ENDPOINT"

# Add team tracking header if specified
if [[ -n "$TEAM_ID" ]]; then
    log_info "Team: $TEAM_ID"
    # Example: Add custom header
    export CLAUDE_TEAM_HEADER="X-Team-ID: $TEAM_ID"
fi

# Check if Claude Code is installed
if ! command -v claude &> /dev/null; then
    log_error "Claude Code CLI not found"
    log_info "Install: npm install -g @anthropic-ai/claude-code"
    exit 1
fi

# Log usage metrics (optional)
if command -v company-metrics &> /dev/null; then
    company-metrics log \
        --service claude \
        --user "$(whoami)" \
        --team "${TEAM_ID:-unknown}" \
        --env "$CONFIG_ENV" \
        &>/dev/null || true
fi

# Execute Claude Code with enterprise settings
log_info "Executing Claude Code..."

exec claude \
    --api-key "$ENTERPRISE_API_KEY" \
    --endpoint "$CLAUDE_API_ENDPOINT" \
    "${CLAUDE_ARGS[@]}"
