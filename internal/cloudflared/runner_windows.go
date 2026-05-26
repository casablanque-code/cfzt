//go:build windows

package cloudflared

import "fmt"

func Start(configPath string) (int, error) {
	return 0, fmt.Errorf("Windows is not supported — run cloudflared manually")
}

func Stop(pid int) error { return nil }

func IsRunning(pid int) bool { return false }
