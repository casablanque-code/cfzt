package cloudflare

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestFindAccessAppByDomain_Found(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[{"id":"app-1","name":"grafana","domain":"app.example.com"}],"success":true,"errors":[]}`))

	app, err := c.FindAccessAppByDomain("app.example.com")
	if err != nil {
		t.Fatalf("FindAccessAppByDomain() error = %v", err)
	}
	if app == nil || app.ID != "app-1" {
		t.Errorf("FindAccessAppByDomain() = %+v, want app-1", app)
	}
}

func TestFindAccessAppByDomain_NotFound(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[{"id":"app-1","name":"other","domain":"other.example.com"}],"success":true,"errors":[]}`))

	app, err := c.FindAccessAppByDomain("app.example.com")
	if err != nil {
		t.Fatalf("FindAccessAppByDomain() error = %v", err)
	}
	if app != nil {
		t.Errorf("FindAccessAppByDomain() = %+v, want nil", app)
	}
}

// Same contract as UpsertCNAME: an existing app for the same hostname must
// be deleted before creating the new one, so retries after a partial
// failure don't pile up duplicate Access apps.
func TestUpsertAccessApp_DeletesExistingFirst(t *testing.T) {
	var calls []string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		switch r.Method {
		case "GET":
			jsonHandler(200, `{"result":[{"id":"old-app","domain":"app.example.com"}],"success":true,"errors":[]}`)(w, r)
		case "DELETE":
			jsonHandler(200, `{"success":true,"errors":[]}`)(w, r)
		case "POST":
			jsonHandler(200, `{"result":{"id":"new-app"},"success":true,"errors":[]}`)(w, r)
		}
	})

	id, err := c.UpsertAccessApp("app.example.com", "grafana")
	if err != nil {
		t.Fatalf("UpsertAccessApp() error = %v", err)
	}
	if id != "new-app" {
		t.Errorf("UpsertAccessApp() = %q, want new-app", id)
	}
	wantOrder := []string{"GET", "DELETE", "POST"}
	if len(calls) != 3 {
		t.Fatalf("calls = %v, want %v", calls, wantOrder)
	}
	for i, want := range wantOrder {
		if calls[i] != want {
			t.Errorf("call[%d] = %q, want %q", i, calls[i], want)
		}
	}
}

func TestCreateBypassPolicy_NoEmails_IsPublicBypass(t *testing.T) {
	var payload map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&payload)
		jsonHandler(200, `{"success":true,"errors":[]}`)(w, r)
	})

	if err := c.CreateBypassPolicy("app-1", nil); err != nil {
		t.Fatalf("CreateBypassPolicy() error = %v", err)
	}
	if payload["decision"] != "bypass" {
		t.Errorf("decision = %v, want bypass when no emails given", payload["decision"])
	}
	include, ok := payload["include"].([]any)
	if !ok || len(include) != 1 {
		t.Fatalf("include = %v, want one 'everyone' rule", payload["include"])
	}
	rule := include[0].(map[string]any)
	if _, ok := rule["everyone"]; !ok {
		t.Errorf("include[0] = %v, want an 'everyone' rule", rule)
	}
}

// This is the security-relevant path: a caller passing specific emails
// must get "allow" + one include entry per email, never silently fall
// back to public bypass.
func TestCreateBypassPolicy_WithEmails_IsAllowList(t *testing.T) {
	var payload map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&payload)
		jsonHandler(200, `{"success":true,"errors":[]}`)(w, r)
	})

	emails := []string{"you@example.com", "them@example.com"}
	if err := c.CreateBypassPolicy("app-1", emails); err != nil {
		t.Fatalf("CreateBypassPolicy() error = %v", err)
	}
	if payload["decision"] != "allow" {
		t.Errorf("decision = %v, want allow when emails are given", payload["decision"])
	}
	include, ok := payload["include"].([]any)
	if !ok || len(include) != len(emails) {
		t.Fatalf("include = %v, want %d entries", payload["include"], len(emails))
	}
	for i, e := range emails {
		rule := include[i].(map[string]any)
		emailRule, ok := rule["email"].(map[string]any)
		if !ok || emailRule["email"] != e {
			t.Errorf("include[%d] = %v, want email=%q", i, rule, e)
		}
	}
}

func TestDeleteAccessApp_APIError(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"success":false,"errors":[{"code":12345,"message":"app not found"}]}`))

	err := c.DeleteAccessApp("gone")
	if err == nil {
		t.Fatal("DeleteAccessApp() = nil, want error")
	}
	if !strings.Contains(err.Error(), "12345") {
		t.Errorf("error = %q, want it to surface the API error code", err.Error())
	}
}
