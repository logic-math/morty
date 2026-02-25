//go:build windows
// +build windows

// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"os"
	"os/exec"
)

// setupProcessGroup is a no-op on Windows as process groups work differently.
func setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use the same process group model as Unix
	// Job objects would be needed for equivalent functionality
}

// signalProcessGroup sends a signal to the process on Windows.
// On Windows, we can only signal the individual process.
func signalProcessGroup(pid int, sig os.Signal) error {
	// Windows doesn't support process group signaling in the same way as Unix
	// Return an error to fall back to individual process signaling
	return os.ErrProcessDone
}

// getProcessGroupID returns the process ID on Windows (no process groups).
func getProcessGroupID(pid int) (int, error) {
	return pid, nil
}
