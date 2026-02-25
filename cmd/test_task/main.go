package main

import (
	"fmt"
	"github.com/morty/morty/internal/parser/markdown"
)

func main() {
	parser := markdown.NewParser()
	doc, err := parser.ParseDocument("- [ ] Task 1\n- [x] Task 2")
	if err != nil {
		fmt.Println("Parse error:", err)
		return
	}
	
	tasks, err := markdown.ExtractTasks(doc)
	if err != nil {
		fmt.Println("Extract error:", err)
		return
	}
	
	fmt.Printf("Found %d tasks\n", len(tasks))
	for i, task := range tasks {
		fmt.Printf("Task %d: %s (completed: %v)\n", i+1, task.Description, task.Completed)
	}
}
