package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/casablanque-code/zt/internal/cloudflared"
	"github.com/casablanque-code/zt/internal/state"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all managed tunnels",
	RunE:    runList,
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show details for a specific tunnel",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func runList(cmd *cobra.Command, args []string) error {
	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	tunnels := store.All()
	if len(tunnels) == 0 {
		fmt.Println("  no tunnels — run `zt up <name> <port>` to create one")
		return nil
	}

	sort.Slice(tunnels, func(i, j int) bool {
		return tunnels[i].Name < tunnels[j].Name
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "URL", "PORT", "STATUS", "PID"})
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	for _, t := range tunnels {
		status := string(t.Status)
		// Re-check live process status
		if t.PID > 0 {
			if cloudflared.IsRunning(t.PID) {
				status = green("running")
			} else {
				status = red("stopped")
			}
		} else {
			status = yellow("unknown")
		}

		pid := "-"
		if t.PID > 0 {
			pid = fmt.Sprintf("%d", t.PID)
		}

		table.Append([]string{
			t.Name,
			"https://" + t.Hostname,
			fmt.Sprintf("%d", t.Port),
			status,
			pid,
		})
	}

	fmt.Println()
	table.Render()
	fmt.Println()
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	t, exists := store.Get(name)
	if !exists {
		return fmt.Errorf("tunnel %q not found", name)
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	alive := cloudflared.IsRunning(t.PID)
	statusStr := red("stopped")
	if alive {
		statusStr = green("running")
	}

	fmt.Println()
	bold.Printf("  %s\n", t.Name)
	fmt.Printf("  URL:        https://%s\n", t.Hostname)
	fmt.Printf("  Port:       %d\n", t.Port)
	fmt.Printf("  Tunnel ID:  %s\n", t.TunnelID)
	fmt.Printf("  PID:        %d\n", t.PID)
	fmt.Printf("  Status:     %s\n", statusStr)
	fmt.Printf("  Created:    %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))

	cfgPath, _ := cloudflared.ConfigPath(t.Name)
	logPath := cfgPath[:len(cfgPath)-len("config.yml")] + "cloudflared.log"
	fmt.Printf("  Log:        %s\n", logPath)
	fmt.Println()

	return nil
}
