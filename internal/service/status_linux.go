//go:build linux

package service

import (
	"os/exec"
	"strings"
)

// IsActive returns true if the systemd user unit is active.
func IsActive(name string) bool {
	out, err := exec.Command("systemctl", "--user", "is-active", unitName(name)).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}
