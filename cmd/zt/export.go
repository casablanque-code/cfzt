package main

import (
	"fmt"
	"sort"

	"github.com/casablanque-code/cfzt/internal/manifest"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var flagExportOut string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export managed tunnels to a portable zt.yaml manifest",
	Long: `Snapshots everything zt currently manages locally (~/.zt-state.json)
into a zt.yaml manifest. The manifest does NOT include credentials or
tunnel IDs — only the intent needed to recreate the same services
elsewhere via 'zt apply'. Commit it to git, copy it to another machine,
run 'zt init' there, then 'zt apply zt.yaml'.`,
	Example: `  zt export
  zt export -o backup/zt.yaml`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&flagExportOut, "output", "o", "zt.yaml", "path to write the manifest to")
}

func runExport(cmd *cobra.Command, args []string) error {
	boldFmt := color.New(color.Bold).SprintFunc()
	okFn := color.New(color.FgGreen).SprintFunc()
	dimFn := color.New(color.FgHiBlack).SprintFunc()

	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	tunnels := store.All()
	if len(tunnels) == 0 {
		fmt.Println("  nothing to export — no tunnels are currently managed by zt")
		return nil
	}

	sort.Slice(tunnels, func(i, j int) bool { return tunnels[i].Name < tunnels[j].Name })

	m := &manifest.Manifest{Services: make(map[string]manifest.ServiceSpec)}
	for _, t := range tunnels {
		m.Services[t.Name] = manifest.ServiceSpec{
			Port:     t.Port,
			Docker:   t.DockerDetect,
			Protocol: protocolForExport(t.Protocol),
			Public:   t.Public,
			Allow:    t.AllowEmails,
		}
	}

	if err := manifest.Save(flagExportOut, m); err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", boldFmt(fmt.Sprintf("⚡ Exported %d service(s) → %s", len(tunnels), flagExportOut)))
	for _, t := range tunnels {
		fmt.Printf("  %s %s\n", okFn("✓"), t.Name)
	}
	fmt.Println()
	fmt.Printf("  %s credentials are not included — run `zt init` on the target machine first\n", dimFn("note:"))
	fmt.Printf("  %s then: zt apply %s\n", dimFn("note:"), flagExportOut)
	fmt.Println()
	return nil
}

// protocolForExport omits the "auto" default to keep the manifest clean —
// ServiceSpec.Protocol is empty unless the user explicitly chose quic/http2.
func protocolForExport(p state.Protocol) string {
	if p == "" || p == state.ProtocolAuto {
		return ""
	}
	return string(p)
}
