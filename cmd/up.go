package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahacop/pgbox/internal/applier"
	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/container"
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/ahacop/pgbox/internal/model"
	"github.com/ahacop/pgbox/internal/render"
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
		Example: `  # Start PostgreSQL 17 (creates container named pgbox-pg17)
  pgbox up

  # Start PostgreSQL 16 (creates container named pgbox-pg16)
  pgbox up -v 16

  # Start PostgreSQL with custom name
  pgbox up -n my-postgres

  # Start with extensions
  pgbox up --ext hypopg,pgvector

  # Start in foreground (attached mode)
  pgbox up --detach=false

  # Start with custom database and user
  pgbox up --database=mydb --user=myuser --password=secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return upPostgres(pgVersion, port, name, password, database, user, detach, extensionList)
		},
	}

	upCmd.Flags().StringVarP(&pgVersion, "version", "v", "17", "PostgreSQL version (16 or 17)")
	upCmd.Flags().StringVarP(&port, "port", "p", "5432", "Port to expose PostgreSQL on")
	upCmd.Flags().StringVarP(&name, "name", "n", "", "Container name (default: pgbox-pg<version>)")
	upCmd.Flags().StringVar(&password, "password", "postgres", "PostgreSQL password")
	upCmd.Flags().StringVar(&database, "database", "postgres", "Default database name")
	upCmd.Flags().StringVar(&user, "user", "postgres", "PostgreSQL user")
	upCmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run container in background")
	upCmd.Flags().StringVar(&extensionList, "ext", "", "Comma-separated list of extensions to install")

	return upCmd
}

func upPostgres(pgVersion, port, name, password, database, user string, detach bool, extensionList string) error {
	// Validate version
	if err := ValidatePostgresVersion(pgVersion); err != nil {
		return err
	}

	// Create config with defaults, then override with user values
	pgConfig := config.NewPostgresConfig()
	pgConfig.Version = pgVersion
	if port != "" {
		pgConfig.Port = port
	}
	if database != "" {
		pgConfig.Database = database
	}
	if user != "" {
		pgConfig.User = user
	}
	if password != "" {
		pgConfig.Password = password
	}

	// Parse extension list
	extNames := ParseExtensionList(extensionList)

	// Determine container name
	containerMgr := container.NewManager()
	containerName := name
	if containerName == "" {
		containerName = containerMgr.Name(pgConfig, extNames)
	}

	// Create Docker client
	client := docker.NewClient()

	// Check if container already exists (stopped)
	existingOutput, _ := client.RunCommandWithOutput("ps", "-a", "--filter", fmt.Sprintf("name=^%s$", containerName), "--format", "{{.Names}}")
	if strings.TrimSpace(existingOutput) == containerName {
		fmt.Printf("Restarting existing container: %s\n", containerName)
		if err := client.RunCommand("start", containerName); err != nil {
			return fmt.Errorf("failed to restart container: %w", err)
		}
		fmt.Printf("Container %s restarted successfully\n", containerName)
		return nil
	}

	// Initialize models
	baseImage := fmt.Sprintf("postgres:%s", pgVersion)
	dockerfileModel := model.NewDockerfileModel(baseImage)
	pgConfModel := model.NewPGConfModel()
	initModel := model.NewInitModel()

	// Process extensions if specified
	if len(extNames) > 0 {
		// Create TOML manager
		tomlMgr := extensions.NewTOMLManager(pgVersion)

		// Validate extensions
		if err := tomlMgr.ValidateExtensions(extNames); err != nil {
			return err
		}

		// Get extension specs
		specs, err := tomlMgr.GetSpecs(extNames)
		if err != nil {
			return fmt.Errorf("failed to load extension specs: %w", err)
		}

		// Apply specs to models
		app := applier.New()
		if err := app.Apply(specs, dockerfileModel, nil, pgConfModel, initModel); err != nil {
			return fmt.Errorf("failed to apply extensions: %w", err)
		}

		// Build custom image with extensions
		customImage, err := buildCustomImage(pgVersion, dockerfileModel, extNames, containerMgr)
		if err != nil {
			return fmt.Errorf("failed to build custom image: %w", err)
		}
		pgConfig.CustomImage = customImage
	}

	// Show the command being run
	fmt.Printf("Starting PostgreSQL %s...\n", pgConfig.Version)
	fmt.Printf("Container: %s\n", containerName)
	fmt.Printf("Port: %s\n", pgConfig.Port)
	fmt.Printf("User: %s\n", pgConfig.User)
	fmt.Printf("Database: %s\n", pgConfig.Database)
	if len(extNames) > 0 {
		fmt.Printf("Extensions: %s\n", strings.Join(extNames, ", "))
	}

	if !detach {
		fmt.Println("\nPress Ctrl+C to stop the container")
	} else {
		fmt.Printf("\nRunning in background. Use 'pgbox down -n %s' to stop.\n", containerName)
	}
	fmt.Println(strings.Repeat("-", 40))

	// Run PostgreSQL with options
	opts := docker.ContainerOptions{
		Name:      containerName,
		ExtraArgs: []string{},
	}
	if detach {
		opts.ExtraArgs = append(opts.ExtraArgs, "-d")
	}

	// Add volume for data persistence
	volumeName := fmt.Sprintf("%s-data", containerName)
	opts.ExtraArgs = append(opts.ExtraArgs, "-v", fmt.Sprintf("%s:/var/lib/postgresql/data", volumeName))

	// Handle extensions configuration
	if len(extNames) > 0 {
		// Generate and mount init.sql
		initFile := filepath.Join(os.TempDir(), fmt.Sprintf("pgbox-init-%s.sql", containerName))
		if err := render.RenderInitSQL(initModel, os.TempDir()); err != nil {
			return fmt.Errorf("failed to render init SQL: %w", err)
		}
		// Move the generated init.sql to the right location
		generatedInitPath := filepath.Join(os.TempDir(), "init.sql")
		initContent, err := os.ReadFile(generatedInitPath)
		if err != nil {
			return fmt.Errorf("failed to read generated init.sql: %w", err)
		}
		if err := os.WriteFile(initFile, initContent, 0644); err != nil {
			return fmt.Errorf("failed to write init.sql: %w", err)
		}
		if err := os.Remove(generatedInitPath); err != nil {
			// Log error but don't fail the command since container is already running
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up temp file %s: %v\n", generatedInitPath, err)
		}
		opts.ExtraArgs = append(opts.ExtraArgs, "-v", fmt.Sprintf("%s:/docker-entrypoint-initdb.d/init.sql:ro", initFile))

		// Add shared_preload_libraries if needed
		if len(pgConfModel.SharedPreload) > 0 {
			preloadStr := pgConfModel.GetSharedPreloadString()
			opts.Command = append(opts.Command, "-c", fmt.Sprintf("shared_preload_libraries=%s", preloadStr))
		}

		// Add other PostgreSQL configuration parameters
		for key, value := range pgConfModel.GUCs {
			// Skip shared_preload_libraries as it's handled above
			if key == "shared_preload_libraries" {
				continue
			}
			opts.Command = append(opts.Command, "-c", fmt.Sprintf("%s=%s", key, value))
		}
	}

	return client.RunPostgres(pgConfig, opts)
}

func buildCustomImage(pgVersion string, dockerfileModel *model.DockerfileModel, extensions []string, containerMgr *container.Manager) (string, error) {
	// Generate temp directory for build context
	buildDir := filepath.Join(os.TempDir(), fmt.Sprintf("pgbox-build-%d", os.Getpid()))
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create build directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(buildDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove build directory %s: %v\n", buildDir, err)
		}
	}()

	// Render Dockerfile
	if err := render.RenderDockerfile(dockerfileModel, buildDir); err != nil {
		return "", fmt.Errorf("failed to render Dockerfile: %w", err)
	}

	// Build image with deterministic name based on extensions
	imageName := containerMgr.ImageName(pgVersion, extensions)
	client := docker.NewClient()

	// Check if image already exists
	existingImages, _ := client.RunCommandWithOutput("images", "-q", imageName)
	if strings.TrimSpace(existingImages) != "" {
		fmt.Printf("Using existing custom image: %s\n", imageName)
		return imageName, nil
	}

	fmt.Println("Building custom PostgreSQL image with extensions...")
	buildArgs := []string{"build", "-t", imageName, "--build-arg", fmt.Sprintf("PG_MAJOR=%s", pgVersion), buildDir}
	if err := client.RunCommand(buildArgs...); err != nil {
		return "", fmt.Errorf("failed to build Docker image: %w", err)
	}

	return imageName, nil
}
