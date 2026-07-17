package cloudflare

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.cloudflare.com/client/v4"

type APIErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	AccountID string
	apiToken  string
	http      *http.Client
	baseURL   string // overridable for tests; defaults to the real API
}

func NewClient(apiToken, accountID string) *Client {
	return &Client{
		AccountID: accountID,
		apiToken:  apiToken,
		http:      &http.Client{Timeout: 30 * time.Second},
		baseURL:   baseURL,
	}
}

// NewClientForTesting returns a Client pointed at an arbitrary base URL
// (e.g. an httptest.Server) instead of the real Cloudflare API.
func NewClientForTesting(apiToken, accountID, testBaseURL string) *Client {
	c := NewClient(apiToken, accountID)
	c.baseURL = testBaseURL
	return c
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

func apiErr(errs []APIErr) error {
	if len(errs) == 0 {
		return fmt.Errorf("unknown API error")
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return fmt.Errorf("cloudflare API error: %s", strings.Join(msgs, "; "))
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

type tokenVerifyResult struct {
	Result struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		NotBefore string `json:"not_before"`
		ExpiresOn string `json:"expires_on"`
	} `json:"result"`
	Success bool     `json:"success"`
	Errors  []APIErr `json:"errors"`
}

// VerifyToken checks that the API token is valid and active.
// Returns a descriptive error if the token is invalid, expired, or lacks permissions.
func (c *Client) VerifyToken() error {
	resp, err := c.do("GET", "/user/tokens/verify", nil)
	if err != nil {
		return fmt.Errorf("could not reach Cloudflare API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result tokenVerifyResult
	if err := decode(resp, &result); err != nil {
		return fmt.Errorf("unexpected API response: %w", err)
	}

	if resp.StatusCode == 401 {
		return fmt.Errorf("token is invalid or revoked — check Cloudflare → My Profile → API Tokens")
	}
	if !result.Success {
		return apiErr(result.Errors)
	}
	if result.Result.Status != "active" {
		return fmt.Errorf("token status is %q (expected active)", result.Result.Status)
	}
	return nil
}

// VerifyZone checks that the given domain exists in the account.
func (c *Client) VerifyZone(domain string) error {
	_, err := c.GetZoneID(domain)
	return err
}
