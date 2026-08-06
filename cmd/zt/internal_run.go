package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// internalCmd groups hidden subcommands that exist only as entrypoints for
// zt's own service integrations (Task Scheduler actions, etc.) — never
// meant to be typed by a user.
var internalCmd = &cobra.Command{
	Use:    "internal",
	Hidden: true,
}

var internalRunLogPath string

// internalRunCmd runs an arbitrary command with its stdout/stderr appended
// to a log file, then exits with the child's own exit code.
//
// This exists because Windows Task Scheduler actions have no redirection
// of their own — there's no ">>" equivalent in an <Exec> action. The
// previous approach wrapped every action in "cmd.exe /c ... >> log 2>&1",
// which meant every argument (including whatever path cloudflared or its
// config file happened to live at) passed through three separate layers
// of quoting before the real process ever saw it: Task Scheduler's XML,
// cmd.exe's own argument parsing, and then cmd.exe re-parsing the /c
// string a second time. `zt internal run` replaces all of that: Task
// Scheduler calls zt directly, zt opens the log file itself and execs the
// real command with a real argv — one quoting layer, not three, using the
// same CommandLineToArgvW rules quoteArg already implements for the outer
// Task Scheduler call.
//
// Exiting with the child's real exit code (instead of swallowing it,
// which is what happened whenever cmd.exe itself launched successfully)
// is also what lets Task Scheduler's RestartOnFailure detect a crash at
// all — see service_windows.go's waitForTaskRunning.
var internalRunCmd = &cobra.Command{
	Use:    "run --log <path> -- <command> [args...]",
	Short:  "Run a command with output appended to a log file (used internally by zt's service integrations)",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE:   runInternalRun,
}

func init() {
	internalRunCmd.Flags().StringVar(&internalRunLogPath, "log", "", "path to append the command's stdout/stderr to")
	_ = internalRunCmd.MarkFlagRequired("log")
	internalCmd.AddCommand(internalRunCmd)
	// internalCmd itself is registered on rootCmd in main() (main.go),
	// alongside the rest of the command-group wiring.
}

func runInternalRun(cmd *cobra.Command, args []string) error {
	hideOwnConsoleWindow()

	logFile, err := os.OpenFile(internalRunLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("zt internal run: failed to open log file %q: %w", internalRunLogPath, err)
	}
	defer logFile.Close()

	child := exec.Command(args[0], args[1:]...)
	child.Stdout = logFile
	child.Stderr = logFile
	child.Stdin = nil
	configureChildForBackground(child)

	runErr := child.Run()
	if runErr == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		// Propagate the real exit code so a caller polling the process
		// tree (or, on Windows, Task Scheduler's Last Result /
		// RestartOnFailure) can tell a crash from a clean exit — cobra's
		// own error path always exits 1 regardless of what the child did.
		os.Exit(exitErr.ExitCode())
	}
	// The command couldn't even be started (not found, permissions, ...).
	return fmt.Errorf("zt internal run: %w", runErr)
}
