//go:build darwin

package service

import (
	"os/exec"
	"strings"
)

// IsActive returns true if the LaunchAgent is loaded and running.
func IsActive(name string) bool {
	out, err := exec.Command("launchctl", "list", "com.zt."+name).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), `"com.zt.`+name+`"`)
}
