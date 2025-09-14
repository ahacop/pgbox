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
	if psqlName == "" {
		psqlName = findRunningPgboxContainer()
		if psqlName == "" {
			psqlName = "pgbox-pg17"
		}
	}

	// Check if container is running
	checkCmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=^%s$", psqlName))
	output, err := checkCmd.Output()
	if err != nil || len(output) == 0 {
		return fmt.Errorf("no pgbox container is running. Start one with: pgbox up")
	}

	// If user and database weren't specified, try to get them from container env vars
	if !cmd.Flags().Changed("user") {
		if envUser := getContainerEnv(psqlName, "POSTGRES_USER"); envUser != "" {
			psqlUser = envUser
		}
	}
	if !cmd.Flags().Changed("database") {
		if envDB := getContainerEnv(psqlName, "POSTGRES_DB"); envDB != "" {
			psqlDatabase = envDB
		}
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

// findRunningPgboxContainer searches for running containers that look like pgbox containers
func findRunningPgboxContainer() string {
	// Get list of running containers
	listCmd := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Image}}")
	output, err := listCmd.Output()
	if err != nil {
		return ""
	}

	containerName := selectPgboxContainer(string(output))
	if containerName != "" {
		return containerName
	}

	// Try common pgbox container names as fallback
	possibleNames := []string{"pgbox-pg17", "pgbox-pg16"}
	for _, name := range possibleNames {
		checkCmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=^%s$", name))
		output, err := checkCmd.Output()
		if err == nil && len(output) > 0 {
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

// getContainerEnv retrieves an environment variable from a running container
func getContainerEnv(containerName, envVar string) string {
	cmd := exec.Command("docker", "inspect", "-f", fmt.Sprintf("{{range .Config.Env}}{{if eq (index (split . \"=\") 0) \"%s\"}}{{index (split . \"=\") 1}}{{end}}{{end}}", envVar), containerName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// parseContainerEnv extracts an environment variable value from docker inspect output
// This is pure business logic with no side effects
func parseContainerEnv(dockerInspectOutput string, envVar string) string {
	// Docker inspect with our template returns just the value or empty
	return strings.TrimSpace(dockerInspectOutput)
}
