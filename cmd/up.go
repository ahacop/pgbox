package cmd

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/container"
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

	// Create config with defaults, then override with user values
	pgConfig := config.NewPostgresConfig()
	pgConfig.Version = pgVersion
	if port != "" {
		pgConfig.Port = port
	}
	if database != "" {
		pgConfig.Database = database
	}
	if user != "" {
		pgConfig.User = user
	}
	if password != "" {
		pgConfig.Password = password
	}

	// Determine container name
	containerMgr := container.NewManager()
	containerName := name
	if containerName == "" {
		containerName = containerMgr.Name(pgConfig)
	}

	// Show the command being run
	fmt.Printf("Starting PostgreSQL %s...\n", pgConfig.Version)
	fmt.Printf("Container: %s\n", containerName)
	fmt.Printf("Port: %s\n", pgConfig.Port)
	fmt.Printf("User: %s\n", pgConfig.User)
	fmt.Printf("Database: %s\n", pgConfig.Database)

	if !detach {
		fmt.Println("\nPress Ctrl+C to stop the container")
	} else {
		fmt.Printf("\nRunning in background. Use 'pgbox down -n %s' to stop.\n", containerName)
	}
	fmt.Println(strings.Repeat("-", 40))

	// Create Docker client and run PostgreSQL
	client := docker.NewClient()
	opts := docker.ContainerOptions{
		Name:      containerName,
		ExtraArgs: []string{"--rm"},
	}
	if detach {
		opts.ExtraArgs = append(opts.ExtraArgs, "-d")
	}

	return client.RunPostgres(pgConfig, opts)
}
