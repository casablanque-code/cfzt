package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/service"
	"github.com/casablanque-code/cfzt/internal/state"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var flagStatusLogs bool
var flagLogsN int
var flagLogsFollow bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all managed tunnels",
	RunE:    runList,
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show tunnel details",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

var logsCmd = &cobra.Command{
	Use:     "logs <name>",
	Aliases: []string{"log"},
	Short:   "Show cloudflared logs for a tunnel",
	Example: `  zt logs grafana
  zt logs grafana -n 100
  zt logs grafana -f`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

func init() {
	statusCmd.Flags().BoolVar(&flagStatusLogs, "logs", false, "show recent log output")
	logsCmd.Flags().IntVarP(&flagLogsN, "lines", "n", 50, "number of lines to show")
	logsCmd.Flags().BoolVarP(&flagLogsFollow, "follow", "f", false, "follow log output")
}

// log line colorizers
var (
	reError = regexp.MustCompile(`(?i)\b(error|err|fatal|fail|failed|failure)\b`)
	reWarn  = regexp.MustCompile(`(?i)\b(warn|warning)\b`)
	reOK    = regexp.MustCompile(`(?i)\b(registered|connected|proxying|success|started|ready)\b`)

	colorError = color.New(color.FgRed).SprintFunc()
	colorWarn  = color.New(color.FgYellow).SprintFunc()
	colorOK    = color.New(color.FgGreen).SprintFunc()
	colorDim   = color.New(color.FgHiBlack).SprintFunc()
	colorBold  = color.New(color.Bold).SprintFunc()
)

func colorizeLine(line string) string {
	switch {
	case reError.MatchString(line):
		return colorError(line)
	case reWarn.MatchString(line):
		return colorWarn(line)
	case reOK.MatchString(line):
		return colorOK(line)
	default:
		// dim the timestamp prefix, keep the rest normal
		if len(line) > 20 {
			return colorDim(line[:20]) + line[20:]
		}
		return line
	}
}

func logPath(name string) (string, error) {
	cfgPath, err := cloudflared.ConfigPath(name)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(cfgPath, "config.yml") + "cloudflared.log", nil
}

// tailFile returns the last n lines of a file.
func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// followFile tails a file and streams new lines until interrupted.
func followFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// seek to end
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(colorizeLine(strings.TrimRight(line, "\n")) + "\n")
		}
		if err != nil {
			if err == io.EOF {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return err
		}
	}
}

func printLogs(name string, n int) error {
	path, err := logPath(name)
	if err != nil {
		return err
	}

	lines, err := tailFile(path, n)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(colorDim("  no log file found — tunnel may not have started yet"))
			return nil
		}
		return err
	}

	if len(lines) == 0 {
		fmt.Println(colorDim("  log is empty"))
		return nil
	}

	fmt.Printf("\n  %s  %s\n\n", colorBold("cloudflared log"), colorDim(path))
	for _, l := range lines {
		fmt.Println("  " + colorizeLine(l))
	}
	fmt.Println()
	return nil
}

func runLogs(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := state.LoadStore()
	if err != nil {
		return err
	}
	if _, exists := store.Get(name); !exists {
		return fmt.Errorf("tunnel %q not found", name)
	}

	if flagLogsFollow {
		path, err := logPath(name)
		if err != nil {
			return err
		}
		fmt.Printf("  %s %s\n\n", colorBold("following"), colorDim(path))
		return followFile(path)
	}

	return printLogs(name, flagLogsN)
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
		if t.PID > 0 {
			if service.IsActive(t.Name) || (t.PID > 0 && cloudflared.IsRunning(t.PID)) {
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

	alive := service.IsActive(t.Name) || (t.PID > 0 && cloudflared.IsRunning(t.PID))
	statusStr := red("stopped")
	if alive {
		statusStr = green("running")
	}

	path, _ := logPath(t.Name)

	fmt.Println()
	bold.Printf("  %s\n", t.Name)
	fmt.Printf("  URL:        https://%s\n", t.Hostname)
	fmt.Printf("  Port:       %d\n", t.Port)
	fmt.Printf("  Tunnel ID:  %s\n", t.TunnelID)
	fmt.Printf("  PID:        %d\n", t.PID)
	fmt.Printf("  Status:     %s\n", statusStr)
	fmt.Printf("  Created:    %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Log:        %s\n", path)
	fmt.Println()

	if flagStatusLogs {
		return printLogs(name, 30)
	}
	return nil
}
