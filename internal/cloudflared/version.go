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

// TooOld reports whether this version predates the minimum supported
// cloudflared release. Year == 0 means the version string didn't match
// the expected format (see parseVersion) — that's deliberately never
// "too old": we don't know what it is, so we don't block on it, we just
// can't give a confident answer either.
func (v Version) TooOld() bool {
	if v.Year == 0 {
		return false
	}
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
	return parseVersion(raw), nil
}

// parseVersion extracts year.month.patch from cloudflared's raw --version
// output. An unrecognized format isn't an error here — GetVersion's
// callers only care whether TooOld() should block, and a Version with no
// year parsed (Year == 0) never reports TooOld() as true, so an unknown
// format degrades to "don't block" rather than failing outright.
func parseVersion(raw string) *Version {
	m := reVersion.FindStringSubmatch(raw)
	if m == nil {
		return &Version{Raw: raw}
	}

	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])

	return &Version{
		Year:  year,
		Month: month,
		Patch: patch,
		Raw:   raw,
	}
}

// MinYear returns the minimum supported cloudflared release year.
func MinYear() int { return minYear }
