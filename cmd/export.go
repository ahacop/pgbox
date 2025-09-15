package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/ahacop/pgbox/pkg/scaffold"
	"github.com/spf13/cobra"
)

func ExportCmd() *cobra.Command {
	var pgVersion string
	var port string
	var extList string

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
  pgbox export ./my-postgres -p 5433`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportScaffold(args[0], pgVersion, port, extList)
		},
	}

	exportCmd.Flags().StringVarP(&pgVersion, "version", "v", "17", "PostgreSQL version (16 or 17)")
	exportCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	exportCmd.Flags().StringVar(&extList, "ext", "", "Comma-separated list of extensions")

	return exportCmd
}

func exportScaffold(targetDir, pgVersion, port, extList string) error {
	// Validate version
	if pgVersion != "16" && pgVersion != "17" {
		return fmt.Errorf("invalid PostgreSQL version: %s (must be 16 or 17)", pgVersion)
	}

	// Create PostgresConfig with defaults and environment overrides
	pgConfig := config.NewPostgresConfig()
	pgConfig.Version = pgVersion
	pgConfig.Port = port

	// Override with environment variables if set
	if user := os.Getenv("PGBOX_USER"); user != "" {
		pgConfig.User = user
	}
	if password := os.Getenv("PGBOX_PASSWORD"); password != "" {
		pgConfig.Password = password
	}
	if database := os.Getenv("PGBOX_DATABASE"); database != "" {
		pgConfig.Database = database
	}

	// Parse and validate extensions
	var extNames []string
	if extList != "" {
		extNames = strings.Split(extList, ",")
		for i := range extNames {
			extNames[i] = strings.TrimSpace(extNames[i])
		}

		// Validate extensions
		mgr := extensions.NewManager(pgVersion)
		if err := mgr.ValidateExtensions(extNames); err != nil {
			return err
		}
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate docker-compose.yml
	if err := generateDockerCompose(targetDir, pgConfig); err != nil {
		return err
	}

	// Generate Dockerfile
	if err := generateDockerfile(targetDir, pgConfig.Version, extNames); err != nil {
		return err
	}

	// Generate init.sql
	if err := generateInitSQL(targetDir, extNames, pgVersion); err != nil {
		return err
	}

	fmt.Printf("Exported Docker configuration to %s\n", targetDir)
	if len(extNames) > 0 {
		fmt.Printf("With extensions: %s\n", strings.Join(extNames, ", "))
	}
	fmt.Printf("\nTo start PostgreSQL:\n")
	fmt.Printf("  cd %s\n", targetDir)
	fmt.Printf("  docker-compose up -d\n")

	return nil
}

func generateDockerCompose(targetDir string, pgConfig *config.PostgresConfig) error {
	composePath := filepath.Join(targetDir, "docker-compose.yml")

	// Get container name from environment or use default
	containerName := os.Getenv("PGBOX_CONTAINER_NAME")
	if containerName == "" {
		containerName = "pgbox-postgres"
	}

	data := scaffold.DockerComposeData{
		PGMajor:       pgConfig.Version,
		ContainerName: containerName,
		Port:          pgConfig.Port,
		User:          pgConfig.User,
		Password:      pgConfig.Password,
		Database:      pgConfig.Database,
		HasExtensions: true, // Always true for export since we always generate init.sql
	}

	content, err := scaffold.GenerateDockerCompose(data)
	if err != nil {
		return fmt.Errorf("failed to generate docker-compose.yml: %w", err)
	}

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}

func generateDockerfile(targetDir, pgVersion string, extList []string) error {
	dockerfilePath := filepath.Join(targetDir, "Dockerfile")

	var packages []string
	if len(extList) > 0 {
		mgr := extensions.NewManager(pgVersion)
		packages = mgr.GetRequiredPackages(extList)
		// Sort packages for consistency
		sort.Strings(packages)
	}

	data := scaffold.DockerfileData{
		PGMajor:     pgVersion,
		HasPackages: len(packages) > 0,
		Packages:    packages,
	}

	content, err := scaffold.GenerateDockerfile(data)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return nil
}

func generateInitSQL(targetDir string, extList []string, pgVersion string) error {
	initPath := filepath.Join(targetDir, "init.sql")

	var exts []scaffold.ExtensionInfo
	if len(extList) > 0 {
		for _, ext := range extList {
			exts = append(exts, scaffold.ExtensionInfo{
				Name:    ext,
				SQLName: extensions.GetSQLName(ext, pgVersion),
			})
		}
	} else {
		// Add a comment placeholder for empty extension list
		// The template will handle this case
	}

	data := scaffold.InitSQLData{
		Extensions: exts,
	}

	content, err := scaffold.GenerateInitSQL(data)
	if err != nil {
		return fmt.Errorf("failed to generate init.sql: %w", err)
	}

	// If no extensions, add example comment
	if len(exts) == 0 {
		content = "-- Initialize PostgreSQL database\n-- Add any custom SQL initialization here\n\n-- Example: CREATE EXTENSION IF NOT EXISTS pg_stat_statements;\n"
	}

	if err := os.WriteFile(initPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write init.sql: %w", err)
	}

	return nil
}
