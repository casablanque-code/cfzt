package main

import (
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestCreateTunnel_InvalidPort(t *testing.T) {
	err := createTunnel(tunnelOpts{name: "grafana", port: "not-a-number", protocol: "auto"})
	if err == nil {
		t.Fatal("createTunnel() = nil error, want an error for a non-numeric port")
	}
	if !strings.Contains(err.Error(), "not-a-number") {
		t.Errorf("createTunnel() error = %q, want it to name the bad port", err.Error())
	}
}

func TestCreateTunnel_InvalidProtocol(t *testing.T) {
	err := createTunnel(tunnelOpts{name: "grafana", port: "3000", protocol: "carrier-pigeon"})
	if err == nil {
		t.Fatal("createTunnel() = nil error, want an error for an unrecognized protocol")
	}
	if !strings.Contains(err.Error(), "carrier-pigeon") {
		t.Errorf("createTunnel() error = %q, want it to name the bad protocol", err.Error())
	}
}

func TestCreateTunnel_NoAccessPolicy(t *testing.T) {
	err := createTunnel(tunnelOpts{name: "grafana", port: "3000", protocol: "auto"})
	if err == nil {
		t.Fatal("createTunnel() = nil error, want an error when neither --allow nor --public is set")
	}
	if !strings.Contains(err.Error(), "--allow") || !strings.Contains(err.Error(), "--public") {
		t.Errorf("createTunnel() error = %q, want it to mention both --allow and --public", err.Error())
	}
}

func TestCreateTunnel_PublicAndAllowConflict(t *testing.T) {
	err := createTunnel(tunnelOpts{
		name: "grafana", port: "3000", protocol: "auto",
		public: true, emails: []string{"you@example.com"},
	})
	if err == nil {
		t.Fatal("createTunnel() = nil error, want an error when --public and --allow are combined")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("createTunnel() error = %q, want it to say --public/--allow are mutually exclusive", err.Error())
	}
}

// TestCreateTunnel_AlreadyExists covers the "tunnel already exists" guard,
// which returns before createTunnel ever constructs a Cloudflare API
// client — safe to run without any network access.
func TestCreateTunnel_AlreadyExists(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	if err := config.Save(&config.Config{APIToken: "tok", AccountID: "acct", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	store, err := state.LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	store.Set(&state.Tunnel{Name: "grafana", Port: 3000})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	err = createTunnel(tunnelOpts{name: "grafana", port: "3001", protocol: "auto", public: true})
	if err == nil {
		t.Fatal("createTunnel() = nil error, want an error for a tunnel that already exists")
	}
	if !strings.Contains(err.Error(), "zt down grafana") {
		t.Errorf("createTunnel() error = %q, want it to suggest `zt down grafana`", err.Error())
	}
}
