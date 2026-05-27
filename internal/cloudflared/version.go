//go:build !windows

package cloudflared

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const minYear = 2023

var reVersion = regexp.MustCompile(`(\d{4})\.(\d+)\.(\d+)`)

type Version struct {
	Year  int
	Month int
	Patch int
	Raw   string
}

func (v Version) String() string { return v.Raw }

func (v Version) TooOld() bool {
	return v.Year < minYear
}

// GetVersion returns the installed cloudflared version.
// Returns an error if cloudflared is not found or version can't be parsed.
func GetVersion() (*Version, error) {
	path, err := exec.LookPath("cloudflared")
	if err != nil {
		return nil, fmt.Errorf("cloudflared not found in PATH\n  install: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/")
	}

	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return nil, fmt.Errorf("cloudflared --version failed: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	m := reVersion.FindStringSubmatch(raw)
	if m == nil {
		// version format unknown — don't block, just return raw
		return &Version{Raw: raw}, nil
	}

	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])

	return &Version{
		Year:  year,
		Month: month,
		Patch: patch,
		Raw:   raw,
	}, nil
}

// MinYear returns the minimum supported cloudflared release year.
func MinYear() int { return minYear }
