//go:build linux

package service

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnitName(t *testing.T) {
	if got := unitName("grafana"); got != "zt-grafana.service" {
		t.Errorf("unitName(grafana) = %q, want zt-grafana.service", got)
	}
}

func TestUnitPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := unitPath("grafana")
	if err != nil {
		t.Fatalf("unitPath() error = %v", err)
	}
	want := filepath.Join(home, ".config", "systemd", "user", "zt-grafana.service")
	if path != want {
		t.Errorf("unitPath() = %q, want %q", path, want)
	}

	// unitPath must create the directory even though it doesn't write the
	// unit file itself - Install() relies on the dir existing before it
	// calls os.WriteFile.
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Errorf("unitPath() did not create %q: %v", filepath.Dir(path), err)
	}
}

func TestWatchdogUnitPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := watchdogUnitPath()
	if err != nil {
		t.Fatalf("watchdogUnitPath() error = %v", err)
	}
	want := filepath.Join(home, ".config", "systemd", "user", "zt-watchdog.service")
	if path != want {
		t.Errorf("watchdogUnitPath() = %q, want %q", path, want)
	}
}

func TestUnitPath_PublicWrapper(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := UnitPath("grafana")
	want := filepath.Join(home, ".config", "systemd", "user", "zt-grafana.service")
	if got != want {
		t.Errorf("UnitPath() = %q, want %q", got, want)
	}
}

func TestIsInstalled_NoUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if IsInstalled("zt-test-tunnel-does-not-exist") {
		t.Error("IsInstalled() = true, want false when no unit file exists")
	}
}

func TestIsInstalled_UnitFilePresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := unitPath("grafana")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[Unit]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if !IsInstalled("grafana") {
		t.Error("IsInstalled() = false, want true when the unit file exists")
	}
}

func TestWatchdogIsInstalled_NoUnitFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if WatchdogIsInstalled() {
		t.Error("WatchdogIsInstalled() = true, want false when no unit file exists")
	}
}

// fakeExecutable creates a file at dir/name that exec.LookPath will find:
// present and marked executable. Content doesn't matter since we never
// actually run it in these tests.
func fakeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCloudflaredBin_Found(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("PATH lookup semantics assumed here are linux-specific")
	}
	dir := t.TempDir()
	fakeExecutable(t, dir, "cloudflared")
	t.Setenv("PATH", dir)

	path, err := cloudflaredBin()
	if err != nil {
		t.Fatalf("cloudflaredBin() error = %v", err)
	}
	want := filepath.Join(dir, "cloudflared")
	if path != want {
		t.Errorf("cloudflaredBin() = %q, want %q", path, want)
	}
}

func TestCloudflaredBin_NotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := cloudflaredBin(); err == nil {
		t.Fatal("cloudflaredBin() = nil error, want error when cloudflared isn't on PATH")
	}
}

// Install() must fail before ever touching systemctl if cloudflared isn't
// on PATH - this is safe to run without a real systemd/dbus present.
func TestInstall_FailsCleanlyWithoutCloudflared(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	err := Install("grafana", "/tmp/config.yml", "/tmp/cloudflared.log")
	if err == nil {
		t.Fatal("Install() = nil error, want error when cloudflared isn't on PATH")
	}

	// Must not have written a unit file for a service it never actually
	// configured.
	if IsInstalled("grafana") {
		t.Error("Install() left behind a unit file despite failing before writing one")
	}
}
