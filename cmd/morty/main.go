package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Build info variables - set by ldflags during build
var (
	Version    = "dev"
	GitCommit  = "unknown"
	BuildTime  = "unknown"
	GoVersion  = runtime.Version()
)

type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Default behavior: show help
	printHelp()
}

func printHelp() {
	fmt.Println("Morty - AI Coding Workflow Orchestrator")
	fmt.Println()
	fmt.Println("Usage: morty [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -version    Show version information")
	fmt.Println("  -help       Show this help message")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  research    Research mode - analyze requirements")
	fmt.Println("  plan        Plan mode - create development plans")
	fmt.Println("  doing       Doing mode - execute tasks")
	fmt.Println("  status      Show current status")
}

func printVersion() {
	buildInfo := BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	// Try to parse and reformat build time
	if t, err := time.Parse(time.RFC3339, BuildTime); err == nil {
		buildInfo.BuildTime = t.Format("2006-01-02 15:04:05")
	}

	data, _ := json.MarshalIndent(buildInfo, "", "  ")
	fmt.Println(string(data))
}
