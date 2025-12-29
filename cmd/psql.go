package cmd

import (
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/orchestrator"
	"github.com/spf13/cobra"
)

func PsqlCmd() *cobra.Command {
	var psqlDatabase string
	var psqlUser string
	var psqlName string

	psqlCmd := &cobra.Command{
		Use:   "psql [flags] [-- psql-args...]",
		Short: "Connect to PostgreSQL with psql",
		Long: `Connect to a running PostgreSQL container using psql client.

This command executes psql inside the container, so no local PostgreSQL client is needed.

You can pass additional arguments to psql after a '--' separator.`,
		Example: `  # Connect to default container with default database and user
  pgbox psql

  # Connect to a specific database
  pgbox psql --database mydb

  # Connect with a specific user
  pgbox psql --user myuser

  # Connect to a container with custom name
  pgbox psql -n my-postgres

  # Pass additional arguments to psql (e.g., execute a command)
  pgbox psql -- -c "SELECT version();"

  # Run psql with specific options
  pgbox psql -- -t -A -c "SELECT current_database();"

  # Execute a SQL file
  pgbox psql -- -f /path/to/file.sql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Collect extra args after --
			var extraArgs []string
			dashPos := cmd.ArgsLenAtDash()
			if dashPos > -1 {
				extraArgs = args[dashPos:]
			}

			// Only pass user/database if explicitly set
			user := ""
			database := ""
			if cmd.Flags().Changed("user") {
				user = psqlUser
			}
			if cmd.Flags().Changed("database") {
				database = psqlDatabase
			}

			orch := orchestrator.NewPsqlOrchestrator(docker.NewClient(), cmd.OutOrStdout())
			return orch.Run(orchestrator.PsqlConfig{
				ContainerName: psqlName,
				Database:      database,
				User:          user,
				ExtraArgs:     extraArgs,
			})
		},
		DisableFlagParsing: false,
		Args:               cobra.ArbitraryArgs,
	}

	psqlCmd.Flags().StringVarP(&psqlDatabase, "database", "d", "postgres", "Database name to connect to")
	psqlCmd.Flags().StringVarP(&psqlUser, "user", "u", "postgres", "Username for connection")
	psqlCmd.Flags().StringVarP(&psqlName, "name", "n", "", "Container name (default: pgbox-pg17)")

	return psqlCmd
}
