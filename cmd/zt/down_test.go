package main

import (
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestRunDown_NoConfig(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	err := runDown(nil, []string{"whatever"})
	if err == nil {
		t.Fatal("runDown() = nil error, want an error when there's no config to load")
	}
}

// TestRunDown_TunnelNotFound covers the "tunnel not in local state" path,
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
