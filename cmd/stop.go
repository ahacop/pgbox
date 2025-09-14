package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func StopCmd() *cobra.Command {
	var containerName string

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running PostgreSQL container",
		Long: `Stop a running PostgreSQL container started with pgbox run.

This command stops and removes the container but preserves any volumes.`,
		Example: `  # Stop the default pgbox container
  pgbox stop

  # Stop a container with a custom name
  pgbox stop -n my-postgres`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopContainer(containerName)
		},
	}

	stopCmd.Flags().StringVarP(&containerName, "name", "n", "pgbox-postgres", "Container name to stop")

	return stopCmd
}

func stopContainer(name string) error {
	fmt.Printf("Stopping container %s...\n", name)

	// Execute docker stop command
	dockerCmd := exec.Command("docker", "stop", name)
	output, err := dockerCmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to stop container: %w\nOutput: %s", err, output)
	}

	fmt.Printf("Container %s stopped successfully\n", name)
	return nil
}
