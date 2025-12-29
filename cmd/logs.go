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

	// Resolve container name (finds running container if not specified)
	resolvedName, err := ResolveRunningContainer(client, containerName)
	if err != nil {
		return err
	}
	if containerName == "" {
		fmt.Printf("Showing logs for container: %s\n", resolvedName)
	}
	containerName = resolvedName

	// Build docker logs command arguments
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	// Show logs
	return client.RunCommand(args...)
}
