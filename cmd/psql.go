package cmd

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

var (
	psqlDatabase string
	psqlUser     string
	psqlName     string
)

func PsqlCmd() *cobra.Command {
	psqlCmd := &cobra.Command{
		Use:   "psql",
		Short: "Connect to PostgreSQL with psql",
		Long: `Connect to a running PostgreSQL container using psql client.

This command executes psql inside the container, so no local PostgreSQL client is needed.`,
		Example: `  # Connect to default container with default database and user
  pgbox psql

  # Connect to a specific database
  pgbox psql --database mydb

  # Connect with a specific user
  pgbox psql --user myuser

  # Connect to a container with custom name
  pgbox psql -n my-postgres`,
		RunE: runPsql,
	}

	psqlCmd.Flags().StringVarP(&psqlDatabase, "database", "d", "postgres", "Database name to connect to")
	psqlCmd.Flags().StringVarP(&psqlUser, "user", "u", "postgres", "Username for connection")
	psqlCmd.Flags().StringVarP(&psqlName, "name", "n", "", "Container name (default: pgbox-pg17)")

	return psqlCmd
}

func runPsql(cmd *cobra.Command, args []string) error {
	client := docker.NewClient()

	// If no container name specified, try to find a running one
	if psqlName == "" {
		foundName, err := client.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		psqlName = foundName
	}

	// Check if the specified container is actually running
	running, err := client.IsContainerRunning(psqlName)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		return fmt.Errorf("container %s is not running. Start it with: pgbox up", psqlName)
	}

	// If user and database weren't specified, try to get them from container env vars
	if !cmd.Flags().Changed("user") {
		if envUser, err := client.GetContainerEnv(psqlName, "POSTGRES_USER"); err == nil && envUser != "" {
			psqlUser = envUser
		}
	}
	if !cmd.Flags().Changed("database") {
		if envDB, err := client.GetContainerEnv(psqlName, "POSTGRES_DB"); err == nil && envDB != "" {
			psqlDatabase = envDB
		}
	}

	fmt.Printf("Connecting to %s as user '%s' to database '%s'...\n", psqlName, psqlUser, psqlDatabase)
	fmt.Println("Type \\q to exit")
	fmt.Println(strings.Repeat("-", 40))

	// Execute psql inside the container
	return client.RunInteractive(
		"exec", "-it", psqlName,
		"psql", "-U", psqlUser, "-d", psqlDatabase,
	)
}
