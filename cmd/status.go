package cmd

import (
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
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
			orch := orchestrator.NewStatusOrchestrator(docker.NewClient(), cmd.OutOrStdout())
			return orch.Run(orchestrator.StatusConfig{
				ContainerName: containerName,
			})
		},
	}

	statusCmd.Flags().StringVarP(&containerName, "name", "n", "", "Container name to check status for")

	return statusCmd
}
