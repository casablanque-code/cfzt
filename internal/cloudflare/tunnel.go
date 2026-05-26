package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TunnelSecret struct {
	TunnelSecret string `json:"tunnel_secret"`
}

type TunnelResponse struct {
	Result struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"result"`
	Success bool     `json:"success"`
	Errors  []APIErr `json:"errors"`
}

type TunnelListResponse struct {
	Result []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"result"`
	Success bool     `json:"success"`
	Errors  []APIErr `json:"errors"`
}

type TunnelConfigPayload struct {
	Config TunnelIngressConfig `json:"config"`
}

type TunnelIngressConfig struct {
	Ingress []IngressRule `json:"ingress"`
}

type IngressRule struct {
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service"`
}

// CreateTunnel creates a named tunnel and returns its ID and credentials JSON.
func (c *Client) CreateTunnel(name string) (tunnelID string, credJSON []byte, err error) {
	secret := generateSecret()
	body, _ := json.Marshal(map[string]string{
		"name":          name,
		"tunnel_secret": secret,
	})

	resp, err := c.do("POST",
		fmt.Sprintf("/accounts/%s/cfd_tunnel", c.AccountID),
		bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var tr TunnelResponse
	if err := decode(resp, &tr); err != nil {
		return "", nil, err
	}
	if !tr.Success {
		return "", nil, apiErr(tr.Errors)
	}

	creds := map[string]string{
		"AccountTag":   c.AccountID,
		"TunnelSecret": secret,
		"TunnelID":     tr.Result.ID,
	}
	credJSON, _ = json.Marshal(creds)
	return tr.Result.ID, credJSON, nil
}

// ConfigureTunnel sets ingress rules for a tunnel.
func (c *Client) ConfigureTunnel(tunnelID, hostname, localPort string) error {
	payload := TunnelConfigPayload{
		Config: TunnelIngressConfig{
			Ingress: []IngressRule{
				{Hostname: hostname, Service: "http://localhost:" + localPort},
				{Service: "http_status:404"},
			},
		},
	}
	body, _ := json.Marshal(payload)

	resp, err := c.do("PUT",
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", c.AccountID, tunnelID),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

// DeleteTunnel deletes a tunnel by ID.
func (c *Client) DeleteTunnel(tunnelID string) error {
	resp, err := c.do("DELETE",
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", c.AccountID, tunnelID),
		nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

// ListTunnels returns all tunnels for the account.
func (c *Client) ListTunnels() ([]struct{ ID, Name, Status string }, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/accounts/%s/cfd_tunnel?status=active", c.AccountID),
		nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tr TunnelListResponse
	if err := decode(resp, &tr); err != nil {
		return nil, err
	}
	if !tr.Success {
		return nil, apiErr(tr.Errors)
	}

	out := make([]struct{ ID, Name, Status string }, len(tr.Result))
	for i, r := range tr.Result {
		out[i] = struct{ ID, Name, Status string }{r.ID, r.Name, r.Status}
	}
	return out, nil
}

func decode(resp *http.Response, v any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// FindTunnelByName finds an existing tunnel by name and returns its ID.
// Returns empty string (no error) if not found.
func (c *Client) FindTunnelByName(name string) (string, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/accounts/%s/cfd_tunnel?name=%s", c.AccountID, name),
		nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tr TunnelListResponse
	if err := decode(resp, &tr); err != nil {
		return "", err
	}
	if !tr.Success {
		return "", apiErr(tr.Errors)
	}
	for _, t := range tr.Result {
		if t.Name == name {
			return t.ID, nil
		}
	}
	return "", nil
}
