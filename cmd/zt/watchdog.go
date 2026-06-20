package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/casablanque-code/cfzt/internal/service"
	"github.com/casablanque-code/cfzt/internal/watchdog"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var watchdogCmd = &cobra.Command{
	Use:   "watchdog",
	Short: "Manage the QUIC/HTTP2 fallback watchdog",
	Long: `cloudflared falls back from QUIC to HTTP/2 when UDP is briefly
blocked, but never retries QUIC again on its own
(see https://github.com/cloudflare/cloudflared/issues/1534).

The watchdog runs in the background, watches each tunnel's log for the
fallback, and restarts the tunnel after a backoff delay so cloudflared
gets a fresh chance to negotiate QUIC. Only tunnels running with
protocol "auto" (the default) are affected — tunnels pinned to a
specific protocol via --protocol or --tcp are left alone.`,
}

var watchdogEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Install and start the watchdog as a background service",
	RunE:  runWatchdogEnable,
}

var watchdogDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Stop and remove the watchdog background service",
	RunE:  runWatchdogDisable,
}

var watchdogStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the watchdog is running",
	RunE:  runWatchdogStatus,
}

// watchdogRunCmd is intentionally not documented prominently — it's the
// entrypoint the systemd/launchd unit calls to run the watchdog loop
// itself (ExecStart=zt watchdog run). Users normally interact via
// `zt watchdog enable/disable/status`, not this directly.
var watchdogRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the watchdog loop in the foreground (used internally by the service unit)",
	Hidden: true,
	RunE:   runWatchdogLoop,
}

func init() {
	watchdogCmd.AddCommand(watchdogEnableCmd)
	watchdogCmd.AddCommand(watchdogDisableCmd)
	watchdogCmd.AddCommand(watchdogStatusCmd)
	watchdogCmd.AddCommand(watchdogRunCmd)
}

func watchdogLogPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".zt", "watchdog.log"), nil
}

func runWatchdogEnable(cmd *cobra.Command, args []string) error {
	boldFmt := color.New(color.Bold).SprintFunc()
	okFn := color.New(color.FgGreen).SprintFunc()

	if service.WatchdogIsInstalled() {
		fmt.Println("  watchdog is already installed — run `zt watchdog status` to check it")
		return nil
	}

	logPath, err := watchdogLogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return fmt.Errorf("failed to create watchdog log dir: %w", err)
	}

	fmt.Printf("\n%s\n\n", boldFmt("⚡ Enabling QUIC/HTTP2 fallback watchdog"))
	fmt.Println("  → installing background service")

	if err := service.InstallWatchdog(logPath); err != nil {
		return err
	}

	fmt.Printf("     %s watchdog running — checks every %s\n", okFn("✓"), watchdog.DefaultPollInterval)
	fmt.Println()
	return nil
}

func runWatchdogDisable(cmd *cobra.Command, args []string) error {
	okFn := color.New(color.FgGreen).SprintFunc()

	if !service.WatchdogIsInstalled() {
		fmt.Println("  watchdog is not installed")
		return nil
	}

	if err := service.UninstallWatchdog(); err != nil {
		return err
	}
	fmt.Printf("  %s watchdog disabled\n", okFn("✓"))
	return nil
}

func runWatchdogStatus(cmd *cobra.Command, args []string) error {
	if !service.WatchdogIsInstalled() {
		fmt.Println("  watchdog is not installed — run `zt watchdog enable` to start it")
		return nil
	}

	statusStr := fail("stopped")
	if service.WatchdogIsActive() {
		statusStr = pass("running")
	}
	fmt.Printf("  watchdog: %s\n", statusStr)

	logPath, err := watchdogLogPath()
	if err == nil {
		fmt.Printf("  log:      %s\n", logPath)
	}
	return nil
}

// runWatchdogLoop is the long-running process started by the service unit.
// It ticks every DefaultPollInterval, evaluating all "auto"-protocol
// tunnels and restarting any that have fallen back to HTTP/2 outside
// their backoff window.
func runWatchdogLoop(cmd *cobra.Command, args []string) error {
	fmt.Printf("zt watchdog starting — polling every %s\n", watchdog.DefaultPollInterval)

	ticker := time.NewTicker(watchdog.DefaultPollInterval)
	defer ticker.Stop()

	// run once immediately on startup, then on every tick
	tick := func() {
		result, err := watchdog.RunOnce(restartTunnel, func(line string) {
			fmt.Println(line)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "watchdog tick error: %v\n", err)
			return
		}
		for name, tickErr := range result.Errors {
			fmt.Fprintf(os.Stderr, "watchdog: %s: %v\n", name, tickErr)
		}
	}

	tick()
	for range ticker.C {
		tick()
	}
	return nil
}
