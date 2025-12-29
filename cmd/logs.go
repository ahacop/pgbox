package cmd

import (
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
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
			orch := orchestrator.NewLogsOrchestrator(docker.NewClient(), cmd.OutOrStdout())
			return orch.Run(orchestrator.LogsConfig{
				ContainerName: containerName,
				Follow:        follow,
			})
		},
	}

	logsCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name (default: auto-detect)")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")

	return logsCmd
}
