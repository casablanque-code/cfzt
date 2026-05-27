//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const unitTemplate = `[Unit]
Description=zt tunnel — %s
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=%s tunnel --config %s run
Restart=on-failure
RestartSec=5s
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=default.target
`

func unitName(name string) string {
	return "zt-" + name + ".service"
}

func unitPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, unitName(name)), nil
}

func cloudflaredBin() (string, error) {
	path, err := exec.LookPath("cloudflared")
	if err != nil {
		return "", fmt.Errorf("cloudflared not found in PATH")
	}
	return path, nil
}

// ensureLinger enables systemd linger for the current user so that
// user services survive without an active login session.
func ensureLinger() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("could not determine current user: %w", err)
	}

	// check current linger status
	out, err := exec.Command("loginctl", "show-user", u.Username, "--property=Linger").Output()
	if err == nil && strings.TrimSpace(string(out)) == "Linger=yes" {
		return nil // already enabled
	}

	// enable linger
	if out, err := exec.Command("loginctl", "enable-linger", u.Username).CombinedOutput(); err != nil {
		return fmt.Errorf("loginctl enable-linger failed: %w\n%s", err, out)
	}
	return nil
}

// Install creates a systemd user unit, enables linger, and starts the service.
func Install(name, configPath, logPath string) error {
	bin, err := cloudflaredBin()
	if err != nil {
		return err
	}

	// ensure linger so service survives after logout / reboot without session
	if err := ensureLinger(); err != nil {
		// non-fatal — warn but continue
		fmt.Printf("     ! linger not enabled: %v\n", err)
		fmt.Println("       tunnel may not start on boot without an active session")
	}

	unit := fmt.Sprintf(unitTemplate, name, bin, configPath, logPath, logPath)

	path, err := unitPath(name)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(unit), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	cmds := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", "--now", unitName(name)},
	}
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemctl %s: %w\n%s", strings.Join(args[1:], " "), err, out)
		}
	}
	return nil
}

// Restart restarts the systemd user unit.
func Restart(name string) error {
	out, err := exec.Command("systemctl", "--user", "restart", unitName(name)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl restart: %w\n%s", err, out)
	}
	return nil
}

// Uninstall stops, disables, and removes the systemd user unit.
func Uninstall(name string) error {
	cmds := [][]string{
		{"systemctl", "--user", "disable", "--now", unitName(name)},
		{"systemctl", "--user", "daemon-reload"},
	}
	for _, args := range cmds {
		exec.Command(args[0], args[1:]...).Run()
	}

	path, err := unitPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}
	return nil
}

// UnitPath returns the path to the systemd unit file.
func UnitPath(name string) string {
	path, _ := unitPath(name)
	return path
}

// IsInstalled returns true if the unit file exists.
func IsInstalled(name string) bool {
	path, err := unitPath(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// LingerEnabled returns true if linger is active for the current user.
func LingerEnabled() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	out, err := exec.Command("loginctl", "show-user", u.Username, "--property=Linger").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "Linger=yes"
}
