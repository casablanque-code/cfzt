package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/cloudflare"
	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/docker"
	"github.com/casablanque-code/cfzt/internal/service"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	flagPublic   bool
	flagEmails   []string
	flagDocker   bool
	flagTCP      bool
	flagProtocol string
)

var upCmd = &cobra.Command{
	Use:   "up <name> [port]",
	Short: "Expose a local service via Zero Trust tunnel",
	Example: `  zt up grafana 3000
  zt up grafana --docker
  zt up portainer --docker --allow you@example.com
  zt up api 8080 --public
  zt up portainer 9000 --tcp`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runUp,
}

func init() {
	upCmd.Flags().BoolVar(&flagPublic, "public", false, "bypass Zero Trust (public access)")
	upCmd.Flags().StringArrayVar(&flagEmails, "allow", nil, "restrict access to specific emails (repeatable)")
	upCmd.Flags().BoolVar(&flagDocker, "docker", false, "auto-detect port from Docker container with this name")
	upCmd.Flags().BoolVar(&flagTCP, "tcp", false, "force TCP (http2) protocol — use if QUIC/UDP is blocked by your ISP")
	upCmd.Flags().StringVar(&flagProtocol, "protocol", "auto", "cloudflared protocol: auto, quic, http2")
}

func runUp(cmd *cobra.Command, args []string) error {
	name := args[0]
	var port string

	step := func(msg string) { fmt.Printf("  → %s\n", msg) }
	okFn := color.New(color.FgGreen).SprintFunc()
	warnFn := color.New(color.FgYellow).SprintFunc()
	boldFmt := color.New(color.Bold).SprintFunc()

	// Check cloudflared version before doing anything
	if ver, err := cloudflared.GetVersion(); err != nil {
		return err
	} else if ver.TooOld() {
		fmt.Printf("  %s cloudflared version %s is too old (minimum: %d.x)\n",
			warnFn("!"), ver, cloudflared.MinYear())
		fmt.Printf("     upgrade: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/\n")
		return fmt.Errorf("unsupported cloudflared version")
	}

	if flagDocker {
		step("Detecting port for Docker container: " + name)
		detected, err := docker.FindContainerPort(name)
		if err != nil {
			return err
		}
		port = detected
		if len(args) == 2 {
			port = args[1]
			fmt.Printf("     %s port override: %s (detected: %s)\n", okFn("✓"), port, detected)
		} else {
			fmt.Printf("     %s detected port: %s\n", okFn("✓"), port)
		}
	} else {
		if len(args) < 2 {
			return fmt.Errorf("port required — usage: zt up <name> <port>\n  or use --docker to auto-detect from container")
		}
		port = args[1]
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("port must be a number, got %q", port)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	store, err := state.LoadStore()
	if err != nil {
		return err
	}
	if _, exists := store.Get(name); exists {
		return fmt.Errorf("tunnel %q already exists — run `zt down %s` first", name, name)
	}

	cf := cloudflare.NewClient(cfg.APIToken, cfg.AccountID)
	hostname := name + "." + cfg.Domain

	fmt.Printf("\n%s\n\n", boldFmt(fmt.Sprintf("⚡ Bringing up %s → localhost:%s", hostname, port)))

	// 1. Resolve zone ID
	step("Resolving zone ID for " + cfg.Domain)
	zoneID, err := cf.GetZoneID(cfg.Domain)
	if err != nil {
		return err
	}
	fmt.Printf("     %s zone: %s\n", okFn("✓"), zoneID)

	// 2. Create tunnel — clean up stale CF tunnel with same name first
	step("Creating Cloudflare tunnel: " + name)
	if staleID, err := cf.FindTunnelByName(name); err == nil && staleID != "" {
		fmt.Printf("     %s found stale tunnel %s — cleaning up\n", warnFn("!"), staleID)
		_ = cf.DeleteTunnel(staleID)
	}
	tunnelID, credJSON, err := cf.CreateTunnel(name)
	if err != nil {
		return err
	}
	fmt.Printf("     %s tunnel ID: %s\n", okFn("✓"), tunnelID)

	rollback := func(dnsRecordID, accessAppID string) {
		if accessAppID != "" {
			_ = cf.DeleteAccessApp(accessAppID)
		}
		if dnsRecordID != "" {
			_ = cf.DeleteDNSRecord(zoneID, dnsRecordID)
		}
		_ = cf.DeleteTunnel(tunnelID)
		_ = cloudflared.CleanTunnelFiles(name)
		_ = service.Uninstall(name)
	}

	// 3. Configure tunnel ingress
	step("Configuring ingress rules")
	if err := cf.ConfigureTunnel(tunnelID, hostname, port); err != nil {
		rollback("", "")
		return err
	}
	fmt.Printf("     %s ingress: %s → localhost:%s\n", okFn("✓"), hostname, port)

	// 4. Upsert DNS record
	step("Upserting CNAME: " + hostname)
	dnsRecordID, err := cf.UpsertCNAME(zoneID, hostname, tunnelID)
	if err != nil {
		rollback("", "")
		return err
	}
	fmt.Printf("     %s DNS record ready\n", okFn("✓"))

	// 5. Zero Trust Access app
	var accessAppID string
	if !flagPublic {
		step("Creating Zero Trust Access application")
		accessAppID, err = cf.UpsertAccessApp(hostname, name)
		if err != nil {
			rollback(dnsRecordID, "")
			return err
		}
		step("Configuring access policy")
		if err := cf.CreateBypassPolicy(accessAppID, flagEmails); err != nil {
			rollback(dnsRecordID, accessAppID)
			return err
		}
		policyDesc := "bypass (public)"
		if len(flagEmails) > 0 {
			policyDesc = fmt.Sprintf("allow %s", strings.Join(flagEmails, ", "))
		}
		fmt.Printf("     %s access policy: %s\n", okFn("✓"), policyDesc)
	} else {
		fmt.Printf("     %s skipping ZT Access (--public flag)\n", okFn("✓"))
	}

	// 6. Write cloudflared config
	protocol := flagProtocol
	if flagTCP {
		protocol = "http2"
	}
	if protocol != "auto" && protocol != "quic" && protocol != "http2" {
		return fmt.Errorf("invalid protocol %q — use: auto, quic, http2", protocol)
	}
	step("Writing cloudflared config")
	cfgPath, err := cloudflared.WriteTunnelConfig(tunnelID, name, hostname, port, protocol, credJSON)
	if err != nil {
		rollback(dnsRecordID, accessAppID)
		return err
	}
	fmt.Printf("     %s config: %s\n", okFn("✓"), cfgPath)

	// 7. Install and start systemd/launchd service
	logPath := strings.TrimSuffix(cfgPath, "config.yml") + "cloudflared.log"
	step("Installing system service")
	if err := service.Install(name, cfgPath, logPath); err != nil {
		// service install failed — fall back to direct process
		fmt.Printf("     %s service install failed, starting directly: %v\n", warnFn("!"), err)
		pid, err2 := cloudflared.Start(cfgPath)
		if err2 != nil {
			rollback(dnsRecordID, accessAppID)
			return err2
		}
		fmt.Printf("     %s pid: %d (no auto-restart)\n", warnFn("!"), pid)
		if err2 := saveTunnel(store, name, tunnelID, hostname, port, protocol, pid); err2 != nil {
			return fmt.Errorf("state save failed: %w", err2)
		}
		fmt.Println()
		fmt.Printf("  %s\n\n", boldFmt(fmt.Sprintf("🎉 Ready: https://%s", hostname)))
		return nil
	}
	fmt.Printf("     %s service: zt-%s.service (auto-start on boot)\n", okFn("✓"), name)

	// 8. Persist state (PID=0 — managed by systemd)
	if err := saveTunnel(store, name, tunnelID, hostname, port, protocol, 0); err != nil {
		return fmt.Errorf("state save failed: %w", err)
	}
	if err := store.Save(); err != nil {
		return fmt.Errorf("state save failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s\n\n", boldFmt(fmt.Sprintf("🎉 Ready: https://%s", hostname)))
	return nil
}

func saveTunnel(store *state.Store, name, tunnelID, hostname, port, protocol string, pid int) error {
	store.Set(&state.Tunnel{
		Name:      name,
		TunnelID:  tunnelID,
		Port:      mustAtoi(port),
		Hostname:  hostname,
		Protocol:  state.Protocol(protocol),
		PID:       pid,
		Status:    state.StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	return store.Save()
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
