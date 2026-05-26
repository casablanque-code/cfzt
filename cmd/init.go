package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/casablanque-code/zt/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure Cloudflare credentials",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	bold.Println("⚡ zt init — Cloudflare Zero Trust setup")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("  API Token (Cloudflare dashboard → My Profile → API Tokens): ")
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
	green.Printf("  ✓ Config saved to %s\n", config.ConfigFilePath())
	fmt.Println()
	fmt.Println("  Next: zt up <service_name> <port>")
	fmt.Println("  Example: zt up grafana 3000")
	return nil
}
