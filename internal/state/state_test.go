package state

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTunnelNewFieldsSerialization(t *testing.T) {
	original := &Tunnel{
		Name:         "grafana",
		TunnelID:     "abc-123",
		Port:         3000,
		Hostname:     "grafana.example.com",
		Protocol:     ProtocolQUIC,
		Status:       StatusRunning,
		Public:       false,
		AllowEmails:  []string{"alice@example.com", "bob@example.com"},
		DockerDetect: true,
		CreatedAt:    time.Now().Truncate(time.Second),
		UpdatedAt:    time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Tunnel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Public != original.Public {
		t.Errorf("Public: got %v, want %v", decoded.Public, original.Public)
	}
	if len(decoded.AllowEmails) != len(original.AllowEmails) {
		t.Fatalf("AllowEmails length: got %d, want %d", len(decoded.AllowEmails), len(original.AllowEmails))
	}
	for i, email := range original.AllowEmails {
		if decoded.AllowEmails[i] != email {
			t.Errorf("AllowEmails[%d]: got %q, want %q", i, decoded.AllowEmails[i], email)
		}
	}
	if decoded.DockerDetect != original.DockerDetect {
		t.Errorf("DockerDetect: got %v, want %v", decoded.DockerDetect, original.DockerDetect)
	}
}

// TestTunnelBackwardCompat verifies that a legacy state file (without the new
// fields) deserializes cleanly — new fields should default to zero values.
func TestTunnelBackwardCompat(t *testing.T) {
	legacy := `{
		"name": "vault",
		"tunnel_id": "xyz-456",
		"port": 8200,
		"hostname": "vault.example.com",
		"status": "running",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z"
	}`

	var t1 Tunnel
	if err := json.Unmarshal([]byte(legacy), &t1); err != nil {
		t.Fatalf("Unmarshal legacy state: %v", err)
	}

	// New fields must be zero/nil — not error out
	if t1.Public {
		t.Error("Public should default to false for legacy state")
	}
	if t1.AllowEmails != nil {
		t.Errorf("AllowEmails should be nil for legacy state, got %v", t1.AllowEmails)
	}
	if t1.DockerDetect {
		t.Error("DockerDetect should default to false for legacy state")
	}
	// Original fields intact
	if t1.Name != "vault" {
		t.Errorf("Name: got %q, want vault", t1.Name)
	}
	if t1.Port != 8200 {
		t.Errorf("Port: got %d, want 8200", t1.Port)
	}
}

func TestTunnelPublicOmitempty(t *testing.T) {
	// Public: false and AllowEmails: nil should be omitted from JSON
	tunnel := &Tunnel{
		Name:     "api",
		TunnelID: "t-1",
		Port:     8080,
		Status:   StatusRunning,
	}

	data, _ := json.Marshal(tunnel)
	raw := string(data)

	if contains(raw, `"public"`) {
		t.Error(`"public" should be omitted when false (omitempty)`)
	}
	if contains(raw, `"allow_emails"`) {
		t.Error(`"allow_emails" should be omitted when nil (omitempty)`)
	}
	if contains(raw, `"docker_detect"`) {
		t.Error(`"docker_detect" should be omitted when false (omitempty)`)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
