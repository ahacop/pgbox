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

	// Resolve container name (finds running container if not specified)
	resolvedName, err := ResolveRunningContainer(client, containerName)
	if err != nil {
		return err
	}
	if containerName == "" {
		fmt.Printf("Restarting container: %s\n", resolvedName)
	}
	containerName = resolvedName

	// Restart the container
	fmt.Printf("Restarting container %s...\n", containerName)
	err = client.RunCommand("restart", containerName)
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	fmt.Printf("Container %s restarted successfully\n", containerName)
	return nil
}
