package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=..." (see Makefile).
// Defaults to "dev" for `go build`/`go run` without the Makefile.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "zt",
	Version: version,
	Short:   "Zero Trust tunnel manager for Cloudflare",
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
  zt watchdog enable           auto-recover from QUIC→HTTP2 fallback

Tips:
  zt up <name> <port> --tcp    force TCP/http2 if QUIC/UDP is blocked
  zt status <name> works as "zt <name> status" too
  zt completion bash|zsh|fish|powershell   shell tab-completion (see --help)

See "zt <command> --help" for full flags on any command.`,
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

	reorderArgs(os.Args, rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// tunnelNameFirstCmds are the subcommands whose only positional argument is
// a tunnel name, so "zt <name> <cmd>" is unambiguous and safe to accept as
// an alias for "zt <cmd> <name>" — useful since the tunnel name is usually
// what's top of mind, not the subcommand.
var tunnelNameFirstCmds = map[string]bool{
	"status":  true,
	"logs":    true,
	"restart": true,
	"down":    true,
}

// reorderArgs rewrites "zt <name> status" (etc.) in place to "zt status <name>"
// so both orders work identically. Only swaps when args[1] isn't itself a
// known subcommand/flag and args[2] is one of tunnelNameFirstCmds, so it
// never interferes with normal invocations or flags like "zt status -h".
func reorderArgs(args []string, root *cobra.Command) {
	if len(args) < 3 {
		return
	}
	first, second := args[1], args[2]
	if strings.HasPrefix(first, "-") {
		return
	}
	if isKnownSubcommand(root, first) {
		// args[1] is already a real subcommand — nothing to reorder.
		return
	}
	if tunnelNameFirstCmds[second] {
		args[1], args[2] = second, first
	}
}

// isKnownSubcommand reports whether name matches a registered subcommand's
// Use token, an alias, or the built-in help/completion commands.
func isKnownSubcommand(root *cobra.Command, name string) bool {
	if name == "help" || name == "completion" || name == "--help" || name == "-h" {
		return true
	}
	for _, c := range root.Commands() {
		if c.Name() == name {
			return true
		}
		for _, alias := range c.Aliases {
			if alias == name {
				return true
			}
		}
	}
	return false
}
