package docker

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"context"
)


func withDockerServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	original := httpClient
	t.Cleanup(func() { httpClient = original })

	addr := srv.Listener.Addr().String()
	httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
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

	httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(context.Context, string, string) (net.Conn, error) {
				return nil, &net.OpError{
					Op:  "dial",
					Err: net.UnknownNetworkError("no docker socket"),
				}
			},
		},
	}

	_, err := FindContainerPort("anything", "")
	if err == nil {
		t.Fatal("FindContainerPort() = nil error, want error when docker is unreachable")
	}
}

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
