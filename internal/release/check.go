// Package release checks GitHub for the latest cfzt release so `zt version`
// and `zt doctor` can nudge people running an old build. It's deliberately
// best-effort: cfzt is meant to work fine on an offline homelab box, so a
// slow or unreachable network must never block or clutter a command —
// callers are expected to swallow errors from CheckLatest and simply skip
// the nudge.
package release

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const releaseAPI = "https://api.github.com/repos/casablanque-code/cfzt/releases/latest"

// checkTimeout keeps a stalled or unreachable network from ever turning
// this into a noticeable delay.
const checkTimeout = 1500 * time.Millisecond

// Info describes the result of a latest-release check.
type Info struct {
	Current   string // as passed in to CheckLatest
	Latest    string // tag_name from the GitHub release, e.g. "v0.8.0"
	Available bool   // true if Latest is newer than Current
	URL       string // HTML URL of the release, for the upgrade hint
}

// CheckLatest queries GitHub for the latest cfzt release and compares it
// against current. A "dev" or empty current (an unversioned local build)
// never reports an update — there's nothing meaningful to compare.
func CheckLatest(current string) (*Info, error) {
	return checkLatestFrom(releaseAPI, current)
}

func checkLatestFrom(apiURL, current string) (*Info, error) {
	if current == "" || current == "dev" {
		return &Info{Current: current}, nil
	}

	client := &http.Client{Timeout: checkTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release API returned %d", resp.StatusCode)
	}

	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	info := &Info{Current: current, Latest: body.TagName, URL: body.HTMLURL}
	newer, err := isNewer(body.TagName, current)
	if err != nil {
		// Unrecognized version format on either side — don't guess.
		return info, nil
	}
	info.Available = newer
	return info, nil
}

// isNewer reports whether a is a newer semver than b. Both may have a
// leading "v" and are expected in X.Y.Z form.
func isNewer(a, b string) (bool, error) {
	av, err := parseSemver(a)
	if err != nil {
		return false, err
	}
	bv, err := parseSemver(b)
	if err != nil {
		return false, err
	}
	for i := range av {
		if av[i] != bv[i] {
			return av[i] > bv[i], nil
		}
	}
	return false, nil
}

func parseSemver(v string) ([3]int, error) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return out, fmt.Errorf("not a semver: %q", v)
	}
	for i, p := range parts {
		if i == 2 {
			// drop a pre-release/build suffix on the patch component, e.g. "2-rc1"
			if idx := strings.IndexAny(p, "-+"); idx >= 0 {
				p = p[:idx]
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, fmt.Errorf("not a semver: %q", v)
		}
		out[i] = n
	}
	return out, nil
}
