package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/docker"
	"github.com/casablanque-code/cfzt/internal/integrity"
	"github.com/casablanque-code/cfzt/internal/manifest"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var flagApplyForce bool

var applyCmd = &cobra.Command{
	Use:   "apply <file>",
	Short: "Apply a zt.yaml manifest — create any services not yet running",
	Long: `Reads a zt.yaml manifest and brings up every service listed in it
that is not already managed locally. Services that already exist in the
local state are skipped with a notice. Services that exist locally but
are absent from the manifest are left untouched and reported — remove
them manually with 'zt down <name>' if needed.

'zt apply' never deletes or modifies existing tunnels automatically.

If zt.yaml changed since it was last applied, 'zt apply' shows the diff
and refuses to proceed until you re-run with --force — this catches an
unexpected or tampered-with manifest before it's used to create tunnels.`,
	Example: `  zt apply zt.yaml
  zt apply ~/backups/home-server.yaml
  zt apply zt.yaml --force   # accept a changed manifest and proceed`,
	Args: cobra.ExactArgs(1),
	RunE: runApply,
}

func init() {
	applyCmd.Flags().BoolVar(&flagApplyForce, "force", false, "accept a zt.yaml that changed since it was last applied")
}

func runApply(cmd *cobra.Command, args []string) error {
	manifestPath := args[0]

	boldFmt := color.New(color.Bold).SprintFunc()
	okFn := color.New(color.FgGreen).SprintFunc()
	warnFn := color.New(color.FgYellow).SprintFunc()
	dimFn := color.New(color.FgHiBlack).SprintFunc()

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest %s: %w", manifestPath, err)
	}
	absPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return err
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return err
	}
	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	currentHash := integrity.Hash(raw)
	prevSnapshot, known := store.ManifestSnapshot(absPath)
	if known && prevSnapshot.Hash != currentHash {
		fmt.Printf("\n%s\n\n", warnFn(fmt.Sprintf("⚠ %s changed since it was last applied", manifestPath)))
		for _, line := range integrity.Diff(prevSnapshot.Content, string(raw)) {
			if len(line) > 0 && line[0] == '-' {
				fmt.Printf("  %s\n", color.New(color.FgRed).Sprint(line))
			} else {
				fmt.Printf("  %s\n", color.New(color.FgGreen).Sprint(line))
			}
		}
		fmt.Println()
		if !flagApplyForce {
			return fmt.Errorf("manifest changed since last apply — re-run with --force to accept these changes and proceed")
		}
		fmt.Printf("  %s --force: accepting the changed manifest\n\n", warnFn("!"))
	}

	if len(m.Services) == 0 {
		fmt.Println("  manifest is empty — nothing to apply")
		return saveManifestSnapshot(store, absPath, currentHash, raw)
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
		return saveManifestSnapshot(store, absPath, currentHash, raw)
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
			name:          name,
			port:          port,
			protocol:      protocol,
			public:        svc.Public,
			emails:        svc.Allow,
			docker:        svc.Docker,
			containerPort: svc.ContainerPort,
			force:         svc.Force,
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
	return saveManifestSnapshot(store, absPath, currentHash, raw)
}

// saveManifestSnapshot records the manifest that was just successfully
// applied, so the next 'zt apply' can detect if it changes on disk before
// then. A failed apply intentionally does not reach this — the stale
// snapshot (or absence of one) is preserved so the next run still flags it.
func saveManifestSnapshot(store *state.Store, absPath, hash string, raw []byte) error {
	store.SetManifestSnapshot(absPath, state.ManifestSnapshot{
		Hash:      hash,
		Content:   string(raw),
		UpdatedAt: time.Now(),
	})
	return store.Save()
}

// resolveApplyPort returns the port string for a service from the manifest.
// For docker services it auto-detects from the running container, unless
// an explicit port is also set in the manifest, in which case that takes
// priority and Docker is never queried at all.
func resolveApplyPort(name string, svc manifest.ServiceSpec) (string, error) {
	if svc.Docker {
		if svc.Port != 0 {
			return strconv.Itoa(svc.Port), nil
		}
		detected, err := docker.FindContainerPort(name, svc.ContainerPort)
		if err != nil {
			return "", fmt.Errorf("docker port detection failed: %w", err)
		}
		return detected, nil
	}
	return strconv.Itoa(svc.Port), nil
}
