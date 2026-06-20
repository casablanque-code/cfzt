package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zt",
	Short: "Zero Trust tunnel manager for Cloudflare",
	Long: `zt — bring up Zero Trust tunnels with a single command.

  zt init                      configure API credentials
  zt up <name> <port>          expose a local service
  zt down <name>               tear down a tunnel
  zt restart <name>            restart cloudflared for a tunnel
  zt list                      list active tunnels
  zt status <name>             show tunnel details
  zt logs <name>               show cloudflared logs
  zt doctor                    check system and tunnel health
  zt export [-o zt.yaml]       export managed tunnels to a portable manifest
  zt apply <file>              apply a zt.yaml manifest on this machine
  zt watchdog enable           auto-recover from QUIC→HTTP2 fallback`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(watchdogCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
