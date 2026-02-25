// Package prompt provides Prompt file parsing functionality.
package prompt

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Prompt represents a parsed prompt template with metadata and variables.
type Prompt struct {
	Name        string            `json:"name"`         // Prompt name from frontmatter
	Description string            `json:"description"`  // Prompt description
	Template    string            `json:"template"`     // The main template content (without frontmatter)
	RawContent  string            `json:"raw_content"`  // Original file content
	Variables   []string          `json:"variables"`    // List of template variable names
	Metadata    map[string]string `json:"metadata"`     // All frontmatter metadata
}

// VariableValue represents a variable name and its replacement value.
type VariableValue struct {
	Name  string
	Value string
}

// Regular expressions for parsing
var (
	// Frontmatter pattern: --- at start of file, followed by content, then ---
	// Supports empty frontmatter (--- followed immediately by ---)
	frontmatterRegex = regexp.MustCompile(`(?s)^\s*---\s*\n(.*?)\n?---\s*(?:\n|$)`)

	// Key-value pair pattern: key: value
	kvRegex = regexp.MustCompile(`^[\s]*([a-zA-Z0-9_-]+)[\s]*:[\s]*(.*)$`)

	// Template variable pattern: {{variable}} or {{ variable }}
	// Supports letters, numbers, underscores, and hyphens
	variableRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_-]+)\s*\}\}`)
)

// ParsePrompt parses a prompt file and returns a structured Prompt.
func ParsePrompt(filepath string) (*Prompt, error) {
	// Read file content
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filepath, err)
	}

	return ParsePromptString(string(content))
}

// ParsePromptString parses prompt content from a string.
func ParsePromptString(content string) (*Prompt, error) {
	prompt := &Prompt{
		RawContent: content,
		Metadata:   make(map[string]string),
	}

	// Extract frontmatter metadata
	metadata, templateContent := extractFrontmatter(content)
	prompt.Metadata = metadata

	// Extract common fields from metadata
	if name, ok := metadata["name"]; ok {
		prompt.Name = name
	}
	if desc, ok := metadata["description"]; ok {
		prompt.Description = desc
	}

	// Set template content (without frontmatter)
	prompt.Template = templateContent

	// Extract template variables
	prompt.Variables = extractVariables(templateContent)

	return prompt, nil
}

// extractFrontmatter extracts YAML frontmatter metadata and returns remaining content.
func extractFrontmatter(content string) (map[string]string, string) {
	metadata := make(map[string]string)

	// Check if content has frontmatter
	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil {
		// No frontmatter found, return empty map and original content
		return metadata, strings.TrimSpace(content)
	}

	// Extract the frontmatter content (without delimiters)
	frontmatterContent := matches[1]

	// Parse each line of the frontmatter
	lines := strings.Split(frontmatterContent, "\n")
	for _, line := range lines {
		// Skip empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Try to parse as key-value pair
		if kvMatches := kvRegex.FindStringSubmatch(line); kvMatches != nil {
			key := strings.TrimSpace(kvMatches[1])
			value := strings.TrimSpace(kvMatches[2])

			// Validate key is not empty
			if key == "" {
				continue
			}

			metadata[key] = value
		}
	}

	// Get the content after frontmatter
	templateContent := frontmatterRegex.ReplaceAllString(content, "")
	templateContent = strings.TrimSpace(templateContent)

	return metadata, templateContent
}

// extractVariables extracts all template variables from content.
// Returns a deduplicated list of variable names.
func extractVariables(content string) []string {
	var variables []string
	seen := make(map[string]bool)

	matches := variableRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				seen[varName] = true
				variables = append(variables, varName)
			}
		}
	}

	return variables
}

// ReplaceVariables replaces template variables with their values.
// Returns the content with all variables replaced.
func (p *Prompt) ReplaceVariables(variables map[string]string) string {
	result := p.Template

	for name, value := range variables {
		// Match the variable with optional whitespace inside braces
		pattern := fmt.Sprintf(`\{\{\s*%s\s*\}\}`, regexp.QuoteMeta(name))
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, value)
	}

	return result
}

// ReplaceVariable replaces a single template variable with its value.
func (p *Prompt) ReplaceVariable(name, value string) string {
	return p.ReplaceVariables(map[string]string{name: value})
}

// GetVariableValues returns the values for the prompt's variables from the provided map.
// Returns error if required variables are missing.
func (p *Prompt) GetVariableValues(values map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	var missing []string

	for _, varName := range p.Variables {
		if val, ok := values[varName]; ok {
			result[varName] = val
		} else {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return result, fmt.Errorf("missing required variables: %s", strings.Join(missing, ", "))
	}

	return result, nil
}

// HasVariable checks if the prompt contains a specific variable.
func (p *Prompt) HasVariable(name string) bool {
	for _, v := range p.Variables {
		if v == name {
			return true
		}
	}
	return false
}

// Validate checks if all required variables have values provided.
func (p *Prompt) Validate(values map[string]string) error {
	_, err := p.GetVariableValues(values)
	return err
}

// Render renders the prompt with the provided variable values.
// Returns the final rendered content.
func (p *Prompt) Render(values map[string]string) (string, error) {
	if err := p.Validate(values); err != nil {
		return "", err
	}
	return p.ReplaceVariables(values), nil
}
