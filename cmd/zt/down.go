package main

import (
	"fmt"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/cloudflare"
	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/service"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var downFlagRemote bool

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Tear down a tunnel and remove all Cloudflare resources",
	Args:  cobra.ExactArgs(1),
	RunE:  runDown,
}

func init() {
	downCmd.Flags().BoolVar(&downFlagRemote, "remote", false,
		"resolve the tunnel directly from Cloudflare by name if it's not in local state — e.g. tearing down from a different machine than \"zt up\" ran on")
}

func runDown(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	cf := newCFClient(cfg.APIToken, cfg.AccountID)
	okFn := color.New(color.FgGreen).SprintFunc()
	warnFn := color.New(color.FgYellow).SprintFunc()
	boldFmt := color.New(color.Bold).SprintFunc()

	tunnel, exists := store.Get(name)
	if !exists {
		if !downFlagRemote {
			return fmt.Errorf("tunnel %q not found in local state\n  if this is a different machine than the one \"zt up %s\" ran on (e.g. CI), pass --remote to resolve it from Cloudflare by name instead", name, name)
		}
		fmt.Printf("  %s %q isn't in local state — resolving it from Cloudflare directly (--remote)\n", warnFn("!"), name)
		resolved, err := resolveTunnelRemote(cf, cfg, name)
		if err != nil {
			return err
		}
		tunnel = resolved
	}

	fmt.Printf("\n%s\n\n", boldFmt("⚡ Tearing down "+name))

	step := func(msg string) { fmt.Printf("  → %s\n", msg) }

	// 1. Stop and remove system service (systemd / launchd)
	step("Removing system service")
	if service.IsInstalled(name) {
		if err := service.Uninstall(name); err != nil {
			fmt.Printf("     %s %v\n", warnFn("!"), err)
		} else {
			fmt.Printf("     %s %s service removed\n", okFn("✓"), service.ManagerName())
		}
	} else {
		// fall back to killing direct process if service wasn't installed
		if tunnel.PID > 0 {
			if err := cloudflared.Stop(tunnel.PID); err != nil {
				fmt.Printf("     %s could not stop pid %d: %v\n", warnFn("!"), tunnel.PID, err)
			} else {
				fmt.Printf("     %s stopped pid %d\n", okFn("✓"), tunnel.PID)
			}
		} else {
			fmt.Printf("     %s no service or PID found, skipping\n", warnFn("!"))
		}
	}

	// 2. Remove local config files
	step("Removing local config files")
	if err := cloudflared.CleanTunnelFiles(name); err != nil {
		fmt.Printf("     %s %v\n", warnFn("!"), err)
	} else {
		fmt.Printf("     %s done\n", okFn("✓"))
	}

	// 3. Delete DNS record
	step("Deleting DNS record")
	zoneID, err := cf.GetZoneID(cfg.Domain)
	if err != nil {
		fmt.Printf("     %s could not resolve zone: %v\n", warnFn("!"), err)
	} else {
		rec, err := cf.FindDNSRecord(zoneID, tunnel.Hostname)
		if err != nil || rec == nil {
			fmt.Printf("     %s DNS record not found, skipping\n", warnFn("!"))
		} else {
			if err := cf.DeleteDNSRecord(zoneID, rec.ID); err != nil {
				fmt.Printf("     %s %v\n", warnFn("!"), err)
			} else {
				fmt.Printf("     %s deleted\n", okFn("✓"))
			}
		}
	}

	// 4. Delete Access app
	step("Removing Zero Trust Access app")
	if existing, err := cf.FindAccessAppByDomain(tunnel.Hostname); err == nil && existing != nil {
		if err := cf.DeleteAccessApp(existing.ID); err != nil {
			fmt.Printf("     %s %v\n", warnFn("!"), err)
		} else {
			fmt.Printf("     %s deleted\n", okFn("✓"))
		}
	} else {
		fmt.Printf("     %s not found, skipping\n", warnFn("!"))
	}

	// 5. Delete Cloudflare tunnel
	step("Deleting Cloudflare tunnel")
	if err := cf.DeleteTunnel(tunnel.TunnelID); err != nil {
		fmt.Printf("     %s %v\n", warnFn("!"), err)
	} else {
		fmt.Printf("     %s tunnel %s deleted\n", okFn("✓"), tunnel.TunnelID)
	}

	// 6. Remove from state
	store.Delete(name)
	if err := store.Save(); err != nil {
		return fmt.Errorf("state save failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s\n\n", boldFmt("✓ "+name+" torn down"))
	return nil
}

// resolveTunnelRemote reconstructs enough of a state.Tunnel to tear down
// when name isn't in local state, by resolving it against Cloudflare
// directly instead. Only TunnelID and Hostname get filled in — the other
// steps in runDown either look things up independently by hostname
// already (DNS record, Access app) or are inherently local-machine-only
// and correctly no-op when there's nothing here to clean up (the
// service/PID removal step, local config files).
//
// Gated behind --remote rather than an automatic fallback: Cloudflare
// Tunnels have no "created by zt" marker the way a zt-created DNS CNAME
// does (looksLikeZtRecord), so resolving by name alone means whatever
// tunnel is on the account with that exact name gets deleted — safe
// when the caller knows that's the one they mean (a CI job tearing down
// its own PR preview), not safe to do implicitly on every plain `zt down`
// typo.
func resolveTunnelRemote(cf *cloudflare.Client, cfg *config.Config, name string) (*state.Tunnel, error) {
	tunnelID, err := cf.FindTunnelByName(name)
	if err != nil {
		return nil, fmt.Errorf("looking up tunnel %q on Cloudflare: %w", name, err)
	}
	if tunnelID == "" {
		return nil, fmt.Errorf("tunnel %q not found in local state or on Cloudflare — nothing to tear down", name)
	}
	return &state.Tunnel{
		Name:     name,
		TunnelID: tunnelID,
		Hostname: name + "." + cfg.Domain,
	}, nil
}
