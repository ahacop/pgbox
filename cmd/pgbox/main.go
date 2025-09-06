package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/pkg/docker"
	"github.com/ahacop/pgbox/pkg/extensions"
	"github.com/ahacop/pgbox/pkg/scaffold"
	"github.com/ahacop/pgbox/pkg/ui"
)

var (
	cfg       *config.Config
	extMgr    *extensions.Manager
	dockerMgr *docker.Manager
)

func init() {
	cfg = config.NewConfig()
	extMgr = extensions.NewManager(cfg.ScriptDir, cfg.PgMajor)
	dockerMgr = docker.NewManager(cfg)
}

var rootCmd = &cobra.Command{
	Use:   "pgbox",
	Short: "Run Postgres-in-Docker with selectable extensions",
	Long:  "pgbox – run Postgres-in-Docker with selectable extensions (hypopg, etc.)",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start PostgreSQL container with extensions",
	RunE:  runUp,
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop PostgreSQL container",
	RunE:  runDown,
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart PostgreSQL container",
	RunE:  runRestart,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show container status",
	RunE:  runStatus,
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show container logs",
	RunE:  runLogs,
}

var psqlCmd = &cobra.Command{
	Use:   "psql",
	Short: "Connect to PostgreSQL with psql",
	RunE:  runPsql,
}

var exportCmd = &cobra.Command{
	Use:   "export [directory]",
	Short: "Export scaffold to directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runExport,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfg.Name, "name", cfg.Name, "Instance name")
	rootCmd.PersistentFlags().StringVar(&cfg.Port, "port", cfg.Port, "Port to bind")
	rootCmd.PersistentFlags().StringVar(&cfg.PgMajor, "pg", cfg.PgMajor, "PostgreSQL major version")

	// Up command flags
	upCmd.Flags().StringVar(&cfg.Extensions, "ext", "", "Comma-separated list of extensions")
	upCmd.Flags().StringVar(&cfg.ExportDir, "export", "", "Export scaffolding to directory")

	// Down command flags
	downCmd.Flags().BoolP("wipe", "w", false, "Remove data volume")

	// Logs command flags
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")

	// Psql command flags
	psqlCmd.Flags().String("db", "appdb", "Database name")
	psqlCmd.Flags().String("user", "appuser", "Username")

	// Export command flags
	exportCmd.Flags().StringVar(&cfg.Extensions, "ext", "", "Comma-separated list of extensions")

	// Add commands
	rootCmd.AddCommand(upCmd, downCmd, restartCmd, statusCmd, logsCmd, psqlCmd, exportCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Update derived values based on flags
	cfg.SetName(cfg.Name)
	extMgr.PgMajor = cfg.PgMajor

	if err := dockerMgr.EnsureTools(); err != nil {
		return err
	}

	// Get extensions list
	var extensionList []string
	if cfg.Extensions == "" {
		// Interactive mode - show main interface with config management
		selectedConfig, selectedExts, err := ui.RunMainInterface(extMgr)
		if err != nil {
			return fmt.Errorf("interface error: %w", err)
		}

		if selectedConfig != nil {
			// User selected an existing configuration
			extensionList = selectedConfig.Extensions
			// Update config to match selected configuration
			cfg.SetName(selectedConfig.Name)
			cfg.Port = selectedConfig.Port
			cfg.PgMajor = selectedConfig.PgMajor
			extMgr.PgMajor = cfg.PgMajor
		} else if selectedExts != nil {
			// User created a new configuration
			extensionList = selectedExts
		} else {
			// User cancelled
			return fmt.Errorf("no configuration selected")
		}
	} else {
		extensionList = extMgr.ParseExtensionList(cfg.Extensions)
	}

	// Create scaffold
	scaffoldObj, err := scaffold.NewScaffold(cfg, extMgr, cfg.ExportDir)
	if err != nil {
		return fmt.Errorf("failed to create scaffold: %w", err)
	}

	if err := scaffoldObj.Generate(extensionList); err != nil {
		return fmt.Errorf("failed to generate scaffold: %w", err)
	}

	fmt.Printf("Bringing up %s (PG %s) with extensions: %s\n", cfg.Name, cfg.PgMajor, strings.Join(extensionList, ", "))

	if err := dockerMgr.ComposeUp(scaffoldObj.Path); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Printf("Ready. Connection: postgres://appuser:changeme@localhost:%s/appdb\n", cfg.Port)

	if cfg.ExportDir == "" {
		// Save scaffold path for runtime commands
		scaffoldPathFile := filepath.Join(cfg.StateDir, "scaffold_path")
		if err := os.WriteFile(scaffoldPathFile, []byte(scaffoldObj.Path), 0644); err != nil {
			return fmt.Errorf("failed to save scaffold path: %w", err)
		}
	} else {
		fmt.Printf("Exported scaffold to: %s\n", cfg.ExportDir)
	}

	return nil
}

func runDown(cmd *cobra.Command, args []string) error {
	cfg.SetName(cfg.Name)

	wipe, _ := cmd.Flags().GetBool("wipe")

	if err := dockerMgr.EnsureTools(); err != nil {
		return err
	}

	scaffoldPath, err := getScaffoldPath()
	if err != nil {
		return err
	}

	if err := dockerMgr.ComposeDown(scaffoldPath); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	if wipe {
		if err := dockerMgr.RemoveVolume(); err != nil {
			fmt.Printf("Warning: failed to remove volume: %v\n", err)
		}
	}

	// Clean up state
	scaffoldPathFile := filepath.Join(cfg.StateDir, "scaffold_path")
	_ = os.Remove(scaffoldPathFile)

	suffix := ""
	if wipe {
		suffix = " (wiped data)"
	}
	fmt.Printf("Stopped %s%s.\n", cfg.Name, suffix)

	return nil
}

func runRestart(cmd *cobra.Command, args []string) error {
	cfg.SetName(cfg.Name)

	if err := dockerMgr.EnsureTools(); err != nil {
		return err
	}

	scaffoldPath, err := getScaffoldPath()
	if err != nil {
		return err
	}

	return dockerMgr.ComposeRestart(scaffoldPath)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg.SetName(cfg.Name)
	return dockerMgr.Status()
}

func runLogs(cmd *cobra.Command, args []string) error {
	cfg.SetName(cfg.Name)

	follow, _ := cmd.Flags().GetBool("follow")

	if err := dockerMgr.EnsureTools(); err != nil {
		return err
	}

	scaffoldPath, err := getScaffoldPath()
	if err != nil {
		return err
	}

	return dockerMgr.ComposeLogs(scaffoldPath, follow)
}

func runPsql(cmd *cobra.Command, args []string) error {
	cfg.SetName(cfg.Name)

	db, _ := cmd.Flags().GetString("db")
	user, _ := cmd.Flags().GetString("user")

	return dockerMgr.Psql(db, user)
}

func runExport(cmd *cobra.Command, args []string) error {
	cfg.SetName(cfg.Name)
	extMgr.PgMajor = cfg.PgMajor

	targetDir := args[0]

	// Get extensions list
	var extensionList []string
	if cfg.Extensions == "" {
		// Interactive mode - show main interface with config management
		selectedConfig, selectedExts, err := ui.RunMainInterface(extMgr)
		if err != nil {
			return fmt.Errorf("interface error: %w", err)
		}

		if selectedConfig != nil {
			// User selected an existing configuration
			extensionList = selectedConfig.Extensions
			// Update config to match selected configuration
			cfg.SetName(selectedConfig.Name)
			cfg.Port = selectedConfig.Port
			cfg.PgMajor = selectedConfig.PgMajor
			extMgr.PgMajor = cfg.PgMajor
		} else if selectedExts != nil {
			// User created a new configuration
			extensionList = selectedExts
		} else {
			// User cancelled
			return fmt.Errorf("no configuration selected")
		}
	} else {
		extensionList = extMgr.ParseExtensionList(cfg.Extensions)
	}

	// Create scaffold
	scaffoldObj, err := scaffold.NewScaffold(cfg, extMgr, targetDir)
	if err != nil {
		return fmt.Errorf("failed to create scaffold: %w", err)
	}

	if err := scaffoldObj.Generate(extensionList); err != nil {
		return fmt.Errorf("failed to generate scaffold: %w", err)
	}

	fmt.Printf("Exported to %s\n", targetDir)
	return nil
}

func getScaffoldPath() (string, error) {
	scaffoldPathFile := filepath.Join(cfg.StateDir, "scaffold_path")

	if data, err := os.ReadFile(scaffoldPathFile); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Fallback: if container exists, create a temp scaffold
	if dockerMgr.ContainerExists() {
		scaffoldObj, err := scaffold.NewScaffold(cfg, extMgr, "")
		if err != nil {
			return "", err
		}
		// Generate minimal scaffold for compose commands
		if err := scaffoldObj.Generate([]string{}); err != nil {
			return "", err
		}
		return scaffoldObj.Path, nil
	}

	return "", fmt.Errorf("no running stack found for --name %s. Start with: pgbox up --name %s", cfg.Name, cfg.Name)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
