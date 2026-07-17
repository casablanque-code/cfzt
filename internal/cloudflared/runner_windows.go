//go:build windows

package cloudflared

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// Start launches cloudflared tunnel in the background.
// Returns the PID of the spawned process.
//
// There is no systemd/launchd equivalent wired up on Windows yet (see
// internal/service/service_windows.go), so every tunnel on Windows runs in
// the same PID-tracked mode that macOS/Linux only fall back to when the
// real service manager is unavailable. That means no auto-restart on
// crash and no start-on-boot — a proper Windows Service (Task Scheduler or
// the SCM) is tracked separately.
func Start(configPath string) (int, error) {
	cfPath, err := which("cloudflared")
	if err != nil {
		return 0, fmt.Errorf("cloudflared not found in PATH — install it: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/")
	}

	logFile, err := openLogFile(configPath)
	if err != nil {
		return 0, err
	}

	cmd := exec.Command(cfPath, "tunnel", "--config", configPath, "run")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// CREATE_NEW_PROCESS_GROUP lets Stop send CTRL_BREAK_EVENT to just this
	// process tree instead of every console process, and detaches it from
	// zt's own console so closing the terminal doesn't kill the tunnel.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return 0, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Detach — don't wait
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
	}()

	return cmd.Process.Pid, nil
}

// Stop terminates a cloudflared process by PID. Windows has no SIGTERM;
// we ask nicely via CTRL_BREAK_EVENT first (cloudflared handles it as a
// graceful shutdown signal same as SIGINT/SIGTERM on other platforms),
// then fall back to a hard TerminateProcess if it's still alive shortly
// after.
func Stop(pid int) error {
	if pid <= 0 {
		return nil
	}

	if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid)); err == nil {
		// Give it a moment to exit gracefully before considering a hard kill.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if !IsRunning(pid) {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if !IsRunning(pid) {
		return nil
	}

	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		// Already gone between our check and here.
		return nil
	}
	defer windows.CloseHandle(h)

	if err := windows.TerminateProcess(h, 1); err != nil {
		return fmt.Errorf("failed to stop cloudflared (pid %d): %w", pid, err)
	}
	return nil
}

// IsRunning returns true if the process with the given PID is alive.
// os.Process.Signal(0) - the POSIX trick used on Linux/macOS - doesn't
// work on Windows, so we open the process and check its exit code instead.
func IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	const stillActive = 259 // STATUS_PENDING / STILL_ACTIVE
	return exitCode == stillActive
}

func which(bin string) (string, error) {
	path, err := exec.LookPath(bin)
	if err != nil {
		return "", err
	}
	return path, nil
}

func openLogFile(configPath string) (*os.File, error) {
	logPath := configPath[:len(configPath)-len("config.yml")] + "cloudflared.log"
	return os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}
