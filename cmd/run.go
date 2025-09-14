package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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

func RunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run PostgreSQL in Docker",
		Long: `Run a PostgreSQL instance in Docker with the specified version.

This command starts a PostgreSQL container with sensible defaults for development.
The container runs in the background by default (detached mode).`,
		Example: `  # Run PostgreSQL 17 (default) on default port 5432
  pgbox run

  # Run PostgreSQL 16 on port 5433
  pgbox run -v 16 -p 5433

  # Run PostgreSQL with custom name
  pgbox run -n my-postgres

  # Run in foreground (attached mode)
  pgbox run --detach=false

  # Run with custom database and user
  pgbox run --database=mydb --user=myuser --password=secret`,
		RunE: runPostgres,
	}

	runCmd.Flags().StringVarP(&pgVersion, "version", "v", "17", "PostgreSQL version (16 or 17)")
	runCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	runCmd.Flags().StringVarP(&name, "name", "n", "pgbox-postgres", "Container name")
	runCmd.Flags().StringVar(&password, "password", "postgres", "PostgreSQL password")
	runCmd.Flags().StringVar(&database, "database", "postgres", "Default database name")
	runCmd.Flags().StringVar(&user, "user", "postgres", "PostgreSQL user")
	runCmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run container in background")

	return runCmd
}

func runPostgres(cmd *cobra.Command, args []string) error {
	// Validate version
	if pgVersion != "16" && pgVersion != "17" {
		return fmt.Errorf("invalid PostgreSQL version: %s (must be 16 or 17)", pgVersion)
	}

	// Build docker run command
	dockerArgs := []string{
		"run",
		"--rm", // Remove container when it stops
		"--name", name,
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		"-e", fmt.Sprintf("POSTGRES_USER=%s", user),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", database),
		"-p", fmt.Sprintf("%s:5432", port),
	}

	if detach {
		dockerArgs = append(dockerArgs, "-d")
	}

	// Add the PostgreSQL image
	image := fmt.Sprintf("postgres:%s", pgVersion)
	dockerArgs = append(dockerArgs, image)

	// Show the command being run
	fmt.Printf("Starting PostgreSQL %s...\n", pgVersion)
	fmt.Printf("Container: %s\n", name)
	fmt.Printf("Port: %s\n", port)
	fmt.Printf("User: %s\n", user)
	fmt.Printf("Database: %s\n", database)

	if !detach {
		fmt.Println("\nPress Ctrl+C to stop the container")
	} else {
		fmt.Printf("\nRunning in background. Use 'pgbox stop -n %s' to stop.\n", name)
	}
	fmt.Println(strings.Repeat("-", 40))

	// Execute docker command
	dockerCmd := exec.Command("docker", dockerArgs...)
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	return dockerCmd.Run()
}
