package cloudflared

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteTunnelConfig writes a cloudflared config.yml and credentials file.
// protocol can be "auto", "quic", or "http2". Empty string defaults to "auto".
func WriteTunnelConfig(tunnelID, tunnelName, hostname, port, protocol string, credJSON []byte) (string, error) {
	dir, err := tunnelDir(tunnelName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create tunnel dir: %w", err)
	}

	credPath := filepath.Join(dir, tunnelID+".json")
	if err := os.WriteFile(credPath, credJSON, 0600); err != nil {
		return "", fmt.Errorf("failed to write credentials: %w", err)
	}

	if protocol == "" || protocol == "auto" {
		protocol = "auto"
	}

	cfgContent := fmt.Sprintf(`tunnel: %s
credentials-file: %s
protocol: %s

ingress:
  - hostname: %s
    service: http://localhost:%s
  - service: http_status:404
`, tunnelID, credPath, protocol, hostname, port)

	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	return cfgPath, nil
}

// CleanTunnelFiles removes the config directory for a tunnel.
func CleanTunnelFiles(tunnelName string) error {
	dir, err := tunnelDir(tunnelName)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// ConfigPath returns the path to the config.yml for a tunnel.
func ConfigPath(tunnelName string) (string, error) {
	dir, err := tunnelDir(tunnelName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// LogPath returns the path to the cloudflared.log for a tunnel.
func LogPath(tunnelName string) (string, error) {
	dir, err := tunnelDir(tunnelName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cloudflared.log"), nil
}

func tunnelDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zt", "tunnels", name), nil
}
