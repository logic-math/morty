#!/bin/bash
# Mock Claude CLI for BDD testing
# This script simulates the Claude Code CLI behavior for testing purposes

# Configuration
MOCK_LOG_FILE="${MOCK_LOG_FILE:-/tmp/mock_claude.log}"
MOCK_DELAY="${MOCK_DELAY:-0.5}"

# Load mock responses
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/mock_responses.sh"

# Initialize log file
mkdir -p "$(dirname "$MOCK_LOG_FILE")"
echo "=== Mock Claude CLI Session Started at $(date) ===" >> "$MOCK_LOG_FILE"

# Parse command line arguments
output_file=""
prompt_content=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output)
            output_file="$2"
            shift 2
            ;;
        -p|--prompt)
            prompt_content="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

# Read input from stdin if no prompt provided
if [ -z "$prompt_content" ]; then
    input=""
    while IFS= read -r line; do
        input+="$line"$'\n'
    done
else
    input="$prompt_content"
fi

# Log the input
{
    echo "--- Input received at $(date) ---"
    echo "Output file: $output_file"
    echo "Input length: ${#input} bytes"
    echo "First 200 chars: ${input:0:200}"
    echo "---"
} >> "$MOCK_LOG_FILE"

# Simulate AI processing delay
sleep "$MOCK_DELAY"

# Generate response based on input
response=$(get_mock_response "$input")

# Log the output
{
    echo "--- Output generated at $(date) ---"
    echo "Response length: ${#response} bytes"
    echo "First 200 chars: ${response:0:200}"
    echo "---"
    echo ""
} >> "$MOCK_LOG_FILE"

# Determine file type from input
file_type=""
if echo "$input" | grep -qi "research topic:"; then
    file_type="research"
elif echo "$input" | grep -qi "plan"; then
    file_type="plan"
elif echo "$input" | grep -qi "task\|doing"; then
    file_type="code"
fi

# Write to output file if specified
if [ -n "$output_file" ]; then
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$output_file")"
    echo "$response" > "$output_file"
    {
        echo "--- File written at $(date) ---"
        echo "File: $output_file"
        echo "Size: $(wc -c < "$output_file") bytes"
        echo "---"
    } >> "$MOCK_LOG_FILE"
elif [ "$file_type" = "research" ]; then
    # Auto-write research files to .morty/research/
    topic=$(echo "$input" | grep -oP "Research Topic: \K.*" | head -n1 | tr -d '\n')
    if [ -n "$topic" ]; then
        # Sanitize filename
        filename=$(echo "$topic" | tr '[:upper:]' '[:lower:]' | tr ' ' '_' | tr -cd '[:alnum:]_' | cut -c1-50)
        timestamp=$(date +%Y%m%d_%H%M%S)
        output_path=".morty/research/${filename}_${timestamp}.md"

        mkdir -p .morty/research
        echo "$response" > "$output_path"

        {
            echo "--- Auto-written research file at $(date) ---"
            echo "File: $output_path"
            echo "Size: $(wc -c < "$output_path") bytes"
            echo "---"
        } >> "$MOCK_LOG_FILE"
    fi
    # Also output to stdout
    echo "$response"
elif [ "$file_type" = "plan" ]; then
    # Auto-write plan files to .morty/plan/
    timestamp=$(date +%Y%m%d_%H%M%S)
    output_path=".morty/plan/plan_${timestamp}.md"

    mkdir -p .morty/plan
    echo "$response" > "$output_path"

    {
        echo "--- Auto-written plan file at $(date) ---"
        echo "File: $output_path"
        echo "Size: $(wc -c < "$output_path") bytes"
        echo "---"
    } >> "$MOCK_LOG_FILE"
    # Also output to stdout
    echo "$response"
else
    # Output to stdout if no output file specified
    echo "$response"
fi

# Exit successfully
exit 0
