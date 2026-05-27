package main

import (
	"fmt"

	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/service"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a tunnel's cloudflared process",
	Args:  cobra.ExactArgs(1),
	RunE:  runRestart,
}

func runRestart(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	t, exists := store.Get(name)
	if !exists {
		return fmt.Errorf("tunnel %q not found — run `zt list` to see active tunnels", name)
	}

	okFn := color.New(color.FgGreen).SprintFunc()
	boldFmt := color.New(color.Bold).SprintFunc()

	fmt.Printf("\n%s\n\n", boldFmt("⚡ Restarting "+name))

	if service.IsInstalled(name) {
		fmt.Printf("  → systemctl --user restart zt-%s\n", name)
		if err := service.Restart(name); err != nil {
			return err
		}
		fmt.Printf("     %s restarted\n", okFn("✓"))
	} else if t.PID > 0 {
		cfgPath, err := cloudflared.ConfigPath(name)
		if err != nil {
			return err
		}
		fmt.Printf("  → stopping pid %d\n", t.PID)
		_ = cloudflared.Stop(t.PID)

		fmt.Printf("  → starting cloudflared\n")
		pid, err := cloudflared.Start(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to restart cloudflared: %w", err)
		}
		t.PID = pid
		store.Set(t)
		if err := store.Save(); err != nil {
			return fmt.Errorf("state save failed: %w", err)
		}
		fmt.Printf("     %s restarted (pid %d)\n", okFn("✓"), pid)
	} else {
		return fmt.Errorf("tunnel %q has no running process or service — run `zt down %s && zt up %s <port>`",
			name, name, name)
	}

	fmt.Println()
	return nil
}
