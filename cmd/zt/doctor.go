package main

import (
	"fmt"
	"net"
	"strconv"
	"time"
	"os"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/cloudflare"
	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/service"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system and tunnel health",
	RunE:  runDoctor,
}

var (
	pass    = color.New(color.FgGreen).SprintFunc()
	fail    = color.New(color.FgRed).SprintFunc()
	warn    = color.New(color.FgYellow).SprintFunc()
	dim     = color.New(color.FgHiBlack).SprintFunc()
	boldFmt = color.New(color.Bold).SprintFunc()
)

func check(label string, err error, hint string) bool {
	if err == nil {
		fmt.Printf("  %s  %s\n", pass("✓"), label)
		return true
	}
	fmt.Printf("  %s  %s\n", fail("✗"), label)
	fmt.Printf("     %s %v\n", dim("→"), err)
	if hint != "" {
		fmt.Printf("     %s %s\n", dim("hint:"), hint)
	}
	return false
}

func checkWarn(label string, err error, hint string) {
	if err == nil {
		fmt.Printf("  %s  %s\n", pass("✓"), label)
		return
	}
	fmt.Printf("  %s  %s\n", warn("!"), label)
	fmt.Printf("     %s %v\n", dim("→"), err)
	if hint != "" {
		fmt.Printf("     %s %s\n", dim("hint:"), hint)
	}
}

func checkCloudflared() (string, error) {
	ver, err := cloudflared.GetVersion()
	if err != nil {
		return "", err
	}
	return ver.Raw, nil
}

func checkPort(port int) error {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("nothing listening on localhost:%d", port)
	}
	_ = conn.Close()
	return nil
}

func checkDNS(hostname string) error {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s", hostname)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("no addresses returned for %s", hostname)
	}
	return nil
}

func runDoctor(cmd *cobra.Command, args []string) error {
	problems := 0

	fmt.Println()
	fmt.Printf("  %s\n", boldFmt("System"))
	fmt.Println()

	// 1. cloudflared installed + version
	version, err := checkCloudflared()
	if !check("cloudflared installed", err,
		"https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/") {
		problems++
	} else {
		fmt.Printf("     %s %s\n", dim("version:"), dim(version))
		if ver, verErr := cloudflared.GetVersion(); verErr == nil && ver.TooOld() {
			fmt.Printf("  %s  cloudflared version %s is too old (minimum: %d.x)\n",
				warn("!"), ver, cloudflared.MinYear())
			fmt.Printf("     %s upgrade: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/\n",
				dim("hint:"))
			problems++
		}
	}

	// linger check
	if !service.LingerEnabled() {
	    fmt.Printf("  %s  linger not enabled — tunnels may not start after reboot\n", warn("!"))
   	 fmt.Printf("     %s run: loginctl enable-linger %s\n", dim("hint:"), os.Getenv("USER"))
   	 problems++
	} else {
   	 fmt.Printf("  %s  systemd linger enabled\n", pass("✓"))
	}

	// 2. Config exists
	fmt.Println()
	fmt.Printf("  %s\n", boldFmt("Cloudflare"))
	fmt.Println()

	cfg, err := config.Load()
	if !check("config file found", err, "run `zt init` to create it") {
		problems++
		fmt.Println()
		printSummary(problems)
		return nil
	}

	// 3. Token valid
	cf := cloudflare.NewClient(cfg.APIToken, cfg.AccountID)
	if !check("API token valid", cf.VerifyToken(),
		"Cloudflare → My Profile → API Tokens — check permissions") {
		problems++
	}

	// 4. Zone exists
	_, zoneErr := cf.GetZoneID(cfg.Domain)
	if !check(fmt.Sprintf("domain %s found in Cloudflare", cfg.Domain), zoneErr,
		"make sure the domain is added to your Cloudflare account") {
		problems++
	}

	// 5. Tunnels
	store, err := state.LoadStore()
	if err != nil {
		return err
	}

	tunnels := store.All()
	if len(tunnels) == 0 {
		fmt.Println()
		fmt.Printf("  %s\n", boldFmt("Tunnels"))
		fmt.Println()
		fmt.Printf("  %s  no tunnels configured\n", dim("–"))
	} else {
		for _, t := range tunnels {
			fmt.Println()
			fmt.Printf("  %s\n", boldFmt("Tunnel: "+t.Name))
			fmt.Println()

			// process / systemd
			var procErr error
			if service.IsInstalled(t.Name) {
				if !service.IsActive(t.Name) {
					procErr = fmt.Errorf("systemd service zt-%s is not active", t.Name)
				}
				label := fmt.Sprintf("systemd service zt-%s.service active", t.Name)
				if !check(label, procErr,
					fmt.Sprintf("run: systemctl --user start zt-%s", t.Name)) {
					problems++
				}
			} else if t.PID > 0 {
				if !cloudflared.IsRunning(t.PID) {
					procErr = fmt.Errorf("cloudflared process (pid %d) is not running", t.PID)
				}
				if !check(fmt.Sprintf("cloudflared running (pid %d)", t.PID), procErr,
					fmt.Sprintf("run: zt down %s && zt up %s %d", t.Name, t.Name, t.Port)) {
					problems++
				}
			} else {
				check("cloudflared process", fmt.Errorf("no service or PID found"),
					fmt.Sprintf("run: zt down %s && zt up %s %d", t.Name, t.Name, t.Port))
				problems++
			}

			// local port
			portErr := checkPort(t.Port)
			checkWarn(
				fmt.Sprintf("local service on port %d reachable", t.Port),
				portErr,
				fmt.Sprintf("is the service running? check `curl http://localhost:%d`", t.Port),
			)

			// DNS
			dnsErr := checkDNS(t.Hostname)
			if !check(fmt.Sprintf("DNS resolves %s", t.Hostname), dnsErr,
				"may take a few minutes after `zt up`; check Cloudflare DNS dashboard") {
				problems++
			}

			// CF tunnel exists
			cfTunnelErr := func() error {
				id, err := cf.FindTunnelByName(t.Name)
				if err != nil {
					return err
				}
				if id == "" {
					return fmt.Errorf("tunnel %q not found in Cloudflare account", t.Name)
				}
				if id != t.TunnelID {
					return fmt.Errorf("tunnel ID mismatch: local=%s CF=%s", t.TunnelID, id)
				}
				return nil
			}()
			if !check("Cloudflare tunnel exists", cfTunnelErr,
				fmt.Sprintf("run: zt down %s && zt up %s %d", t.Name, t.Name, t.Port)) {
				problems++
			}

			// port is warn-only — local service might be intentionally down
			_ = portErr
		}
	}

	fmt.Println()
	printSummary(problems)
	return nil
}

func printSummary(problems int) {
	if problems == 0 {
		fmt.Printf("  %s all checks passed\n\n", boldFmt(pass("✓")))
	} else {
		fmt.Printf("  %s %s found\n\n", boldFmt(fail("✗")), pluralize(problems, "problem"))
	}
}

func pluralize(n int, word string) string {
	if n == 1 {
		return strconv.Itoa(n) + " " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}
