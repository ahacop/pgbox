package cmd

import (
	"fmt"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

func LogsCmd() *cobra.Command {
	var containerName string
	var follow bool

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Show container logs",
		Long: `Display logs from a running PostgreSQL container.

By default shows recent logs and exits. Use -f/--follow to stream logs continuously.`,
		Example: `  # Show logs from the default container
  pgbox logs

  # Follow logs from a specific container
  pgbox logs -n my-postgres -f

  # Show logs from a specific container
  pgbox logs -n my-postgres`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showLogs(containerName, follow)
		},
	}

	logsCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name (default: auto-detect)")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")

	return logsCmd
}

func showLogs(containerName string, follow bool) error {
	client := docker.NewClient()

	// If no container name specified, try to find a running one
	if containerName == "" {
		foundName, err := client.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		containerName = foundName
		fmt.Printf("Showing logs for container: %s\n", containerName)
	}

	// Check if the specified container is actually running
	running, err := client.IsContainerRunning(containerName)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		return fmt.Errorf("container %s is not running. Start it with: pgbox up", containerName)
	}

	// Build docker logs command arguments
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	// Show logs
	return client.RunCommand(args...)
}
