package markdown

import (
	"testing"
)

func TestExtractTasks_NilDocument(t *testing.T) {
	tasks, err := ExtractTasks(nil)
	if err == nil {
		t.Error("Expected error for nil document, got nil")
	}
	if tasks != nil {
		t.Errorf("Expected nil tasks for nil document, got %v", tasks)
	}
}

func TestExtractTasks_EmptyDocument(t *testing.T) {
	doc := &Document{Nodes: []Node{}}
	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestExtractTasks_PendingTasks(t *testing.T) {
	content := "- [ ] Task 1\n- [ ] Task 2\n- [ ] Task 3"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	for i, task := range tasks {
		if task.Completed {
			t.Errorf("Task %d should be pending, but is completed", i+1)
		}
		if task.Status != TaskStatusPending {
			t.Errorf("Task %d status should be pending, got %s", i+1, task.Status)
		}
	}
}

func TestExtractTasks_CompletedTasks(t *testing.T) {
	content := "- [x] Completed Task 1\n- [X] Completed Task 2"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	for i, task := range tasks {
		if !task.Completed {
			t.Errorf("Task %d should be completed, but is pending", i+1)
		}
		if task.Status != TaskStatusCompleted {
			t.Errorf("Task %d status should be completed, got %s", i+1, task.Status)
		}
	}
}

func TestExtractTasks_MixedTasks(t *testing.T) {
	content := "- [ ] Pending Task\n- [x] Completed Task\n- [ ] Another Pending\n- [X] Another Completed"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(tasks))
	}

	expected := []bool{false, true, false, true}
	for i, task := range tasks {
		if task.Completed != expected[i] {
			t.Errorf("Task %d completion status: expected %v, got %v", i+1, expected[i], task.Completed)
		}
	}
}

func TestExtractTasks_DifferentBulletStyles(t *testing.T) {
	content := "- [ ] Dash task\n* [x] Asterisk task\n+ [ ] Plus task"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	expectedDescriptions := []string{"Dash task", "Asterisk task", "Plus task"}
	for i, task := range tasks {
		if task.Description != expectedDescriptions[i] {
			t.Errorf("Task %d description: expected %q, got %q", i+1, expectedDescriptions[i], task.Description)
		}
	}
}

func TestExtractTasks_TaskDescription(t *testing.T) {
	content := "- [ ] This is a simple task\n- [x] This is a completed task with more words"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	expected := []string{"This is a simple task", "This is a completed task with more words"}
	for i, task := range tasks {
		if task.Description != expected[i] {
			t.Errorf("Task %d description: expected %q, got %q", i+1, expected[i], task.Description)
		}
	}
}

func TestExtractTasks_IndentationLevels(t *testing.T) {
	content := "- [ ] Root level task\n  - [ ] Level 1 task\n    - [ ] Level 2 task\n- [ ] Another root task"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(tasks))
	}

	expectedLevels := []int{0, 1, 2, 0}
	for i, task := range tasks {
		if task.Level != expectedLevels[i] {
			t.Errorf("Task %d level: expected %d, got %d", i+1, expectedLevels[i], task.Level)
		}
	}
}

func TestExtractTasks_HierarchicalDescription(t *testing.T) {
	content := "- [ ] Parent task\n  - [ ] Child task\n    - [ ] Grandchild task"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	expectedDescriptions := []string{"Parent task", "Child task", "Grandchild task"}
	for i, task := range tasks {
		if task.Description != expectedDescriptions[i] {
			t.Errorf("Task %d description: expected %q, got %q", i+1, expectedDescriptions[i], task.Description)
		}
	}
}

func TestExtractTasks_NoTasks(t *testing.T) {
	content := "# Heading\nThis is a paragraph.\n- Regular list item\n- Another regular item"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks for document without tasks, got %d", len(tasks))
	}
}

func TestExtractTasks_EmptyCheckbox(t *testing.T) {
	content := "- [ ]\n- [x] "
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks (empty descriptions), got %d", len(tasks))
	}
}

func TestFindPendingTasks(t *testing.T) {
	content := "- [ ] Pending 1\n- [x] Completed 1\n- [ ] Pending 2\n- [x] Completed 2"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	pending, err := FindPendingTasks(doc)
	if err != nil {
		t.Fatalf("FindPendingTasks failed: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(pending))
	}

	for _, task := range pending {
		if task.Completed {
			t.Error("Found completed task in pending results")
		}
	}
}

func TestFindCompletedTasks(t *testing.T) {
	content := "- [ ] Pending 1\n- [x] Completed 1\n- [ ] Pending 2\n- [x] Completed 2"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	completed, err := FindCompletedTasks(doc)
	if err != nil {
		t.Fatalf("FindCompletedTasks failed: %v", err)
	}

	if len(completed) != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", len(completed))
	}

	for _, task := range completed {
		if !task.Completed {
			t.Error("Found pending task in completed results")
		}
	}
}

func TestCountTasks(t *testing.T) {
	content := "- [ ] Pending 1\n- [x] Completed 1\n- [ ] Pending 2\n- [x] Completed 2\n- [x] Completed 3"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	total, pending, completed, err := CountTasks(doc)
	if err != nil {
		t.Fatalf("CountTasks failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if pending != 2 {
		t.Errorf("Expected pending 2, got %d", pending)
	}
	if completed != 3 {
		t.Errorf("Expected completed 3, got %d", completed)
	}
}

func TestGetTasksByLevel(t *testing.T) {
	content := "- [ ] Root 1\n  - [ ] Level 1 - A\n  - [ ] Level 1 - B\n    - [ ] Level 2\n- [ ] Root 2"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	level0, err := GetTasksByLevel(doc, 0)
	if err != nil {
		t.Fatalf("GetTasksByLevel failed: %v", err)
	}
	if len(level0) != 2 {
		t.Errorf("Expected 2 level 0 tasks, got %d", len(level0))
	}

	level1, err := GetTasksByLevel(doc, 1)
	if err != nil {
		t.Fatalf("GetTasksByLevel failed: %v", err)
	}
	if len(level1) != 2 {
		t.Errorf("Expected 2 level 1 tasks, got %d", len(level1))
	}

	level2, err := GetTasksByLevel(doc, 2)
	if err != nil {
		t.Fatalf("GetTasksByLevel failed: %v", err)
	}
	if len(level2) != 1 {
		t.Errorf("Expected 1 level 2 task, got %d", len(level2))
	}
}

func TestGetTaskHierarchy(t *testing.T) {
	content := "- [ ] Root 1\n  - [ ] Child 1\n  - [ ] Child 2\n    - [ ] Grandchild\n- [ ] Root 2"
	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	hierarchy, err := GetTaskHierarchy(doc)
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	if len(hierarchy[0]) != 2 {
		t.Errorf("Expected 2 root tasks, got %d", len(hierarchy[0]))
	}
	if len(hierarchy[1]) != 2 {
		t.Errorf("Expected 2 level 1 tasks, got %d", len(hierarchy[1]))
	}
	if len(hierarchy[2]) != 1 {
		t.Errorf("Expected 1 level 2 task, got %d", len(hierarchy[2]))
	}
}

func TestParseTaskFromLine_InvalidFormats(t *testing.T) {
	invalidLines := []string{
		"Regular text",
		"- Regular list item",
		"* Regular list item",
		"1. Ordered list item",
		"[ ] No bullet",
		"-[] No space",
		"- [  ] Extra spaces",
		"",
	}

	for _, line := range invalidLines {
		task := parseTaskFromLine(line)
		if task != nil {
			t.Errorf("Expected nil for line %q, got %v", line, task)
		}
	}
}

func TestParseTaskFromLine_ValidFormats(t *testing.T) {
	tests := []struct {
		line              string
		expectedDesc      string
		expectedCompleted bool
		expectedLevel     int
	}{
		{"- [ ] Simple task", "Simple task", false, 0},
		{"- [x] Completed task", "Completed task", true, 0},
		{"- [X] Capital X", "Capital X", true, 0},
		{"* [ ] Asterisk", "Asterisk", false, 0},
		{"+ [x] Plus", "Plus", true, 0},
		{"  - [ ] Indented", "Indented", false, 1},
		{"    - [x] Double indent", "Double indent", true, 2},
		{"- [ ] Task with [link](url)", "Task with [link](url)", false, 0},
		{"- [ ]   Extra spaces  ", "Extra spaces", false, 0},
	}

	for _, test := range tests {
		task := parseTaskFromLine(test.line)
		if task == nil {
			t.Errorf("Expected task for line %q, got nil", test.line)
			continue
		}
		if task.Description != test.expectedDesc {
			t.Errorf("Description mismatch for %q: expected %q, got %q", test.line, test.expectedDesc, task.Description)
		}
		if task.Completed != test.expectedCompleted {
			t.Errorf("Completed mismatch for %q: expected %v, got %v", test.line, test.expectedCompleted, task.Completed)
		}
		if task.Level != test.expectedLevel {
			t.Errorf("Level mismatch for %q: expected %d, got %d", test.line, test.expectedLevel, task.Level)
		}
	}
}

func TestCalculateIndentLevel(t *testing.T) {
	tests := []struct {
		indent   string
		expected int
	}{
		{"", 0},
		{" ", 0},
		{"  ", 1},
		{"    ", 2},
		{"      ", 3},
		{"\t", 2},
		{"\t\t", 4},
		{" \t ", 3},
	}

	for _, test := range tests {
		level := calculateIndentLevel(test.indent)
		if level != test.expected {
			t.Errorf("Indent %q: expected level %d, got %d", test.indent, test.expected, level)
		}
	}
}

func TestExtractTasks_WithOtherContent(t *testing.T) {
	content := "# Project Tasks\n\n## Todo\n- [ ] Implement feature A\n- [ ] Implement feature B\n  - [ ] Subtask B1\n  - [x] Subtask B2\n\n## Done\n- [x] Setup project\n- [x] Initial commit\n\nSome paragraph text here.\n- [ ] Task in paragraph context"

	parser := NewParser()
	doc, err := parser.ParseDocument(content)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tasks, err := ExtractTasks(doc)
	if err != nil {
		t.Fatalf("ExtractTasks failed: %v", err)
	}

	if len(tasks) != 7 {
		t.Errorf("Expected 7 tasks, got %d", len(tasks))
	}

	pending := 0
	completed := 0
	for _, task := range tasks {
		if task.Completed {
			completed++
		} else {
			pending++
		}
	}

	if pending != 4 {
		t.Errorf("Expected 4 pending tasks, got %d", pending)
	}
	if completed != 3 {
		t.Errorf("Expected 3 completed tasks, got %d", completed)
	}
}

func BenchmarkExtractTasks(b *testing.B) {
	content := "- [ ] Task 1\n- [x] Task 2\n  - [ ] Subtask 2.1\n  - [x] Subtask 2.2\n- [ ] Task 3\n  - [ ] Subtask 3.1\n    - [ ] Sub-subtask 3.1.1"

	parser := NewParser()
	doc, _ := parser.ParseDocument(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractTasks(doc)
	}
}
