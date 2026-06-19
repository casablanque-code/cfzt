package cloudflare

import "net/http"

func closeBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}
