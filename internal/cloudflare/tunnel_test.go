package cloudflare

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCreateTunnel_Success(t *testing.T) {
	var gotBody map[string]string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/accounts/test-account/cfd_tunnel") {
			t.Errorf("path = %q, want .../accounts/test-account/cfd_tunnel", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		jsonHandler(200, `{"result":{"id":"tun-1","name":"grafana","status":"inactive"},"success":true,"errors":[]}`)(w, r)
	})

	id, credJSON, err := c.CreateTunnel("grafana")
	if err != nil {
		t.Fatalf("CreateTunnel() error = %v", err)
	}
	if id != "tun-1" {
		t.Errorf("CreateTunnel() id = %q, want tun-1", id)
	}
	if gotBody["name"] != "grafana" {
		t.Errorf("request body name = %q, want grafana", gotBody["name"])
	}
	if gotBody["tunnel_secret"] == "" {
		t.Error("request body missing tunnel_secret")
	}

	var creds map[string]string
	if err := json.Unmarshal(credJSON, &creds); err != nil {
		t.Fatalf("credJSON is not valid JSON: %v", err)
	}
	if creds["TunnelID"] != "tun-1" || creds["AccountTag"] != "test-account" {
		t.Errorf("creds = %+v, want TunnelID=tun-1 AccountTag=test-account", creds)
	}
}

func TestCreateTunnel_APIError(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":{},"success":false,"errors":[{"code":1003,"message":"tunnel name already exists"}]}`))

	_, _, err := c.CreateTunnel("grafana")
	if err == nil {
		t.Fatal("CreateTunnel() = nil, want error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want it to surface the API message", err.Error())
	}
}

func TestConfigureTunnel_Success(t *testing.T) {
	var payload TunnelConfigPayload
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		jsonHandler(200, `{"success":true,"errors":[]}`)(w, r)
	})

	if err := c.ConfigureTunnel("tun-1", "app.example.com", "8080"); err != nil {
		t.Fatalf("ConfigureTunnel() error = %v", err)
	}
	if len(payload.Config.Ingress) != 2 {
		t.Fatalf("ingress rules = %d, want 2 (hostname rule + catch-all)", len(payload.Config.Ingress))
	}
	if payload.Config.Ingress[0].Hostname != "app.example.com" || payload.Config.Ingress[0].Service != "http://localhost:8080" {
		t.Errorf("ingress[0] = %+v, want hostname=app.example.com service=http://localhost:8080", payload.Config.Ingress[0])
	}
	if payload.Config.Ingress[1].Hostname != "" || payload.Config.Ingress[1].Service != "http_status:404" {
		t.Errorf("ingress[1] (catch-all) = %+v, want empty hostname + http_status:404", payload.Config.Ingress[1])
	}
}

func TestDeleteTunnel_APIError(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"success":false,"errors":[{"code":1010,"message":"tunnel not found"}]}`))

	if err := c.DeleteTunnel("tun-missing"); err == nil {
		t.Fatal("DeleteTunnel() = nil, want error")
	}
}

func TestFindTunnelByName_Match(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[{"id":"tun-1","name":"grafana","status":"active"},{"id":"tun-2","name":"other","status":"active"}],"success":true,"errors":[]}`))

	id, err := c.FindTunnelByName("grafana")
	if err != nil {
		t.Fatalf("FindTunnelByName() error = %v", err)
	}
	if id != "tun-1" {
		t.Errorf("FindTunnelByName() = %q, want tun-1", id)
	}
}

func TestFindTunnelByName_NoMatch(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[{"id":"tun-2","name":"other","status":"active"}],"success":true,"errors":[]}`))

	id, err := c.FindTunnelByName("grafana")
	if err != nil {
		t.Fatalf("FindTunnelByName() error = %v", err)
	}
	if id != "" {
		t.Errorf("FindTunnelByName() = %q, want empty string for no match", id)
	}
}

func TestGetTunnelStatus(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":{"status":"degraded"},"success":true,"errors":[]}`))

	status, err := c.GetTunnelStatus("tun-1")
	if err != nil {
		t.Fatalf("GetTunnelStatus() error = %v", err)
	}
	if status != "degraded" {
		t.Errorf("GetTunnelStatus() = %q, want degraded", status)
	}
}

func TestListTunnels(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[{"id":"tun-1","name":"grafana","status":"active"}],"success":true,"errors":[]}`))

	out, err := c.ListTunnels()
	if err != nil {
		t.Fatalf("ListTunnels() error = %v", err)
	}
	if len(out) != 1 || out[0].ID != "tun-1" || out[0].Name != "grafana" {
		t.Errorf("ListTunnels() = %+v, want one entry tun-1/grafana", out)
	}
}
