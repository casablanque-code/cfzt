package release

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestCheckLatestFrom_DevBuild_NeverReportsUpdate(t *testing.T) {
	// No server needed at all — a dev build must short-circuit before
	// ever making a network call.
	info, err := checkLatestFrom("http://should-not-be-hit.invalid", "dev")
	if err != nil {
		t.Fatalf("checkLatestFrom() error = %v", err)
	}
	if info.Available {
		t.Error("checkLatestFrom() reported an update available for a dev build")
	}
}

func TestCheckLatestFrom_NewerReleaseAvailable(t *testing.T) {
	srv := testServer(t, jsonHandler(200, `{"tag_name":"v0.8.0","html_url":"https://github.com/casablanque-code/cfzt/releases/tag/v0.8.0"}`))

	info, err := checkLatestFrom(srv.URL, "v0.7.2")
	if err != nil {
		t.Fatalf("checkLatestFrom() error = %v", err)
	}
	if !info.Available {
		t.Error("checkLatestFrom() = Available false, want true (v0.8.0 > v0.7.2)")
	}
	if info.Latest != "v0.8.0" {
		t.Errorf("Latest = %q, want v0.8.0", info.Latest)
	}
}

func TestCheckLatestFrom_AlreadyLatest(t *testing.T) {
	srv := testServer(t, jsonHandler(200, `{"tag_name":"v0.7.2","html_url":"https://example.com"}`))

	info, err := checkLatestFrom(srv.URL, "v0.7.2")
	if err != nil {
		t.Fatalf("checkLatestFrom() error = %v", err)
	}
	if info.Available {
		t.Error("checkLatestFrom() = Available true, want false when already on the latest version")
	}
}

func TestCheckLatestFrom_RunningNewerThanLatest(t *testing.T) {
	// e.g. a pre-release build running ahead of the last published tag —
	// must not claim an "update" is available.
	srv := testServer(t, jsonHandler(200, `{"tag_name":"v0.7.0","html_url":"https://example.com"}`))

	info, err := checkLatestFrom(srv.URL, "v0.8.0")
	if err != nil {
		t.Fatalf("checkLatestFrom() error = %v", err)
	}
	if info.Available {
		t.Error("checkLatestFrom() = Available true, want false when current is already newer than the latest tag")
	}
}

func TestCheckLatestFrom_ServerError(t *testing.T) {
	srv := testServer(t, jsonHandler(500, `{}`))

	_, err := checkLatestFrom(srv.URL, "v0.7.2")
	if err == nil {
		t.Fatal("checkLatestFrom() = nil error, want error on a non-200 response")
	}
}

func TestCheckLatestFrom_UnreachableServer(t *testing.T) {
	srv := testServer(t, jsonHandler(200, `{}`))
	badURL := srv.URL
	srv.Close() // close immediately so the connection is refused

	_, err := checkLatestFrom(badURL, "v0.7.2")
	if err == nil {
		t.Fatal("checkLatestFrom() = nil error, want error when the server is unreachable")
	}
}

func TestCheckLatestFrom_UnrecognizedVersionFormat(t *testing.T) {
	// A malformed tag (or a "current" build string that isn't semver)
	// must degrade to "no update reported", not an error — see
	// checkLatestFrom's comment on isNewer.
	srv := testServer(t, jsonHandler(200, `{"tag_name":"not-a-version","html_url":"https://example.com"}`))

	info, err := checkLatestFrom(srv.URL, "v0.7.2")
	if err != nil {
		t.Fatalf("checkLatestFrom() error = %v", err)
	}
	if info.Available {
		t.Error("checkLatestFrom() = Available true, want false for an unparseable release tag")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.8.0", "v0.7.2", true},
		{"v0.7.2", "v0.8.0", false},
		{"v0.7.2", "v0.7.2", false},
		{"v1.0.0", "v0.99.99", true},
		{"0.8.0", "v0.7.2", true}, // no leading "v" on either side is fine
		{"v0.7.10", "v0.7.9", true},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_vs_%s", c.a, c.b), func(t *testing.T) {
			got, err := isNewer(c.a, c.b)
			if err != nil {
				t.Fatalf("isNewer(%q, %q) error = %v", c.a, c.b, err)
			}
			if got != c.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
