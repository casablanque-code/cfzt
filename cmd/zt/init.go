package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/casablanque-code/cfzt/config"
	"github.com/casablanque-code/cfzt/internal/cloudflare"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure Cloudflare credentials",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s\n\n", bold("⚡ zt init — Cloudflare Zero Trust setup"))

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("  API Token (Cloudflare → My Profile → API Tokens): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	fmt.Print("  Account ID (right sidebar on any CF dashboard page): ")
	accountID, _ := reader.ReadString('\n')
	accountID = strings.TrimSpace(accountID)

	fmt.Print("  Domain (e.g. example.com — must be on Cloudflare): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)

	if token == "" || accountID == "" || domain == "" {
		return fmt.Errorf("all fields are required")
	}

	cfg := &config.Config{
		APIToken:  token,
		AccountID: accountID,
		Domain:    domain,
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  → Verifying API token... ")
	cf := cloudflare.NewClient(token, accountID)
	if err := cf.VerifyToken(); err != nil {
		fmt.Println()
		fmt.Printf("  %s Token verification failed: %v\n", warn("!"), err)
		fmt.Printf("  %s Config saved, but check your token before running zt up\n", warn("!"))
		fmt.Println()
		fmt.Println("  Required token permissions:")
		fmt.Println("    Account / Cloudflare Tunnel / Edit")
		fmt.Println("    Zone / DNS / Edit")
		fmt.Println("    Account / Access: Apps and Policies / Edit")
		return nil
	}
	fmt.Printf("%s\n", green("✓"))

	fmt.Printf("  → Verifying domain %s... ", domain)
	if err := cf.VerifyZone(domain); err != nil {
		fmt.Println()
		fmt.Printf("  %s Domain check failed: %v\n", warn("!"), err)
		fmt.Printf("  %s Config saved, but make sure the domain is added to Cloudflare\n", warn("!"))
		return nil
	}
	fmt.Printf("%s\n", green("✓"))

	fmt.Println()
	fmt.Printf("  %s Config saved to %s\n", green("✓"), config.ConfigFilePath())
	fmt.Println()
	fmt.Println("  Next: zt up <service_name> <port>")
	fmt.Println("  Example: zt up portainer --docker --allow you@example.com")
	return nil
}
