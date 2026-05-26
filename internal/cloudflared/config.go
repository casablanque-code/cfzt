package cloudflared

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteTunnelConfig writes a cloudflared config.yml and credentials file.
// Returns the path to the config file.
func WriteTunnelConfig(tunnelID, tunnelName, hostname, port string, credJSON []byte) (string, error) {
	dir, err := tunnelDir(tunnelName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create tunnel dir: %w", err)
	}

	// Write credentials file
	credPath := filepath.Join(dir, tunnelID+".json")
	if err := os.WriteFile(credPath, credJSON, 0600); err != nil {
		return "", fmt.Errorf("failed to write credentials: %w", err)
	}

	// Write config.yml
	cfgContent := fmt.Sprintf(`tunnel: %s
credentials-file: %s

ingress:
  - hostname: %s
    service: http://localhost:%s
  - service: http_status:404
`, tunnelID, credPath, hostname, port)

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

func tunnelDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zt", "tunnels", name), nil
}
