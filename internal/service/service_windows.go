//go:build windows

package service

import (
	"fmt"
	"os/exec"
	"strings"
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

// Install always fails for now — task creation (the mutating half of the
// Task Scheduler integration) lands in a follow-up patch. Callers (zt up,
// zt restart) fall back to PID-tracked mode automatically in the
// meantime, same as they would if systemd/launchd were unavailable on
// Linux/macOS. See runner_windows.go.
func Install(name, configPath, logPath string) error {
	return fmt.Errorf("no service manager integration on Windows yet — running in PID mode (no auto-restart on crash, no start-on-boot)")
}

func Restart(name string) error {
	return fmt.Errorf("service restart is not supported on Windows")
}

func Uninstall(name string) error { return nil }

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
