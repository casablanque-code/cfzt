package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/cloudflare"
	"github.com/casablanque-code/cfzt/internal/testutil"
)

// withStdin replaces os.Stdin with a pipe pre-loaded with input, restoring
// the original after the test. init.go reads credentials interactively
// via bufio.NewReader(os.Stdin) with no injection point of its own, so
// substituting the global is the only way to drive it without a source
// change.
func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })
	go func() {
		defer w.Close()
		_, _ = w.WriteString(input)
	}()
}

// withFakeCFClient points newCFClient (see cloudflare_client.go) at an
// httptest.Server for the duration of the test.
func withFakeCFClient(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	orig := newCFClient
	newCFClient = func(token, accountID string) *cloudflare.Client {
		return cloudflare.NewClientForTesting(token, accountID, srv.URL)
	}
	t.Cleanup(func() { newCFClient = orig })
	return srv
}

func TestRunInit_MissingFields(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	withStdin(t, "\n\n\n") // empty token, account ID, domain

	if err := runInit(nil, nil); err == nil {
		t.Fatal("runInit() = nil error, want an error when all fields are left blank")
	}
}

func TestRunInit_Success(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	withStdin(t, "tok-abc\nacct-123\nexample.com\n")

	withFakeCFClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/user/tokens/verify"):
			_, _ = w.Write([]byte(`{"result":{"id":"abc","status":"active"},"success":true,"errors":[]}`))
		case strings.HasPrefix(r.URL.Path, "/zones"):
			_, _ = w.Write([]byte(`{"result":[{"id":"zone-123"}],"success":true,"errors":[]}`))
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() after runInit() error = %v", err)
	}
	if cfg.APIToken != "tok-abc" || cfg.AccountID != "acct-123" || cfg.Domain != "example.com" {
		t.Errorf("saved config = %+v, want the values entered at the prompts", cfg)
	}
}

// TestRunInit_InvalidToken_StillSavesConfig covers runInit's deliberate
// choice to save the config and return nil even when token verification
// fails — the user gets a warning, not a hard failure, and can fix the
// token without re-entering everything.
func TestRunInit_InvalidToken_StillSavesConfig(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	withStdin(t, "bad-token\nacct-123\nexample.com\n")

	withFakeCFClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"result":null,"success":false,"errors":[{"code":1000,"message":"Invalid API Token"}]}`))
	})

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit() error = %v, want nil (config should still save on a bad token)", err)
	}

	if _, err := config.Load(); err != nil {
		t.Errorf("config.Load() after a failed token check = %v, want the config to have been saved anyway", err)
	}
}
