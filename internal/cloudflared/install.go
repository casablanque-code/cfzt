package cloudflared

import (
	"os/exec"
	"runtime"
	"strings"
)

// InstallMethod identifies how cloudflared appears to have been installed,
// so we can suggest the matching upgrade command instead of a generic
// "go download it" link.
type InstallMethod string

const (
	InstallUnknown  InstallMethod = ""
	InstallHomebrew InstallMethod = "homebrew"
	InstallAPT      InstallMethod = "apt"
	InstallBinary   InstallMethod = "binary" // manually downloaded / not package-managed
)

// DetectInstallMethod best-effort sniffs how the cloudflared on PATH was
// installed. It never errors - an unknown result just means we fall back
// to the generic downloads page.
func DetectInstallMethod() InstallMethod {
	path, err := exec.LookPath("cloudflared")
	if err != nil {
		return InstallUnknown
	}

	switch runtime.GOOS {
	case "darwin":
		if out, err := exec.Command("brew", "list", "--formula", "cloudflared").CombinedOutput(); err == nil && len(out) > 0 {
			return InstallHomebrew
		}
	case "linux":
		if out, err := exec.Command("dpkg", "-S", path).CombinedOutput(); err == nil && strings.Contains(string(out), "cloudflared") {
			return InstallAPT
		}
	}

	return InstallBinary
}

// UpgradeCommand returns the shell command to upgrade cloudflared given how
// it was installed, and ok=false if there's no better suggestion than the
// generic downloads page.
func UpgradeCommand(m InstallMethod) (cmd string, ok bool) {
	switch m {
	case InstallHomebrew:
		return "brew upgrade cloudflared", true
	case InstallAPT:
		return "sudo apt update && sudo apt install --only-upgrade cloudflared", true
	default:
		return "", false
	}
}
