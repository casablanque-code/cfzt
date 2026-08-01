//go:build windows

package service

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func buildTaskXML(description, cmd string, args ...string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = quoteArg(a)
	}
	return fmt.Sprintf(taskXMLTemplate, description, cmd, strings.Join(quoted, " "))
}

// quoteArg wraps an argument in double quotes for the Task Scheduler
// <Arguments> string, escaping any embedded double quotes. Paths on
// Windows routinely contain spaces (Program Files, user profile dirs),
// so unquoted arguments would be split incorrectly.
func quoteArg(a string) string {
	return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
}

func taskExists(name string) bool {
	err := exec.Command("schtasks", "/query", "/tn", name).Run()
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

	out, err := exec.Command("schtasks", "/create", "/tn", name, "/xml", tmpPath, "/f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks /create: %w\n%s", err, out)
	}
	return nil
}

// Install registers a Task Scheduler task that runs cloudflared at user
// logon and restarts it on failure. Stdout/stderr are redirected to
// logPath via a cmd.exe wrapper since Task Scheduler actions have no
// native output-redirection field.
func Install(name, configPath, logPath string) error {
	bin, err := cloudflaredBin()
	if err != nil {
		return err
	}

	inner := fmt.Sprintf(`"%s" tunnel --config "%s" run >> "%s" 2>&1`, bin, configPath, logPath)
	xml := buildTaskXML("zt tunnel — "+name, "cmd.exe", "/c", inner)

	return createTask(taskName(name), xml)
}

// Restart stops and re-runs the task immediately (RestartOnFailure only
// fires when the action process exits, not on demand).
func Restart(name string) error {
	tn := taskName(name)
	_ = exec.Command("schtasks", "/end", "/tn", tn).Run()
	out, err := exec.Command("schtasks", "/run", "/tn", tn).CombinedOutput()
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
	out, err := exec.Command("schtasks", "/delete", "/tn", tn, "/f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks /delete: %w\n%s", err, out)
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

func InstallWatchdog(logPath string) error {
	return fmt.Errorf("watchdog is not supported on Windows yet — run `zt watchdog run` manually if needed")
}
func UninstallWatchdog() error { return nil }

// WatchdogIsInstalled returns true if the watchdog task exists.
func WatchdogIsInstalled() bool {
	return taskExists(watchdogTaskName)
}
func WatchdogIsActive() bool { return false }
