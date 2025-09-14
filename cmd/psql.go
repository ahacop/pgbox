package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	// Use default name if not provided
	if psqlName == "" {
		psqlName = "pgbox-pg17"
	}

	// Check if container is running
	checkCmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%s", psqlName))
	output, err := checkCmd.Output()
	if err != nil || len(output) == 0 {
		return fmt.Errorf("container %s is not running. Start it with: pgbox up -n %s", psqlName, psqlName)
	}

	fmt.Printf("Connecting to %s as user '%s' to database '%s'...\n", psqlName, psqlUser, psqlDatabase)
	fmt.Println("Type \\q to exit")
	fmt.Println(strings.Repeat("-", 40))

	// Execute psql inside the container
	dockerArgs := []string{
		"exec",
		"-it",
		psqlName,
		"psql",
		"-U", psqlUser,
		"-d", psqlDatabase,
	}

	dockerCmd := exec.Command("docker", dockerArgs...)
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	return dockerCmd.Run()
}