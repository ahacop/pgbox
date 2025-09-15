package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/extensions"
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
	if err := generateDockerCompose(targetDir, pgVersion, port); err != nil {
		return err
	}

	// Generate Dockerfile
	if err := generateDockerfile(targetDir, pgVersion, extNames); err != nil {
		return err
	}

	// Generate init.sql
	if err := generateInitSQL(targetDir, extNames); err != nil {
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

func generateDockerCompose(targetDir, pgVersion, port string) error {
	composePath := filepath.Join(targetDir, "docker-compose.yml")

	var compose strings.Builder
	compose.WriteString("version: '3.8'\n\n")
	compose.WriteString("services:\n")
	compose.WriteString("  postgres:\n")
	compose.WriteString("    build:\n")
	compose.WriteString("      context: .\n")
	compose.WriteString("      dockerfile: Dockerfile\n")
	compose.WriteString("      args:\n")
	compose.WriteString(fmt.Sprintf("        PG_MAJOR: %s\n", pgVersion))
	compose.WriteString("    container_name: pgbox-postgres\n")
	compose.WriteString("    environment:\n")
	compose.WriteString("      POSTGRES_DB: postgres\n")
	compose.WriteString("      POSTGRES_USER: postgres\n")
	compose.WriteString("      POSTGRES_PASSWORD: postgres\n")
	compose.WriteString("    ports:\n")
	compose.WriteString(fmt.Sprintf("      - \"%s:5432\"\n", port))
	compose.WriteString("    volumes:\n")
	compose.WriteString("      - postgres_data:/var/lib/postgresql/data\n")
	compose.WriteString("      - ./init.sql:/docker-entrypoint-initdb.d/init.sql\n")
	compose.WriteString("\nvolumes:\n")
	compose.WriteString("  postgres_data:\n")

	if err := os.WriteFile(composePath, []byte(compose.String()), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}

func generateDockerfile(targetDir, pgVersion string, extList []string) error {
	dockerfilePath := filepath.Join(targetDir, "Dockerfile")

	var dockerfile strings.Builder
	dockerfile.WriteString("ARG PG_MAJOR=17\n")
	dockerfile.WriteString("FROM postgres:${PG_MAJOR}\n\n")

	if len(extList) > 0 {
		mgr := extensions.NewManager(pgVersion)
		packages := mgr.GetRequiredPackages(extList)

		if len(packages) > 0 {
			dockerfile.WriteString("# Install PostgreSQL extensions\n")
			dockerfile.WriteString("RUN set -eux; \\\n")
			dockerfile.WriteString("    apt-get update; \\\n")

			// Add PostgreSQL apt repository if we have packages
			dockerfile.WriteString("    apt-get install -y --no-install-recommends curl gnupg ca-certificates lsb-release; \\\n")
			dockerfile.WriteString("    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql.gpg; \\\n")
			dockerfile.WriteString("    echo \"deb [signed-by=/usr/share/keyrings/postgresql.gpg] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main\" > /etc/apt/sources.list.d/pgdg.list; \\\n")
			dockerfile.WriteString("    apt-get update; \\\n")
			dockerfile.WriteString("    apt-get install -y --no-install-recommends \\\n")

			// Sort packages for consistency
			sort.Strings(packages)
			for i, pkg := range packages {
				if i < len(packages)-1 {
					dockerfile.WriteString(fmt.Sprintf("        %s \\\n", pkg))
				} else {
					dockerfile.WriteString(fmt.Sprintf("        %s; \\\n", pkg))
				}
			}

			dockerfile.WriteString("    apt-get purge -y --auto-remove curl gnupg lsb-release; \\\n")
			dockerfile.WriteString("    rm -rf /var/lib/apt/lists/*\n")
		} else if len(extList) > 0 {
			dockerfile.WriteString("# Extensions are builtin - no additional packages needed\n")
		}
	} else {
		dockerfile.WriteString("# This Dockerfile can be customized to add extensions or other PostgreSQL configurations\n")
	}

	if err := os.WriteFile(dockerfilePath, []byte(dockerfile.String()), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return nil
}

func generateInitSQL(targetDir string, extList []string) error {
	initPath := filepath.Join(targetDir, "init.sql")

	var sql strings.Builder
	sql.WriteString("-- Initialize PostgreSQL database\n")

	if len(extList) > 0 {
		sql.WriteString("-- Create extensions\n\n")
		for _, ext := range extList {
			// Map extension names to their actual PostgreSQL names
			pgExtName := ext
			if ext == "pgvector" {
				pgExtName = "vector"
			}
			sql.WriteString(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\";\n", pgExtName))
		}
	} else {
		sql.WriteString("-- Add any custom SQL initialization here\n\n")
		sql.WriteString("-- Example: CREATE EXTENSION IF NOT EXISTS pg_stat_statements;\n")
	}

	if err := os.WriteFile(initPath, []byte(sql.String()), 0644); err != nil {
		return fmt.Errorf("failed to write init.sql: %w", err)
	}

	return nil
}
