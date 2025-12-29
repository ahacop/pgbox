package orchestrator

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/ahacop/pgbox/internal/model"
	"github.com/ahacop/pgbox/internal/render"
	"github.com/ahacop/pgbox/internal/util"
)

// ExportConfig holds configuration for the export command.
type ExportConfig struct {
	TargetDir  string
	Version    string
	Port       string
	Extensions []string
	BaseImage  string
	// Environment overrides
	User     string
	Password string
	Database string
}

// ExportOrchestrator handles exporting Docker configurations.
type ExportOrchestrator struct {
	output io.Writer
}

// NewExportOrchestrator creates a new ExportOrchestrator.
func NewExportOrchestrator(w io.Writer) *ExportOrchestrator {
	return &ExportOrchestrator{output: w}
}

// Run exports Docker configuration to the target directory.
func (o *ExportOrchestrator) Run(cfg ExportConfig) error {
	baseImage := cfg.BaseImage
	if baseImage == "" {
		baseImage = extensions.GetBaseImage(cfg.Extensions, cfg.Version)
		if baseImage == "" {
			baseImage = fmt.Sprintf("postgres:%s", cfg.Version)
		}
	}

	pgConfig := config.NewPostgresConfig()
	pgConfig.Version = cfg.Version
	pgConfig.Port = cfg.Port
	if cfg.User != "" {
		pgConfig.User = cfg.User
	}
	if cfg.Password != "" {
		pgConfig.Password = cfg.Password
	}
	if cfg.Database != "" {
		pgConfig.Database = cfg.Database
	}

	if err := os.MkdirAll(cfg.TargetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dockerfileModel := model.NewDockerfileModel(baseImage)
	composeModel := model.NewComposeModel("db")
	pgConfModel := model.NewPGConfModel()
	initModel := model.NewInitModel()

	composeModel.BuildPath = "."
	composeModel.Image = baseImage
	composeModel.AddPort(fmt.Sprintf("%s:5432", cfg.Port))
	composeModel.AddVolume("postgres_data:/var/lib/postgresql/data")
	composeModel.AddVolume("./init.sql:/docker-entrypoint-initdb.d/init.sql:ro")
	composeModel.SetEnv("POSTGRES_USER", pgConfig.User)
	composeModel.SetEnv("POSTGRES_PASSWORD", pgConfig.Password)
	composeModel.SetEnv("POSTGRES_DB", pgConfig.Database)

	if len(cfg.Extensions) > 0 {
		if err := o.processExtensions(cfg.Version, cfg.Extensions, dockerfileModel, pgConfModel, initModel); err != nil {
			return err
		}
	}

	if err := render.RenderDockerfile(dockerfileModel, cfg.TargetDir); err != nil {
		return fmt.Errorf("failed to render Dockerfile: %w", err)
	}

	if err := render.RenderCompose(composeModel, pgConfModel, cfg.TargetDir); err != nil {
		return fmt.Errorf("failed to render docker-compose.yml: %w", err)
	}

	if err := render.RenderInitSQL(initModel, cfg.TargetDir); err != nil {
		return fmt.Errorf("failed to render init.sql: %w", err)
	}

	if len(pgConfModel.SharedPreload) > 0 || len(pgConfModel.GUCs) > 0 {
		if err := render.RenderPostgreSQLConf(pgConfModel, cfg.TargetDir); err != nil {
			return fmt.Errorf("failed to render postgresql.conf: %w", err)
		}
	}

	o.printSuccess(cfg, pgConfModel)

	return nil
}

// processExtensions loads and applies extension configurations.
func (o *ExportOrchestrator) processExtensions(
	pgVersion string,
	extNames []string,
	dockerfileModel *model.DockerfileModel,
	pgConfModel *model.PGConfModel,
	initModel *model.InitModel,
) error {
	if err := extensions.ValidateExtensions(extNames); err != nil {
		return err
	}

	packages := extensions.GetPackages(extNames, pgVersion)
	if len(packages) > 0 {
		dockerfileModel.AddPackages(packages, "apt")
	}

	debURLs := extensions.GetDebURLs(extNames, pgVersion, util.GetDebArch())
	if len(debURLs) > 0 {
		dockerfileModel.AddDebURLs(debURLs...)
	}

	zipURLs := extensions.GetZipURLs(extNames, pgVersion, util.GetDebArch())
	if len(zipURLs) > 0 {
		dockerfileModel.AddZipURLs(zipURLs...)
	}

	preload := extensions.GetPreloadLibraries(extNames)
	if len(preload) > 0 {
		pgConfModel.AddSharedPreload(preload...)
	}

	gucs, err := extensions.GetGUCs(extNames)
	if err != nil {
		return fmt.Errorf("extension configuration conflict: %w", err)
	}
	for key, value := range gucs {
		pgConfModel.GUCs[key] = value
	}

	for _, name := range extNames {
		sql := extensions.GetInitSQL(name)
		if sql != "" {
			initModel.AddFragment(name+"-init", sql)
		}
	}

	return nil
}

// printSuccess prints the success message.
func (o *ExportOrchestrator) printSuccess(cfg ExportConfig, pgConfModel *model.PGConfModel) {
	fmt.Fprintf(o.output, "Exported Docker configuration to %s\n", cfg.TargetDir)
	if len(cfg.Extensions) > 0 {
		fmt.Fprintf(o.output, "With extensions: %s\n", strings.Join(cfg.Extensions, ", "))
	}
	fmt.Fprintf(o.output, "\nTo start PostgreSQL:\n")
	fmt.Fprintf(o.output, "  cd %s\n", cfg.TargetDir)
	fmt.Fprintf(o.output, "  docker-compose up -d\n")

	if pgConfModel.RequireRestart {
		fmt.Fprintf(o.output, "\nNote: Some extensions require server configuration changes.\n")
		fmt.Fprintf(o.output, "The container will start with the required settings.\n")
	}
}
