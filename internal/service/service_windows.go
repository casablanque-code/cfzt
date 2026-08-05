//go:build windows

package service

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf16"
)

// taskXMLTemplate defines a Task Scheduler task that runs at user logon and
// restarts itself if the process exits (crash, killed, etc). This mirrors
// the Restart=on-failure behavior of the systemd unit / LaunchAgent
// KeepAlive used on Linux/macOS.
//
// Deliberately does NOT set a stored-password LogonType or UserId, so task
// creation never needs admin rights or a password prompt — same
// permission model as the user-level systemd --user / LaunchAgent
// services already in use.
const taskXMLTemplate = `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>%s</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>999</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <Arguments>%s</Arguments>
    </Exec>
  </Actions>
</Task>
`

// taskName returns the Task Scheduler task name for a tunnel, matching the
// zt-<name> convention used for the systemd unit / LaunchAgent label.
func taskName(name string) string {
	return "zt-" + name
}

const watchdogTaskName = "zt-watchdog"

// buildTaskXML renders a Task Scheduler task definition for running cmd
// with args at user logon, restarting on failure. description is used for
// the task's RegistrationInfo only — purely informational.
//
// description/cmd/args are XML-escaped before insertion. Without this,
// any value containing an XML special character produces a task file
// Windows rejects outright — an unescaped '&' anywhere in an argument
// (a path, a hostname, ...) makes every Install()/InstallWatchdog() call
// fail with "ERROR: The task XML is malformed."
func buildTaskXML(description, cmd string, args ...string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = quoteArg(a)
	}
	return fmt.Sprintf(taskXMLTemplate,
		xmlEscapeText(description),
		xmlEscapeText(cmd),
		xmlEscapeText(strings.Join(quoted, " ")),
	)
}

// xmlEscapeText escapes s for use as XML element text content.
func xmlEscapeText(s string) string {
	var buf strings.Builder
	// xml.EscapeText never returns an error for a strings.Builder target.
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// quoteArg wraps an argument in double quotes for the Task Scheduler
// <Arguments> string, escaping any embedded double quotes. Paths on
// Windows routinely contain spaces (Program Files, user profile dirs),
// so unquoted arguments would be split incorrectly. This is CRT
// (CommandLineToArgvW) argument-quoting syntax, applied before the XML
// escaping in buildTaskXML — the two operate on different layers and both
// are needed. It's also the *only* quoting layer an argument passes
// through: Task Scheduler invokes the Command directly (zt.exe, see
// tunnelRunArgs/watchdogRunArgs below), not via cmd.exe, so there's no
// second shell re-parsing the already-quoted string a second time.
func quoteArg(a string) string {
	return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
}

// tunnelRunArgs builds the argv Task Scheduler uses to launch cloudflared
// under `zt internal run` — zt opens the log file itself and execs bin
// with real argv, so Task Scheduler never has to go through cmd.exe (and
// its own argument re-parsing) just to redirect output to a file. See
// cmd/zt/internal_run.go for the run side of this.
func tunnelRunArgs(bin, configPath, logPath string) []string {
	return []string{"internal", "run", "--log", logPath, "--", bin, "tunnel", "--config", configPath, "run"}
}

// watchdogRunArgs is tunnelRunArgs' counterpart for the watchdog task —
// same wrapper, running `zt watchdog run` (which already writes its own
// progress to stdout) as the child instead of cloudflared.
func watchdogRunArgs(zt, logPath string) []string {
	return []string{"internal", "run", "--log", logPath, "--", zt, "watchdog", "run"}
}

var setConsoleUTF8Once sync.Once

// ensureUTF8Console switches the current console's active output codepage
// to UTF-8 (65001). schtasks always emits its (often localized) error text
// in whatever codepage the console is running — CP866 on ru-RU Windows,
// for instance — so capturing that output as a Go string and printing it
// assuming UTF-8 produces mojibake. Idempotent per process; chcp.com is a
// real standalone executable (not a cmd.exe builtin), so no shell wrapper
// is needed.
func ensureUTF8Console() {
	setConsoleUTF8Once.Do(func() {
		_ = exec.Command("chcp.com", "65001").Run()
	})
}

// runSchtasks runs schtasks with the given args after ensuring the console
// is in UTF-8, so any error text it captures is decodable.
func runSchtasks(args ...string) ([]byte, error) {
	ensureUTF8Console()
	return exec.Command("schtasks", args...).CombinedOutput()
}

// accessDeniedHint is appended to /create and /delete failures. Task
// Scheduler ties task ownership to whichever principal created it — if a
// zt-* task was ever created or touched from an elevated (Administrator)
// prompt, a later attempt from a normal session gets Access Denied even
// with /f, since it can't take over a task it doesn't own. This is the
// single most common cause of a "no admin needed" install suddenly
// failing, so it's worth a hint even when a given failure turns out to be
// something else — it's cheap to check and confirms or rules itself out.
func accessDeniedHint(taskName string) string {
	return fmt.Sprintf("(if this is a permissions error: a %s task created or modified from an elevated/Administrator prompt can't be overwritten from a normal session, even with /f — delete it once from an elevated prompt with `schtasks /delete /tn %s /f`, then retry normally)", taskName, taskName)
}

func taskExists(name string) bool {
	_, err := runSchtasks("/query", "/tn", name)
	return err == nil
}

func cloudflaredBin() (string, error) {
	path, err := exec.LookPath("cloudflared")
	if err != nil {
		return "", fmt.Errorf("cloudflared not found in PATH")
	}
	return path, nil
}

// createTask writes xml to a temp file and registers it via
// `schtasks /create /xml`, then removes the temp file. /xml is used
// instead of building the task from flags because RestartOnFailure and
// the no-stored-credentials logon type aren't reachable from schtasks'
// flag surface.
func createTask(name, xml string) error {
	f, err := os.CreateTemp("", "zt-task-*.xml")
	if err != nil {
		return fmt.Errorf("failed to create temp task file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)

	// schtasks expects the XML file as UTF-16LE with a BOM.
	if _, err := f.Write([]byte{0xFF, 0xFE}); err != nil {
		f.Close()
		return fmt.Errorf("failed to write task file: %w", err)
	}
	for _, u := range utf16.Encode([]rune(xml)) {
		if err := binary.Write(f, binary.LittleEndian, u); err != nil {
			f.Close()
			return fmt.Errorf("failed to write task file: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	out, err := runSchtasks("/create", "/tn", name, "/xml", tmpPath, "/f")
	if err != nil {
		return fmt.Errorf("schtasks /create: %w\n%s\n%s", err, out, accessDeniedHint(name))
	}
	return nil
}

// Install registers a Task Scheduler task that runs cloudflared at user
// logon and restarts it on failure, then starts it immediately.
//
// A LogonTrigger only fires at the *next* logon — since `zt up` is
// normally run from an already-logged-in session, the task would
// otherwise sit in "Ready" state, never actually launching cloudflared,
// until the next reboot/logon. That's silent: schtasks /create succeeds
// either way, so without this the tunnel looks fully set up (DNS, ingress,
// task registered) while nothing is actually listening — `zt doctor` and
// `zt status` would report it as inactive right after a successful-looking
// `zt up`. This mirrors `systemctl --user enable --now` on Linux and
// LaunchAgent's immediate load on macOS, both of which start the service
// as part of Install(), not just register it.
func Install(name, configPath, logPath string) error {
	bin, err := cloudflaredBin()
	if err != nil {
		return err
	}
	zt, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine zt binary path: %w", err)
	}

	tn := taskName(name)
	xml := buildTaskXML("zt tunnel — "+name, zt, tunnelRunArgs(bin, configPath, logPath)...)

	if err := createTask(tn, xml); err != nil {
		return err
	}

	if out, err := runSchtasks("/run", "/tn", tn); err != nil {
		return fmt.Errorf("task created but failed to start: schtasks /run: %w\n%s", err, out)
	}
	return nil
}

// Restart stops and re-runs the task immediately (RestartOnFailure only
// fires when the action process exits, not on demand).
func Restart(name string) error {
	tn := taskName(name)
	_, _ = runSchtasks("/end", "/tn", tn)
	out, err := runSchtasks("/run", "/tn", tn)
	if err != nil {
		return fmt.Errorf("schtasks /run: %w\n%s", err, out)
	}
	return nil
}

// Uninstall removes the task. Missing task is not an error — mirrors
// Uninstall's os.IsNotExist handling on Linux/macOS.
func Uninstall(name string) error {
	tn := taskName(name)
	if !taskExists(tn) {
		return nil
	}
	out, err := runSchtasks("/delete", "/tn", tn, "/f")
	if err != nil {
		return fmt.Errorf("schtasks /delete: %w\n%s\n%s", err, out, accessDeniedHint(tn))
	}
	return nil
}

// UnitPath returns the Task Scheduler task path (there is no unit file on
// disk to point to, unlike systemd/launchd, so this is informational).
func UnitPath(name string) string {
	return `\` + taskName(name)
}

// IsInstalled returns true if the Task Scheduler task exists.
func IsInstalled(name string) bool {
	return taskExists(taskName(name))
}

// LingerEnabled always returns true — like LaunchAgents, a logon-trigger
// Task Scheduler task runs automatically once the user logs in, with no
// separate "linger" toggle to check.
func LingerEnabled() bool { return true }

// Description returns a human-readable label for the installed service,
// used in zt up's output instead of a systemd/launchd-shaped message.
func Description(name string) string {
	return fmt.Sprintf("%s scheduled task (auto-start at logon)", taskName(name))
}

// ManagerName returns a short label for the service manager in use, for
// generic status/error messages that shouldn't hardcode "systemd".
func ManagerName() string { return "scheduled task" }

// InstallWatchdog registers and immediately starts the watchdog task —
// see Install's comment on why a LogonTrigger task needs an explicit
// /run after /create to actually be running, not just registered.
func InstallWatchdog(logPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine zt binary path: %w", err)
	}

	xml := buildTaskXML("zt QUIC/HTTP2 fallback watchdog", exe, watchdogRunArgs(exe, logPath)...)

	if err := createTask(watchdogTaskName, xml); err != nil {
		return err
	}
	if out, err := runSchtasks("/run", "/tn", watchdogTaskName); err != nil {
		return fmt.Errorf("task created but failed to start: schtasks /run: %w\n%s", err, out)
	}
	return nil
}

// UninstallWatchdog removes the watchdog task. Missing task is a no-op.
func UninstallWatchdog() error {
	if !taskExists(watchdogTaskName) {
		return nil
	}
	out, err := runSchtasks("/delete", "/tn", watchdogTaskName, "/f")
	if err != nil {
		return fmt.Errorf("schtasks /delete: %w\n%s\n%s", err, out, accessDeniedHint(watchdogTaskName))
	}
	return nil
}

// WatchdogIsInstalled returns true if the watchdog task exists.
func WatchdogIsInstalled() bool {
	return taskExists(watchdogTaskName)
}

// taskIsRunning reports whether the named Task Scheduler task is
// currently running.
//
// This shells out to PowerShell's Get-ScheduledTask rather than parsing
// `schtasks /query ... /fo list /v` output, because schtasks localizes
// both its field labels ("Status:") and values ("Running") to the OS
// display language — on a ru-RU system those come back as something
// like "Состояние:"/"Выполняется", so a hardcoded English match against
// them silently always returns false. Get-ScheduledTask's State is a
// .NET enum; its symbolic name (Running/Ready/Disabled/...) doesn't
// change with display language.
func taskIsRunning(name string) bool {
	ensureUTF8Console()
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("(Get-ScheduledTask -TaskName %s -ErrorAction SilentlyContinue).State", psQuote(name)),
	).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "Running"
}

// psQuote single-quotes a string for interpolation into a PowerShell
// command, escaping embedded single quotes by doubling them (PowerShell's
// single-quoted string escaping rule).
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
