#!/bin/bash
#
# build.sh - Morty Build Script
# Compiles Go source code into executable binary
#

set -e

# Default values
OUTPUT="./bin/morty"
VERSION="dev"
TARGET_OS=""
TARGET_ARCH=""
MAIN_PACKAGE="github.com/morty/morty/cmd/morty"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Task 1: Detect Go environment
detect_go() {
    log_info "Task 1: Detecting Go environment..."

    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
    GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)

    log_info "Found Go version: $GO_VERSION"

    # Check Go >= 1.21
    if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]); then
        log_error "Go version $GO_VERSION is too old. Required: >= 1.21"
        exit 1
    fi

    log_success "Go version $GO_VERSION meets requirement (>= 1.21)"
}

# Task 2: Parse build parameters
parse_args() {
    log_info "Task 2: Parsing build parameters..."

    while [[ $# -gt 0 ]]; do
        case $1 in
            --output)
                OUTPUT="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --os)
                TARGET_OS="$2"
                shift 2
                ;;
            --arch)
                TARGET_ARCH="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_warn "Unknown option: $1"
                shift
                ;;
        esac
    done

    log_info "Build parameters:"
    log_info "  Output: $OUTPUT"
    log_info "  Version: $VERSION"
    if [ -n "$TARGET_OS" ]; then
        log_info "  Target OS: $TARGET_OS"
    fi
    if [ -n "$TARGET_ARCH" ]; then
        log_info "  Target Arch: $TARGET_ARCH"
    fi
}

show_help() {
    cat << EOF
Morty Build Script

Usage: ./scripts/build.sh [options]

Options:
    --output PATH      Output binary path (default: ./bin/morty)
    --version VERSION  Build version (default: dev)
    --os OS            Target operating system (linux, darwin, windows)
    --arch ARCH        Target architecture (amd64, arm64)
    --help             Show this help message

Examples:
    ./scripts/build.sh                          # Default build
    ./scripts/build.sh --output ./bin/morty     # Specify output path
    ./scripts/build.sh --version 2.0.0          # Specify version
    ./scripts/build.sh --os linux --arch arm64  # Cross compile for Linux ARM64
EOF
}

# Task 3: Execute go mod tidy
tidy_deps() {
    log_info "Task 3: Running go mod tidy..."

    cd "$(dirname "$0")/.."

    if [ ! -f "go.mod" ]; then
        log_error "go.mod not found. Are you in the right directory?"
        exit 1
    fi

    go mod tidy
    log_success "Dependencies tidied"
}

# Task 4 & 5: Execute go build with version injection
build_binary() {
    log_info "Task 4 & 5: Building binary with version injection..."

    # Create output directory
    mkdir -p "$(dirname "$OUTPUT")"

    # Get git commit hash
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

    # Get build time
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Build ldflags for version injection
    LDFLAGS="-X main.Version=$VERSION"
    LDFLAGS="$LDFLAGS -X main.GitCommit=$GIT_COMMIT"
    LDFLAGS="$LDFLAGS -X main.BuildTime=$BUILD_TIME"

    log_info "Version info to inject:"
    log_info "  Version: $VERSION"
    log_info "  GitCommit: $GIT_COMMIT"
    log_info "  BuildTime: $BUILD_TIME"

    # Set cross-compilation environment if specified
    if [ -n "$TARGET_OS" ]; then
        export GOOS="$TARGET_OS"
    fi
    if [ -n "$TARGET_ARCH" ]; then
        export GOARCH="$TARGET_ARCH"
    fi

    # Build
    log_info "Building: $MAIN_PACKAGE"
    log_info "Output: $OUTPUT"

    go build -ldflags "$LDFLAGS" -o "$OUTPUT" "$MAIN_PACKAGE"

    log_success "Build completed"
}

# Task 6: Verify compilation results
verify_build() {
    log_info "Task 6: Verifying build results..."

    # Check file exists
    if [ ! -f "$OUTPUT" ]; then
        log_error "Binary not found at $OUTPUT"
        exit 1
    fi
    log_success "Binary file exists: $OUTPUT"

    # Check file is executable
    if [ ! -x "$OUTPUT" ]; then
        log_error "Binary is not executable"
        exit 1
    fi
    log_success "Binary is executable"

    # Check file size
    FILE_SIZE=$(stat -c%s "$OUTPUT" 2>/dev/null || stat -f%z "$OUTPUT" 2>/dev/null)
    log_info "Binary size: $FILE_SIZE bytes"

    # Test version output
    VERSION_OUTPUT=$("$OUTPUT" -version 2>&1)
    if [ $? -ne 0 ]; then
        log_error "Binary execution failed"
        exit 1
    fi

    # Verify version info is injected
    if echo "$VERSION_OUTPUT" | grep -q "$VERSION"; then
        log_success "Version info correctly injected"
    else
        log_warn "Version info may not be correctly injected"
    fi
}

# Task 6.5: Copy prompts directory
copy_prompts() {
    log_info "Task 6.5: Copying prompt templates..."

    # Determine prompts destination (next to binary)
    local OUTPUT_DIR=$(dirname "$OUTPUT")
    local PROMPTS_DEST="${OUTPUT_DIR}/prompts"

    # Check if prompts source exists
    if [ ! -d "./prompts" ]; then
        log_warn "Prompts directory not found, skipping..."
        return 0
    fi

    # Create prompts directory and copy files
    mkdir -p "$PROMPTS_DEST"
    cp -r ./prompts/* "$PROMPTS_DEST/" 2>/dev/null || true

    # Verify key prompt files
    local prompt_count=0
    for prompt in research.md plan.md doing.md; do
        if [ -f "${PROMPTS_DEST}/${prompt}" ]; then
            ((prompt_count++))
        fi
    done

    if [ $prompt_count -gt 0 ]; then
        log_success "Copied ${prompt_count} prompt templates to ${PROMPTS_DEST}"
    else
        log_warn "No prompt files copied"
    fi
}

# Task 7: Output build information
output_info() {
    log_info "Task 7: Build information..."

    echo ""
    echo "========================================"
    echo "           Build Summary"
    echo "========================================"
    echo "Binary:     $OUTPUT"
    echo "Version:    $VERSION"
    echo "Git Commit: $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
    echo "Build Time: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    echo "Go Version: $(go version | awk '{print $3}')"
    echo "OS/Arch:    ${TARGET_OS:-$(go env GOOS)}/${TARGET_ARCH:-$(go env GOARCH)}"
    echo ""
    echo "File Info:"
    ls -lh "$OUTPUT"
    echo ""
    echo "Version Output:"
    "$OUTPUT" -version
    echo "========================================"
    log_success "Build completed successfully!"
}

# Main execution
main() {
    echo "========================================"
    echo "      Morty Build Script"
    echo "========================================"
    echo ""

    parse_args "$@"
    detect_go
    tidy_deps
    build_binary
    verify_build
    copy_prompts
    output_info
}

main "$@"
