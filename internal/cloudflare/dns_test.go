package cloudflare

import (
	"net/http"
	"strings"
	"testing"
)

func TestGetZoneID_Found(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "name=example.com") {
			t.Errorf("query = %q, want name=example.com", r.URL.RawQuery)
		}
		jsonHandler(200, `{"result":[{"id":"zone-123"}],"success":true,"errors":[]}`)(w, r)
	})

	id, err := c.GetZoneID("example.com")
	if err != nil {
		t.Fatalf("GetZoneID() error = %v", err)
	}
	if id != "zone-123" {
		t.Errorf("GetZoneID() = %q, want zone-123", id)
	}
}

func TestGetZoneID_NotFound(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[],"success":true,"errors":[]}`))

	_, err := c.GetZoneID("nope.example.com")
	if err == nil {
		t.Fatal("GetZoneID() = nil error, want error for empty result")
	}
	if !strings.Contains(err.Error(), "nope.example.com") {
		t.Errorf("error = %q, want it to name the domain", err.Error())
	}
}

func TestFindDNSRecord_None(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[],"success":true,"errors":[]}`))

	rec, err := c.FindDNSRecord("zone-1", "app.example.com")
	if err != nil {
		t.Fatalf("FindDNSRecord() error = %v", err)
	}
	if rec != nil {
		t.Errorf("FindDNSRecord() = %+v, want nil for no match", rec)
	}
}

func TestFindDNSRecord_Found(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"result":[{"id":"rec-1","type":"CNAME","name":"app.example.com","content":"x.cfargotunnel.com","proxied":true,"ttl":1}],"success":true,"errors":[]}`))

	rec, err := c.FindDNSRecord("zone-1", "app.example.com")
	if err != nil {
		t.Fatalf("FindDNSRecord() error = %v", err)
	}
	if rec == nil || rec.ID != "rec-1" {
		t.Errorf("FindDNSRecord() = %+v, want record with ID rec-1", rec)
	}
}

// UpsertCNAME must delete a pre-existing record with the same name before
// creating the new one — this is the behavior the up.go rollback path
// depends on not leaving duplicate/stale records behind on retry.
func TestUpsertCNAME_DeletesExistingFirst(t *testing.T) {
	var calls []string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch r.Method {
		case "GET":
			jsonHandler(200, `{"result":[{"id":"old-rec","type":"CNAME","name":"app.example.com","content":"old.cfargotunnel.com"}],"success":true,"errors":[]}`)(w, r)
		case "DELETE":
			jsonHandler(200, `{"success":true,"errors":[]}`)(w, r)
		case "POST":
			jsonHandler(200, `{"result":{"id":"new-rec"},"success":true,"errors":[]}`)(w, r)
		}
	})

	id, err := c.UpsertCNAME("zone-1", "app.example.com", "tunnel-abc")
	if err != nil {
		t.Fatalf("UpsertCNAME() error = %v", err)
	}
	if id != "new-rec" {
		t.Errorf("UpsertCNAME() = %q, want new-rec", id)
	}

	wantOrder := []string{"GET /zones/zone-1/dns_records", "DELETE /zones/zone-1/dns_records/old-rec", "POST /zones/zone-1/dns_records"}
	if len(calls) != len(wantOrder) {
		t.Fatalf("calls = %v, want %v", calls, wantOrder)
	}
	for i, want := range wantOrder {
		if calls[i] != want {
			t.Errorf("call[%d] = %q, want %q", i, calls[i], want)
		}
	}
}

func TestUpsertCNAME_NoExisting(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			t.Error("UpsertCNAME() called DELETE when no existing record was found")
		}
		if r.Method == "GET" {
			jsonHandler(200, `{"result":[],"success":true,"errors":[]}`)(w, r)
			return
		}
		jsonHandler(200, `{"result":{"id":"new-rec"},"success":true,"errors":[]}`)(w, r)
	})

	if _, err := c.UpsertCNAME("zone-1", "app.example.com", "tunnel-abc"); err != nil {
		t.Fatalf("UpsertCNAME() error = %v", err)
	}
}

func TestDeleteDNSRecord_APIError(t *testing.T) {
	c, _ := testServer(t, jsonHandler(200, `{"success":false,"errors":[{"code":81044,"message":"Record does not exist"}]}`))

	err := c.DeleteDNSRecord("zone-1", "gone-rec")
	if err == nil {
		t.Fatal("DeleteDNSRecord() = nil, want error when API reports success=false")
	}
	if !strings.Contains(err.Error(), "81044") {
		t.Errorf("error = %q, want it to surface the API error code", err.Error())
	}
}
