//go:build windows

package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procShowWindow       = user32.NewProc("ShowWindow")
	procGetConsoleWindow = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
)

const swHide = 0

// hideOwnConsoleWindow hides the console window Windows allocates for our
// own process, if any.
//
// `zt internal run` is meant to be invisible: it's launched by Task
// Scheduler as the tunnel/watchdog action, not by a user sitting at a
// terminal. But zt.exe is a console-subsystem binary (it has to be —
// that's what makes normal interactive `zt up`/`zt status`/etc. work at
// all), and Windows allocates a brand-new console window for any
// console-subsystem process that doesn't inherit one, which is exactly
// what happens when Task Scheduler starts it. Without this, every tunnel
// popped up a visible console window on the user's desktop — and closing
// that window sent a close signal to everything attached to that console,
// taking cloudflared down with it, since it inherits the same console as
// its parent by default.
//
// Hiding after the fact (rather than never creating the console at all)
// is the standard approach for a single binary that needs to run both
// interactively and headless — building zt.exe itself as a GUI-subsystem
// binary would silence normal CLI usage, and Task Scheduler gives us no
// way to pass CREATE_NO_WINDOW for its own action's CreateProcess call.
func hideOwnConsoleWindow() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	_, _, _ = procShowWindow.Call(hwnd, swHide)
}

// configureChildForBackground puts cmd in its own process group so it
// doesn't share fate with our (now hidden) console: closing or signaling
// our console/process no longer takes the child down with it, and it's
// what lets a future Stop-equivalent target this process tree specifically
// with CTRL_BREAK_EVENT, the same technique runner_windows.go's PID-mode
// Start() already uses for the same reason.
func configureChildForBackground(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}
