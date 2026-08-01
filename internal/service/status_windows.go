//go:build windows

package service

// IsActive returns true if the Task Scheduler task for the tunnel is
// currently running.
func IsActive(name string) bool {
	return taskIsRunning(taskName(name))
}

// WatchdogIsActive returns true if the watchdog Task Scheduler task is
// currently running.
func WatchdogIsActive() bool {
	return taskIsRunning(watchdogTaskName)
}
