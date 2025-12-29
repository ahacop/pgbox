package cmd

import (
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
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
			orch := orchestrator.NewDownOrchestrator(docker.NewClient(), cmd.OutOrStdout())
			return orch.Run(orchestrator.DownConfig{
				ContainerName: containerName,
			})
		},
	}

	downCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name to stop (default: pgbox-pg<version>)")

	return downCmd
}
