package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
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
// If any record with the same name already exists, it is deleted first.
func (c *Client) UpsertCNAME(zoneID, subdomain, tunnelID string) (string, error) {
	existing, err := c.FindDNSRecord(zoneID, subdomain)
	if err != nil {
		return "", fmt.Errorf("checking existing DNS records: %w", err)
	}
	if existing != nil {
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
