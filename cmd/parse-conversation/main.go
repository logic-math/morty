// Command parse-conversation parses Claude Code conversation JSON and generates formatted logs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/morty/morty/internal/callcli"
)

func main() {
	// Parse command line flags
	inputFile := flag.String("input", "", "Input JSON file (Claude Code output)")
	outputDir := flag.String("output", ".morty/logs", "Output directory for logs")
	module := flag.String("module", "default_module", "Module name")
	job := flag.String("job", "default_job", "Job name")
	verbose := flag.Bool("verbose", false, "Verbose output")

	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <json-file> [-output <dir>] [-module <name>] [-job <name>]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Read input file
	if *verbose {
		fmt.Printf("Reading input file: %s\n", *inputFile)
	}

	jsonData, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Create parser
	parser := callcli.NewConversationParser(*outputDir)

	// Parse JSON
	if *verbose {
		fmt.Printf("Parsing JSON data (%d bytes)...\n", len(jsonData))
	}

	conversation, err := parser.Parse(string(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("\n=== Parsing Summary ===\n")
	fmt.Printf("Session ID: %s\n", conversation.SessionID)
	fmt.Printf("Model: %s\n", conversation.Model)
	fmt.Printf("Events: %d\n", len(conversation.Events))
	if conversation.TotalCostUSD > 0 {
		fmt.Printf("Cost: $%.4f USD\n", conversation.TotalCostUSD)
	}
	if conversation.TotalDurationMs > 0 {
		fmt.Printf("Duration: %.2f seconds\n", float64(conversation.TotalDurationMs)/1000.0)
	}
	if conversation.NumTurns > 0 {
		fmt.Printf("Turns: %d\n", conversation.NumTurns)
	}
	if conversation.TotalInputTokens > 0 || conversation.TotalOutputTokens > 0 {
		fmt.Printf("Tokens: %d input + %d output = %d total\n",
			conversation.TotalInputTokens,
			conversation.TotalOutputTokens,
			conversation.TotalInputTokens+conversation.TotalOutputTokens)
	}

	// Extract logs
	logs := parser.ExtractLogs(conversation)
	fmt.Printf("Extracted Logs: %d\n", len(logs))

	// Count log types
	logTypes := make(map[string]int)
	for _, log := range logs {
		logTypes[log.MessageType]++
	}
	fmt.Printf("\nLog Types:\n")
	for typ, count := range logTypes {
		fmt.Printf("  - %s: %d\n", typ, count)
	}

	// Save formatted log
	logPath, err := parser.ParseAndSave(string(jsonData), *module, *job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving logs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Success ===\n")
	fmt.Printf("Formatted log saved to: %s\n", logPath)

	// Also save formatted JSON
	jsonPath, err := parser.SaveFormattedJSON(conversation, *module, *job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save formatted JSON: %v\n", err)
	} else {
		fmt.Printf("Formatted JSON saved to: %s\n", jsonPath)
	}

	// Print absolute paths
	absLogPath, _ := filepath.Abs(logPath)
	absJSONPath, _ := filepath.Abs(jsonPath)
	fmt.Printf("\nAbsolute paths:\n")
	fmt.Printf("  Log:  %s\n", absLogPath)
	fmt.Printf("  JSON: %s\n", absJSONPath)
}
