//go:build !windows

package cloudflared

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Start launches cloudflared tunnel in the background.
// Returns the PID of the spawned process.
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Detach — don't wait
	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	return cmd.Process.Pid, nil
}

// Stop sends SIGTERM to a cloudflared process by PID.
func Stop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil // already gone
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone {
			return nil
		}
		return fmt.Errorf("failed to stop cloudflared (pid %d): %w", pid, err)
	}
	return nil
}

// IsRunning returns true if the process with the given PID is alive.
func IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
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
