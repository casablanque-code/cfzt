package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestWatchdogLogPath(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	got, err := watchdogLogPath()
	if err != nil {
		t.Fatalf("watchdogLogPath() error = %v", err)
	}
	want := filepath.Join(home, ".zt", "watchdog.log")
	if got != want {
		t.Errorf("watchdogLogPath() = %q, want %q", got, want)
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// TestRunWatchdogDisable_NotInstalled relies on a fresh test/CI
// environment genuinely having no zt-watchdog systemd unit / scheduled
// task registered — service.WatchdogIsInstalled() has no injection point,
// so this is the only branch reachable without actually installing (or
// faking) a real background service.
func TestRunWatchdogDisable_NotInstalled(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	out := captureStdout(t, func() {
		if err := runWatchdogDisable(nil, nil); err != nil {
			t.Fatalf("runWatchdogDisable() error = %v", err)
		}
	})
	if !strings.Contains(out, "not installed") {
		t.Errorf("runWatchdogDisable() output = %q, want it to mention the watchdog isn't installed", out)
	}
}

func TestRunWatchdogStatus_NotInstalled(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	out := captureStdout(t, func() {
		if err := runWatchdogStatus(nil, nil); err != nil {
			t.Fatalf("runWatchdogStatus() error = %v", err)
		}
	})
	if !strings.Contains(out, "not installed") {
		t.Errorf("runWatchdogStatus() output = %q, want it to mention the watchdog isn't installed", out)
	}
}
