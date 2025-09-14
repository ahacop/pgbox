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

	if psqlName == "" {
		psqlName = findRunningPgboxContainer(client)
		if psqlName == "" {
			psqlName = "pgbox-pg17"
		}
	}

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

// findRunningPgboxContainer searches for running containers that look like pgbox containers
func findRunningPgboxContainer(client *docker.Client) string {
	// Get list of running containers
	output, err := client.RunCommandWithOutput("ps", "--format", "{{.Names}}\t{{.Image}}")
	if err != nil {
		return ""
	}

	containerName := selectPgboxContainer(output)
	if containerName != "" {
		return containerName
	}

	// Try common pgbox container names as fallback
	possibleNames := []string{"pgbox-pg17", "pgbox-pg16"}
	for _, name := range possibleNames {
		if running, err := client.IsContainerRunning(name); err == nil && running {
			return name
		}
	}

	return ""
}

// selectPgboxContainer is pure business logic that selects the best container from docker ps output
// This function has no side effects and is easily testable
func selectPgboxContainer(dockerPsOutput string) string {
	if dockerPsOutput == "" {
		return ""
	}

	lines := strings.Split(dockerPsOutput, "\n")

	// First priority: containers starting with "pgbox-"
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			if strings.HasPrefix(name, "pgbox-") {
				return name
			}
		}
	}

	// Second priority: any container with postgres image
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			image := strings.TrimSpace(parts[1])
			if strings.HasPrefix(image, "postgres:") {
				return name
			}
		}
	}

	return ""
}
