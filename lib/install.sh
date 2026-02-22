#!/usr/bin/env bash
#
# install.sh - Installation module for Morty
#
# Provides dependency checking, installation path management,
# and installation/upgrade/uninstall functionality.
#

# Prevent duplicate loading
[[ -n "${_INSTALL_SH_LOADED:-}" ]] && return 0
_INSTALL_SH_LOADED=1

# Get script directory
_INSTALL_SH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required modules
source "${_INSTALL_SH_DIR}/common.sh"
source "${_INSTALL_SH_DIR}/logging.sh"

# ============================================
# Configuration
# ============================================

# Minimum required versions
readonly INSTALL_MIN_BASH_VERSION="4.0"
readonly INSTALL_MIN_GIT_VERSION="2.0"

# Default installation paths
readonly INSTALL_DEFAULT_PREFIX="${HOME}/.morty"
readonly INSTALL_DEFAULT_BIN_DIR="${HOME}/.local/bin"

# AI CLI command (can be overridden via config)
INSTALL_AI_CLI="${MORTY_AI_CLI:-claude}"

# ============================================
# Result Codes
# ============================================

readonly INSTALL_STATUS_PASS=0
readonly INSTALL_STATUS_FAIL=1
readonly INSTALL_STATUS_WARN=2

# ============================================
# Dependency Checking
# ============================================

# Check all required dependencies
# Returns: 0 if all required deps pass, 1 otherwise
# Outputs: JSON-like structured result to stdout
install_check_deps() {
    local results=()
    local all_passed=true

    log_info "Checking dependencies..."

    # Check Bash version
    local bash_result
    bash_result=$(install_check_bash_version)
    results+=("$bash_result")
    if [[ "$(echo "$bash_result" | jq -r '.status')" == "FAIL" ]]; then
        all_passed=false
    fi

    # Check Git version
    local git_result
    git_result=$(install_check_git_version)
    results+=("$git_result")
    if [[ "$(echo "$git_result" | jq -r '.status')" == "FAIL" ]]; then
        all_passed=false
    fi

    # Check AI CLI
    local ai_result
    ai_result=$(install_check_ai_cli)
    results+=("$ai_result")
    if [[ "$(echo "$ai_result" | jq -r '.status')" == "FAIL" ]]; then
        all_passed=false
    fi

    # Check optional dependencies
    local optional_result
    optional_result=$(install_check_optional_deps)
    results+=("$optional_result")

    # Output combined results - flatten array, optional deps are separate entries
    local json_results="["
    local first=true
    for result in "${results[@]}"; do
        # Check if result is an array (optional deps)
        if [[ "$result" == \[* ]]; then
            # Extract items from array and add them individually
            local items=$(echo "$result" | jq -c '.[]' 2>/dev/null)
            while IFS= read -r item; do
                if [[ -n "$item" ]]; then
                    if [[ "$first" == true ]]; then
                        first=false
                    else
                        json_results+=","
                    fi
                    json_results+="$item"
                fi
            done <<< "$items"
        else
            if [[ "$first" == true ]]; then
                first=false
            else
                json_results+=","
            fi
            json_results+="$result"
        fi
    done
    json_results+="]"

    echo "$json_results"

    if [[ "$all_passed" == true ]]; then
        return 0
    else
        return 1
    fi
}

# Check Bash version
# Returns: JSON object with check results
install_check_bash_version() {
    local status="PASS"
    local message=""
    local version=""

    # Get Bash version (e.g., "5.1.16")
    version="${BASH_VERSION%%(*}"
    version="${version%%-*}"

    # Compare versions
    if install_compare_versions "$version" "$INSTALL_MIN_BASH_VERSION"; then
        # version >= min_version
        status="PASS"
        message="Bash version $version is supported (>= $INSTALL_MIN_BASH_VERSION)"
    else
        status="FAIL"
        message="Bash version $version is too old. Required: >= $INSTALL_MIN_BASH_VERSION"
    fi

    # Output JSON result
    cat <<EOF
{
  "name": "bash",
  "status": "$status",
  "version": "$version",
  "required": ">= $INSTALL_MIN_BASH_VERSION",
  "message": "$message"
}
EOF
}

# Check Git version
# Returns: JSON object with check results
install_check_git_version() {
    local status="PASS"
    local message=""
    local version=""

    # Check if git is installed
    if ! command -v git &>/dev/null; then
        status="FAIL"
        message="Git is not installed. Please install Git >= $INSTALL_MIN_GIT_VERSION"
        version=""
    else
        # Get Git version (e.g., "2.34.1")
        version=$(git --version 2>/dev/null | awk '{print $3}')

        if install_compare_versions "$version" "$INSTALL_MIN_GIT_VERSION"; then
            status="PASS"
            message="Git version $version is supported (>= $INSTALL_MIN_GIT_VERSION)"
        else
            status="FAIL"
            message="Git version $version is too old. Required: >= $INSTALL_MIN_GIT_VERSION"
        fi
    fi

    # Output JSON result
    cat <<EOF
{
  "name": "git",
  "status": "$status",
  "version": "${version:-null}",
  "required": ">= $INSTALL_MIN_GIT_VERSION",
  "message": "$message"
}
EOF
}

# Check AI CLI (Claude Code)
# Returns: JSON object with check results
install_check_ai_cli() {
    local status="PASS"
    local message=""
    local version=""
    local ai_cmd="$INSTALL_AI_CLI"

    # Check if AI CLI is installed
    if ! command -v "$ai_cmd" &>/dev/null; then
        status="FAIL"
        message="Claude Code CLI ($ai_cmd) is not found. Please install Claude Code: https://claude.ai/code"
        version=""
    else
        # Try to get version (claude --version)
        # Expected format: "2.1.50 (Claude Code)" or "Claude Code 2.1.50"
        local raw_version=$($ai_cmd --version 2>/dev/null | head -1)
        if [[ "$raw_version" =~ ^[0-9]+\.[0-9]+ ]]; then
            # Format: "2.1.50 (Claude Code)" - extract version number
            version=$(echo "$raw_version" | awk '{print $1}')
        elif [[ "$raw_version" =~ Claude[[:space:]]Code[[:space:]][0-9]+\.[0-9]+ ]]; then
            # Format: "Claude Code 2.1.50" - extract last field
            version=$(echo "$raw_version" | awk '{print $NF}')
        else
            version="unknown"
        fi
        status="PASS"
        message="Claude Code CLI is available ($ai_cmd)"
    fi

    # Output JSON result
    cat <<EOF
{
  "name": "ai_cli",
  "status": "$status",
  "version": "${version:-null}",
  "required": "required",
  "command": "$ai_cmd",
  "message": "$message"
}
EOF
}

# Check optional dependencies
# Returns: JSON object with check results (array)
install_check_optional_deps() {
    local optional_deps=("jq" "tmux")
    local results=()

    for dep in "${optional_deps[@]}"; do
        local status="WARN"
        local message=""
        local version=""

        if command -v "$dep" &>/dev/null; then
            status="PASS"
            case "$dep" in
                jq)
                    version=$(jq --version 2>/dev/null | head -1)
                    message="jq is available (JSON processing enhanced)"
                    ;;
                tmux)
                    version=$(tmux -V 2>/dev/null | head -1)
                    message="tmux is available (loop monitoring mode available)"
                    ;;
            esac
        else
            status="WARN"
            version=""
            case "$dep" in
                jq)
                    message="jq is not installed. JSON processing will be limited (optional)"
                    ;;
                tmux)
                    message="tmux is not installed. Loop monitoring mode unavailable (optional)"
                    ;;
            esac
        fi

        # Build individual result
        results+=("{\"name\": \"$dep\", \"status\": \"$status\", \"version\": \"${version:-null}\", \"message\": \"$message\"}")
    done

    # Output JSON array - join with commas
    local json_array=""
    local first=true
    for result in "${results[@]}"; do
        if [[ "$first" == true ]]; then
            first=false
            json_array="$result"
        else
            json_array="$json_array, $result"
        fi
    done
    echo "[$json_array]"
}

# ============================================
# Installation Guidance
# ============================================

# Print friendly installation guidance for missing dependencies
# Usage: install_print_guidance <json_check_results>
install_print_guidance() {
    local results="$1"

    echo ""
    log_error "Dependency check failed!"
    echo ""

    # Parse and display each failed dependency
    local count=$(echo "$results" | jq '. | length')
    local has_failed=false

    for ((i=0; i<count; i++)); do
        local item=$(echo "$results" | jq ".[$i]")
        local name=$(echo "$item" | jq -r '.name')
        local status=$(echo "$item" | jq -r '.status')
        local message=$(echo "$item" | jq -r '.message')

        if [[ "$status" == "FAIL" ]]; then
            has_failed=true
            echo -e "\033[0;31m✗ $name\033[0m"
            echo "  $message"
            echo ""

            # Print specific installation guidance
            case "$name" in
                bash)
                    echo "  To upgrade Bash:"
                    echo "    macOS:   brew install bash"
                    echo "    Ubuntu:  sudo apt-get install bash"
                    echo "    CentOS:  sudo yum install bash"
                    echo ""
                    ;;
                git)
                    echo "  To install Git:"
                    echo "    macOS:   brew install git"
                    echo "    Ubuntu:  sudo apt-get install git"
                    echo "    CentOS:  sudo yum install git"
                    echo ""
                    echo "  Or download from: https://git-scm.com/downloads"
                    echo ""
                    ;;
                ai_cli)
                    echo "  To install Claude Code:"
                    echo "    Visit: https://claude.ai/code"
                    echo ""
                    echo "  Or use an alternative AI CLI and configure it:"
                    echo "    export MORTY_AI_CLI=<your-cli>"
                    echo ""
                    ;;
            esac
        elif [[ "$status" == "WARN" ]]; then
            echo -e "\033[1;33m⚠ $name (optional)\033[0m"
            echo "  $message"
            echo ""
        fi
    done

    # Print optional dependency guidance
    echo ""
    echo "Optional dependencies (installation recommended):"
    echo "  jq:    brew install jq    (or apt-get install jq)"
    echo "  tmux:  brew install tmux  (or apt-get install tmux)"
    echo ""
}

# Print structured check results
# Usage: install_print_results <json_check_results>
install_print_results() {
    local results="$1"

    echo ""
    echo "Dependency Check Results:"
    echo "========================"
    echo ""

    local count=$(echo "$results" | jq '. | length')

    for ((i=0; i<count; i++)); do
        local item=$(echo "$results" | jq ".[$i]")
        local name=$(echo "$item" | jq -r '.name')
        local status=$(echo "$item" | jq -r '.status')
        local version=$(echo "$item" | jq -r '.version // empty')
        local message=$(echo "$item" | jq -r '.message')

        case "$status" in
            PASS)
                echo -e "\033[0;32m✓ $name\033[0m ${version:+($version)}"
                ;;
            FAIL)
                echo -e "\033[0;31m✗ $name\033[0m"
                echo "  $message"
                ;;
            WARN)
                echo -e "\033[1;33m⚠ $name (optional)\033[0m"
                echo "  $message"
                ;;
        esac
    done

    echo ""
}

# ============================================
# Path Management
# ============================================

# Get default installation prefix
# Returns: path to default prefix
install_get_default_prefix() {
    echo "$INSTALL_DEFAULT_PREFIX"
}

# Get default binary directory
# Returns: path to default bin directory
install_get_default_bin_dir() {
    echo "$INSTALL_DEFAULT_BIN_DIR"
}

# Validate installation prefix
# Usage: install_validate_prefix <path>
# Returns: 0 if valid, 1 otherwise
install_validate_prefix() {
    local prefix="$1"

    if [[ -z "$prefix" ]]; then
        log_error "Installation path cannot be empty"
        return 1
    fi

    # Expand ~ to $HOME
    prefix="${prefix/#\~/$HOME}"

    # Check if path is absolute
    if [[ ! "$prefix" =~ ^/ ]]; then
        log_error "Installation path must be absolute: $prefix"
        return 1
    fi

    # Check if parent directory exists and is writable
    local parent_dir=$(dirname "$prefix")
    if [[ ! -d "$parent_dir" ]]; then
        log_error "Parent directory does not exist: $parent_dir"
        return 1
    fi

    if [[ ! -w "$parent_dir" ]]; then
        log_error "Parent directory is not writable: $parent_dir"
        return 1
    fi

    return 0
}

# Check for existing installation
# Usage: install_check_existing <prefix>
# Returns: 0 if no existing installation, 1 if exists
# Outputs: JSON object with installation status details
install_check_existing() {
    local prefix="$1"
    prefix="${prefix/#\~/$HOME}"

    local exists="false"
    local has_bin="false"
    local has_lib="false"
    local has_prompts="false"
    local has_version="false"
    local version=""
    local install_time=""

    if [[ -d "$prefix" ]]; then
        exists="true"

        # Check for required subdirectories
        [[ -d "$prefix/bin" ]] && has_bin="true"
        [[ -d "$prefix/lib" ]] && has_lib="true"
        [[ -d "$prefix/prompts" ]] && has_prompts="true"

        # Check for version file
        if [[ -f "$prefix/VERSION" ]]; then
            has_version="true"
            version=$(cat "$prefix/VERSION" 2>/dev/null | head -1)
        fi

        # Get installation time (directory modification time)
        if [[ -d "$prefix/bin" ]]; then
            install_time=$(stat -c %Y "$prefix/bin" 2>/dev/null || stat -f %m "$prefix/bin" 2>/dev/null)
        fi
    fi

    # Output JSON result
    cat <<EOF
{
  "exists": $exists,
  "prefix": "$prefix",
  "has_bin": $has_bin,
  "has_lib": $has_lib,
  "has_prompts": $has_prompts,
  "has_version": $has_version,
  "version": "${version:-null}",
  "install_time": ${install_time:-null},
  "is_complete": $([[ "$has_bin" == "true" && "$has_lib" == "true" && "$has_prompts" == "true" ]] && echo "true" || echo "false")
}
EOF

    if [[ "$exists" == "true" ]]; then
        return 1
    fi

    return 0
}

# Ensure installation directories exist
# Usage: install_ensure_dirs <prefix>
# Returns: 0 on success, 1 on failure
install_ensure_dirs() {
    local prefix="$1"
    prefix="${prefix/#\~/$HOME}"

    local dirs=("bin" "lib" "prompts")
    local created_dirs=()

    for dir in "${dirs[@]}"; do
        local full_path="$prefix/$dir"
        if [[ ! -d "$full_path" ]]; then
            mkdir -p "$full_path" || {
                log_error "Failed to create directory: $full_path"
                # Cleanup partially created directories
                for created in "${created_dirs[@]}"; do
                    [[ -d "$created" ]] && rmdir "$created" 2>/dev/null
                done
                return 1
            }
            created_dirs+=("$full_path")
        fi
    done

    return 0
}

# ============================================
# Conflict Handling
# ============================================

# Backup existing installation
# Usage: install_backup_existing <prefix>
# Returns: 0 on success, 1 on failure
# Outputs: Path to backup directory
install_backup_existing() {
    local prefix="$1"
    prefix="${prefix/#\~/$HOME}"

    if [[ ! -d "$prefix" ]]; then
        log_warn "No existing installation to backup at $prefix"
        echo ""
        return 0
    fi

    # Create backup with timestamp
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_dir="${prefix}.backup.${timestamp}"

    # Check if version file exists to include in backup name
    if [[ -f "$prefix/VERSION" ]]; then
        local version=$(cat "$prefix/VERSION" 2>/dev/null | head -1 | tr -d '[:space:]')
        if [[ -n "$version" ]]; then
            backup_dir="${prefix}.backup.${version}.${timestamp}"
        fi
    fi

    # Copy existing installation to backup
    cp -r "$prefix" "$backup_dir" || {
        log_error "Failed to create backup at $backup_dir"
        return 1
    }

    log_info "Created backup: $backup_dir"
    echo "$backup_dir"
    return 0
}

# Handle installation conflict (existing installation found)
# Usage: install_handle_conflict <prefix> <force> [action]
#   prefix: installation path
#   force: "true" to force overwrite (backup first), "false" to prompt
#   action: optional - "backup", "overwrite", "cancel"
# Returns: 0 to proceed, 1 to cancel
install_handle_conflict() {
    local prefix="$1"
    local force="${2:-false}"
    local action="${3:-}"

    prefix="${prefix/#\~/$HOME}"

    # Check if installation exists
    local check_result
    check_result=$(install_check_existing "$prefix")
    local exists=$(echo "$check_result" | jq -r '.exists')

    if [[ "$exists" != "true" ]]; then
        # No existing installation, proceed
        return 0
    fi

    local version=$(echo "$check_result" | jq -r '.version // "unknown"')
    local is_complete=$(echo "$check_result" | jq -r '.is_complete')

    log_warn "Existing installation found at $prefix"

    if [[ "$version" != "null" && "$version" != "unknown" ]]; then
        log_info "Version: $version"
    fi

    if [[ "$force" == "true" ]]; then
        # Force mode: backup and proceed
        log_info "Force mode enabled - creating backup before overwrite"
        local backup_dir
        backup_dir=$(install_backup_existing "$prefix")
        if [[ $? -ne 0 ]]; then
            log_error "Failed to create backup, aborting"
            return 1
        fi
        log_info "Backup created at: $backup_dir"

        # Remove existing installation
        rm -rf "$prefix" || {
            log_error "Failed to remove existing installation"
            return 1
        }

        return 0
    fi

    # Non-force mode: check action parameter or prompt
    if [[ -n "$action" ]]; then
        case "$action" in
            backup)
                local backup_dir
                backup_dir=$(install_backup_existing "$prefix")
                if [[ $? -ne 0 ]]; then
                    return 1
                fi
                rm -rf "$prefix" || {
                    log_error "Failed to remove existing installation"
                    return 1
                }
                return 0
                ;;
            overwrite)
                rm -rf "$prefix" || {
                    log_error "Failed to remove existing installation"
                    return 1
                }
                return 0
                ;;
            cancel)
                log_info "Installation cancelled by user"
                return 1
                ;;
            *)
                log_error "Unknown action: $action"
                return 1
                ;;
        esac
    fi

    # No action specified and not in force mode - return error with guidance
    log_error "Installation already exists at $prefix"
    echo ""
    echo "Options:"
    echo "  1. Use --force to backup and overwrite"
    echo "  2. Use --action backup   to backup then overwrite"
    echo "  3. Use --action overwrite to overwrite without backup"
    echo "  4. Use --action cancel   to cancel installation"
    echo "  5. Choose a different --prefix"
    echo ""

    return 1
}

# ============================================
# File Installation
# ============================================

# Copy all necessary files to target directory
# Usage: install_copy_files <source_dir> <target_dir>
# Returns: 0 on success, 1 on failure
install_copy_files() {
    local source_dir="$1"
    local target_dir="$2"

    if [[ -z "$source_dir" || -z "$target_dir" ]]; then
        log_error "Source and target directories are required"
        return 1
    fi

    # Expand ~ to $HOME
    source_dir="${source_dir/#\~/$HOME}"
    target_dir="${target_dir/#\~/$HOME}"

    # Validate source directory exists
    if [[ ! -d "$source_dir" ]]; then
        log_error "Source directory does not exist: $source_dir"
        return 1
    fi

    # Validate target directory exists (should be created by install_ensure_dirs)
    if [[ ! -d "$target_dir" ]]; then
        log_error "Target directory does not exist: $target_dir"
        return 1
    fi

    log_info "Copying files from $source_dir to $target_dir..."

    # Define files to copy to bin/
    local bin_scripts=(
        "morty_fix.sh"
        "morty_loop.sh"
        "morty_reset.sh"
        "morty_research.sh"
        "morty_plan.sh"
        "morty_doing.sh"
    )

    # Copy bin scripts
    local bin_dir="$target_dir/bin"
    if [[ ! -d "$bin_dir" ]]; then
        mkdir -p "$bin_dir" || {
            log_error "Failed to create bin directory: $bin_dir"
            return 1
        }
    fi

    for script in "${bin_scripts[@]}"; do
        local src="$source_dir/$script"
        local dst="$bin_dir/$script"

        if [[ -f "$src" ]]; then
            cp "$src" "$dst" || {
                log_error "Failed to copy $script to $bin_dir"
                return 1
            }
            log_debug "Copied $script to $bin_dir"
        else
            log_warn "Source file not found: $src"
        fi
    done

    # Create main morty command
    install_create_main_command "$bin_dir" || {
        log_error "Failed to create main morty command"
        return 1
    }

    # Copy lib files
    local lib_src="$source_dir/lib"
    local lib_dst="$target_dir/lib"

    if [[ -d "$lib_src" ]]; then
        # Create lib directory if not exists
        if [[ ! -d "$lib_dst" ]]; then
            mkdir -p "$lib_dst" || {
                log_error "Failed to create lib directory: $lib_dst"
                return 1
            }
        fi

        # Copy all .sh files from lib/
        for src_file in "$lib_src"/*.sh; do
            if [[ -f "$src_file" ]]; then
                local filename=$(basename "$src_file")
                cp "$src_file" "$lib_dst/$filename" || {
                    log_error "Failed to copy lib/$filename"
                    return 1
                }
                log_debug "Copied lib/$filename"
            fi
        done
    else
        log_warn "Source lib directory not found: $lib_src"
    fi

    # Copy prompts files
    local prompts_src="$source_dir/prompts"
    local prompts_dst="$target_dir/prompts"

    if [[ -d "$prompts_src" ]]; then
        # Create prompts directory if not exists
        if [[ ! -d "$prompts_dst" ]]; then
            mkdir -p "$prompts_dst" || {
                log_error "Failed to create prompts directory: $prompts_dst"
                return 1
            }
        fi

        # Copy all .md files from prompts/
        for src_file in "$prompts_src"/*.md; do
            if [[ -f "$src_file" ]]; then
                local filename=$(basename "$src_file")
                cp "$src_file" "$prompts_dst/$filename" || {
                    log_error "Failed to copy prompts/$filename"
                    return 1
                }
                log_debug "Copied prompts/$filename"
            fi
        done
    else
        log_warn "Source prompts directory not found: $prompts_src"
    fi

    log_success "All files copied successfully"
    return 0
}

# Create main morty command script
# Usage: install_create_main_command <bin_dir>
# Returns: 0 on success, 1 on failure
install_create_main_command() {
    local bin_dir="$1"

    if [[ -z "$bin_dir" ]]; then
        log_error "Bin directory is required"
        return 1
    fi

    bin_dir="${bin_dir/#\~/$HOME}"

    if [[ ! -d "$bin_dir" ]]; then
        log_error "Bin directory does not exist: $bin_dir"
        return 1
    fi

    local morty_cmd="$bin_dir/morty"

    cat > "$morty_cmd" << 'EOF'
#!/usr/bin/env bash
# Morty - 简化的 AI 开发循环
# Main command wrapper

MORTY_HOME="${MORTY_HOME:-$HOME/.morty}"
VERSION_FILE="$MORTY_HOME/VERSION"

# Get version
if [[ -f "$VERSION_FILE" ]]; then
    VERSION=$(head -1 "$VERSION_FILE")
else
    VERSION="unknown"
fi

# Source common functions
source "$MORTY_HOME/lib/common.sh" 2>/dev/null || true
source "$MORTY_HOME/lib/logging.sh" 2>/dev/null || true

# Show help
show_help() {
    cat << 'HELP'
Morty - 简化的 AI 开发循环

用法: morty <command> [options]

命令:
    research [topic]        交互式代码库/文档库研究
    plan                    基于研究结果创建 TDD 开发计划
    doing [options]         执行 Plan 的分层 TDD 开发
    fix <prd.md>            迭代式 PRD 改进(问题修复/功能增强/架构优化)
    loop [options]          启动开发循环(集成监控)
    reset [options]         版本回滚和循环管理
    version                 显示版本

示例:
    morty research                     # 启动研究模式
    morty research "api架构"           # 研究指定主题
    morty plan                         # 基于研究结果创建 TDD 计划
    morty doing                        # 执行分层 TDD 开发
    morty fix prd.md                   # 改进 PRD 并生成 .morty/ 目录
    morty loop                         # 启动带监控的开发循环
    morty reset -l                     # 查看循环提交历史
    morty reset -c abc123              # 回滚到指定 commit

HELP
}

# Show version
show_version() {
    echo "Morty version $VERSION"
}

# Command routing
case "${1:-}" in
    research)
        shift
        exec "$MORTY_HOME/bin/morty_research.sh" "$@"
        ;;
    plan)
        shift
        exec "$MORTY_HOME/bin/morty_plan.sh" "$@"
        ;;
    doing)
        shift
        exec "$MORTY_HOME/bin/morty_doing.sh" "$@"
        ;;
    fix)
        shift
        exec "$MORTY_HOME/bin/morty_fix.sh" "$@"
        ;;
    loop)
        shift
        exec "$MORTY_HOME/bin/morty_loop.sh" "$@"
        ;;
    reset)
        shift
        exec "$MORTY_HOME/bin/morty_reset.sh" "$@"
        ;;
    version|--version|-v)
        show_version
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo "错误: 未知命令 '$1'" >&2
        echo ""
        show_help
        exit 1
        ;;
esac
EOF

    chmod +x "$morty_cmd" || {
        log_error "Failed to set permissions on morty command"
        return 1
    }

    log_debug "Created main morty command at $morty_cmd"
    return 0
}

# ============================================
# Permission Management
# ============================================

# Set executable permissions on all installed files
# Usage: install_set_permissions <prefix>
# Returns: 0 on success, 1 on failure
install_set_permissions() {
    local prefix="$1"

    if [[ -z "$prefix" ]]; then
        log_error "Installation prefix is required"
        return 1
    fi

    prefix="${prefix/#\~/$HOME}"

    if [[ ! -d "$prefix" ]]; then
        log_error "Installation directory does not exist: $prefix"
        return 1
    fi

    log_info "Setting file permissions..."

    # Set permissions on bin scripts (755)
    local bin_dir="$prefix/bin"
    if [[ -d "$bin_dir" ]]; then
        chmod -R 755 "$bin_dir" 2>/dev/null || {
            log_error "Failed to set permissions on $bin_dir"
            return 1
        }
    fi

    # Set permissions on lib scripts (644 for files, 755 for directories)
    local lib_dir="$prefix/lib"
    if [[ -d "$lib_dir" ]]; then
        find "$lib_dir" -type f -name "*.sh" -exec chmod 644 {} \; 2>/dev/null || true
        chmod 755 "$lib_dir" 2>/dev/null || true
    fi

    # Set permissions on prompts (644)
    local prompts_dir="$prefix/prompts"
    if [[ -d "$prompts_dir" ]]; then
        find "$prompts_dir" -type f -name "*.md" -exec chmod 644 {} \; 2>/dev/null || true
        chmod 755 "$prompts_dir" 2>/dev/null || true
    fi

    log_debug "File permissions set"
    return 0
}

# ============================================
# Symlink Management
# ============================================

# Create symlink for morty command
# Usage: install_create_symlink <target> <link_name>
# Returns: 0 on success, 1 on failure
install_create_symlink() {
    local target="$1"
    local link_name="$2"

    if [[ -z "$target" || -z "$link_name" ]]; then
        log_error "Target and link name are required"
        return 1
    fi

    target="${target/#\~/$HOME}"
    link_name="${link_name/#\~/$HOME}"

    if [[ ! -f "$target" ]]; then
        log_error "Target does not exist: $target"
        return 1
    fi

    # Create parent directory if needed
    local link_parent=$(dirname "$link_name")
    if [[ ! -d "$link_parent" ]]; then
        mkdir -p "$link_parent" || {
            log_error "Failed to create parent directory: $link_parent"
            return 1
        }
    fi

    # Remove existing symlink if it exists
    if [[ -L "$link_name" ]]; then
        rm "$link_name" || {
            log_error "Failed to remove existing symlink: $link_name"
            return 1
        }
    fi

    # Remove existing file if it exists
    if [[ -f "$link_name" ]]; then
        rm "$link_name" || {
            log_error "Failed to remove existing file: $link_name"
            return 1
        }
    fi

    # Create symlink
    ln -s "$target" "$link_name" || {
        log_error "Failed to create symlink: $link_name -> $target"
        return 1
    }

    log_info "Created symlink: $link_name -> $target"
    return 0
}

# ============================================
# Version Management
# ============================================

# Get the repository root directory
# Returns: path to repository root
install_get_repo_root() {
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local repo_root="$(dirname "$script_dir")"
    echo "$repo_root"
}

# Get the current version from the repository
# Returns: version string
install_get_current_version() {
    local repo_root
    repo_root=$(install_get_repo_root)

    # Try to get version from VERSION file
    if [[ -f "$repo_root/VERSION" ]]; then
        head -1 "$repo_root/VERSION"
        return 0
    fi

    # Try to get version from git
    if command -v git &>/dev/null && [[ -d "$repo_root/.git" ]]; then
        git -C "$repo_root" describe --tags --always 2>/dev/null || echo "2.0.0-dev"
        return 0
    fi

    # Default version
    echo "2.0.0"
}

# Write version file to installation directory
# Usage: install_write_version <prefix> <version>
# Returns: 0 on success, 1 on failure
install_write_version() {
    local prefix="$1"
    local version="$2"

    if [[ -z "$prefix" ]]; then
        log_error "Installation prefix is required"
        return 1
    fi

    prefix="${prefix/#\~/$HOME}"

    if [[ ! -d "$prefix" ]]; then
        log_error "Installation directory does not exist: $prefix"
        return 1
    fi

    # Write version file
    echo "${version:-2.0.0}" > "$prefix/VERSION" || {
        log_error "Failed to write version file"
        return 1
    }

    log_debug "Version file written: $version"
    return 0
}

# ============================================
# Configuration Initialization
# ============================================

# Initialize default configuration
# Usage: install_init_config() [prefix]
# Returns: 0 on success, 1 on failure
install_init_config() {
    local prefix="${1:-$INSTALL_DEFAULT_PREFIX}"
    prefix="${prefix/#\~/$HOME}"

    log_info "Initializing configuration..."

    # Create .morty work directory (for runtime state)
    local morty_dir="$prefix/.morty"
    if [[ ! -d "$morty_dir" ]]; then
        mkdir -p "$morty_dir" || {
            log_error "Failed to create .morty directory"
            return 1
        }
        log_debug "Created .morty directory: $morty_dir"
    fi

    # Create subdirectories
    local subdirs=("logs" "plan" "research" "doing")
    for subdir in "${subdirs[@]}"; do
        local full_path="$morty_dir/$subdir"
        if [[ ! -d "$full_path" ]]; then
            mkdir -p "$full_path" || {
                log_error "Failed to create directory: $full_path"
                return 1
            }
            log_debug "Created directory: $full_path"
        fi
    done

    # Initialize status.json if it doesn't exist
    local status_file="$morty_dir/status.json"
    if [[ ! -f "$status_file" ]]; then
        local timestamp
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        cat > "$status_file" <<EOF
{
  "version": "2.0",
  "state": "initialized",
  "current": {
    "module": null,
    "job": null,
    "status": null
  },
  "session": {
    "start_time": "$timestamp",
    "last_update": "$timestamp",
    "total_loops": 0
  },
  "modules": {},
  "summary": {
    "total_modules": 0,
    "completed_modules": 0,
    "running_modules": 0,
    "pending_modules": 0,
    "blocked_modules": 0,
    "total_jobs": 0,
    "completed_jobs": 0,
    "running_jobs": 0,
    "failed_jobs": 0,
    "blocked_jobs": 0,
    "progress_percentage": 0
  }
}
EOF
        if [[ $? -ne 0 ]]; then
            log_error "Failed to create status.json"
            return 1
        fi
        log_debug "Created status.json"
    fi

    # Create empty log file
    local log_file="$morty_dir/logs/morty.log"
    if [[ ! -f "$log_file" ]]; then
        touch "$log_file" || {
            log_warn "Failed to create morty.log"
        }
    fi

    log_debug "Configuration initialized successfully"
    return 0
}

# ============================================
# Version Utilities
# ============================================

# Compare two version strings
# Usage: install_compare_versions <version1> <version2>
# Returns: 0 if version1 >= version2, 1 otherwise
install_compare_versions() {
    local v1="$1"
    local v2="$2"

    # Normalize versions by padding with zeros
    local IFS=.
    local v1_parts=($v1)
    local v2_parts=($v2)

    local max_len=${#v1_parts[@]}
    if [[ ${#v2_parts[@]} -gt $max_len ]]; then
        max_len=${#v2_parts[@]}
    fi

    for ((i=0; i<max_len; i++)); do
        local p1=${v1_parts[$i]:-0}
        local p2=${v2_parts[$i]:-0}

        # Remove any non-numeric suffix
        p1=${p1%%[^0-9]*}
        p2=${p2%%[^0-9]*}

        if [[ $p1 -gt $p2 ]]; then
            return 0
        elif [[ $p1 -lt $p2 ]]; then
            return 1
        fi
    done

    # Versions are equal
    return 0
}

# ============================================
# Installation Execution (Stubs for future Jobs)
# ============================================

# Perform installation
# Usage: install_do_install <prefix> <bin_dir> [force=false]
install_do_install() {
    local prefix="${1:-$INSTALL_DEFAULT_PREFIX}"
    local bin_dir="${2:-$INSTALL_DEFAULT_BIN_DIR}"
    local force="${3:-false}"

    log_info "Installing Morty to $prefix..."

    # Check dependencies first
    local deps_result
    if ! deps_result=$(install_check_deps); then
        install_print_guidance "$deps_result"
        return 1
    fi

    # Check existing installation
    if ! install_check_existing "$prefix"; then
        if [[ "$force" != true ]]; then
            log_error "Installation already exists at $prefix"
            log_info "Use --force to overwrite"
            return 1
        fi
        log_warn "Overwriting existing installation at $prefix"
    fi

    # Create directories
    if ! install_ensure_dirs "$prefix"; then
        return 1
    fi

    log_success "Installation directories created"

    # Copy files from source to installation directory
    local source_dir
    source_dir=$(install_get_repo_root)
    if [[ $? -ne 0 || -z "$source_dir" ]]; then
        log_error "Failed to determine source directory"
        return 1
    fi

    if ! install_copy_files "$source_dir" "$prefix"; then
        log_error "Failed to copy files"
        return 1
    fi

    log_success "Files copied successfully"

    # Set permissions on all installed files
    if ! install_set_permissions "$prefix"; then
        log_error "Failed to set permissions"
        return 1
    fi

    log_success "Permissions set successfully"

    # Create symlink to bin directory
    local morty_cmd="$prefix/bin/morty"
    local symlink_path="$bin_dir/morty"

    if ! install_create_symlink "$morty_cmd" "$symlink_path"; then
        log_error "Failed to create symlink"
        return 1
    fi

    log_success "Symlink created successfully"

    # Write version file
    local version
    version=$(install_get_current_version)
    if ! install_write_version "$prefix" "$version"; then
        log_error "Failed to write version file"
        return 1
    fi

    log_success "Version file written"

    # Initialize configuration
    if ! install_init_config "$prefix"; then
        log_error "Failed to initialize configuration"
        return 1
    fi

    log_success "Configuration initialized"

    # Print PATH instructions
    install_print_path_instructions "$bin_dir"

    log_success "Installation completed successfully!"
    return 0
}

# ============================================
# PATH Management
# ============================================

# Check if a directory is in PATH
# Usage: install_path_contains <directory>
# Returns: 0 if in PATH, 1 otherwise
install_path_contains() {
    local dir="$1"
    [[ ":$PATH:" == *":$dir:"* ]]
}

# Print PATH setup instructions
# Usage: install_print_path_instructions <bin_dir>
install_print_path_instructions() {
    local bin_dir="$1"

    if install_path_contains "$bin_dir"; then
        return 0
    fi

    echo ""
    log_warn "$bin_dir is not in your PATH"
    echo ""
    echo "Add the following to your shell configuration:"
    echo ""

    local shell_rc=""
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_rc="~/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]]; then
        shell_rc="~/.bashrc"
    else
        shell_rc="your shell rc file"
    fi

    echo "  echo 'export PATH=\"$bin_dir:\$PATH\"' >> $shell_rc"
    echo ""
    echo "Then reload your shell configuration:"
    echo "  source $shell_rc"
    echo ""
}

# ============================================
# Module Initialization
# ============================================

log_debug "install.sh module loaded"
