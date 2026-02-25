//go:build !windows
// +build !windows

// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"os"
	"os/exec"
	"syscall"
)

// setupProcessGroup sets up the process group for the command.
// This allows signals to be sent to all processes in the group.
func setupProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Setpgid creates a new process group for the child process
	cmd.SysProcAttr.Setpgid = true
}

// signalProcessGroup sends a signal to the process group.
func signalProcessGroup(pid int, sig os.Signal) error {
	// Negative PID signals the entire process group
	return syscall.Kill(-pid, sig.(syscall.Signal))
}

// getProcessGroupID returns the process group ID for the given PID.
func getProcessGroupID(pid int) (int, error) {
	// Get process group ID
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return 0, err
	}
	return pgid, nil
}
