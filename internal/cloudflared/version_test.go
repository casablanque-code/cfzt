package cloudflared

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name                           string
		raw                            string
		wantYear, wantMonth, wantPatch int
	}{
		{"standard release", "cloudflared version 2024.6.1 (built 2024-06-10-1200 UTC)", 2024, 6, 1},
		{"different command name in output", "2025.12.3", 2025, 12, 3},
		{"double-digit month and patch", "cloudflared version 2023.11.22 (built ...)", 2023, 11, 22},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := parseVersion(tt.raw)
			if v.Year != tt.wantYear || v.Month != tt.wantMonth || v.Patch != tt.wantPatch {
				t.Errorf("parseVersion(%q) = {%d %d %d}, want {%d %d %d}",
					tt.raw, v.Year, v.Month, v.Patch, tt.wantYear, tt.wantMonth, tt.wantPatch)
			}
			if v.Raw != tt.raw {
				t.Errorf("parseVersion(%q).Raw = %q, want unchanged input", tt.raw, v.Raw)
			}
		})
	}
}

func TestParseVersion_UnrecognizedFormat(t *testing.T) {
	v := parseVersion("some unexpected output with no version number")
	if v.Year != 0 || v.Month != 0 || v.Patch != 0 {
		t.Errorf("parseVersion(unrecognized) = %+v, want all-zero fields", v)
	}
	// An unparseable version must not be treated as too old — GetVersion's
	// callers use TooOld() to decide whether to block, and blocking on a
	// format we simply don't understand yet would break every future
	// cloudflared release before we update the regex for it.
	if v.TooOld() {
		t.Error("parseVersion(unrecognized).TooOld() = true, want false — unknown format should never block")
	}
}

func TestVersion_TooOld(t *testing.T) {
	tests := []struct {
		year int
		want bool
	}{
		{minYear - 1, true},
		{minYear, false},
		{minYear + 1, false},
	}
	for _, tt := range tests {
		v := Version{Year: tt.year}
		if got := v.TooOld(); got != tt.want {
			t.Errorf("Version{Year: %d}.TooOld() = %v, want %v", tt.year, got, tt.want)
		}
	}
}

func TestVersion_String(t *testing.T) {
	v := Version{Raw: "cloudflared version 2024.6.1"}
	if got := v.String(); got != v.Raw {
		t.Errorf("Version.String() = %q, want Raw %q", got, v.Raw)
	}
}

func TestMinYear(t *testing.T) {
	if MinYear() != minYear {
		t.Errorf("MinYear() = %d, want %d", MinYear(), minYear)
	}
}
