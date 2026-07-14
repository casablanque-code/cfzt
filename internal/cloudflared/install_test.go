package cloudflared

import "testing"

func TestUpgradeCommand(t *testing.T) {
	cases := []struct {
		method InstallMethod
		wantOK bool
	}{
		{InstallHomebrew, true},
		{InstallAPT, true},
		{InstallBinary, false},
		{InstallUnknown, false},
	}
	for _, c := range cases {
		cmd, ok := UpgradeCommand(c.method)
		if ok != c.wantOK {
			t.Errorf("UpgradeCommand(%q) ok = %v, want %v", c.method, ok, c.wantOK)
		}
		if ok && cmd == "" {
			t.Errorf("UpgradeCommand(%q) returned ok=true but empty command", c.method)
		}
	}
}
