package cmd

import (
	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
	"github.com/spf13/cobra"
)

func UpCmd() *cobra.Command {
	var pgVersion string
	var port string
	var name string
	var password string
	var database string
	var user string
	var detach bool
	var extensionList string

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Start PostgreSQL in Docker",
		Long: `Start a PostgreSQL instance in Docker with the specified version.

This command starts a PostgreSQL container with sensible defaults for development.
The container runs in the background by default (detached mode).`,
		Example: `  # Start PostgreSQL 18 (creates container named pgbox-pg18)
  pgbox up

  # Start PostgreSQL 17 (creates container named pgbox-pg17)
  pgbox up -v 17

  # Start PostgreSQL with custom name
  pgbox up -n my-postgres

  # Start with extensions
  pgbox up --ext hypopg,pgvector

  # Start in foreground (attached mode)
  pgbox up --detach=false

  # Start with custom database and user
  pgbox up --database=mydb --user=myuser --password=secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidatePostgresVersion(pgVersion); err != nil {
				return err
			}

			extensions := ParseExtensionList(extensionList)
			orch := orchestrator.NewUpOrchestrator(docker.NewClient(), cmd.OutOrStdout())

			return orch.Run(orchestrator.UpConfig{
				Version:       pgVersion,
				Port:          port,
				ContainerName: name,
				Password:      password,
				Database:      database,
				User:          user,
				Detach:        detach,
				Extensions:    extensions,
			})
		},
	}

	upCmd.Flags().StringVarP(&pgVersion, "version", "v", config.DefaultVersion, "PostgreSQL version (16, 17, or 18)")
	upCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	upCmd.Flags().StringVarP(&name, "name", "n", "", "Container name (default: pgbox-pg<version>)")
	upCmd.Flags().StringVar(&password, "password", "postgres", "PostgreSQL password")
	upCmd.Flags().StringVar(&database, "database", "postgres", "Default database name")
	upCmd.Flags().StringVar(&user, "user", "postgres", "PostgreSQL user")
	upCmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run container in background")
	upCmd.Flags().StringVar(&extensionList, "ext", "", "Comma-separated list of extensions to install")

	return upCmd
}
