package cloudflare

import (
	"net/http"
	"strings"
	"testing"
)

func TestVerifyToken_Active(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want Bearer test-token", got)
		}
		if r.URL.Path != "/user/tokens/verify" {
			t.Errorf("path = %q, want /user/tokens/verify", r.URL.Path)
		}
		jsonHandler(200, `{"result":{"id":"abc","status":"active"},"success":true,"errors":[]}`)(w, r)
	})

	if err := c.VerifyToken(); err != nil {
		t.Fatalf("VerifyToken() = %v, want nil", err)
	}
}

func TestVerifyToken_Unauthorized(t *testing.T) {
	c, _ := testServer(t, jsonHandler(401, `{"result":null,"success":false,"errors":[{"code":1000,"message":"Invalid API Token"}]}`))

	err := c.VerifyToken()
	if err == nil {
		t.Fatal("VerifyToken() = nil, want error for 401")
	}
	if !strings.Contains(err.Error(), "invalid or revoked") {
		t.Errorf("error = %q, want mention of invalid/revoked token", err.Error())
	}
}

func TestVerifyToken_InactiveStatus(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":{"id":"abc","status":"expired"},"success":true,"errors":[]}`))

	err := c.VerifyToken()
	if err == nil {
		t.Fatal("VerifyToken() = nil, want error for non-active status")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error = %q, want it to mention the actual status", err.Error())
	}
}

func TestVerifyToken_APIError(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":null,"success":false,"errors":[{"code":9109,"message":"Uncaught exception"}]}`))

	err := c.VerifyToken()
	if err == nil {
		t.Fatal("VerifyToken() = nil, want error when success=false")
	}
	if !strings.Contains(err.Error(), "9109") || !strings.Contains(err.Error(), "Uncaught exception") {
		t.Errorf("error = %q, want it to surface the API error code/message", err.Error())
	}
}

func TestVerifyZone_NotFound(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[],"success":true,"errors":[]}`))

	err := c.VerifyZone("example.com")
	if err == nil {
		t.Fatal("VerifyZone() = nil, want error when zone list is empty")
	}
	if !strings.Contains(err.Error(), "example.com") {
		t.Errorf("error = %q, want it to name the domain", err.Error())
	}
}

func TestGenerateSecret_UniqueAndDecodable(t *testing.T) {
	a := generateSecret()
	b := generateSecret()
	if a == b {
		t.Fatal("generateSecret() returned the same value twice in a row")
	}
	if len(a) == 0 {
		t.Fatal("generateSecret() returned empty string")
	}
}
