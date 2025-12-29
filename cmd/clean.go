package cmd

import (
	"os"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
	"github.com/spf13/cobra"
)

func CleanCmd() *cobra.Command {
	var force bool
	var all bool

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove pgbox containers and images",
		Long: `Remove pgbox-related Docker containers and images to free up space and clear cache.

By default, this command will:
- Stop and remove all running pgbox containers
- Remove all pgbox Docker images

Use --all to also remove PostgreSQL base images.`,
		Example: `  # Clean pgbox containers and images
  pgbox clean

  # Clean without confirmation prompt
  pgbox clean --force

  # Clean everything including PostgreSQL base images
  pgbox clean --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orch := orchestrator.NewCleanOrchestrator(docker.NewClient(), cmd.OutOrStdout(), os.Stdin)
			return orch.Run(orchestrator.CleanConfig{
				Force: force,
				All:   all,
			})
		},
	}

	cleanCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cleanCmd.Flags().BoolVarP(&all, "all", "a", false, "Also remove PostgreSQL base images")

	return cleanCmd
}
