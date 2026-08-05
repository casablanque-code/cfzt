package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/casablanque-code/cfzt/internal/release"
	"github.com/fatih/color"
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
  zt version                   show the running version
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

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the running version",
	// A plain command (not printVersion's --version shortcut) so cobra's
	// own tab-completion machinery knows "version" exists — see
	// reorderArgs' __complete handling for why that matters.
	RunE: func(cmd *cobra.Command, args []string) error {
		printVersion()
		return nil
	},
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
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(watchdogCmd)

	reorderArgs(os.Args, rootCmd)

	// --version specifically is handled here rather than left to cobra's
	// built-in version flag, so it goes through the same printVersion (and
	// its update check) as the "zt version" subcommand above instead of
	// cobra's plain built-in template.
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		printVersion()
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// printVersion prints the running version and, unless ZT_NO_UPDATE_CHECK is
// set, does a best-effort check for a newer release. The check is silent on
// any failure (offline machine, GitHub unreachable, rate limited) — cfzt is
// meant to work fine without ever phoning home, so this can only add a
// helpful line, never block or clutter the output.
func printVersion() {
	fmt.Printf("zt version %s\n", version)

	if os.Getenv("ZT_NO_UPDATE_CHECK") != "" {
		return
	}
	info, err := release.CheckLatest(version)
	if err != nil || info == nil || !info.Available {
		return
	}
	warnFn := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("  %s a newer version is available: %s (you have %s)\n", warnFn("!"), info.Latest, info.Current)
	fmt.Printf("     upgrade: %s\n", info.URL)
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
	// Shell tab-completion invokes the binary as "zt __complete down d" (or
	// __completeNoDesc) — cobra's own hidden completion commands, not user
	// input. args[2] there is often the partial word being completed,
	// which can collide with tunnelNameFirstCmds (e.g. completing "zt
	// __complete down ''" has second == "down"), and swapping it out from
	// position 1 breaks the completion protocol entirely: shells stop
	// offering suggestions with no visible error. Bail out before any of
	// that logic runs.
	if args[1] == "__complete" || args[1] == "__completeNoDesc" {
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
	if name == "help" || name == "completion" || name == "--help" || name == "-h" ||
		name == "__complete" || name == "__completeNoDesc" {
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
