package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestRunDown_NoConfig(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	err := runDown(nil, []string{"whatever"})
	if err == nil {
		t.Fatal("runDown() = nil error, want an error when there's no config to load")
	}
}

// TestRunDown_FullTeardown drives runDown's entire flow — DNS record,
// Access app, and tunnel deletion, plus local state cleanup — against a
// fake server standing in for every Cloudflare endpoint it calls.
func TestRunDown_FullTeardown(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	if err := config.Save(&config.Config{APIToken: "tok", AccountID: "acct-1", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}

	store, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	store.Set(&state.Tunnel{
		Name:     "grafana",
		Port:     3000,
		TunnelID: "tunnel-abc",
		Hostname: "grafana.example.com",
	})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	var deletedDNS, deletedAccess, deletedTunnel bool

	withFakeCFClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/zones":
			_, _ = w.Write([]byte(`{"result":[{"id":"zone-1"}],"success":true,"errors":[]}`))
		case r.Method == "GET" && r.URL.Path == "/zones/zone-1/dns_records":
			_, _ = w.Write([]byte(`{"result":[{"id":"rec-1","name":"grafana.example.com"}],"success":true,"errors":[]}`))
		case r.Method == "DELETE" && r.URL.Path == "/zones/zone-1/dns_records/rec-1":
			deletedDNS = true
			_, _ = w.Write([]byte(`{"success":true,"errors":[]}`))
		case r.Method == "GET" && r.URL.Path == "/accounts/acct-1/access/apps":
			_, _ = w.Write([]byte(`{"result":[{"id":"app-1","domain":"grafana.example.com"}],"success":true,"errors":[]}`))
		case r.Method == "DELETE" && r.URL.Path == "/accounts/acct-1/access/apps/app-1":
			deletedAccess = true
			_, _ = w.Write([]byte(`{"success":true,"errors":[]}`))
		case r.Method == "DELETE" && r.URL.Path == "/accounts/acct-1/cfd_tunnel/tunnel-abc":
			deletedTunnel = true
			_, _ = w.Write([]byte(`{"success":true,"errors":[]}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	})

	if err := runDown(nil, []string{"grafana"}); err != nil {
		t.Fatalf("runDown() error = %v", err)
	}

	if !deletedDNS || !deletedAccess || !deletedTunnel {
		t.Errorf("runDown() didn't delete everything: DNS=%v Access=%v Tunnel=%v", deletedDNS, deletedAccess, deletedTunnel)
	}

	store, err = state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := store.Get("grafana"); exists {
		t.Error("runDown() should have removed the tunnel from local state")
	}
}
// which returns before runDown ever constructs a Cloudflare API client —
// safe to run without any network access or mocked API responses.
func TestRunDown_TunnelNotFound(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	if err := config.Save(&config.Config{APIToken: "tok", AccountID: "acct", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}

	err := runDown(nil, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("runDown() = nil error, want an error for a tunnel not in local state")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("runDown() error = %q, want it to name the tunnel", err.Error())
	}
}
