//go:build windows

package cloudflared

type Version struct {
	Year  int
	Month int
	Patch int
	Raw   string
}

func (v Version) String() string { return v.Raw }
func (v Version) TooOld() bool   { return false }

func GetVersion() (*Version, error) {
	return &Version{Raw: "unknown"}, nil
}

func MinYear() int { return 2023 }
