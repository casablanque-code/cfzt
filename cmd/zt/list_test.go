package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
)

func init() {
	// Deterministic string assertions regardless of whether tests run in a
	// TTY - color adds ANSI escapes that would otherwise leak into the
	// expected-value comparisons below.
	color.NoColor = true
}

// A tunnel with no PID and (in this sandbox) definitely no matching
// systemd unit falls all the way through to the "none"/"stopped" case.
func TestTunnelStatus_NoServiceNoPID(t *testing.T) {
	tun := &state.Tunnel{Name: "zt-test-tunnel-does-not-exist", PID: 0}
	status, managedBy := tunnelStatus(tun)
	if managedBy != "none" {
		t.Errorf("managedBy = %q, want none", managedBy)
	}
	if status != "stopped" {
		t.Errorf("status = %q, want stopped", status)
	}
}

// A PID that doesn't correspond to a real running process should report
// stopped, not panic or hang.
func TestTunnelStatus_StalePID(t *testing.T) {
	tun := &state.Tunnel{Name: "zt-test-tunnel-stale-pid", PID: 999999}
	status, managedBy := tunnelStatus(tun)
	if managedBy != "pid 999999" {
		t.Errorf("managedBy = %q, want %q", managedBy, "pid 999999")
	}
	if status != "stopped" {
		t.Errorf("status = %q, want stopped for a PID that isn't a real running process", status)
	}
}

func TestProtocolLabel_Pinned(t *testing.T) {
	cases := []struct {
		proto state.Protocol
		want  string
	}{
		{state.ProtocolHTTP2, "http2 (TCP)"},
		{state.ProtocolQUIC, "quic (UDP)"},
	}
	for _, c := range cases {
		got := protocolLabel(c.proto, "/does/not/matter")
		if got != c.want {
			t.Errorf("protocolLabel(%q, ...) = %q, want %q", c.proto, got, c.want)
		}
	}
}

func TestProtocolLabel_AutoNoLog(t *testing.T) {
	got := protocolLabel(state.ProtocolAuto, filepath.Join(t.TempDir(), "does-not-exist.log"))
	if got != "auto" {
		t.Errorf("protocolLabel(auto, missing log) = %q, want auto", got)
	}
}

func TestProtocolLabel_AutoWithLiveSignal(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cloudflared.log")
	content := "2026-07-13T20:24:17Z INF Registered tunnel connection connIndex=1 connection=aaa event=0 protocol=http2\n"
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got := protocolLabel(state.ProtocolAuto, logPath)
	if got != "auto (http2)" {
		t.Errorf("protocolLabel(auto, live http2 log) = %q, want auto (http2)", got)
	}
}

func TestProtocolLabel_DefaultCase(t *testing.T) {
	// Empty Protocol (unset/zero value) should behave like ProtocolAuto,
	// not panic or fall through unexpectedly.
	got := protocolLabel(state.Protocol(""), filepath.Join(t.TempDir(), "missing.log"))
	if !strings.HasPrefix(got, "auto") {
		t.Errorf("protocolLabel(\"\", ...) = %q, want it to start with auto", got)
	}
}
