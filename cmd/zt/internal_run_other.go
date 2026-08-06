//go:build !windows

package main

import "os/exec"

// hideOwnConsoleWindow is a no-op outside Windows — there's no console
// window allocation to fight with. See internal_run_windows.go.
func hideOwnConsoleWindow() {}

// configureChildForBackground is a no-op outside Windows — process groups
// and job control there already do the right thing without extra flags.
// See internal_run_windows.go.
func configureChildForBackground(cmd *exec.Cmd) {}
