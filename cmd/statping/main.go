package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/template"

	"github.com/ankityadav/statping/internal/checker"
	"github.com/ankityadav/statping/internal/config"
	"github.com/ankityadav/statping/internal/notifier"
	"github.com/ankityadav/statping/internal/storage"
	"github.com/ankityadav/statping/internal/tray"
	"github.com/ankityadav/statping/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "statping",
	Short: "Website monitoring CLI with TUI",
	Long:  "A beautiful terminal-based website monitoring tool with notifications",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the monitoring service with TUI",
	Run:   runStart,
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run monitoring service in background (no TUI)",
	Run:   runDaemon,
}

var addCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Add a new monitor",
	Args:  cobra.ExactArgs(1),
	Run:   runAdd,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all monitors",
	Run:   runList,
}

var removeCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove a monitor by ID",
	Args:  cobra.ExactArgs(1),
	Run:   runRemove,
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show real-time dashboard with response time graphs",
	Run:   runDashboard,
}

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Run in system tray (persistent background monitoring)",
	Run:   runTray,
}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable auto-start on login (registers LaunchAgent)",
	Run:   runEnable,
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable auto-start on login (removes LaunchAgent)",
	Run:   runDisable,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if auto-start is enabled",
	Run:   runStatus,
}

var (
	addName          string
	addInterval      int
	addTimeout       int
	addExpectedCodes string
	addKeywords      string
)

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(trayCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(statusCmd)

	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Monitor name")
	addCmd.Flags().IntVarP(&addInterval, "interval", "i", config.DefaultCheckInterval, "Check interval in seconds")
	addCmd.Flags().IntVarP(&addTimeout, "timeout", "t", config.DefaultTimeout, "Request timeout in seconds")
	addCmd.Flags().StringVarP(&addExpectedCodes, "codes", "c", "200", "Expected status codes (comma-separated)")
	addCmd.Flags().StringVarP(&addKeywords, "keywords", "k", "", "Keywords to find in response (comma-separated)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initDatabase() (*storage.Database, error) {
	dbPath, err := config.GetDatabasePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	db, err := storage.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

func runStart(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	n := notifier.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := checker.New(db, n)
	if err := c.Start(ctx); err != nil {
		log.Fatalf("Failed to start checker: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	p := tea.NewProgram(
		tui.New(db),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}

	c.Stop()
}

func runDaemon(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	n := notifier.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := checker.New(db, n)
	if err := c.Start(ctx); err != nil {
		log.Fatalf("Failed to start checker: %v", err)
	}

	log.Println("Monitoring service started in daemon mode")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	c.Stop()
}

func runAdd(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	url := args[0]
	name := addName
	if name == "" {
		name = url
	}

	monitor := &storage.Monitor{
		Name:          name,
		URL:           url,
		CheckInterval: addInterval,
		Timeout:       addTimeout,
		ExpectedCodes: addExpectedCodes,
		Keywords:      addKeywords,
		Enabled:       true,
	}

	if err := db.CreateMonitor(monitor); err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}

	fmt.Printf("Monitor created successfully (ID: %d)\n", monitor.ID)
}

func runList(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	monitors, err := db.ListMonitors()
	if err != nil {
		log.Fatalf("Failed to list monitors: %v", err)
	}

	if len(monitors) == 0 {
		fmt.Println("No monitors configured")
		return
	}

	fmt.Printf("%-4s %-20s %-40s %-10s %-8s\n", "ID", "Name", "URL", "Status", "Enabled")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, m := range monitors {
		enabled := "No"
		if m.Enabled {
			enabled = "Yes"
		}
		fmt.Printf("%-4d %-20s %-40s %-10s %-8s\n", m.ID, m.Name, m.URL, m.CurrentStatus, enabled)
	}
}

func runRemove(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	var id uint
	fmt.Sscanf(args[0], "%d", &id)

	if err := db.DeleteMonitor(id); err != nil {
		log.Fatalf("Failed to remove monitor: %v", err)
	}

	fmt.Printf("Monitor %d removed successfully\n", id)
}

func runDashboard(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Start checker in background
	n := notifier.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := checker.New(db, n)
	if err := c.Start(ctx); err != nil {
		log.Fatalf("Failed to start checker: %v", err)
	}

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start dashboard TUI
	p := tea.NewProgram(
		tui.NewDashboard(db),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("Dashboard error: %v", err)
	}

	c.Stop()
}

func runTray(cmd *cobra.Command, args []string) {
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	t := tray.New(db)
	t.Run()

	db.Close()
}

const launchAgentLabel = "com.statping.tray"

const launchAgentTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExePath}}</string>
        <string>tray</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}/statping.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}/statping.err</string>
</dict>
</plist>
`

func getLaunchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist"), nil
}

func getExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

func runEnable(cmd *cobra.Command, args []string) {
	plistPath, err := getLaunchAgentPath()
	if err != nil {
		log.Fatalf("Failed to get LaunchAgent path: %v", err)
	}

	exePath, err := getExecutablePath()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	logPath, err := config.GetConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config dir: %v", err)
	}

	// Ensure LaunchAgents directory exists
	launchAgentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		log.Fatalf("Failed to create LaunchAgents directory: %v", err)
	}

	// Generate plist content
	tmpl, err := template.New("plist").Parse(launchAgentTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	file, err := os.Create(plistPath)
	if err != nil {
		log.Fatalf("Failed to create plist file: %v", err)
	}
	defer file.Close()

	data := struct {
		Label   string
		ExePath string
		LogPath string
	}{
		Label:   launchAgentLabel,
		ExePath: exePath,
		LogPath: logPath,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to write plist: %v", err)
	}

	// Load the LaunchAgent
	loadCmd := exec.Command("launchctl", "load", plistPath)
	if err := loadCmd.Run(); err != nil {
		fmt.Printf("⚠️  Created plist but failed to load: %v\n", err)
		fmt.Printf("   You may need to run: launchctl load %s\n", plistPath)
	} else {
		fmt.Println("✅ Auto-start enabled! Statping will start on login.")
		fmt.Printf("   Plist: %s\n", plistPath)
		fmt.Printf("   Binary: %s\n", exePath)
	}
}

func runDisable(cmd *cobra.Command, args []string) {
	plistPath, err := getLaunchAgentPath()
	if err != nil {
		log.Fatalf("Failed to get LaunchAgent path: %v", err)
	}

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("ℹ️  Auto-start is not enabled (no LaunchAgent found)")
		return
	}

	// Unload the LaunchAgent
	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	_ = unloadCmd.Run() // Ignore error if not loaded

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil {
		log.Fatalf("Failed to remove plist: %v", err)
	}

	fmt.Println("✅ Auto-start disabled. Statping will no longer start on login.")
}

func runStatus(cmd *cobra.Command, args []string) {
	plistPath, err := getLaunchAgentPath()
	if err != nil {
		log.Fatalf("Failed to get LaunchAgent path: %v", err)
	}

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("❌ Auto-start: Disabled")
		fmt.Println("   Run 'statping enable' to enable auto-start on login")
		return
	}

	// Check if loaded
	checkCmd := exec.Command("launchctl", "list", launchAgentLabel)
	if err := checkCmd.Run(); err != nil {
		fmt.Println("⚠️  Auto-start: Enabled but not loaded")
		fmt.Printf("   Plist exists at: %s\n", plistPath)
		fmt.Println("   Run 'launchctl load <plist>' to load it")
		return
	}

	fmt.Println("✅ Auto-start: Enabled and running")
	fmt.Printf("   Plist: %s\n", plistPath)
}
