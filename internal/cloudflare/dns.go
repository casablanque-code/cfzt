package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type DNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

type dnsListResponse struct {
	Result  []DNSRecord `json:"result"`
	Success bool        `json:"success"`
	Errors  []APIErr    `json:"errors"`
}

type dnsCreateResponse struct {
	Result  DNSRecord `json:"result"`
	Success bool      `json:"success"`
	Errors  []APIErr  `json:"errors"`
}

// GetZoneID resolves zone ID for a domain.
func (c *Client) GetZoneID(domain string) (string, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/zones?name=%s", domain),
		nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
		Success bool     `json:"success"`
		Errors  []APIErr `json:"errors"`
	}
	if err := decode(resp, &result); err != nil {
		return "", err
	}
	if !result.Success {
		return "", apiErr(result.Errors)
	}
	if len(result.Result) == 0 {
		return "", fmt.Errorf("zone not found for domain %q — make sure it's added to Cloudflare", domain)
	}
	return result.Result[0].ID, nil
}

// UpsertCNAME creates or replaces a CNAME pointing subdomain → tunnel.cfargotunnel.com.
// If a record with the same name already exists and looks like a record zt
// itself created (a CNAME to *.cfargotunnel.com), it's replaced automatically —
// that's the normal "re-run zt up on a stale tunnel" case. Any other existing
// record (a real A/AAAA/CNAME the caller didn't create with zt) is left alone
// unless force is true, since deleting it would be destructive and silent.
func (c *Client) UpsertCNAME(zoneID, subdomain, tunnelID string, force bool) (string, error) {
	existing, err := c.FindDNSRecord(zoneID, subdomain)
	if err != nil {
		return "", fmt.Errorf("checking existing DNS records: %w", err)
	}
	if existing != nil {
		if !force && !looksLikeZtRecord(existing) {
			return "", fmt.Errorf(
				"existing DNS record found for %s (type %s, content %q) that zt didn't create — refusing to replace it\nuse --force to replace it anyway",
				subdomain, existing.Type, existing.Content)
		}
		if err := c.DeleteDNSRecord(zoneID, existing.ID); err != nil {
			return "", fmt.Errorf("removing existing DNS record: %w", err)
		}
	}

	content := tunnelID + ".cfargotunnel.com"
	payload := map[string]any{
		"type":    "CNAME",
		"name":    subdomain,
		"content": content,
		"proxied": true,
		"ttl":     1,
	}
	body, _ := json.Marshal(payload)

	resp, err := c.do("POST",
		fmt.Sprintf("/zones/%s/dns_records", zoneID),
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var dr dnsCreateResponse
	if err := decode(resp, &dr); err != nil {
		return "", err
	}
	if !dr.Success {
		return "", apiErr(dr.Errors)
	}
	return dr.Result.ID, nil
}

// DeleteDNSRecord deletes a DNS record by ID.
func (c *Client) DeleteDNSRecord(zoneID, recordID string) error {
	resp, err := c.do("DELETE",
		fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID),
		nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Success bool     `json:"success"`
		Errors  []APIErr `json:"errors"`
	}
	if err := decode(resp, &result); err != nil {
		return err
	}
	if !result.Success {
		return apiErr(result.Errors)
	}
	return nil
}

// FindDNSRecord looks up a DNS record by name.
func (c *Client) FindDNSRecord(zoneID, name string) (*DNSRecord, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/zones/%s/dns_records?name=%s", zoneID, name),
		nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var lr dnsListResponse
	if err := decode(resp, &lr); err != nil {
		return nil, err
	}
	if !lr.Success {
		return nil, apiErr(lr.Errors)
	}
	if len(lr.Result) == 0 {
		return nil, nil
	}
	return &lr.Result[0], nil
}

// looksLikeZtRecord reports whether a DNS record looks like something zt
// itself created — a CNAME pointing at a Cloudflare Tunnel. These are safe
// to replace automatically (e.g. re-running `zt up` after a stale tunnel).
// Anything else (A, AAAA, or a CNAME to something that isn't a tunnel) is
// left alone unless the caller passes --force.
func looksLikeZtRecord(r *DNSRecord) bool {
	return r.Type == "CNAME" && strings.HasSuffix(r.Content, ".cfargotunnel.com")
}
