package cmd

import (
	"os"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/orchestrator"
	"github.com/spf13/cobra"
)

func ExportCmd() *cobra.Command {
	var pgVersion string
	var port string
	var extList string
	var baseImage string

	exportCmd := &cobra.Command{
		Use:   "export [directory]",
		Short: "Export Docker configuration to directory",
		Long: `Export a Docker Compose configuration for PostgreSQL with optional extensions.

This command generates a docker-compose.yml, Dockerfile, and init.sql that can be
used independently of pgbox to run PostgreSQL with your chosen configuration.`,
		Example: `  # Export basic PostgreSQL 17 configuration
  pgbox export ./my-postgres

  # Export with specific version and extensions
  pgbox export ./my-postgres -v 16 --ext hypopg,pgvector

  # Export with custom port
  pgbox export ./my-postgres -p 5433

  # Export with custom base image
  pgbox export ./my-postgres --base-image postgres:17-alpine`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate version
			if err := ValidatePostgresVersion(pgVersion); err != nil {
				return err
			}

			// Parse extensions
			extensions := ParseExtensionList(extList)

			// Create orchestrator
			orch := orchestrator.NewExportOrchestrator(cmd.OutOrStdout())

			// Run the orchestrator
			return orch.Run(orchestrator.ExportConfig{
				TargetDir:  args[0],
				Version:    pgVersion,
				Port:       port,
				Extensions: extensions,
				BaseImage:  baseImage,
				User:       os.Getenv("PGBOX_USER"),
				Password:   os.Getenv("PGBOX_PASSWORD"),
				Database:   os.Getenv("PGBOX_DATABASE"),
			})
		},
	}

	exportCmd.Flags().StringVarP(&pgVersion, "version", "v", config.DefaultVersion, "PostgreSQL version (16, 17, or 18)")
	exportCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	exportCmd.Flags().StringVar(&extList, "ext", "", "Comma-separated list of extensions")
	exportCmd.Flags().StringVar(&baseImage, "base-image", "", "Base Docker image (default: postgres:<version>)")

	return exportCmd
}
