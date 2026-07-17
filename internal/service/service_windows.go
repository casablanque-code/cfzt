//go:build windows

package service

import "fmt"

// Install always fails on Windows — there's no systemd/launchd-equivalent
// service integration yet, so callers (zt up, zt restart) fall back to
// PID-tracked mode automatically, same as they would if systemd/launchd
// were simply unavailable on Linux/macOS. See runner_windows.go.
func Install(name, configPath, logPath string) error {
	return fmt.Errorf("no service manager integration on Windows yet — running in PID mode (no auto-restart on crash, no start-on-boot)")
}

func Restart(name string) error {
	return fmt.Errorf("service restart is not supported on Windows")
}

func Uninstall(name string) error  { return nil }
func UnitPath(name string) string  { return "" }
func IsInstalled(name string) bool { return false }
func LingerEnabled() bool          { return false }

func InstallWatchdog(logPath string) error {
	return fmt.Errorf("watchdog is not supported on Windows — run `zt watchdog run` manually if needed")
}
func UninstallWatchdog() error  { return nil }
func WatchdogIsInstalled() bool { return false }
func WatchdogIsActive() bool    { return false }
