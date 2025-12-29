// Package orchestrator contains the business logic for pgbox commands.
package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/container"
	"github.com/ahacop/pgbox/internal/docker"
	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/ahacop/pgbox/internal/model"
	"github.com/ahacop/pgbox/internal/render"
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

// UpConfig holds the configuration for starting a PostgreSQL container.
type UpConfig struct {
	Version       string
	Port          string
	ContainerName string
	Password      string
	Database      string
	User          string
	Detach        bool
	Extensions    []string
}

// UpOrchestrator handles the business logic for starting PostgreSQL containers.
type UpOrchestrator struct {
	docker       docker.Docker
	containerMgr *container.Manager
}

// NewUpOrchestrator creates a new UpOrchestrator with the given dependencies.
func NewUpOrchestrator(d docker.Docker) *UpOrchestrator {
	return &UpOrchestrator{
		docker:       d,
		containerMgr: container.NewManager(),
	}
}

// Run starts a PostgreSQL container with the given configuration.
func (o *UpOrchestrator) Run(cfg UpConfig) error {
	// Create PostgreSQL config
	pgConfig := config.NewPostgresConfig()
	pgConfig.Version = cfg.Version
	if cfg.Port != "" {
		pgConfig.Port = cfg.Port
	}
	if cfg.Database != "" {
		pgConfig.Database = cfg.Database
	}
	if cfg.User != "" {
		pgConfig.User = cfg.User
	}
	if cfg.Password != "" {
		pgConfig.Password = cfg.Password
	}

	// Determine container name
	containerName := cfg.ContainerName
	if containerName == "" {
		containerName = o.containerMgr.Name(pgConfig, cfg.Extensions)
	}

	// Check if container already exists (stopped)
	if restarted, err := o.tryRestartExisting(containerName); err != nil {
		return err
	} else if restarted {
		return nil
	}

	// Initialize models
	// Check if extensions require a specific base image
	baseImage := extensions.GetBaseImage(cfg.Extensions, cfg.Version)
	if baseImage == "" {
		baseImage = fmt.Sprintf("postgres:%s", cfg.Version)
	}
	dockerfileModel := model.NewDockerfileModel(baseImage)
	pgConfModel := model.NewPGConfModel()
	initModel := model.NewInitModel()

	// Process extensions if specified
	if len(cfg.Extensions) > 0 {
		if err := o.processExtensions(cfg.Version, cfg.Extensions, dockerfileModel, pgConfModel, initModel, pgConfig); err != nil {
			return err
		}
	}

	// Print status
	o.printStatus(pgConfig, containerName, cfg.Extensions, cfg.Detach)

	// Build container options
	opts := o.buildContainerOptions(containerName, cfg.Detach, cfg.Extensions, pgConfModel, initModel)

	return o.docker.RunPostgres(pgConfig, opts)
}

// tryRestartExisting checks if a container exists and restarts it if so.
// Returns (restarted, error).
func (o *UpOrchestrator) tryRestartExisting(containerName string) (bool, error) {
	existingOutput, _ := o.docker.RunCommandWithOutput("ps", "-a", "--filter", fmt.Sprintf("name=^%s$", containerName), "--format", "{{.Names}}")
	if strings.TrimSpace(existingOutput) == containerName {
		fmt.Printf("Restarting existing container: %s\n", containerName)
		if err := o.docker.RunCommand("start", containerName); err != nil {
			return false, fmt.Errorf("failed to restart container: %w", err)
		}
		fmt.Printf("Container %s restarted successfully\n", containerName)
		return true, nil
	}
	return false, nil
}

// processExtensions loads and applies extension configurations using the Go catalog.
func (o *UpOrchestrator) processExtensions(
	pgVersion string,
	extNames []string,
	dockerfileModel *model.DockerfileModel,
	pgConfModel *model.PGConfModel,
	initModel *model.InitModel,
	pgConfig *config.PostgresConfig,
) error {
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

	// Build custom image if packages or .deb URLs are needed
	if len(packages) > 0 || len(debURLs) > 0 {
		customImage, err := o.buildCustomImage(pgVersion, dockerfileModel, extNames)
		if err != nil {
			return fmt.Errorf("failed to build custom image: %w", err)
		}
		pgConfig.CustomImage = customImage
	}

	return nil
}

// buildCustomImage builds a Docker image with the specified extensions.
func (o *UpOrchestrator) buildCustomImage(pgVersion string, dockerfileModel *model.DockerfileModel, extensions []string) (string, error) {
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
	imageName := o.containerMgr.ImageName(pgVersion, extensions)

	// Check if image already exists
	existingImages, _ := o.docker.RunCommandWithOutput("images", "-q", imageName)
	if strings.TrimSpace(existingImages) != "" {
		fmt.Printf("Using existing custom image: %s\n", imageName)
		return imageName, nil
	}

	fmt.Println("Building custom PostgreSQL image with extensions...")
	buildArgs := []string{"build", "-t", imageName, "--build-arg", fmt.Sprintf("PG_MAJOR=%s", pgVersion), buildDir}
	if err := o.docker.RunCommand(buildArgs...); err != nil {
		return "", fmt.Errorf("failed to build Docker image: %w", err)
	}

	return imageName, nil
}

// printStatus prints the startup status to stdout.
func (o *UpOrchestrator) printStatus(pgConfig *config.PostgresConfig, containerName string, extensions []string, detach bool) {
	fmt.Printf("Starting PostgreSQL %s...\n", pgConfig.Version)
	fmt.Printf("Container: %s\n", containerName)
	fmt.Printf("Port: %s\n", pgConfig.Port)
	fmt.Printf("User: %s\n", pgConfig.User)
	fmt.Printf("Database: %s\n", pgConfig.Database)
	if len(extensions) > 0 {
		fmt.Printf("Extensions: %s\n", strings.Join(extensions, ", "))
	}

	if !detach {
		fmt.Println("\nPress Ctrl+C to stop the container")
	} else {
		fmt.Printf("\nRunning in background. Use 'pgbox down -n %s' to stop.\n", containerName)
	}
	fmt.Println(strings.Repeat("-", 40))
}

// buildContainerOptions builds the Docker container options.
func (o *UpOrchestrator) buildContainerOptions(
	containerName string,
	detach bool,
	extensions []string,
	pgConfModel *model.PGConfModel,
	initModel *model.InitModel,
) docker.ContainerOptions {
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
	if len(extensions) > 0 {
		o.configureExtensions(&opts, containerName, pgConfModel, initModel)
	}

	return opts
}

// configureExtensions adds extension-specific configuration to container options.
func (o *UpOrchestrator) configureExtensions(
	opts *docker.ContainerOptions,
	containerName string,
	pgConfModel *model.PGConfModel,
	initModel *model.InitModel,
) {
	// Generate and mount init.sql
	initFile := filepath.Join(os.TempDir(), fmt.Sprintf("pgbox-init-%s.sql", containerName))
	if err := render.RenderInitSQL(initModel, os.TempDir()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to render init SQL: %v\n", err)
		return
	}

	// Move the generated init.sql to the right location
	generatedInitPath := filepath.Join(os.TempDir(), "init.sql")
	initContent, err := os.ReadFile(generatedInitPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read generated init.sql: %v\n", err)
		return
	}
	if err := os.WriteFile(initFile, initContent, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write init.sql: %v\n", err)
		return
	}
	if err := os.Remove(generatedInitPath); err != nil {
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
		if key == "shared_preload_libraries" {
			continue
		}
		opts.Command = append(opts.Command, "-c", fmt.Sprintf("%s=%s", key, value))
	}
}
