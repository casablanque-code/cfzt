package main

import (
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestRestartTunnel_NotFound(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	err := restartTunnel("does-not-exist")
	if err == nil {
		t.Fatal("restartTunnel() = nil error, want an error for an unknown tunnel")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("restartTunnel() error = %q, want it to name the tunnel", err.Error())
	}
}

// TestRestartTunnel_NoProcessOrService covers a tunnel that's tracked in
// state but has neither an installed service nor a running PID — the
// "zt down && zt up" recovery-hint path. This is safe to run in any CI
// environment: service.IsInstalled() genuinely returns false for a task/
// unit name that was never created, no real process management happens.
func TestRestartTunnel_NoProcessOrService(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	store, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	store.Set(&state.Tunnel{Name: "orphaned", Port: 3000, PID: 0})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	err = restartTunnel("orphaned")
	if err == nil {
		t.Fatal("restartTunnel() = nil error, want an error when there's no service or PID to restart")
	}
	if !strings.Contains(err.Error(), "zt down orphaned") || !strings.Contains(err.Error(), "zt up orphaned") {
		t.Errorf("restartTunnel() error = %q, want it to suggest 'zt down orphaned && zt up orphaned'", err.Error())
	}
}
