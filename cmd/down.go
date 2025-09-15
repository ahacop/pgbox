package cmd

import (
	"fmt"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

func DownCmd() *cobra.Command {
	var containerName string

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop a running PostgreSQL container",
		Long: `Stop a running PostgreSQL container started with pgbox up.

This command stops and removes the container but preserves any volumes.`,
		Example: `  # Stop the default pgbox container
  pgbox down

  # Stop a container with a custom name
  pgbox down -n my-postgres`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return downContainer(containerName)
		},
	}

	downCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name to stop (default: pgbox-pg17)")

	return downCmd
}

func downContainer(name string) error {
	client := docker.NewClient()

	// Try to find a running container if name not specified
	if name == "" {
		foundName, err := client.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Specify container name with -n flag")
		}
		name = foundName
		fmt.Printf("Found running container: %s\n", name)
	}

	fmt.Printf("Stopping container %s...\n", name)

	err := client.StopContainer(name)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	fmt.Printf("Container %s stopped successfully\n", name)
	return nil
}
