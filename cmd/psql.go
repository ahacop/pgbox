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

	// Use the new GetOrFindContainerName method
	psqlName = client.GetOrFindContainerName(psqlName)

	// Check if container is running
	running, err := client.IsContainerRunning(psqlName)
	if err != nil || !running {
		return fmt.Errorf("no pgbox container is running. Start one with: pgbox up")
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
