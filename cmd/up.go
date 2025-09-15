package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/container"
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/ahacop/pgbox/pkg/scaffold"
	"github.com/spf13/cobra"
)

var (
	pgVersion     string
	port          string
	name          string
	password      string
	database      string
	user          string
	detach        bool
	extensionList string
)

func UpCmd() *cobra.Command {
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
		RunE: upPostgres,
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

func upPostgres(cmd *cobra.Command, args []string) error {
	// Validate version
	if pgVersion != "16" && pgVersion != "17" {
		return fmt.Errorf("invalid PostgreSQL version: %s (must be 16 or 17)", pgVersion)
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

	// Parse and validate extensions
	var extNames []string
	if extensionList != "" {
		extNames = strings.Split(extensionList, ",")
		for i := range extNames {
			extNames[i] = strings.TrimSpace(extNames[i])
		}

		// Validate extensions
		mgr := extensions.NewManager(pgConfig.Version)
		if err := mgr.ValidateExtensions(extNames); err != nil {
			return err
		}

		// Build custom image with extensions
		customImage, err := buildCustomImage(pgConfig.Version, extNames)
		if err != nil {
			return fmt.Errorf("failed to build custom image: %w", err)
		}
		pgConfig.CustomImage = customImage
	}

	// Determine container name
	containerMgr := container.NewManager()
	containerName := name
	if containerName == "" {
		containerName = containerMgr.Name(pgConfig)
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

	// Create Docker client and run PostgreSQL
	client := docker.NewClient()
	opts := docker.ContainerOptions{
		Name:      containerName,
		ExtraArgs: []string{"--rm"},
	}
	if detach {
		opts.ExtraArgs = append(opts.ExtraArgs, "-d")
	}

	// Mount init.sql if we have extensions
	if len(extNames) > 0 {
		initSQL, err := generateInitSQLContent(extNames)
		if err != nil {
			return fmt.Errorf("failed to generate init SQL: %w", err)
		}
		initFile := filepath.Join(os.TempDir(), fmt.Sprintf("pgbox-init-%s.sql", containerName))
		if err := os.WriteFile(initFile, []byte(initSQL), 0644); err != nil {
			return fmt.Errorf("failed to write init.sql: %w", err)
		}
		// Don't remove the file - let it persist in temp directory
		// The file is small and /tmp is usually cleaned on reboot

		opts.ExtraArgs = append(opts.ExtraArgs, "-v", fmt.Sprintf("%s:/docker-entrypoint-initdb.d/init.sql:ro", initFile))
	}

	return client.RunPostgres(pgConfig, opts)
}

func buildCustomImage(pgVersion string, extNames []string) (string, error) {
	mgr := extensions.NewManager(pgVersion)
	packages := mgr.GetRequiredPackages(extNames)

	// Generate temp directory for build context
	buildDir := filepath.Join(os.TempDir(), fmt.Sprintf("pgbox-build-%d", os.Getpid()))
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	// Write Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	dockerfile, err := generateDockerfileContent(pgVersion, packages)
	if err != nil {
		return "", fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return "", fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Build image
	imageName := fmt.Sprintf("pgbox-pg%s-custom:%d", pgVersion, os.Getpid())
	client := docker.NewClient()

	fmt.Println("Building custom PostgreSQL image with extensions...")
	buildArgs := []string{"build", "-t", imageName, "--build-arg", fmt.Sprintf("PG_MAJOR=%s", pgVersion), buildDir}
	if err := client.RunCommand(buildArgs...); err != nil {
		return "", fmt.Errorf("failed to build Docker image: %w", err)
	}

	return imageName, nil
}

func generateDockerfileContent(pgVersion string, packages []string) (string, error) {
	// Sort packages for consistency
	sort.Strings(packages)

	data := scaffold.DockerfileData{
		PGMajor:     pgVersion,
		HasPackages: len(packages) > 0,
		Packages:    packages,
	}

	content, err := scaffold.GenerateDockerfile(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	return content, nil
}

func generateInitSQLContent(extNames []string) (string, error) {
	var extensions []scaffold.ExtensionInfo
	for _, ext := range extNames {
		extensions = append(extensions, scaffold.ExtensionInfo{
			Name:    ext,
			SQLName: scaffold.MapExtensionToSQLName(ext),
		})
	}

	data := scaffold.InitSQLData{
		Extensions: extensions,
	}

	content, err := scaffold.GenerateInitSQL(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate init SQL: %w", err)
	}

	return content, nil
}
