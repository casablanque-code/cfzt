package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/internal/manifest"
	"github.com/casablanque-code/cfzt/internal/state"
)

func writeManifest(t *testing.T, dir string, m *manifest.Manifest) string {
	t.Helper()
	path := filepath.Join(dir, "zt.yaml")
	if err := manifest.Save(path, m); err != nil {
		t.Fatalf("manifest.Save() error = %v", err)
	}
	return path
}

func TestRunApply_EmptyManifest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	path := writeManifest(t, dir, &manifest.Manifest{Services: map[string]manifest.ServiceSpec{}})

	if err := runApply(nil, []string{path}); err != nil {
		t.Fatalf("runApply() error = %v, want nil for an empty manifest", err)
	}
}

func TestRunApply_AllServicesAlreadyExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	store, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	store.Set(&state.Tunnel{Name: "grafana", Port: 3000})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := writeManifest(t, dir, &manifest.Manifest{Services: map[string]manifest.ServiceSpec{
		"grafana": {Port: 3000},
	}})

	if err := runApply(nil, []string{path}); err != nil {
		t.Fatalf("runApply() error = %v, want nil when everything already exists (nothing to create)", err)
	}
}

func TestRunApply_UntrackedLocalTunnelIsReportedNotTouched(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	store, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	store.Set(&state.Tunnel{Name: "leftover-service", Port: 1234})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	// Manifest doesn't mention leftover-service at all.
	path := writeManifest(t, dir, &manifest.Manifest{Services: map[string]manifest.ServiceSpec{}})

	if err := runApply(nil, []string{path}); err != nil {
		t.Fatalf("runApply() error = %v, want nil", err)
	}

	// runApply must never delete or modify state on its own - confirm the
	// untracked tunnel is still exactly as it was.
	reloaded, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reloaded.Get("leftover-service")
	if !ok || got.Port != 1234 {
		t.Errorf("runApply() modified or removed the untracked local tunnel: %+v, ok=%v", got, ok)
	}
}

// With a service that needs creating, runApply must check the cloudflared
// version before touching Cloudflare's API. In this sandbox there's no
// cloudflared binary at all, so this exercises that guard deterministically
// without needing network access or real credentials.
func TestRunApply_NewService_FailsCleanlyWithoutCloudflared(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Make sure a real cloudflared on the test runner's PATH can't leak in
	// and change this test's behavior.
	t.Setenv("PATH", t.TempDir())

	dir := t.TempDir()
	path := writeManifest(t, dir, &manifest.Manifest{Services: map[string]manifest.ServiceSpec{
		"grafana": {Port: 3000},
	}})

	err := runApply(nil, []string{path})
	if err == nil {
		t.Fatal("runApply() = nil error, want an error since cloudflared isn't on PATH")
	}
}

func TestRunApply_MissingManifestFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runApply(nil, []string{filepath.Join(t.TempDir(), "does-not-exist.yaml")})
	if err == nil {
		t.Fatal("runApply() = nil error, want error for a missing manifest file")
	}
}

func TestResolveApplyPort_ExplicitPort(t *testing.T) {
	port, err := resolveApplyPort("grafana", manifest.ServiceSpec{Port: 3000})
	if err != nil {
		t.Fatalf("resolveApplyPort() error = %v", err)
	}
	if port != "3000" {
		t.Errorf("resolveApplyPort() = %q, want 3000", port)
	}
}

func TestResolveApplyPort_DockerWithExplicitOverride(t *testing.T) {
	// Docker=true but Port is also set - the explicit port must win without
	// ever needing to actually query Docker.
	port, err := resolveApplyPort("portainer", manifest.ServiceSpec{Docker: true, Port: 9999})
	if err != nil {
		t.Fatalf("resolveApplyPort() error = %v", err)
	}
	if port != "9999" {
		t.Errorf("resolveApplyPort() = %q, want 9999 (explicit override)", port)
	}
}

func TestResolveApplyPort_DockerNoOverride_NoDockerAvailable(t *testing.T) {
	// No explicit port, Docker=true - this sandbox has no docker socket, so
	// FindContainerPort must fail and the error must be surfaced, not
	// swallowed or defaulted to an empty port.
	_, err := resolveApplyPort("portainer", manifest.ServiceSpec{Docker: true})
	if err == nil {
		t.Fatal("resolveApplyPort() = nil error, want error when docker detection fails")
	}
	if !strings.Contains(err.Error(), "docker") {
		t.Errorf("error = %q, want it to mention docker port detection", err.Error())
	}
}

func TestWriteManifestHelperProducesLoadableFile(t *testing.T) {
	// Sanity check on the test helper itself.
	dir := t.TempDir()
	path := writeManifest(t, dir, &manifest.Manifest{Services: map[string]manifest.ServiceSpec{
		"x": {Port: 1},
	}})
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("writeManifest() didn't create a file: %v", err)
	}
}
