package cmd

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

func StatusCmd() *cobra.Command {
	var containerName string

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of PostgreSQL containers",
		Long: `Display the status of running pgbox PostgreSQL containers.

Shows information about running containers including:
- Container name
- PostgreSQL version
- Port mapping
- Running time`,
		Example: `  # Show status of all pgbox containers
  pgbox status

  # Show status of a specific container
  pgbox status -n my-postgres`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showStatus(containerName)
		},
	}

	statusCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name to check status for")

	return statusCmd
}

func showStatus(containerName string) error {
	client := docker.NewClient()

	// If no container specified, try to find running pgbox containers
	if containerName == "" {
		containers, err := client.ListContainers("pgbox")
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers) == 0 {
			fmt.Println("No pgbox containers are running.")
			fmt.Println("\nStart a container with: pgbox up")
			return nil
		}

		// Show status for all pgbox containers
		fmt.Println("PostgreSQL containers:")
		output, err := client.RunCommandWithOutput("ps", "--filter", "name=pgbox", "--format", "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
		if err != nil {
			return fmt.Errorf("failed to get container status: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Check specific container - verify it's running
	resolvedName, err := ResolveRunningContainer(client, containerName)
	if err != nil {
		// Container not running is not an error for status command
		fmt.Printf("Container '%s' is not running.\n", containerName)
		return nil
	}
	containerName = resolvedName

	// Get detailed container info
	output, err := client.RunCommandWithOutput("ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
	if err != nil {
		return fmt.Errorf("failed to get container details: %w", err)
	}

	fmt.Println("Container status:")
	fmt.Println(output)

	// Get database info from the container using existing functions
	dbName, _ := client.GetContainerEnv(containerName, "POSTGRES_DB")
	userName, _ := client.GetContainerEnv(containerName, "POSTGRES_USER")

	if dbName != "" || userName != "" {
		fmt.Println("\nDatabase configuration:")
		if dbName != "" {
			fmt.Printf("  Database: %s\n", dbName)
		}
		if userName != "" {
			fmt.Printf("  User: %s\n", userName)
		}

		// Extract port for connection string
		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				ports := fields[3]
				if strings.Contains(ports, "->") {
					portMapping := strings.Split(ports, "->")[0]
					port := strings.TrimPrefix(portMapping, "0.0.0.0:")
					port = strings.TrimPrefix(port, ":")

					fmt.Printf("\nConnection string:\n")
					fmt.Printf("  postgres://%s@localhost:%s/%s\n", userName, port, dbName)
				}
			}
		}
	}

	return nil
}
