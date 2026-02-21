#!/bin/bash
# morty_research.sh - Research command for Morty
# Usage: morty research <topic>
# Starts a research process using Claude CLI with the research system prompt

set -e

# Get script directory
MORTY_BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MORTY_ROOT_DIR="$(dirname "$MORTY_BIN_DIR")"
MORTY_LIB_DIR="$MORTY_ROOT_DIR/lib"

# Source required libraries
source "$MORTY_LIB_DIR/common.sh"
source "$MORTY_LIB_DIR/config.sh"
source "$MORTY_LIB_DIR/logging.sh"

# Version
MORTY_RESEARCH_VERSION="1.0.0"

# ============================================================================
# Research Command Functions
# ============================================================================

# Display usage information
research_usage() {
    cat << EOF
Usage: morty research <topic>

Start a research process on the given topic.

Arguments:
  topic    The research topic to investigate

Options:
  -h, --help       Show this help message
  -v, --version    Show version information

Examples:
  morty research "project architecture"
  morty research "database optimization"

The research process will:
  1. Load the research system prompt from prompts/research.md
  2. Invoke the AI CLI in Plan mode
  3. Generate a research report to .morty/research/[topic].md

EOF
}

# Display version information
research_version() {
    echo "morty research version $MORTY_RESEARCH_VERSION"
}

# Parse command arguments
# Usage: research_parse_args "$@"
# Sets: RESEARCH_TOPIC
research_parse_args() {
    RESEARCH_TOPIC=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                research_usage
                exit 0
                ;;
            -v|--version)
                research_version
                exit 0
                ;;
            -*)
                log ERROR "Unknown option: $1"
                research_usage
                return 1
                ;;
            *)
                if [[ -z "$RESEARCH_TOPIC" ]]; then
                    RESEARCH_TOPIC="$1"
                else
                    RESEARCH_TOPIC="$RESEARCH_TOPIC $1"
                fi
                shift
                ;;
        esac
    done

    # Validate topic is provided
    if [[ -z "$RESEARCH_TOPIC" ]]; then
        log ERROR "Research topic is required"
        research_usage
        return 1
    fi

    return 0
}

# Load research system prompt
# Returns: Prompt content via stdout
research_load_prompt() {
    local prompt_file="$MORTY_ROOT_DIR/prompts/research.md"

    if [[ ! -f "$prompt_file" ]]; then
        log ERROR "Research prompt file not found: $prompt_file"
        return 1
    fi

    cat "$prompt_file"
}

# Get AI CLI command from config
# Returns: CLI command via stdout
research_get_ai_cli() {
    # Ensure config is loaded
    if [[ -z "$MORTY_CONFIG_FILE" ]]; then
        config_load || {
            log WARN "Failed to load config, using default"
            echo "claude"
            return 0
        }
    fi

    # Get CLI command from config with default fallback
    local ai_cli
    ai_cli=$(config_get "cli.command" "claude")

    echo "$ai_cli"
}

# Create research directory if not exists
research_ensure_directory() {
    local research_dir
    research_dir=$(config_get_research_dir)

    if [[ ! -d "$research_dir" ]]; then
        mkdir -p "$research_dir" || {
            log ERROR "Failed to create research directory: $research_dir"
            return 1
        }
        log INFO "Created research directory: $research_dir"
    fi

    return 0
}

# Build the AI CLI command for research
# Usage: research_build_cli_cmd <topic> <prompt>
# Returns: Full command string via stdout
research_build_cli_cmd() {
    local topic="$1"
    local prompt="$2"
    local ai_cli="$3"

    # Escape the prompt for safe shell execution
    local escaped_prompt
    escaped_prompt=$(printf '%q' "$prompt")

    # Build command:
    # - Use Plan mode (--permission-mode plan)
    # - Pass system prompt via -p flag
    # - Pass topic as additional context
    echo "$ai_cli --permission-mode plan -p \"=== Research Topic: $topic ===\n\n$escaped_prompt\""
}

# Execute research process
# Usage: research_execute <topic>
research_execute() {
    local topic="$1"

    log INFO "Starting research on: $topic"

    # Step 1: Ensure work directory exists
    config_ensure_work_dir || {
        log ERROR "Failed to initialize work directory"
        return 1
    }

    # Step 2: Load system prompt
    local system_prompt
    system_prompt=$(research_load_prompt) || {
        log ERROR "Failed to load research system prompt"
        return 1
    }
    log INFO "Loaded research system prompt"

    # Step 3: Get AI CLI command from config
    local ai_cli
    ai_cli=$(research_get_ai_cli)
    log INFO "Using AI CLI: $ai_cli"

    # Step 4: Ensure research directory exists
    research_ensure_directory || {
        log ERROR "Failed to create research directory"
        return 1
    }

    # Step 5: Build the full prompt with topic context
    local full_prompt
    full_prompt="# Research Task

**Topic**: $topic

**Instructions**:
Please conduct a comprehensive research on the topic above. Follow the research methodology defined in the system prompt.

**Output**: Save your findings to \`.morty/research/${topic// /_}.md\`

---

$system_prompt"

    # Step 6: Build and execute the CLI command
    log INFO "Invoking AI CLI in Plan mode..."

    # Create a temporary file for the prompt to avoid shell escaping issues
    local prompt_file
    prompt_file=$(mktemp)
    echo "$full_prompt" > "$prompt_file"

    # Execute the AI CLI
    local cmd="$ai_cli --permission-mode plan < \"$prompt_file\""
    log DEBUG "Executing: $cmd"

    # Run the command
    if eval "$cmd"; then
        log INFO "Research process completed"
        rm -f "$prompt_file"

        # Verify output was created
        local research_dir
        research_dir=$(config_get_research_dir)
        local output_file="$research_dir/${topic// /_}.md"

        if [[ -f "$output_file" ]]; then
            log INFO "Research report generated: $output_file"
            return 0
        else
            log WARN "Research completed but output file not found at expected location: $output_file"
            return 0
        fi
    else
        local exit_code=$?
        rm -f "$prompt_file"
        log ERROR "Research process failed with exit code: $exit_code"
        return $exit_code
    fi
}

# ============================================================================
# Main Entry Point
# ============================================================================

main() {
    # Parse arguments
    research_parse_args "$@" || exit 1

    # Initialize logging
    log_init "research"

    # Execute research
    research_execute "$RESEARCH_TOPIC"
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
