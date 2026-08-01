//go:build windows

package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskName(t *testing.T) {
	if got := taskName("grafana"); got != "zt-grafana" {
		t.Errorf("taskName(grafana) = %q, want zt-grafana", got)
	}
}

func TestUnitPath(t *testing.T) {
	if got := UnitPath("grafana"); got != `\zt-grafana` {
		t.Errorf(`UnitPath(grafana) = %q, want \zt-grafana`, got)
	}
}

func TestLingerEnabled(t *testing.T) {
	// No systemd/launchd equivalent to toggle on Windows - a logon-trigger
	// task always "lingers" in that sense, so this is always true.
	if !LingerEnabled() {
		t.Error("LingerEnabled() = false, want true")
	}
}

func TestQuoteArg(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`C:\Program Files\cloudflared\cloudflared.exe`, `"C:\Program Files\cloudflared\cloudflared.exe"`},
		{`say "hi"`, `"say \"hi\""`},
		{"plain", `"plain"`},
	}
	for _, tt := range tests {
		if got := quoteArg(tt.in); got != tt.want {
			t.Errorf("quoteArg(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildTaskXML(t *testing.T) {
	xml := buildTaskXML("zt tunnel — grafana", "cmd.exe", "/c", `"C:\cloudflared.exe" tunnel run`)

	for _, want := range []string{
		"<Description>zt tunnel — grafana</Description>",
		"<Command>cmd.exe</Command>",
		`<Arguments>"/c" "\"C:\cloudflared.exe\" tunnel run"</Arguments>`,
		"<LogonTrigger>",
		"<RestartOnFailure>",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("buildTaskXML() missing %q in output:\n%s", want, xml)
		}
	}

	// The XML must never request stored credentials - that's what keeps
	// Install() from needing admin rights or a password prompt.
	for _, unwanted := range []string{"<LogonType>Password</LogonType>", "<UserId>"} {
		if strings.Contains(xml, unwanted) {
			t.Errorf("buildTaskXML() contains %q, task should not require stored credentials", unwanted)
		}
	}
}

// fakeExecutable creates a file at dir/name.exe that exec.LookPath will
// find via PATHEXT. Content doesn't matter since we never run it.
func fakeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name+".exe")
	if err := os.WriteFile(path, []byte("not a real binary"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCloudflaredBin_Found(t *testing.T) {
	dir := t.TempDir()
	fakeExecutable(t, dir, "cloudflared")
	t.Setenv("PATH", dir)
	t.Setenv("PATHEXT", ".EXE")

	path, err := cloudflaredBin()
	if err != nil {
		t.Fatalf("cloudflaredBin() error = %v", err)
	}
	want := filepath.Join(dir, "cloudflared.exe")
	if path != want {
		t.Errorf("cloudflaredBin() = %q, want %q", path, want)
	}
}

func TestCloudflaredBin_NotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PATHEXT", ".EXE")

	if _, err := cloudflaredBin(); err == nil {
		t.Fatal("cloudflaredBin() = nil error, want error when cloudflared isn't on PATH")
	}
}

// Install() must fail before ever touching schtasks if cloudflared isn't
// on PATH - this must be safe to run on any Windows CI runner without
// leaving a real scheduled task behind.
func TestInstall_FailsCleanlyWithoutCloudflared(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PATHEXT", ".EXE")

	err := Install("zt-test-tunnel-does-not-exist", `C:\config.yml`, `C:\cloudflared.log`)
	if err == nil {
		t.Fatal("Install() = nil error, want error when cloudflared isn't on PATH")
	}

	// Must not have registered a task for a service it never actually
	// configured.
	if IsInstalled("zt-test-tunnel-does-not-exist") {
		t.Error("Install() registered a task despite failing before creating one")
	}
}

func TestTaskExists_NotInstalled(t *testing.T) {
	if taskExists("zt-test-task-does-not-exist") {
		t.Error("taskExists() = true, want false for a task that was never created")
	}
}
