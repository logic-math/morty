#!/bin/bash
# Config management module for Morty
# Provides unified configuration management through settings.json

# Load common utilities
MORTY_LIB_DIR="${MORTY_LIB_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
source "$MORTY_LIB_DIR/common.sh"

# Configuration file path (set by config_load)
MORTY_CONFIG_FILE=""

# Default configuration values
declare -A MORTY_CONFIG_DEFAULTS=(
    ["cli.command"]="claude"
    ["defaults.max_loops"]=50
    ["defaults.loop_delay"]=5
    ["defaults.log_level"]="INFO"
    ["defaults.stat_refresh_interval"]=60
    ["paths.work_dir"]=".morty"
    ["paths.log_dir"]=".morty/logs"
    ["paths.research_dir"]=".morty/research"
    ["paths.plan_dir"]=".morty/plan"
    ["paths.status_file"]=".morty/status.json"
)

# ============================================================================
# Path and Initialization
# ============================================================================

# Get MORTY_HOME environment variable
# Returns: path to MORTY_HOME or empty if not set
config_get_morty_home() {
    if [[ -z "${MORTY_HOME:-}" ]]; then
        return 1
    fi
    echo "$MORTY_HOME"
}

# Load configuration from settings.json
# Usage: config_load
# Returns: 0 on success, 1 on failure
config_load() {
    local morty_home
    morty_home=$(config_get_morty_home) || {
        log ERROR "MORTY_HOME environment variable is not set"
        return 1
    }

    # Ensure config directory exists
    if [[ ! -d "$morty_home" ]]; then
        mkdir -p "$morty_home" || {
            log ERROR "Failed to create MORTY_HOME directory: $morty_home"
            return 1
        }
    fi

    MORTY_CONFIG_FILE="$morty_home/settings.json"

    # Create default config if not exists
    if [[ ! -f "$MORTY_CONFIG_FILE" ]]; then
        config_init_settings
    fi

    return 0
}

# Initialize default settings.json
# Usage: config_init_settings
config_init_settings() {
    local morty_home
    morty_home=$(config_get_morty_home) || return 1

    local config_file="$morty_home/settings.json"

    cat > "$config_file" << 'EOF'
{
  "version": "2.0",
  "cli": {
    "command": "claude"
  },
  "defaults": {
    "max_loops": 50,
    "loop_delay": 5,
    "log_level": "INFO",
    "stat_refresh_interval": 60
  },
  "paths": {
    "work_dir": ".morty",
    "log_dir": ".morty/logs",
    "research_dir": ".morty/research",
    "plan_dir": ".morty/plan",
    "status_file": ".morty/status.json"
  }
}
EOF

    log INFO "Created default configuration file: $config_file"
}

# ============================================================================
# Configuration Read/Write
# ============================================================================

# Get configuration value by key
# Usage: config_get <key> [default]
# Example: config_get "cli.command" "claude"
config_get() {
    local key="$1"
    local default_value="${2:-}"

    if [[ -z "$MORTY_CONFIG_FILE" ]] || [[ ! -f "$MORTY_CONFIG_FILE" ]]; then
        echo "$default_value"
        return 1
    fi

    # Convert dot notation to jq path
    local jq_path=$(echo "$key" | sed 's/\./\./g')

    local value
    value=$(jq -r ".${jq_path} // empty" "$MORTY_CONFIG_FILE" 2>/dev/null)

    if [[ -z "$value" ]] || [[ "$value" == "null" ]]; then
        echo "$default_value"
        return 1
    fi

    echo "$value"
    return 0
}

# Get integer configuration value
# Usage: config_get_int <key> [default]
config_get_int() {
    local key="$1"
    local default_value="${2:-0}"

    local value
    value=$(config_get "$key" "")

    if [[ -z "$value" ]]; then
        echo "$default_value"
        return 1
    fi

    echo "$value"
}

# Get boolean configuration value
# Usage: config_get_bool <key> [default]
config_get_bool() {
    local key="$1"
    local default_value="${2:-false}"

    local value
    value=$(config_get "$key" "")

    if [[ -z "$value" ]]; then
        echo "$default_value"
        return 1
    fi

    # Normalize boolean
    case "$value" in
        true|True|TRUE|1|yes|YES)
            echo "true"
            ;;
        false|False|FALSE|0|no|NO)
            echo "false"
            ;;
        *)
            echo "$default_value"
            ;;
    esac
}

# Set configuration value
# Usage: config_set <key> <value>
# Example: config_set "cli.command" "mc --code"
config_set() {
    local key="$1"
    local value="$2"

    if [[ -z "$MORTY_CONFIG_FILE" ]] || [[ ! -f "$MORTY_CONFIG_FILE" ]]; then
        log ERROR "Configuration file not loaded"
        return 1
    fi

    # Convert dot notation to jq path
    local jq_path=$(echo "$key" | sed 's/\./\./g')

    # Determine if value is a number or string
    local jq_filter
    if [[ "$value" =~ ^-?[0-9]+$ ]]; then
        jq_filter=".${jq_path} = $value"
    elif [[ "$value" == "true" ]] || [[ "$value" == "false" ]]; then
        jq_filter=".${jq_path} = $value"
    else
        jq_filter=".${jq_path} = \"$value\""
    fi

    local temp_file
    temp_file=$(mktemp)

    if jq "$jq_filter" "$MORTY_CONFIG_FILE" > "$temp_file" 2>/dev/null; then
        mv "$temp_file" "$MORTY_CONFIG_FILE"
        log INFO "Configuration updated: $key = $value"
        return 0
    else
        rm -f "$temp_file"
        log ERROR "Failed to update configuration: $key"
        return 1
    fi
}

# ============================================================================
# Work Directory Management
# ============================================================================

# Check if work directory (.morty) exists in current directory
# Usage: config_check_work_dir
# Returns: 0 if exists, 1 if not exists
config_check_work_dir() {
    local work_dir=".morty"

    if [[ -d "$work_dir" ]]; then
        return 0
    else
        return 1
    fi
}

# Initialize work directory structure
# Usage: config_init_work_dir
# Creates: .morty/, .morty/logs/, .morty/research/, .morty/plan/
config_init_work_dir() {
    local work_dir=".morty"

    # Create main work directory
    if [[ ! -d "$work_dir" ]]; then
        mkdir -p "$work_dir" || {
            log ERROR "Failed to create work directory: $work_dir"
            return 1
        }
        log INFO "Created work directory: $work_dir"
    fi

    # Create subdirectories
    local subdirs=("logs" "research" "plan")
    for subdir in "${subdirs[@]}"; do
        local full_path="$work_dir/$subdir"
        if [[ ! -d "$full_path" ]]; then
            mkdir -p "$full_path" || {
                log ERROR "Failed to create subdirectory: $full_path"
                return 1
            }
            log INFO "Created subdirectory: $full_path"
        fi
    done

    return 0
}

# Ensure work directory exists (create if not exists)
# Usage: config_ensure_work_dir
# Returns: 0 on success, 1 on failure
config_ensure_work_dir() {
    if ! config_check_work_dir; then
        log INFO "Work directory not found, initializing..."
        config_init_work_dir
    fi

    # Verify work directory is writable
    local work_dir=".morty"
    if [[ ! -w "$work_dir" ]]; then
        log ERROR "Work directory is not writable: $work_dir"
        return 1
    fi

    # Check subdirectories
    local subdirs=("logs" "research" "plan")
    for subdir in "${subdirs[@]}"; do
        local full_path="$work_dir/$subdir"
        if [[ ! -d "$full_path" ]]; then
            mkdir -p "$full_path" || {
                log ERROR "Failed to create subdirectory: $full_path"
                return 1
            }
        fi
        if [[ ! -w "$full_path" ]]; then
            log ERROR "Subdirectory is not writable: $full_path"
            return 1
        fi
    done

    return 0
}

# Get work directory path
# Usage: config_get_work_dir
# Returns: path to work directory
config_get_work_dir() {
    echo ".morty"
}

# Get log directory path
# Usage: config_get_log_dir
# Returns: path to log directory
config_get_log_dir() {
    echo ".morty/logs"
}

# Get research directory path
# Usage: config_get_research_dir
# Returns: path to research directory
config_get_research_dir() {
    echo ".morty/research"
}

# Get plan directory path
# Usage: config_get_plan_dir
# Returns: path to plan directory
config_get_plan_dir() {
    echo ".morty/plan"
}

# Get status file path
# Usage: config_get_status_file
# Returns: path to status file
config_get_status_file() {
    echo ".morty/status.json"
}

# ============================================================================
# Precondition Checks
# ============================================================================

# Check if research directory exists and has files
# Usage: config_check_research_exists
# Returns: 0 if exists and has files, 1 otherwise
config_check_research_exists() {
    local research_dir
    research_dir=$(config_get_research_dir)

    if [[ ! -d "$research_dir" ]]; then
        return 1
    fi

    # Check if directory has any files
    local file_count
    file_count=$(find "$research_dir" -type f 2>/dev/null | wc -l)

    if [[ $file_count -eq 0 ]]; then
        return 1
    fi

    return 0
}

# Check if plan directory exists and has files
# Usage: config_check_plan_done
# Returns: 0 if plan is done, 1 otherwise
config_check_plan_done() {
    local plan_dir
    plan_dir=$(config_get_plan_dir)

    if [[ ! -d "$plan_dir" ]]; then
        return 1
    fi

    # Check if directory has any .md files
    local file_count
    file_count=$(find "$plan_dir" -name "*.md" -type f 2>/dev/null | wc -l)

    if [[ $file_count -eq 0 ]]; then
        return 1
    fi

    return 0
}

# Require plan to be completed before proceeding
# Usage: config_require_plan
# Returns: 0 if plan exists, 1 otherwise (prints error)
config_require_plan() {
    if ! config_check_plan_done; then
        log ERROR "请先运行 morty plan"
        return 1
    fi
    return 0
}

# Load research facts from research directory
# Usage: config_load_research_facts
# Returns: list of research files and their content
config_load_research_facts() {
    local research_dir
    research_dir=$(config_get_research_dir)

    if [[ ! -d "$research_dir" ]]; then
        return 1
    fi

    local files=()
    while IFS= read -r -d '' file; do
        files+=("$file")
    done < <(find "$research_dir" -name "*.md" -type f -print0 2>/dev/null)

    if [[ ${#files[@]} -eq 0 ]]; then
        return 1
    fi

    # Output files and content
    for file in "${files[@]}"; do
        echo "=== $(basename "$file") ==="
        cat "$file"
        echo ""
    done

    return 0
}
