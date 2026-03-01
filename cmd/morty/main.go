package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	// Load global configuration
	// Try to load from multiple locations:
	// 1) ~/.morty/config.json (user home)
	// 2) <binary_dir>/../config.json (installation directory)
	// 3) ./.morty/config.json (project local)
	var cfgLoader *config.Loader

	configPaths := []string{
		os.ExpandEnv("${HOME}/.morty/config.json"),
	}

	// Add binary directory config path
	if execPath, err := os.Executable(); err == nil {
		binDir := filepath.Dir(execPath)
		installDir := filepath.Dir(binDir)
		configPaths = append(configPaths, filepath.Join(installDir, "config.json"))
	}

	configPaths = append(configPaths, "./.morty/config.json")

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			cfgLoader = config.NewLoader()
			if err := cfgLoader.Load(configPath); err == nil {
				// Successfully loaded
				// Debug: print loaded config path and prompts dir
				if os.Getenv("MORTY_DEBUG") != "" {
					fmt.Fprintf(os.Stderr, "DEBUG: Loaded config from: %s\n", configPath)
					cfg := cfgLoader.Config()
					if cfg != nil {
						fmt.Fprintf(os.Stderr, "DEBUG: Config.Prompts.Dir = %q\n", cfg.Prompts.Dir)
					} else {
						fmt.Fprintf(os.Stderr, "DEBUG: Config is nil!\n")
					}
				}
				break
			}
			cfgLoader = nil
		}
	}

	// Setup logging
	var logConfig *config.LoggingConfig
	if cfgLoader != nil && cfgLoader.Config() != nil {
		// Use logging config from loaded configuration
		logConfig = &cfgLoader.Config().Logging
	} else {
		// Use default logging config
		logConfig = &config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}
	}
	logger, _, err := logging.NewLoggerFromConfig(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create paths with config loader
	var cfg *config.Paths
	if cfgLoader != nil {
		cfg = config.NewPathsWithLoader(cfgLoader)
	} else {
		cfg = config.NewPaths()
	}

	// Route to appropriate handler
	switch command {
	case "research":
		handleResearch(cfg, cfgLoader, logger, os.Args[2:])
	case "plan":
		handlePlan(cfg, cfgLoader, logger, os.Args[2:])
	case "doing":
		handleDoing(cfg, cfgLoader, logger, os.Args[2:])
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

func handleResearch(cfg *config.Paths, cfgLoader *config.Loader, logger logging.Logger, args []string) {
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

	// Use loader if available, otherwise use paths wrapper
	var cfgMgr config.Manager
	if cfgLoader != nil {
		cfgMgr = cfgLoader
	} else {
		cfgMgr = &pathsConfigManager{paths: cfg}
	}

	handler := cmd.NewResearchHandler(cfgMgr, logger)
	ctx := context.Background()

	_, err := handler.Execute(ctx, fs.Args())
	if err != nil {
		logger.Error("Research failed", logging.String("error", err.Error()))
		os.Exit(1)
	}

	fmt.Println("✓ Research completed")
}

func handlePlan(cfg *config.Paths, cfgLoader *config.Loader, logger logging.Logger, args []string) {
	// Check for subcommands
	if len(args) > 0 && args[0] == "validate" {
		handlePlanValidate(cfg, cfgLoader, logger, args[1:])
		return
	}

	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	_ = fs.String("module", "", "Target module name") // reserved for future use
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty plan [subcommand] [options]")
		fmt.Println()
		fmt.Println("Create a development plan based on research.")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  validate    Validate plan file format")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -module string    Target module name")
		os.Exit(0)
	}

	// Use loader if available, otherwise use paths wrapper
	var cfgMgr config.Manager
	if cfgLoader != nil {
		cfgMgr = cfgLoader
	} else {
		cfgMgr = &pathsConfigManager{paths: cfg}
	}

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

func handleDoing(cfg *config.Paths, cfgLoader *config.Loader, logger logging.Logger, args []string) {
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

	// Use loader if available, otherwise use paths wrapper
	var cfgMgr config.Manager
	if cfgLoader != nil {
		cfgMgr = cfgLoader
	} else {
		cfgMgr = &pathsConfigManager{paths: cfg}
	}

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
	list := fs.Bool("l", false, "List recent commits")
	clean := fs.Bool("c", false, "Clean reset")
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty reset [options]")
		fmt.Println()
		fmt.Println("Reset workflow state.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -l [count]    List recent commits")
		fmt.Println("  -c            Clean reset")
		fmt.Println("  hash          Reset to specific commit")
		os.Exit(0)
	}

	// Build args for handler - include parsed flags
	handlerArgs := []string{}
	if *list {
		handlerArgs = append(handlerArgs, "-l")
		// Check if there's a count argument after -l
		remaining := fs.Args()
		if len(remaining) > 0 {
			if _, err := fmt.Sscanf(remaining[0], "%d", new(int)); err == nil {
				handlerArgs = append(handlerArgs, remaining[0])
			}
		}
	}
	if *clean {
		handlerArgs = append(handlerArgs, "-c")
	}
	// Add remaining args (commit hash, etc.)
	handlerArgs = append(handlerArgs, fs.Args()...)

	cfgMgr := &pathsConfigManager{paths: cfg}
	handler := cmd.NewResetHandler(cfgMgr, logger)
	ctx := context.Background()

	_, err := handler.Execute(ctx, handlerArgs)
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

// handlePlanValidate handles the 'morty plan validate' subcommand.
func handlePlanValidate(cfg *config.Paths, cfgLoader *config.Loader, logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("plan validate", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	verbose := fs.Bool("verbose", false, "Show verbose output")
	verboseShort := fs.Bool("v", false, "Show verbose output (shorthand)")
	fix := fs.Bool("fix", false, "Auto-fix format issues if possible")
	fixShort := fs.Bool("f", false, "Auto-fix format issues (shorthand)")
	fs.Parse(args)

	if *help {
		fmt.Println("Usage: morty plan validate [options] [file]")
		fmt.Println()
		fmt.Println("Validate plan file format against specification.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -v, --verbose    Show detailed error information")
		fmt.Println("  -f, --fix        Auto-fix format issues if possible")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  file            Validate single file (optional)")
		fmt.Println("                  If not specified, validates all files in plan directory")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  morty plan validate                  # Validate all plan files")
		fmt.Println("  morty plan validate user_auth.md     # Validate single file")
		fmt.Println("  morty plan validate --verbose        # Show detailed errors")
		fmt.Println("  morty plan validate --fix            # Auto-fix issues")
		os.Exit(0)
	}

	// Combine verbose flags
	isVerbose := *verbose || *verboseShort
	isFix := *fix || *fixShort

	// Use loader if available, otherwise use paths wrapper
	var cfgMgr config.Manager
	if cfgLoader != nil {
		cfgMgr = cfgLoader
	} else {
		cfgMgr = &pathsConfigManager{paths: cfg}
	}

	handler := cmd.NewPlanHandler(cfgMgr, logger, nil)
	ctx := context.Background()

	// Build args for Validate method
	validateArgs := fs.Args()
	if isVerbose {
		validateArgs = append(validateArgs, "--verbose")
	}
	if isFix {
		validateArgs = append(validateArgs, "--fix")
	}

	result, err := handler.Validate(ctx, validateArgs)
	if err != nil {
		logger.Error("Validation failed", logging.String("error", err.Error()))
		os.Exit(1)
	}

	// Print result
	fmt.Print(result.Message)

	// Exit with appropriate code
	if !result.Success {
		os.Exit(1)
	}
}
