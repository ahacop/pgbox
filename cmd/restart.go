package cmd

import (
	"fmt"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

func RestartCmd() *cobra.Command {
	var containerName string

	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart PostgreSQL container",
		Long: `Restart a running PostgreSQL container.

This command stops and then starts the container, preserving all data and configuration.`,
		Example: `  # Restart the default container
  pgbox restart

  # Restart a specific container
  pgbox restart -n my-postgres`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return restartContainer(containerName)
		},
	}

	restartCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name (default: auto-detect)")

	return restartCmd
}

func restartContainer(containerName string) error {
	client := docker.NewClient()

	// If no container name specified, try to find a running one
	if containerName == "" {
		foundName, err := client.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		containerName = foundName
		fmt.Printf("Restarting container: %s\n", containerName)
	}

	// Check if the specified container is actually running
	running, err := client.IsContainerRunning(containerName)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		return fmt.Errorf("container %s is not running. Start it with: pgbox up", containerName)
	}

	// Restart the container
	fmt.Printf("Restarting container %s...\n", containerName)
	err = client.RunCommand("restart", containerName)
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	fmt.Printf("Container %s restarted successfully\n", containerName)
	return nil
}
