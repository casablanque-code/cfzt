package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	content := `
services:
  grafana:
    port: 3000
  portainer:
    docker: true
    allow: [you@example.com]
  api:
    port: 8080
    public: true
    protocol: quic
`
	path := writeTmp(t, content)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(m.Services) != 3 {
		t.Fatalf("expected 3 services, got %d", len(m.Services))
	}

	grafana := m.Services["grafana"]
	if grafana.Port != 3000 {
		t.Errorf("grafana.Port = %d, want 3000", grafana.Port)
	}
	if grafana.Docker {
		t.Error("grafana.Docker should be false")
	}

	portainer := m.Services["portainer"]
	if !portainer.Docker {
		t.Error("portainer.Docker should be true")
	}
	if len(portainer.Allow) != 1 || portainer.Allow[0] != "you@example.com" {
		t.Errorf("portainer.Allow = %v, want [you@example.com]", portainer.Allow)
	}

	api := m.Services["api"]
	if !api.Public {
		t.Error("api.Public should be true")
	}
	if api.Protocol != "quic" {
		t.Errorf("api.Protocol = %q, want quic", api.Protocol)
	}
}

func TestLoadMissingPortAndDocker(t *testing.T) {
	content := `
services:
  broken:
    public: true
`
	path := writeTmp(t, content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should return error when port and docker are both missing")
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/zt.yaml")
	if err == nil {
		t.Fatal("Load() should return error for missing file")
	}
}

func TestLoadEmptyServices(t *testing.T) {
	content := "services: {}\n"
	path := writeTmp(t, content)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(m.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(m.Services))
	}
}

func TestSaveRoundtrip(t *testing.T) {
	original := &Manifest{
		Services: map[string]ServiceSpec{
			"grafana": {Port: 3000},
			"vault":   {Port: 8200, Protocol: "quic"},
			"agent":   {Docker: true, Allow: []string{"alice@example.com", "bob@example.com"}},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "zt.yaml")

	if err := Save(path, original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}

	if len(loaded.Services) != len(original.Services) {
		t.Fatalf("service count mismatch: got %d, want %d", len(loaded.Services), len(original.Services))
	}

	for name, want := range original.Services {
		got, ok := loaded.Services[name]
		if !ok {
			t.Errorf("service %q missing after roundtrip", name)
			continue
		}
		if got.Port != want.Port {
			t.Errorf("%s.Port: got %d, want %d", name, got.Port, want.Port)
		}
		if got.Protocol != want.Protocol {
			t.Errorf("%s.Protocol: got %q, want %q", name, got.Protocol, want.Protocol)
		}
		if got.Docker != want.Docker {
			t.Errorf("%s.Docker: got %v, want %v", name, got.Docker, want.Docker)
		}
		if len(got.Allow) != len(want.Allow) {
			t.Errorf("%s.Allow: got %v, want %v", name, got.Allow, want.Allow)
		}
	}
}

func TestSaveHeaderPresent(t *testing.T) {
	m := &Manifest{Services: map[string]ServiceSpec{"svc": {Port: 9000}}}
	dir := t.TempDir()
	path := filepath.Join(dir, "zt.yaml")

	if err := Save(path, m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if content[:2] != "# " {
		t.Error("saved file should start with a comment header")
	}
}

// writeTmp writes content to a temp file and returns its path.
func writeTmp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "zt-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}
