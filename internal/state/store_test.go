package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadStore_MissingFileReturnsEmptyStore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	s, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v, want nil for a missing file", err)
	}
	if s.Tunnels == nil {
		t.Fatal("LoadStore() Tunnels map is nil, want an initialized empty map")
	}
	if len(s.All()) != 0 {
		t.Errorf("LoadStore() has %d tunnels, want 0 for a fresh store", len(s.All()))
	}
}

func TestLoadStore_Malformed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(home, stateFileName), []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadStore(); err == nil {
		t.Fatal("LoadStore() = nil error, want error for malformed JSON")
	}
}

func TestLoadStore_NullTunnelsFieldBecomesEmptyMap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// A state file saved with zero tunnels serializes "tunnels":null via
	// encoding/json for a nil map - LoadStore must not hand back a nil
	// map that panics on the next Set().
	if err := os.WriteFile(filepath.Join(home, stateFileName), []byte(`{"tunnels":null}`), 0600); err != nil {
		t.Fatal(err)
	}

	s, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if s.Tunnels == nil {
		t.Fatal("LoadStore() left Tunnels nil for a null field")
	}
	// Would panic here if Tunnels were nil.
	s.Set(&Tunnel{Name: "grafana"})
}

func TestStore_SetGetDelete(t *testing.T) {
	s := &Store{Tunnels: make(map[string]*Tunnel)}

	tun := &Tunnel{Name: "grafana", Port: 3000}
	s.Set(tun)

	got, ok := s.Get("grafana")
	if !ok || got.Name != "grafana" || got.Port != 3000 {
		t.Errorf("Get(grafana) = %+v, %v, want the tunnel we just Set", got, ok)
	}

	if _, ok := s.Get("nonexistent"); ok {
		t.Error("Get(nonexistent) = true, want false")
	}

	s.Delete("grafana")
	if _, ok := s.Get("grafana"); ok {
		t.Error("Get(grafana) after Delete = true, want false")
	}
}

func TestStore_All(t *testing.T) {
	s := &Store{Tunnels: make(map[string]*Tunnel)}
	s.Set(&Tunnel{Name: "grafana"})
	s.Set(&Tunnel{Name: "portainer"})

	all := s.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d tunnels, want 2", len(all))
	}
}

func TestSaveThenLoad_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	s := &Store{Tunnels: make(map[string]*Tunnel)}
	want := &Tunnel{
		Name:      "grafana",
		TunnelID:  "tun-123",
		Port:      3000,
		Hostname:  "grafana.example.com",
		Protocol:  ProtocolAuto,
		Status:    StatusRunning,
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
	}
	s.Set(want)

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	got, ok := loaded.Get("grafana")
	if !ok {
		t.Fatal("LoadStore() after Save() is missing the tunnel")
	}
	if got.TunnelID != want.TunnelID || got.Port != want.Port || got.Hostname != want.Hostname {
		t.Errorf("loaded tunnel = %+v, want %+v", got, want)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s := &Store{Tunnels: make(map[string]*Tunnel)}
	s.Set(&Tunnel{Name: "grafana"})
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(home, stateFileName))
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("state file permissions = %o, want 0600", perm)
	}
}
