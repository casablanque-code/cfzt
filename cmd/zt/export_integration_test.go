package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/casablanque-code/cfzt/internal/manifest"
	"github.com/casablanque-code/cfzt/internal/state"
)

func TestRunExport_NoTunnels(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	flagExportOut = filepath.Join(dir, "zt.yaml")
	t.Cleanup(func() { flagExportOut = "zt.yaml" })

	if err := runExport(nil, nil); err != nil {
		t.Fatalf("runExport() error = %v", err)
	}
	if _, err := os.Stat(flagExportOut); err == nil {
		t.Error("runExport() wrote a manifest file with zero tunnels, want no file written")
	}
}

func TestRunExport_WritesManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	store, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	store.Set(&state.Tunnel{Name: "grafana", Port: 3000, Protocol: state.ProtocolAuto})
	store.Set(&state.Tunnel{Name: "portainer", DockerDetect: true, Protocol: state.ProtocolQUIC, AllowEmails: []string{"you@example.com"}})
	store.Set(&state.Tunnel{Name: "api", Port: 8080, Public: true})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	flagExportOut = filepath.Join(outDir, "zt.yaml")
	t.Cleanup(func() { flagExportOut = "zt.yaml" })

	if err := runExport(nil, nil); err != nil {
		t.Fatalf("runExport() error = %v", err)
	}

	m, err := manifest.Load(flagExportOut)
	if err != nil {
		t.Fatalf("manifest.Load() of runExport's output: %v", err)
	}
	if len(m.Services) != 3 {
		t.Fatalf("exported manifest has %d services, want 3", len(m.Services))
	}

	grafana, ok := m.Services["grafana"]
	if !ok || grafana.Port != 3000 || grafana.Protocol != "" {
		// "auto" must be omitted from the exported manifest - see protocolForExport.
		t.Errorf("grafana = %+v, want Port=3000 Protocol=\"\" (auto omitted)", grafana)
	}

	portainer, ok := m.Services["portainer"]
	if !ok || !portainer.Docker || portainer.Protocol != "quic" || len(portainer.Allow) != 1 || portainer.Allow[0] != "you@example.com" {
		t.Errorf("portainer = %+v, want Docker=true Protocol=quic Allow=[you@example.com]", portainer)
	}

	api, ok := m.Services["api"]
	if !ok || !api.Public || api.Port != 8080 {
		t.Errorf("api = %+v, want Public=true Port=8080", api)
	}
}
