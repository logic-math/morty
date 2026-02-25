package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}
	if router.commands == nil {
		t.Error("NewRouter() commands map is nil")
	}
}

func TestRouter_Register(t *testing.T) {
	router := NewRouter()

	tests := []struct {
		name    string
		cmd     Command
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid command",
			cmd: Command{
				Name:        "test",
				Description: "Test command",
				Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
			},
			wantErr: false,
		},
		{
			name: "command with options",
			cmd: Command{
				Name:        "options",
				Description: "Command with options",
				Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
				Options: []Option{
					{Name: "verbose", Short: "-v", Description: "Verbose output", HasValue: false},
					{Name: "output", Short: "-o", Description: "Output file", HasValue: true},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate command",
			cmd: Command{
				Name:        "test",
				Description: "Duplicate command",
				Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
			},
			wantErr: true,
			errMsg:  "already registered",
		},
		{
			name: "empty command name",
			cmd: Command{
				Name:        "",
				Description: "Empty name",
				Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
			},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name: "nil handler",
			cmd: Command{
				Name:        "nohandler",
				Description: "No handler",
				Handler:     nil,
			},
			wantErr: true,
			errMsg:  "must have a handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.Register(tt.cmd)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Register() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Register() error = %v, should contain %v", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Register() unexpected error = %v", err)
			}
		})
	}
}

func TestRouter_Register_CaseInsensitive(t *testing.T) {
	router := NewRouter()

	// Register command with uppercase
	err := router.Register(Command{
		Name:        "Test",
		Description: "Test command",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Try to register with different case
	err = router.Register(Command{
		Name:        "test",
		Description: "Duplicate case",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err == nil {
		t.Error("Register() should reject case-insensitive duplicate")
	}
}

func TestRouter_GetHandler(t *testing.T) {
	router := NewRouter()

	// Register a test command
	testHandler := func(ctx context.Context, args []string, opts ParseResult) error {
		return nil
	}

	err := router.Register(Command{
		Name:        "test",
		Description: "Test command",
		Handler:     testHandler,
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	tests := []struct {
		name       string
		cmdName    string
		wantExists bool
	}{
		{
			name:       "existing command",
			cmdName:    "test",
			wantExists: true,
		},
		{
			name:       "case insensitive lookup lowercase",
			cmdName:    "test",
			wantExists: true,
		},
		{
			name:       "case insensitive lookup uppercase",
			cmdName:    "TEST",
			wantExists: true,
		},
		{
			name:       "case insensitive lookup mixed",
			cmdName:    "Test",
			wantExists: true,
		},
		{
			name:       "non-existing command",
			cmdName:    "unknown",
			wantExists: false,
		},
		{
			name:       "empty command name",
			cmdName:    "",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, exists := router.GetHandler(tt.cmdName)
			if exists != tt.wantExists {
				t.Errorf("GetHandler() exists = %v, want %v", exists, tt.wantExists)
			}
			if tt.wantExists && handler == nil {
				t.Error("GetHandler() returned nil handler for existing command")
			}
		})
	}
}

func TestRouter_ListCommands(t *testing.T) {
	router := NewRouter()

	// Initially should return empty slice
	commands := router.ListCommands()
	if len(commands) != 0 {
		t.Errorf("ListCommands() initial length = %d, want 0", len(commands))
	}

	// Register some commands
	cmds := []Command{
		{
			Name:        "cmd1",
			Description: "Command 1",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		},
		{
			Name:        "cmd2",
			Description: "Command 2",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		},
		{
			Name:        "cmd3",
			Description: "Command 3",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		},
	}

	for _, cmd := range cmds {
		if err := router.Register(cmd); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// List commands
	commands = router.ListCommands()
	if len(commands) != 3 {
		t.Errorf("ListCommands() length = %d, want 3", len(commands))
	}

	// Verify all commands are present
	cmdMap := make(map[string]bool)
	for _, cmd := range commands {
		cmdMap[cmd.Name] = true
	}

	for _, cmd := range cmds {
		if !cmdMap[cmd.Name] {
			t.Errorf("ListCommands() missing command %s", cmd.Name)
		}
	}
}

func TestRouter_Execute(t *testing.T) {
	router := NewRouter()

	// Track if handler was called
	handlerCalled := false
	var receivedArgs []string
	var receivedOpts ParseResult

	// Register a test command
	testHandler := func(ctx context.Context, args []string, opts ParseResult) error {
		handlerCalled = true
		receivedArgs = args
		receivedOpts = opts
		return nil
	}

	err := router.Register(Command{
		Name:        "run",
		Description: "Run command",
		Handler:     testHandler,
		Options: []Option{
			{Name: "verbose", Short: "-v", HasValue: false},
			{Name: "output", Short: "-o", HasValue: true},
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Execute the command
	ctx := context.Background()
	err = router.Execute(ctx, []string{"run", "arg1", "arg2", "--verbose", "-o", "file.txt"})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if !handlerCalled {
		t.Error("Execute() handler was not called")
	}

	if len(receivedArgs) != 2 || receivedArgs[0] != "arg1" || receivedArgs[1] != "arg2" {
		t.Errorf("Execute() args = %v, want [arg1 arg2]", receivedArgs)
	}

	if !receivedOpts.HasOption("verbose") {
		t.Error("Execute() verbose option not passed")
	}

	if val, exists := receivedOpts.GetOption("output"); !exists || val != "file.txt" {
		t.Errorf("Execute() output option = %v, want file.txt", val)
	}
}

func TestRouter_Execute_UnknownCommand(t *testing.T) {
	router := NewRouter()

	// Register a known command
	err := router.Register(Command{
		Name:        "known",
		Description: "Known command",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Execute unknown command
	ctx := context.Background()
	err = router.Execute(ctx, []string{"unknown"})

	if err == nil {
		t.Fatal("Execute() expected error for unknown command")
	}

	// Check error message
	if !strings.Contains(err.Error(), "未知命令") {
		t.Errorf("Execute() error = %v, should contain '未知命令'", err)
	}

	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("Execute() error = %v, should contain 'unknown'", err)
	}

	if !strings.Contains(err.Error(), "known") {
		t.Errorf("Execute() error = %v, should list available commands", err)
	}
}

func TestRouter_Execute_NoArgs(t *testing.T) {
	router := NewRouter()

	ctx := context.Background()
	err := router.Execute(ctx, []string{})

	if err == nil {
		t.Fatal("Execute() expected error for no args")
	}

	if !strings.Contains(err.Error(), "no command") {
		t.Errorf("Execute() error = %v, should contain 'no command'", err)
	}
}

func TestRouter_Execute_HandlerError(t *testing.T) {
	router := NewRouter()

	expectedErr := errors.New("handler error")
	err := router.Register(Command{
		Name:        "error",
		Description: "Error command",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return expectedErr },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx := context.Background()
	err = router.Execute(ctx, []string{"error"})

	if err == nil {
		t.Fatal("Execute() expected error from handler")
	}

	if !errors.Is(err, expectedErr) && !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestRouter_Execute_CaseInsensitive(t *testing.T) {
	router := NewRouter()

	handlerCalled := false
	err := router.Register(Command{
		Name:        "Test",
		Description: "Test command",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { handlerCalled = true; return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx := context.Background()

	// Test with different cases
	cases := []string{"test", "TEST", "Test", "tEsT"}
	for _, c := range cases {
		handlerCalled = false
		err = router.Execute(ctx, []string{c})
		if err != nil {
			t.Errorf("Execute(%s) error = %v", c, err)
		}
		if !handlerCalled {
			t.Errorf("Execute(%s) handler not called", c)
		}
	}
}

func TestRouter_ConcurrentAccess(t *testing.T) {
	router := NewRouter()
	ctx := context.Background()

	// Register initial command
	err := router.Register(Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Run concurrent operations
	done := make(chan bool, 10)

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				router.GetHandler("test")
				router.ListCommands()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			for j := 0; j < 10; j++ {
				cmd := Command{
					Name:        fmt.Sprintf("cmd%d_%d", idx, j),
					Description: "Concurrent command",
					Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
				}
				router.Register(cmd)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify router is still functional
	_, exists := router.GetHandler("test")
	if !exists {
		t.Error("GetHandler() test command not found after concurrent access")
	}

	commands := router.ListCommands()
	if len(commands) < 1 {
		t.Error("ListCommands() returned empty after concurrent access")
	}

	// Test execution still works
	err = router.Execute(ctx, []string{"test"})
	if err != nil {
		t.Errorf("Execute() error after concurrent access = %v", err)
	}
}

func TestRouter_Register_Multiple(t *testing.T) {
	router := NewRouter()

	commands := []Command{
		{
			Name:        "cmd1",
			Description: "Command 1",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		},
		{
			Name:        "cmd2",
			Description: "Command 2",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		},
		{
			Name:        "cmd3",
			Description: "Command 3",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		},
	}

	for _, cmd := range commands {
		if err := router.Register(cmd); err != nil {
			t.Errorf("Register(%s) error = %v", cmd.Name, err)
		}
	}

	// Verify all commands are registered
	for _, cmd := range commands {
		_, exists := router.GetHandler(cmd.Name)
		if !exists {
			t.Errorf("GetHandler(%s) not found", cmd.Name)
		}
	}
}

func TestRouter_Execute_Context(t *testing.T) {
	router := NewRouter()

	var receivedCtx context.Context
	err := router.Register(Command{
		Name:        "ctx",
		Description: "Context test",
		Handler: func(ctx context.Context, args []string, opts ParseResult) error {
			receivedCtx = ctx
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx := context.WithValue(context.Background(), "key", "value")
	err = router.Execute(ctx, []string{"ctx"})
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if receivedCtx == nil {
		t.Error("Execute() context not passed to handler")
	}

	if receivedCtx.Value("key") != "value" {
		t.Error("Execute() context value not preserved")
	}
}

func TestRouter_Execute_ParsingError(t *testing.T) {
	router := NewRouter()

	err := router.Register(Command{
		Name:        "parse",
		Description: "Parse test",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		Options: []Option{
			{Name: "required", Short: "-r", HasValue: true, Required: false},
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create a parser that requires values for certain options
	ctx := context.Background()

	// This should work (no value after --required is okay for unknown options)
	err = router.Execute(ctx, []string{"parse", "--unknown"})
	if err != nil {
		// This is expected to work since unknown options are treated as bool
		t.Logf("Execute() error (may be expected): %v", err)
	}
}

func TestRouter_getAvailableCommands(t *testing.T) {
	router := NewRouter()

	// Empty router
	available := router.getAvailableCommands()
	if available != "(none)" {
		t.Errorf("getAvailableCommands() = %v, want (none)", available)
	}

	// Add commands
	err := router.Register(Command{
		Name:        "alpha",
		Description: "Alpha",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err = router.Register(Command{
		Name:        "beta",
		Description: "Beta",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	available = router.getAvailableCommands()
	if !strings.Contains(available, "alpha") || !strings.Contains(available, "beta") {
		t.Errorf("getAvailableCommands() = %v, should contain alpha and beta", available)
	}
}

func TestRouter_GetHandler_Concurrent(t *testing.T) {
	router := NewRouter()

	// Register commands
	for i := 0; i < 10; i++ {
		err := router.Register(Command{
			Name:        fmt.Sprintf("cmd%d", i),
			Description: fmt.Sprintf("Command %d", i),
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Concurrent reads
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(idx int) {
			for j := 0; j < 50; j++ {
				cmdName := fmt.Sprintf("cmd%d", idx%10)
				router.GetHandler(cmdName)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestCommand_Struct(t *testing.T) {
	cmd := Command{
		Name:        "test",
		Description: "Test description",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		Options: []Option{
			{Name: "opt1", Short: "-o", Description: "Option 1", HasValue: true, Required: true},
		},
	}

	if cmd.Name != "test" {
		t.Errorf("Command.Name = %v, want test", cmd.Name)
	}
	if cmd.Description != "Test description" {
		t.Errorf("Command.Description = %v, want 'Test description'", cmd.Description)
	}
	if cmd.Handler == nil {
		t.Error("Command.Handler is nil")
	}
	if len(cmd.Options) != 1 {
		t.Errorf("Command.Options length = %v, want 1", len(cmd.Options))
	}
}

func TestOption_Struct(t *testing.T) {
	opt := Option{
		Name:        "verbose",
		Short:       "-v",
		Description: "Verbose output",
		HasValue:    false,
		Required:    false,
	}

	if opt.Name != "verbose" {
		t.Errorf("Option.Name = %v, want verbose", opt.Name)
	}
	if opt.Short != "-v" {
		t.Errorf("Option.Short = %v, want -v", opt.Short)
	}
	if opt.HasValue != false {
		t.Errorf("Option.HasValue = %v, want false", opt.HasValue)
	}
}

func TestRouter_Execute_PreservesCommandName(t *testing.T) {
	router := NewRouter()

	var capturedResult ParseResult
	err := router.Register(Command{
		Name:        "test",
		Description: "Test command",
		Handler: func(ctx context.Context, args []string, opts ParseResult) error {
			capturedResult = opts
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx := context.Background()
	err = router.Execute(ctx, []string{"test", "arg1"})
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if capturedResult.Command != "test" {
		t.Errorf("ParseResult.Command = %v, want test", capturedResult.Command)
	}
}

func TestRouter_ListCommands_ReturnsCopy(t *testing.T) {
	router := NewRouter()

	err := router.Register(Command{
		Name:        "test",
		Description: "Original",
		Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	commands1 := router.ListCommands()
	if len(commands1) != 1 {
		t.Fatalf("ListCommands() length = %d, want 1", len(commands1))
	}

	// Modify the returned slice
	commands1[0].Description = "Modified"

	// Get commands again
	commands2 := router.ListCommands()
	if commands2[0].Description != "Original" {
		t.Error("ListCommands() returns reference to internal state")
	}
}

func BenchmarkRouter_Register(b *testing.B) {
	router := NewRouter()

	for i := 0; i < b.N; i++ {
		cmd := Command{
			Name:        fmt.Sprintf("cmd%d", i),
			Description: "Benchmark command",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		}
		router.Register(cmd)
	}
}

func BenchmarkRouter_GetHandler(b *testing.B) {
	router := NewRouter()

	// Register some commands
	for i := 0; i < 100; i++ {
		router.Register(Command{
			Name:        fmt.Sprintf("cmd%d", i),
			Description: "Test command",
			Handler:     func(ctx context.Context, args []string, opts ParseResult) error { return nil },
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.GetHandler(fmt.Sprintf("cmd%d", i%100))
	}
}
