package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/morty/morty/internal/cmd"
	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
)

// Build info variables - set by ldflags during build
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
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
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	// Check for global flags
	if os.Args[1] == "-version" || os.Args[1] == "--version" || os.Args[1] == "version" {
		printVersion()
		os.Exit(0)
	}

	if os.Args[1] == "-help" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printHelp()
		os.Exit(0)
	}

	// Get the command
	command := os.Args[1]

	// Setup logging
	logger, _, err := logging.NewLoggerFromConfig(&config.LoggingConfig{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	cfg := config.NewPaths()

	// Route to appropriate handler
	switch command {
	case "research":
		handleResearch(cfg, logger, os.Args[2:])
	case "plan":
		handlePlan(cfg, logger, os.Args[2:])
	case "doing":
		handleDoing(cfg, logger, os.Args[2:])
	case "stat", "status":
		handleStat(cfg, logger, os.Args[2:])
	case "reset":
		handleReset(cfg, logger, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Morty - AI Coding Workflow Orchestrator")
	fmt.Println()
	fmt.Println("Usage: morty <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  research    Research mode - analyze requirements")
	fmt.Println("  plan        Plan mode - create development plans")
	fmt.Println("  doing       Doing mode - execute tasks")
	fmt.Println("  stat        Show current status")
	fmt.Println("  reset       Reset workflow state")
	fmt.Println("  version     Show version information")
	fmt.Println("  help        Show this help message")
	fmt.Println()
	fmt.Println("Use 'morty <command> --help' for more information about a command.")
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

func handleResearch(cfg *config.Paths, logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("research", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty research [topic]")
		fmt.Println()
		fmt.Println("Start research mode to analyze requirements.")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  topic    Optional research topic")
		os.Exit(0)
	}

	// Create a simple config manager wrapper
	cfgMgr := &pathsConfigManager{paths: cfg}
	handler := cmd.NewResearchHandler(cfgMgr, logger)
	ctx := context.Background()

	_, err := handler.Execute(ctx, fs.Args())
	if err != nil {
		logger.Error("Research failed", logging.String("error", err.Error()))
		os.Exit(1)
	}

	fmt.Println("✓ Research completed")
}

func handlePlan(cfg *config.Paths, logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	_ = fs.String("module", "", "Target module name") // reserved for future use
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty plan [options] [research-topic]")
		fmt.Println()
		fmt.Println("Create a development plan based on research.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -module string    Target module name")
		os.Exit(0)
	}

	cfgMgr := &pathsConfigManager{paths: cfg}

	// Plan handler requires an executor parameter
	var executor interface{} // TODO: create actual executor if needed
	handler := cmd.NewPlanHandler(cfgMgr, logger, executor)
	ctx := context.Background()

	_, err := handler.Execute(ctx, fs.Args())
	if err != nil {
		logger.Error("Plan failed", logging.String("error", err.Error()))
		os.Exit(1)
	}

	fmt.Println("✓ Plan completed")
}

func handleDoing(cfg *config.Paths, logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("doing", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	restart := fs.Bool("restart", false, "Restart mode")
	module := fs.String("module", "", "Target module")
	job := fs.String("job", "", "Target job (requires -module)")
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty doing [options]")
		fmt.Println()
		fmt.Println("Execute tasks from the development plan.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -restart          Restart mode - reset state before execution")
		fmt.Println("  -module string    Target specific module")
		fmt.Println("  -job string       Target specific job (requires -module)")
		os.Exit(0)
	}

	cfgMgr := &pathsConfigManager{paths: cfg}
	handler := cmd.NewDoingHandler(cfgMgr, logger)
	ctx := context.Background()

	// Build args list for handler
	handlerArgs := []string{}
	if *restart {
		handlerArgs = append(handlerArgs, "--restart")
	}
	if *module != "" {
		handlerArgs = append(handlerArgs, "--module", *module)
	}
	if *job != "" {
		handlerArgs = append(handlerArgs, "--job", *job)
	}
	handlerArgs = append(handlerArgs, fs.Args()...)

	result, err := handler.Execute(ctx, handlerArgs)
	if err != nil {
		handler.PrintDoingSummary(result)
		os.Exit(1)
	}

	handler.PrintDoingSummary(result)
}

func handleStat(cfg *config.Paths, logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("stat", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty stat [options]")
		fmt.Println()
		fmt.Println("Show current execution status.")
		os.Exit(0)
	}

	cfgMgr := &pathsConfigManager{paths: cfg}
	handler := cmd.NewStatHandler(cfgMgr, logger)
	ctx := context.Background()

	_, err := handler.Execute(ctx, fs.Args())
	if err != nil {
		logger.Error("Stat failed", logging.String("error", err.Error()))
		os.Exit(1)
	}
}

func handleReset(cfg *config.Paths, logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty reset [options]")
		fmt.Println()
		fmt.Println("Reset workflow state.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -l [count]    List recent commits")
		fmt.Println("  -c hash       Reset to specific commit")
		os.Exit(0)
	}

	cfgMgr := &pathsConfigManager{paths: cfg}
	handler := cmd.NewResetHandler(cfgMgr, logger)
	ctx := context.Background()

	_, err := handler.Execute(ctx, fs.Args())
	if err != nil {
		logger.Error("Reset failed", logging.String("error", err.Error()))
		os.Exit(1)
	}
}

func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

// pathsConfigManager adapts *config.Paths to config.Manager interface
type pathsConfigManager struct {
	paths *config.Paths
}

func (p *pathsConfigManager) Load(path string) error {
	return nil
}

func (p *pathsConfigManager) LoadWithMerge(userConfigPath string) error {
	return nil
}

func (p *pathsConfigManager) Get(key string, defaultValue ...interface{}) (interface{}, error) {
	return nil, nil
}

func (p *pathsConfigManager) GetString(key string, defaultValue ...string) string {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (p *pathsConfigManager) GetInt(key string, defaultValue ...int) int {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (p *pathsConfigManager) GetBool(key string, defaultValue ...bool) bool {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

func (p *pathsConfigManager) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (p *pathsConfigManager) Set(key string, value interface{}) error {
	return nil
}

func (p *pathsConfigManager) Save() error {
	return nil
}

func (p *pathsConfigManager) SaveTo(path string) error {
	return nil
}

func (p *pathsConfigManager) GetWorkDir() string {
	return p.paths.GetWorkDir()
}

func (p *pathsConfigManager) GetPlanDir() string {
	return p.paths.GetPlanDir()
}

func (p *pathsConfigManager) GetStatusFile() string {
	return p.paths.GetStatusFile()
}

func (p *pathsConfigManager) GetLogDir() string {
	return p.paths.GetLogDir()
}

func (p *pathsConfigManager) GetResearchDir() string {
	return p.paths.GetResearchDir()
}

func (p *pathsConfigManager) GetConfigDir() string {
	return ""
}

func (p *pathsConfigManager) GetConfigFile() string {
	return ""
}

func (p *pathsConfigManager) GetPromptsDir() string {
	return p.paths.GetPromptsDir()
}

func (p *pathsConfigManager) GetAll() map[string]interface{} {
	return nil
}
