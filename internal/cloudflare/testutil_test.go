package cloudflare

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer spins up an httptest.Server backed by handler and returns a
// Client wired to it, plus the server for further inspection (e.g.
// asserting on captured requests). The server is closed automatically via
// t.Cleanup.
func testServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClientForTesting("test-token", "test-account", srv.URL)
	return c, srv
}

// jsonHandler returns an http.HandlerFunc that always responds with the
// given status code and raw JSON body, ignoring the request. Useful for
// single-call tests where the request itself isn't under test.
func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}
