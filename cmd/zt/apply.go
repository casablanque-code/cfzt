package main

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/docker"
	"github.com/casablanque-code/cfzt/internal/manifest"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply <file>",
	Short: "Apply a zt.yaml manifest — create any services not yet running",
	Long: `Reads a zt.yaml manifest and brings up every service listed in it
that is not already managed locally. Services that already exist in the
local state are skipped with a notice. Services that exist locally but
are absent from the manifest are left untouched and reported — remove
them manually with 'zt down <name>' if needed.

'zt apply' never deletes or modifies existing tunnels automatically.`,
	Example: `  zt apply zt.yaml
  zt apply ~/backups/home-server.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	manifestPath := args[0]

	boldFmt := color.New(color.Bold).SprintFunc()
	okFn := color.New(color.FgGreen).SprintFunc()
	warnFn := color.New(color.FgYellow).SprintFunc()
	dimFn := color.New(color.FgHiBlack).SprintFunc()

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return err
	}
	if len(m.Services) == 0 {
		fmt.Println("  manifest is empty — nothing to apply")
		return nil
	}

	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	// Sort service names for deterministic output
	names := make([]string, 0, len(m.Services))
	for name := range m.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	var toCreate []string
	var skipped []string
	var untracked []string

	for _, name := range names {
		if _, exists := store.Get(name); exists {
			skipped = append(skipped, name)
		} else {
			toCreate = append(toCreate, name)
		}
	}

	// Report services in local state that are absent from the manifest
	for _, t := range store.All() {
		if _, inManifest := m.Services[t.Name]; !inManifest {
			untracked = append(untracked, t.Name)
		}
	}
	sort.Strings(untracked)

	fmt.Printf("\n%s\n\n", boldFmt(fmt.Sprintf("⚡ Applying %s", manifestPath)))
	fmt.Printf("  %s to create: %d   skipped: %d   untracked: %d\n\n",
		dimFn("plan:"), len(toCreate), len(skipped), len(untracked))

	if len(skipped) > 0 {
		for _, name := range skipped {
			fmt.Printf("  %s %-20s already exists — skipping\n", warnFn("~"), name)
		}
		fmt.Println()
	}

	if len(untracked) > 0 {
		for _, name := range untracked {
			fmt.Printf("  %s %-20s exists locally but not in manifest — run `zt down %s` to remove\n",
				dimFn("?"), name, name)
		}
		fmt.Println()
	}

	if len(toCreate) == 0 {
		fmt.Printf("  %s all services already running — nothing to do\n\n", okFn("✓"))
		return nil
	}

	// Verify cloudflared version once before the loop, not once per service
	ver, err := cloudflared.GetVersion()
	if err != nil {
		return err
	}
	if ver.TooOld() {
		return fmt.Errorf("cloudflared %s is too old (minimum: %d.x) — upgrade before applying",
			ver, cloudflared.MinYear())
	}

	var failed []string
	for _, name := range toCreate {
		svc := m.Services[name]
		port, err := resolveApplyPort(name, svc)
		if err != nil {
			fmt.Printf("  %s %-20s skipping: %v\n", warnFn("!"), name, err)
			failed = append(failed, name)
			continue
		}

		protocol := svc.Protocol
		if protocol == "" {
			protocol = "auto"
		}

		if err := createTunnel(tunnelOpts{
			name:     name,
			port:     port,
			protocol: protocol,
			public:   svc.Public,
			emails:   svc.Allow,
			docker:   svc.Docker,
		}); err != nil {
			fmt.Printf("  %s %-20s failed: %v\n", warnFn("!"), name, err)
			failed = append(failed, name)
		}
	}

	fmt.Println()
	if len(failed) > 0 {
		fmt.Printf("  %s %d service(s) failed — check output above\n\n", warnFn("!"), len(failed))
		return fmt.Errorf("%d service(s) failed to apply", len(failed))
	}

	created := len(toCreate)
	fmt.Printf("  %s\n\n", boldFmt(fmt.Sprintf("✅ Done — %d service(s) created", created)))
	return nil
}

// resolveApplyPort returns the port string for a service from the manifest.
// For docker services it auto-detects from the running container.
func resolveApplyPort(name string, svc manifest.ServiceSpec) (string, error) {
	if svc.Docker {
		detected, err := docker.FindContainerPort(name)
		if err != nil {
			return "", fmt.Errorf("docker port detection failed: %w", err)
		}
		if svc.Port != 0 {
			// explicit port in manifest overrides docker detection
			return strconv.Itoa(svc.Port), nil
		}
		return detected, nil
	}
	return strconv.Itoa(svc.Port), nil
}
