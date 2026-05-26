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
}

func NewClient(apiToken, accountID string) *Client {
	return &Client{
		AccountID: accountID,
		apiToken:  apiToken,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := baseURL + path
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
