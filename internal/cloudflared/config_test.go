package cloudflared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestWriteTunnelConfig(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	cfgPath, err := WriteTunnelConfig("tunnel-id-123", "grafana", "grafana.example.com", "3000", "auto", []byte(`{"fake":"creds"}`))
	if err != nil {
		t.Fatalf("WriteTunnelConfig() error = %v", err)
	}

	wantCfgPath := filepath.Join(home, ".zt", "tunnels", "grafana", "config.yml")
	if cfgPath != wantCfgPath {
		t.Errorf("WriteTunnelConfig() path = %q, want %q", cfgPath, wantCfgPath)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading generated config: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"tunnel: tunnel-id-123",
		"protocol: auto",
		"hostname: grafana.example.com",
		"service: http://localhost:3000",
		"service: http_status:404",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("generated config missing %q:\n%s", want, content)
		}
	}

	credPath := filepath.Join(home, ".zt", "tunnels", "grafana", "tunnel-id-123.json")
	credData, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("reading credentials file: %v", err)
	}
	if string(credData) != `{"fake":"creds"}` {
		t.Errorf("credentials file content = %q, want the raw credJSON unchanged", credData)
	}
}

func TestWriteTunnelConfig_EmptyProtocolDefaultsToAuto(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	cfgPath, err := WriteTunnelConfig("id", "svc", "svc.example.com", "8080", "", nil)
	if err != nil {
		t.Fatalf("WriteTunnelConfig() error = %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading generated config: %v", err)
	}
	if !strings.Contains(string(data), "protocol: auto") {
		t.Errorf("empty protocol should default to auto, got:\n%s", data)
	}
}

func TestWriteTunnelConfig_ExplicitProtocolPreserved(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	for _, proto := range []string{"quic", "http2"} {
		cfgPath, err := WriteTunnelConfig("id", "svc-"+proto, "svc.example.com", "8080", proto, nil)
		if err != nil {
			t.Fatalf("WriteTunnelConfig(%q) error = %v", proto, err)
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("reading generated config: %v", err)
		}
		if !strings.Contains(string(data), "protocol: "+proto) {
			t.Errorf("protocol %q was not preserved, got:\n%s", proto, data)
		}
	}
}

func TestConfigPath(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	got, err := ConfigPath("grafana")
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	want := filepath.Join(home, ".zt", "tunnels", "grafana", "config.yml")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestLogPath(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	got, err := LogPath("grafana")
	if err != nil {
		t.Fatalf("LogPath() error = %v", err)
	}
	want := filepath.Join(home, ".zt", "tunnels", "grafana", "cloudflared.log")
	if got != want {
		t.Errorf("LogPath() = %q, want %q", got, want)
	}
}

func TestCleanTunnelFiles(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	if _, err := WriteTunnelConfig("id", "grafana", "grafana.example.com", "3000", "auto", []byte("creds")); err != nil {
		t.Fatalf("WriteTunnelConfig() error = %v", err)
	}
	dir := filepath.Join(home, ".zt", "tunnels", "grafana")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("tunnel dir should exist before CleanTunnelFiles(): %v", err)
	}

	if err := CleanTunnelFiles("grafana"); err != nil {
		t.Fatalf("CleanTunnelFiles() error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("tunnel dir should be removed after CleanTunnelFiles(), stat err = %v", err)
	}
}

func TestCleanTunnelFiles_MissingDirIsNotAnError(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	if err := CleanTunnelFiles("never-existed"); err != nil {
		t.Errorf("CleanTunnelFiles() on a nonexistent tunnel = %v, want nil (os.RemoveAll is idempotent)", err)
	}
}
