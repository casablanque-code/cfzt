//go:build windows

package service

import "fmt"

func Install(name, configPath, logPath string) error {
	return fmt.Errorf("systemd/launchd integration is not supported on Windows\n  run cloudflared manually: cloudflared tunnel --config %s run", configPath)
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
