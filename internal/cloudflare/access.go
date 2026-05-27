package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type AccessApp struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Domain          string `json:"domain"`
	SessionDuration string `json:"session_duration"`
	Type            string `json:"type"`
}

type accessAppResponse struct {
	Result  AccessApp `json:"result"`
	Success bool      `json:"success"`
	Errors  []APIErr  `json:"errors"`
}

type accessListResponse struct {
	Result  []AccessApp `json:"result"`
	Success bool        `json:"success"`
	Errors  []APIErr    `json:"errors"`
}

type AccessPolicy struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Decision   string       `json:"decision"`
	Include    []AccessRule `json:"include"`
	Precedence int          `json:"precedence"`
}

type emailVal struct {
	Email string `json:"email"`
}

type emailDomainVal struct {
	Domain string `json:"domain"`
}

type AccessRule struct {
	Everyone    *struct{}       `json:"everyone,omitempty"`
	Email       *emailVal       `json:"email,omitempty"`
	EmailDomain *emailDomainVal `json:"email_domain,omitempty"`
}

// FindAccessAppByDomain finds an existing Access app by hostname.
// Returns nil (no error) if not found.
func (c *Client) FindAccessAppByDomain(hostname string) (*AccessApp, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/accounts/%s/access/apps", c.AccountID),
		nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var lr accessListResponse
	if err := decode(resp, &lr); err != nil {
		return nil, err
	}
	if !lr.Success {
		return nil, apiErr(lr.Errors)
	}
	for _, app := range lr.Result {
		if app.Domain == hostname {
			a := app
			return &a, nil
		}
	}
	return nil, nil
}

// UpsertAccessApp creates or replaces a Zero Trust Access application for a hostname.
// If an app for the same domain already exists it is deleted first.
func (c *Client) UpsertAccessApp(hostname, name string) (string, error) {
	existing, err := c.FindAccessAppByDomain(hostname)
	if err != nil {
		return "", fmt.Errorf("checking existing Access apps: %w", err)
	}
	if existing != nil {
		if err := c.DeleteAccessApp(existing.ID); err != nil {
			return "", fmt.Errorf("removing existing Access app: %w", err)
		}
	}

	payload := map[string]any{
		"name":                      name,
		"domain":                    hostname,
		"type":                      "self_hosted",
		"session_duration":          "24h",
		"auto_redirect_to_identity": false,
	}
	body, _ := json.Marshal(payload)

	resp, err := c.do("POST",
		fmt.Sprintf("/accounts/%s/access/apps", c.AccountID),
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var ar accessAppResponse
	if err := decode(resp, &ar); err != nil {
		return "", err
	}
	if !ar.Success {
		return "", apiErr(ar.Errors)
	}
	return ar.Result.ID, nil
}

// CreateBypassPolicy creates an access policy on an app.
// Empty emails = bypass (public). With emails = allow specific addresses.
func (c *Client) CreateBypassPolicy(appID string, emails []string) error {
	var include []AccessRule
	if len(emails) == 0 {
		include = []AccessRule{{Everyone: &struct{}{}}}
	} else {
		for _, e := range emails {
			email := e
			include = append(include, AccessRule{
				Email: &emailVal{Email: email},
			})
		}
	}

	decision := "bypass"
	if len(emails) > 0 {
		decision = "allow"
	}

	payload := map[string]any{
		"name":       "default",
		"decision":   decision,
		"include":    include,
		"precedence": 1,
	}
	body, _ := json.Marshal(payload)

	resp, err := c.do("POST",
		fmt.Sprintf("/accounts/%s/access/apps/%s/policies", c.AccountID, appID),
		bytes.NewReader(body))
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

// DeleteAccessApp removes an Access application.
func (c *Client) DeleteAccessApp(appID string) error {
	resp, err := c.do("DELETE",
		fmt.Sprintf("/accounts/%s/access/apps/%s", c.AccountID, appID),
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
