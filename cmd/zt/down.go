package main

import (
	"fmt"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/cloudflare"
	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Tear down a tunnel and remove all Cloudflare resources",
	Args:  cobra.ExactArgs(1),
	RunE:  runDown,
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

	tunnel, exists := store.Get(name)
	if !exists {
		return fmt.Errorf("tunnel %q not found in local state", name)
	}

	cf := cloudflare.NewClient(cfg.APIToken, cfg.AccountID)
	ok := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold)

	bold.Printf("\n⚡ Tearing down %s\n\n", name)

	step := func(msg string) { fmt.Printf("  → %s\n", msg) }

	// 1. Stop cloudflared process
	step("Stopping cloudflared")
	if tunnel.PID > 0 {
		if err := cloudflared.Stop(tunnel.PID); err != nil {
			fmt.Printf("     %s could not stop process: %v\n", warn("!"), err)
		} else {
			fmt.Printf("     %s stopped pid %d\n", ok("✓"), tunnel.PID)
		}
	} else {
		fmt.Printf("     %s no PID recorded, skipping\n", warn("!"))
	}

	// 2. Delete cloudflared local config
	step("Removing local config files")
	if err := cloudflared.CleanTunnelFiles(name); err != nil {
		fmt.Printf("     %s %v\n", warn("!"), err)
	} else {
		fmt.Printf("     %s done\n", ok("✓"))
	}

	// 3. Delete DNS record
	step("Deleting DNS record")
	zoneID, err := cf.GetZoneID(cfg.Domain)
	if err != nil {
		fmt.Printf("     %s could not resolve zone: %v\n", warn("!"), err)
	} else {
		rec, err := cf.FindDNSRecord(zoneID, tunnel.Hostname)
		if err != nil || rec == nil {
			fmt.Printf("     %s DNS record not found, skipping\n", warn("!"))
		} else {
			if err := cf.DeleteDNSRecord(zoneID, rec.ID); err != nil {
				fmt.Printf("     %s %v\n", warn("!"), err)
			} else {
				fmt.Printf("     %s deleted\n", ok("✓"))
			}
		}
	}

	// 4. Delete tunnel
	step("Deleting Cloudflare tunnel")
	if err := cf.DeleteTunnel(tunnel.TunnelID); err != nil {
		fmt.Printf("     %s %v\n", warn("!"), err)
	} else {
		fmt.Printf("     %s tunnel %s deleted\n", ok("✓"), tunnel.TunnelID)
	}

	// 5. Remove from state
	store.Delete(name)
	if err := store.Save(); err != nil {
		return fmt.Errorf("state save failed: %w", err)
	}

	fmt.Println()
	bold.Printf("  ✓ %s torn down\n\n", name)
	return nil
}
