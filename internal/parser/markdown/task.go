// Package markdown provides Markdown task extraction functionality.
package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	// TaskStatusPending represents an incomplete task
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusCompleted represents a completed task
	TaskStatusCompleted TaskStatus = "completed"
)

// Task represents a markdown task item (- [ ] or - [x])
type Task struct {
	Description string     `json:"description"` // Task description text
	Completed   bool       `json:"completed"`   // Whether the task is completed
	Level       int        `json:"level"`       // Indentation level (0 = root, 1 = indented once, etc.)
	Status      TaskStatus `json:"status"`      // Task status (pending/completed)
	Raw         string     `json:"raw"`         // Raw task line
}

// TaskExtractor provides functionality to extract tasks from markdown documents
type TaskExtractor struct {
	doc *Document
}

// Regular expressions for parsing tasks
var (
	// Task pattern for raw lines: - [ ] task or - [x] task (case insensitive for x)
	// Captures: leading whitespace, checkbox status, and description
	taskRegex = regexp.MustCompile(`^(\s*)[-\*\+]\s*\[([ xX])\]\s*(.*)$`)
)

// NewTaskExtractor creates a new task extractor for the given document
func NewTaskExtractor(doc *Document) *TaskExtractor {
	return &TaskExtractor{doc: doc}
}


// ExtractTasks extracts all tasks from a markdown document.
// Returns a slice of Task structs containing the description, completion status, and indentation level.
func ExtractTasks(doc *Document) ([]Task, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	extractor := NewTaskExtractor(doc)
	return extractor.extractAllTasks(), nil
}

// extractAllTasks extracts all tasks from the document
func (te *TaskExtractor) extractAllTasks() []Task {
	if te.doc == nil || len(te.doc.Nodes) == 0 {
		return []Task{}
	}

	var tasks []Task

	// Iterate through parsed nodes
	for _, node := range te.doc.Nodes {
		tasks = append(tasks, te.extractTasksFromNode(node)...)
	}

	return tasks
}

// extractTasksFromNode extracts tasks from a single node
func (te *TaskExtractor) extractTasksFromNode(node Node) []Task {
	var tasks []Task

	switch node.Type {
	case NodeTypeList:
		// Extract tasks from list items, using ItemIndents for level information
		// Need to reconstruct the full task line with bullet prefix for parsing
		for i, item := range node.Items {
			level := 0
			if i < len(node.ItemIndents) {
				level = node.ItemIndents[i]
			}
			// Reconstruct the task line with bullet prefix based on list type
			bullet := "-"
			if node.ListType == ListTypeOrdered {
				bullet = "1."
			}
			// Add indentation spaces based on level
			indent := strings.Repeat("  ", level)
			fullLine := indent + bullet + " " + item
			if task := parseTaskFromLineWithLevel(fullLine, level); task != nil {
				tasks = append(tasks, *task)
			}
		}
	case NodeTypeParagraph:
		// Check each line in the paragraph for tasks
		lines := strings.Split(node.Content, "\n")
		for _, line := range lines {
			if task := parseTaskFromLine(line); task != nil {
				tasks = append(tasks, *task)
			}
		}
	}

	return tasks
}

// parseTaskFromLine parses a single line and returns a Task if it matches the task pattern
func parseTaskFromLine(line string) *Task {
	return parseTaskFromLineWithLevel(line, 0)
}

// parseTaskFromLineWithLevel parses a single line with a specified indentation level
func parseTaskFromLineWithLevel(line string, level int) *Task {
	trimmed := strings.TrimRight(line, "\r\n")

	// Only match the full task pattern with bullet prefix
	matches := taskRegex.FindStringSubmatch(trimmed)
	if matches != nil {
		indent := matches[1]
		status := strings.ToLower(matches[2])
		description := strings.TrimSpace(matches[3])

		// Skip empty descriptions
		if description == "" {
			return nil
		}

		// Use provided level or calculate from indent
		if level == 0 && indent != "" {
			level = calculateIndentLevel(indent)
		}
		completed := status == "x"

		return &Task{
			Description: description,
			Completed:   completed,
			Level:       level,
			Status:      getTaskStatus(completed),
			Raw:         trimmed,
		}
	}

	return nil
}

// getTaskStatus returns the TaskStatus based on completion
func getTaskStatus(completed bool) TaskStatus {
	if completed {
		return TaskStatusCompleted
	}
	return TaskStatusPending
}

// calculateIndentLevel calculates the indentation level from leading whitespace
// Uses 2 spaces as the standard indent unit (common in markdown)
func calculateIndentLevel(indent string) int {
	if indent == "" {
		return 0
	}

	// Convert tabs to spaces (1 tab = 4 spaces)
	expanded := strings.ReplaceAll(indent, "\t", "    ")

	// Calculate level based on 2-space indentation
	level := len(expanded) / 2
	if level < 0 {
		level = 0
	}

	return level
}

// FindPendingTasks returns only pending (incomplete) tasks
func FindPendingTasks(doc *Document) ([]Task, error) {
	tasks, err := ExtractTasks(doc)
	if err != nil {
		return nil, err
	}

	var pending []Task
	for _, task := range tasks {
		if !task.Completed {
			pending = append(pending, task)
		}
	}

	return pending, nil
}

// FindCompletedTasks returns only completed tasks
func FindCompletedTasks(doc *Document) ([]Task, error) {
	tasks, err := ExtractTasks(doc)
	if err != nil {
		return nil, err
	}

	var completed []Task
	for _, task := range tasks {
		if task.Completed {
			completed = append(completed, task)
		}
	}

	return completed, nil
}

// CountTasks returns the total number of tasks and breakdown by status
func CountTasks(doc *Document) (total, pending, completed int, err error) {
	tasks, err := ExtractTasks(doc)
	if err != nil {
		return 0, 0, 0, err
	}

	total = len(tasks)
	for _, task := range tasks {
		if task.Completed {
			completed++
		} else {
			pending++
		}
	}

	return total, pending, completed, nil
}

// GetTasksByLevel returns tasks at a specific indentation level
func GetTasksByLevel(doc *Document, level int) ([]Task, error) {
	tasks, err := ExtractTasks(doc)
	if err != nil {
		return nil, err
	}

	var result []Task
	for _, task := range tasks {
		if task.Level == level {
			result = append(result, task)
		}
	}

	return result, nil
}

// GetTaskHierarchy returns tasks organized by their parent-child relationships
// Tasks are grouped by their parent task (based on indentation level)
func GetTaskHierarchy(doc *Document) (map[int][]Task, error) {
	tasks, err := ExtractTasks(doc)
	if err != nil {
		return nil, err
	}

	hierarchy := make(map[int][]Task)
	for _, task := range tasks {
		hierarchy[task.Level] = append(hierarchy[task.Level], task)
	}

	return hierarchy, nil
}
