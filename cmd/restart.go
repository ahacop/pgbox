package cmd

import (
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
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
			orch := orchestrator.NewRestartOrchestrator(docker.NewClient(), cmd.OutOrStdout())
			return orch.Run(orchestrator.RestartConfig{
				ContainerName: containerName,
			})
		},
	}

	restartCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name (default: auto-detect)")

	return restartCmd
}
