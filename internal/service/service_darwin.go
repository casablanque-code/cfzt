//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.zt.%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>tunnel</string>
		<string>--config</string>
		<string>%s</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`

func plistName(name string) string {
	return "com.zt." + name + ".plist"
}

func plistPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, plistName(name)), nil
}

func cloudflaredBin() (string, error) {
	path, err := exec.LookPath("cloudflared")
	if err != nil {
		return "", fmt.Errorf("cloudflared not found in PATH")
	}
	return path, nil
}

// Install creates a LaunchAgent plist and loads it.
func Install(name, configPath, logPath string) error {
	bin, err := cloudflaredBin()
	if err != nil {
		return err
	}

	plist := fmt.Sprintf(plistTemplate, name, bin, configPath, logPath, logPath)

	path, err := plistPath(name)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	out, err := exec.Command("launchctl", "load", "-w", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	return nil
}

// Restart unloads and reloads the LaunchAgent.
func Restart(name string) error {
	path, err := plistPath(name)
	if err != nil {
		return err
	}
	exec.Command("launchctl", "unload", path).Run()
	out, err := exec.Command("launchctl", "load", "-w", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	return nil
}

// Uninstall unloads and removes the LaunchAgent plist.
func Uninstall(name string) error {
	path, err := plistPath(name)
	if err != nil {
		return err
	}
	exec.Command("launchctl", "unload", "-w", path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}
	return nil
}

// UnitPath returns the path to the plist file.
func UnitPath(name string) string {
	path, _ := plistPath(name)
	return path
}

// IsInstalled returns true if the plist exists.
func IsInstalled(name string) bool {
	path, err := plistPath(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

const watchdogLabel = "com.zt.watchdog"

const watchdogPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.zt.watchdog</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>watchdog</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`

func watchdogPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, watchdogLabel+".plist"), nil
}

// InstallWatchdog creates and loads the watchdog LaunchAgent.
func InstallWatchdog(logPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine zt binary path: %w", err)
	}

	plist := fmt.Sprintf(watchdogPlistTemplate, exe, logPath, logPath)

	path, err := watchdogPlistPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return fmt.Errorf("failed to write watchdog plist: %w", err)
	}

	out, err := exec.Command("launchctl", "load", "-w", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	return nil
}

// UninstallWatchdog unloads and removes the watchdog LaunchAgent.
func UninstallWatchdog() error {
	path, err := watchdogPlistPath()
	if err != nil {
		return err
	}
	exec.Command("launchctl", "unload", "-w", path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove watchdog plist: %w", err)
	}
	return nil
}

// WatchdogIsInstalled returns true if the watchdog plist exists.
func WatchdogIsInstalled() bool {
	path, err := watchdogPlistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// LingerEnabled always returns true on macOS — LaunchAgents run at login automatically.
func LingerEnabled() bool { return true }
