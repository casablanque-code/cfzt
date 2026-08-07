//go:build windows

package service

import (
	"encoding/xml"
	"fmt"
	"io"
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
	taskXML := buildTaskXML("zt tunnel — grafana", "cmd.exe", "/c", `"C:\cloudflared.exe" tunnel run`)

	for _, want := range []string{
		"<Description>zt tunnel — grafana</Description>",
		"<Command>cmd.exe</Command>",
		// quoteArg's CRT-quoting happens first, then xmlEscapeText escapes
		// the resulting double quotes for XML text content. buildTaskXML
		// itself is generic — it doesn't care what "cmd" is — so this is
		// just exercising it with an arbitrary command/args pair.
		`<Arguments>&#34;/c&#34; &#34;\&#34;C:\cloudflared.exe\&#34; tunnel run&#34;</Arguments>`,
		"<LogonTrigger>",
		"<RestartOnFailure>",
	} {
		if !strings.Contains(taskXML, want) {
			t.Errorf("buildTaskXML() missing %q in output:\n%s", want, taskXML)
		}
	}

	// The XML must never request stored credentials - that's what keeps
	// Install() from needing admin rights or a password prompt.
	for _, unwanted := range []string{"<LogonType>Password</LogonType>", "<UserId>"} {
		if strings.Contains(taskXML, unwanted) {
			t.Errorf("buildTaskXML() contains %q, task should not require stored credentials", unwanted)
		}
	}
}

// TestBuildTaskXML_EscapesAmpersand is a regression test for a bare '&' in
// any argument — a hostname, a path, whatever — breaking the generated XML.
// Task Scheduler used to see this via the old cmd.exe log-redirection
// wrapper's "2>&1"; that wrapper is gone (see tunnelRunArgs/
// watchdogRunArgs), but buildTaskXML must still escape '&' correctly for
// any argument that happens to contain one, so this stays as a direct
// regression test on buildTaskXML/xmlEscapeText themselves.
func TestBuildTaskXML_EscapesAmpersand(t *testing.T) {
	got := buildTaskXML("zt tunnel — grafana", `C:\zt.exe`, "internal", "run", "--log", `C:\log&file.log`, "--", `C:\cloudflared.exe`)

	if strings.Contains(got, `log&file`) {
		t.Errorf("buildTaskXML() contains a raw unescaped '&', want it escaped as &amp;:\n%s", got)
	}
	if !strings.Contains(got, `log&amp;file`) {
		t.Errorf("buildTaskXML() = %s, want the '&' escaped as &amp;", got)
	}

	dec := xml.NewDecoder(strings.NewReader(strings.Replace(got, `encoding="UTF-16"`, `encoding="UTF-8"`, 1)))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("buildTaskXML() output does not parse as XML: %v\n%s", err, got)
			break
		}
	}
}

// TestBuildTaskXML_NoCmdExe is a regression test for the old three-layer
// quoting bug: Task Scheduler used to always invoke cmd.exe /c "<...>",
// which meant every argument was parsed twice (once by cmd.exe, once by
// whatever cmd.exe launched) before the real process ever saw it. As long
// as buildTaskXML is called with the real target executable as cmd — which
// Install()/InstallWatchdog() now do — cmd.exe never appears anywhere in
// the generated task.
func TestBuildTaskXML_NoCmdExe(t *testing.T) {
	got := buildTaskXML("zt tunnel — grafana", `C:\zt.exe`, tunnelRunArgs(`C:\cloudflared.exe`, `C:\config.yml`, `C:\cloudflared.log`)...)

	if strings.Contains(strings.ToLower(got), "cmd.exe") {
		t.Errorf("buildTaskXML() output still references cmd.exe:\n%s", got)
	}
	if !strings.Contains(got, `<Command>C:\zt.exe</Command>`) {
		t.Errorf("buildTaskXML() = %s, want <Command> to be the zt executable itself", got)
	}
}

func TestTunnelRunArgs(t *testing.T) {
	got := tunnelRunArgs(`C:\cloudflared.exe`, `C:\config.yml`, `C:\cloudflared.log`)
	want := []string{"internal", "run", "--log", `C:\cloudflared.log`, "--", `C:\cloudflared.exe`, "tunnel", "--config", `C:\config.yml`, "run"}
	if len(got) != len(want) {
		t.Fatalf("tunnelRunArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tunnelRunArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWatchdogRunArgs(t *testing.T) {
	got := watchdogRunArgs(`C:\zt.exe`, `C:\watchdog.log`)
	want := []string{"internal", "run", "--log", `C:\watchdog.log`, "--", `C:\zt.exe`, "watchdog", "run"}
	if len(got) != len(want) {
		t.Fatalf("watchdogRunArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("watchdogRunArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTailLog(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		if got := tailLog(filepath.Join(dir, "does-not-exist.log"), 10); got != "  (no log output captured)" {
			t.Errorf("tailLog() = %q, want the placeholder for a missing file", got)
		}
	})

	t.Run("fewer lines than n", func(t *testing.T) {
		p := filepath.Join(dir, "short.log")
		if err := os.WriteFile(p, []byte("one\ntwo\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got := tailLog(p, 10)
		if !strings.Contains(got, "one") || !strings.Contains(got, "two") {
			t.Errorf("tailLog() = %q, want it to contain all lines when there are fewer than n", got)
		}
	})

	t.Run("more lines than n keeps only the tail", func(t *testing.T) {
		p := filepath.Join(dir, "long.log")
		var lines []string
		for i := 1; i <= 20; i++ {
			lines = append(lines, fmt.Sprintf("line%02d", i))
		}
		if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got := tailLog(p, 5)
		if strings.Contains(got, lines[0]) {
			t.Errorf("tailLog(n=5) = %q, should not contain the first line of a 20-line file", got)
		}
		if !strings.Contains(got, lines[len(lines)-1]) {
			t.Errorf("tailLog(n=5) = %q, should contain the last line", got)
		}
	})
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

func TestPSQuote(t *testing.T) {
	tests := []struct{ in, want string }{
		{"zt-watchdog", "'zt-watchdog'"},
		{"it's", "'it''s'"},
	}
	for _, tt := range tests {
		if got := psQuote(tt.in); got != tt.want {
			t.Errorf("psQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAccessDeniedHint(t *testing.T) {
	hint := accessDeniedHint("zt-watchdog")
	if !strings.Contains(hint, "zt-watchdog") {
		t.Errorf("accessDeniedHint() = %q, want it to mention the task name", hint)
	}
	if !strings.Contains(hint, "schtasks /delete") {
		t.Errorf("accessDeniedHint() = %q, want a concrete schtasks /delete command", hint)
	}
}

func TestTaskExists_NotInstalled(t *testing.T) {
	if taskExists("zt-test-task-does-not-exist") {
		t.Error("taskExists() = true, want false for a task that was never created")
	}
}

// TestUninstall_NoTaskIsNoop and TestUninstallWatchdog_NoTaskIsNoop are
// regression tests for the "ends the task before deleting it" change:
// they exercise the early return for a task that was never created,
// which must stay a clean no-op and must NOT reach the /end call with a
// task name that was never registered.
func TestUninstall_NoTaskIsNoop(t *testing.T) {
	if err := Uninstall("zt-test-task-does-not-exist"); err != nil {
		t.Errorf("Uninstall() error = %v, want nil for a tunnel with no task installed", err)
	}
}

func TestUninstallWatchdog_NoTaskIsNoop(t *testing.T) {
	if taskExists(watchdogTaskName) {
		t.Skip("a real watchdog task is installed on this machine — skipping to avoid touching it")
	}
	if err := UninstallWatchdog(); err != nil {
		t.Errorf("UninstallWatchdog() error = %v, want nil when no watchdog task is installed", err)
	}
}
