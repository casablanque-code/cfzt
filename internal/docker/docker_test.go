package docker

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// withDockerServer starts an httptest.Server, points the package-level
// httpClient's dialer at it instead of the real Docker unix socket for the
// duration of the test, and restores the original client afterward.
// docker.go always requests "http://localhost/...", so the exact address
// dialed here just needs to reach the test server - the request path is
// what the handler actually inspects.
func withDockerServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	original := httpClient
	t.Cleanup(func() { httpClient = original })

	addr := srv.Listener.Addr().String()
	httpClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(_, _ string) (net.Conn, error) {
				return net.Dial("tcp", addr)
			},
		},
	}
	return srv
}

func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestFindContainerPort_RunningWithPublishedPort(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
			return
		}
		jsonHandler(200, `{"State":{"Running":true},"NetworkSettings":{"Ports":{"9000/tcp":[{"HostPort":"9999"}]}}}`)(w, r)
	})

	port, err := FindContainerPort("portainer", "")
	if err != nil {
		t.Fatalf("FindContainerPort() error = %v", err)
	}
	if port != "9999" {
		t.Errorf("FindContainerPort() = %q, want 9999", port)
	}
}

func TestFindContainerPort_NotRunning(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
			return
		}
		jsonHandler(200, `{"State":{"Running":false},"NetworkSettings":{"Ports":{}}}`)(w, r)
	})

	_, err := FindContainerPort("stopped-container", "")
	if err == nil {
		t.Fatal("FindContainerPort() = nil error, want error for a non-running container")
	}
}

// When /containers/<name>/json 404s (name lookup style container ID vs
// name mismatch, or similar), FindContainerPort falls back to listing all
// containers and matching by name instead of failing outright.
func TestFindContainerPort_FallsBackToList(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/version"):
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/portainer/json"):
			jsonHandler(404, `{"message":"no such container"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/json"):
			jsonHandler(200, `[{"Names":["/portainer"],"State":"running","Ports":[{"PublicPort":9999,"Type":"tcp"}]}]`)(w, r)
		}
	})

	port, err := FindContainerPort("portainer", "")
	if err != nil {
		t.Fatalf("FindContainerPort() error = %v", err)
	}
	if port != "9999" {
		t.Errorf("FindContainerPort() = %q, want 9999 from list fallback", port)
	}
}

func TestFindContainerPort_ListFallback_NotFound(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/version"):
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/nope/json"):
			jsonHandler(404, `{"message":"no such container"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/json"):
			jsonHandler(200, `[]`)(w, r)
		}
	})

	_, err := FindContainerPort("nope", "")
	if err == nil {
		t.Fatal("FindContainerPort() = nil error, want error when container isn't found anywhere")
	}
}

func TestFindContainerPort_ListFallback_NoPublishedPorts(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/version"):
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/portainer/json"):
			jsonHandler(404, `{"message":"no such container"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/json"):
			jsonHandler(200, `[{"Names":["/portainer"],"State":"running","Ports":[]}]`)(w, r)
		}
	})

	_, err := FindContainerPort("portainer", "")
	if err == nil {
		t.Fatal("FindContainerPort() = nil error, want error for a running container with no published TCP ports")
	}
}

func TestFindContainerPort_DockerUnreachable(t *testing.T) {
	original := httpClient
	t.Cleanup(func() { httpClient = original })
	// Point at a closed listener so every request fails to connect,
	// simulating "docker not running".
	httpClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(_, _ string) (net.Conn, error) {
				return nil, &net.OpError{Op: "dial", Err: net.UnknownNetworkError("no docker socket")}
			},
		},
	}

	_, err := FindContainerPort("anything", "")
	if err == nil {
		t.Fatal("FindContainerPort() = nil error, want error when docker is unreachable")
	}
}

// Multiple published TCP ports must resolve to the same port every time
// regardless of Go's randomized map iteration order — a container with
// e.g. -p 8080:80 -p 8443:443 should always pick the lowest container
// port (80/tcp here), not whichever key the map happened to yield first.
func TestFindContainerPort_MultiplePorts_Deterministic(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
			return
		}
		jsonHandler(200, `{"State":{"Running":true},"NetworkSettings":{"Ports":{
			"443/tcp":[{"HostPort":"8443"}],
			"80/tcp":[{"HostPort":"8080"}],
			"9000/tcp":[{"HostPort":"9999"}]
		}}}`)(w, r)
	})

	for i := 0; i < 20; i++ {
		port, err := FindContainerPort("multi-port", "")
		if err != nil {
			t.Fatalf("FindContainerPort() error = %v", err)
		}
		if port != "8080" {
			t.Fatalf("FindContainerPort() = %q, want 8080 (lowest container port 80/tcp) on iteration %d", port, i)
		}
	}
}

// An explicit container port selects that binding exactly, overriding the
// lowest-port default — this is how a caller picks a non-default port on
// a container that publishes several.
func TestFindContainerPort_ExplicitContainerPort(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
			return
		}
		jsonHandler(200, `{"State":{"Running":true},"NetworkSettings":{"Ports":{
			"443/tcp":[{"HostPort":"8443"}],
			"80/tcp":[{"HostPort":"8080"}]
		}}}`)(w, r)
	})

	port, err := FindContainerPort("multi-port", "443")
	if err != nil {
		t.Fatalf("FindContainerPort() error = %v", err)
	}
	if port != "8443" {
		t.Fatalf("FindContainerPort() = %q, want 8443 for --container-port 443", port)
	}
}

// Requesting a container port the container doesn't actually publish is
// an error, not a silent fallback to some other port.
func TestFindContainerPort_ExplicitContainerPort_NotPublished(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
			return
		}
		jsonHandler(200, `{"State":{"Running":true},"NetworkSettings":{"Ports":{
			"80/tcp":[{"HostPort":"8080"}]
		}}}`)(w, r)
	})

	_, err := FindContainerPort("single-port", "443")
	if err == nil {
		t.Fatal("FindContainerPort() = nil error, want error when the requested container port isn't published")
	}
}

// The /containers/json list-fallback path (used when /containers/<n>/json
// 404s) must honor an explicit container port the same way the primary
// inspect path does.
func TestFindContainerPort_ExplicitContainerPort_ListFallback(t *testing.T) {
	withDockerServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/version"):
			jsonHandler(200, `{"ApiVersion":"1.44"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/portainer/json"):
			jsonHandler(404, `{"message":"no such container"}`)(w, r)
		case strings.Contains(r.URL.Path, "/containers/json"):
			jsonHandler(200, `[{"Names":["/portainer"],"State":"running","Ports":[
				{"PrivatePort":80,"PublicPort":8080,"Type":"tcp"},
				{"PrivatePort":443,"PublicPort":8443,"Type":"tcp"}
			]}]`)(w, r)
		}
	})

	port, err := FindContainerPort("portainer", "443")
	if err != nil {
		t.Fatalf("FindContainerPort() error = %v", err)
	}
	if port != "8443" {
		t.Fatalf("FindContainerPort() = %q, want 8443 for --container-port 443 via list fallback", port)
	}
}
