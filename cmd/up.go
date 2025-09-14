package cmd

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

var (
	pgVersion string
	port      string
	name      string
	password  string
	database  string
	user      string
	detach    bool
)

func UpCmd() *cobra.Command {
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Start PostgreSQL in Docker",
		Long: `Start a PostgreSQL instance in Docker with the specified version.

This command starts a PostgreSQL container with sensible defaults for development.
The container runs in the background by default (detached mode).`,
		Example: `  # Start PostgreSQL 17 (creates container named pgbox-pg17)
  pgbox up

  # Start PostgreSQL 16 (creates container named pgbox-pg16)
  pgbox up -v 16

  # Start PostgreSQL with custom name
  pgbox up -n my-postgres

  # Start in foreground (attached mode)
  pgbox up --detach=false

  # Start with custom database and user
  pgbox up --database=mydb --user=myuser --password=secret`,
		RunE: upPostgres,
	}

	upCmd.Flags().StringVarP(&pgVersion, "version", "v", "17", "PostgreSQL version (16 or 17)")
	upCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	upCmd.Flags().StringVarP(&name, "name", "n", "", "Container name (default: pgbox-pg<version>)")
	upCmd.Flags().StringVar(&password, "password", "postgres", "PostgreSQL password")
	upCmd.Flags().StringVar(&database, "database", "postgres", "Default database name")
	upCmd.Flags().StringVar(&user, "user", "postgres", "PostgreSQL user")
	upCmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run container in background")

	return upCmd
}

func upPostgres(cmd *cobra.Command, args []string) error {
	// Validate version
	if pgVersion != "16" && pgVersion != "17" {
		return fmt.Errorf("invalid PostgreSQL version: %s (must be 16 or 17)", pgVersion)
	}

	// Use default name if not provided
	if name == "" {
		name = fmt.Sprintf("pgbox-pg%s", pgVersion)
	}

	// Show the command being run
	fmt.Printf("Starting PostgreSQL %s...\n", pgVersion)
	fmt.Printf("Container: %s\n", name)
	fmt.Printf("Port: %s\n", port)
	fmt.Printf("User: %s\n", user)
	fmt.Printf("Database: %s\n", database)

	if !detach {
		fmt.Println("\nPress Ctrl+C to stop the container")
	} else {
		fmt.Printf("\nRunning in background. Use 'pgbox down -n %s' to stop.\n", name)
	}
	fmt.Println(strings.Repeat("-", 40))

	// Create Docker client and run PostgreSQL
	client := docker.NewClient()
	config := docker.PostgresConfig{
		Name:     name,
		Image:    fmt.Sprintf("postgres:%s", pgVersion),
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
	}

	// Add --rm flag and -d if detaching
	extraArgs := []string{"--rm"}
	if detach {
		extraArgs = append(extraArgs, "-d")
	}
	config.ExtraArgs = extraArgs

	return client.RunPostgres(config)
}
