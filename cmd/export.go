package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/ahacop/pgbox/internal/model"
	"github.com/ahacop/pgbox/internal/render"
	"github.com/spf13/cobra"
)

// getDebArch returns the Debian architecture string for the current system
func getDebArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return "amd64" // fallback
	}
}

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
			return exportScaffold(args[0], pgVersion, port, extList, baseImage)
		},
	}

	exportCmd.Flags().StringVarP(&pgVersion, "version", "v", "17", "PostgreSQL version (16 or 17)")
	exportCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	exportCmd.Flags().StringVar(&extList, "ext", "", "Comma-separated list of extensions")
	exportCmd.Flags().StringVar(&baseImage, "base-image", "", "Base Docker image (default: postgres:<version>)")

	return exportCmd
}

func exportScaffold(targetDir, pgVersion, port, extList, baseImage string) error {
	// Validate version
	if err := ValidatePostgresVersion(pgVersion); err != nil {
		return err
	}

	// Parse extension list early to check for base image requirements
	extNames := ParseExtensionList(extList)

	// Set base image - check if extensions require a specific one
	if baseImage == "" {
		baseImage = extensions.GetBaseImage(extNames, pgVersion)
		if baseImage == "" {
			baseImage = fmt.Sprintf("postgres:%s", pgVersion)
		}
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

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Initialize models
	dockerfileModel := model.NewDockerfileModel(baseImage)
	composeModel := model.NewComposeModel("db")
	pgConfModel := model.NewPGConfModel()
	initModel := model.NewInitModel()

	// Configure compose model basics
	composeModel.BuildPath = "."
	composeModel.Image = baseImage
	composeModel.AddPort(fmt.Sprintf("%s:5432", port))
	composeModel.AddVolume("postgres_data:/var/lib/postgresql/data")
	composeModel.AddVolume("./init.sql:/docker-entrypoint-initdb.d/init.sql:ro")
	composeModel.SetEnv("POSTGRES_USER", pgConfig.User)
	composeModel.SetEnv("POSTGRES_PASSWORD", pgConfig.Password)
	composeModel.SetEnv("POSTGRES_DB", pgConfig.Database)

	// Process extensions if specified
	if len(extNames) > 0 {
		// Validate extensions exist in catalog
		if err := extensions.ValidateExtensions(extNames); err != nil {
			return err
		}

		// Add packages to Dockerfile model (apt packages)
		packages := extensions.GetPackages(extNames, pgVersion)
		if len(packages) > 0 {
			dockerfileModel.AddPackages(packages, "apt")
		}

		// Add .deb URLs to Dockerfile model
		debURLs := extensions.GetDebURLs(extNames, pgVersion, getDebArch())
		if len(debURLs) > 0 {
			dockerfileModel.AddDebURLs(debURLs...)
		}

		// Add shared_preload_libraries
		preload := extensions.GetPreloadLibraries(extNames)
		if len(preload) > 0 {
			pgConfModel.AddSharedPreload(preload...)
		}

		// Add GUCs (with conflict detection)
		gucs, err := extensions.GetGUCs(extNames)
		if err != nil {
			return fmt.Errorf("extension configuration conflict: %w", err)
		}
		for key, value := range gucs {
			pgConfModel.GUCs[key] = value
		}

		// Add init SQL for each extension
		for _, name := range extNames {
			sql := extensions.GetInitSQL(name)
			if sql != "" {
				initModel.AddFragment(name+"-init", sql)
			}
		}
	}

	// Render files
	if err := render.RenderDockerfile(dockerfileModel, targetDir); err != nil {
		return fmt.Errorf("failed to render Dockerfile: %w", err)
	}

	if err := render.RenderCompose(composeModel, pgConfModel, targetDir); err != nil {
		return fmt.Errorf("failed to render docker-compose.yml: %w", err)
	}

	if err := render.RenderInitSQL(initModel, targetDir); err != nil {
		return fmt.Errorf("failed to render init.sql: %w", err)
	}

	// Optionally render postgresql.conf snippet if there are config changes
	if len(pgConfModel.SharedPreload) > 0 || len(pgConfModel.GUCs) > 0 {
		if err := render.RenderPostgreSQLConf(pgConfModel, targetDir); err != nil {
			return fmt.Errorf("failed to render postgresql.conf: %w", err)
		}
	}

	// Success message
	fmt.Printf("Exported Docker configuration to %s\n", targetDir)
	if len(extNames) > 0 {
		fmt.Printf("With extensions: %s\n", strings.Join(extNames, ", "))
	}
	fmt.Printf("\nTo start PostgreSQL:\n")
	fmt.Printf("  cd %s\n", targetDir)
	fmt.Printf("  docker-compose up -d\n")

	if pgConfModel.RequireRestart {
		fmt.Printf("\nNote: Some extensions require server configuration changes.\n")
		fmt.Printf("The container will start with the required settings.\n")
	}

	return nil
}
