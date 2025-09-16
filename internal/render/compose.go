package render

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/model"
)

// RenderCompose renders a docker-compose.yml from the model
func RenderCompose(m *model.ComposeModel, pgConf *model.PGConfModel, outputPath string) error {
	composePath := filepath.Join(outputPath, "docker-compose.yml")

	// Parse existing file if it exists
	parsed, err := ParseFileWithAnchors(composePath, ComposeAnchors)
	if err != nil {
		return fmt.Errorf("failed to parse existing docker-compose.yml: %w", err)
	}

	// Generate new anchored content
	anchoredContent := generateComposeService(m, pgConf)

	// If no existing file, create default structure
	if !parsed.HasAnchor && len(parsed.PreAnchor) == 0 {
		parsed.PreAnchor = []string{
			"version: '3.8'",
			"",
		}
		// Add volumes section after anchored content
		parsed.PostAnchor = []string{
			"",
			"volumes:",
			"  postgres_data:",
		}
	}

	// Replace anchored content
	lines := ReplaceAnchored(parsed, ComposeAnchors, anchoredContent)

	// Write the file
	return WriteLines(composePath, lines)
}

// generateComposeService generates the service configuration
func generateComposeService(m *model.ComposeModel, pgConf *model.PGConfModel) []string {
	lines := []string{
		"services:",
		fmt.Sprintf("  %s:", m.ServiceName),
	}

	// Image or build configuration
	if m.BuildPath != "" {
		lines = append(lines,
			"    build:",
			fmt.Sprintf("      context: %s", m.BuildPath),
			"      dockerfile: Dockerfile",
		)
		// Extract PG major from image if possible
		if strings.Contains(m.Image, "17") {
			lines = append(lines, "      args:")
			lines = append(lines, "        PG_MAJOR: \"17\"")
		} else if strings.Contains(m.Image, "16") {
			lines = append(lines, "      args:")
			lines = append(lines, "        PG_MAJOR: \"16\"")
		}
	} else if m.Image != "" {
		lines = append(lines, fmt.Sprintf("    image: %s", m.Image))
	}

	// Container name
	containerName := fmt.Sprintf("pgbox-%s", m.ServiceName)
	if m.ServiceName == "db" {
		containerName = "pgbox-postgres"
	}
	lines = append(lines, fmt.Sprintf("    container_name: %s", containerName))

	// Environment variables
	if len(m.Env) > 0 {
		lines = append(lines, "    environment:")
		// Sort keys for consistent output
		var keys []string
		for k := range m.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("      %s: %s", k, m.Env[k]))
		}
	}

	// PostgreSQL configuration via command
	if pgConf != nil && (len(pgConf.SharedPreload) > 0 || len(pgConf.GUCs) > 0) {
		lines = append(lines, "    command:")
		lines = append(lines, "      - postgres")

		// Add shared_preload_libraries if set
		if len(pgConf.SharedPreload) > 0 {
			preloadStr := pgConf.GetSharedPreloadString()
			lines = append(lines, fmt.Sprintf("      - -c"))
			lines = append(lines, fmt.Sprintf("      - shared_preload_libraries=%s", preloadStr))
		}

		// Add other GUCs
		if len(pgConf.GUCs) > 0 {
			// Sort GUC keys for consistent output
			var gucKeys []string
			for k := range pgConf.GUCs {
				gucKeys = append(gucKeys, k)
			}
			sort.Strings(gucKeys)

			for _, k := range gucKeys {
				lines = append(lines, "      - -c")
				lines = append(lines, fmt.Sprintf("      - %s=%s", k, pgConf.GUCs[k]))
			}
		}
	}

	// Ports
	if len(m.Ports) > 0 {
		lines = append(lines, "    ports:")
		for _, port := range m.Ports {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", port))
		}
	}

	// Volumes
	if len(m.Volumes) > 0 {
		lines = append(lines, "    volumes:")
		for _, vol := range m.Volumes {
			lines = append(lines, fmt.Sprintf("      - %s", vol))
		}
	}

	// Health check
	lines = append(lines,
		"    healthcheck:",
		"      test: [\"CMD-SHELL\", \"pg_isready -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-postgres}\"]",
		"      interval: 10s",
		"      timeout: 5s",
		"      retries: 5",
	)

	// Networks (if specified)
	if len(m.Networks) > 0 {
		lines = append(lines, "    networks:")
		for _, net := range m.Networks {
			lines = append(lines, fmt.Sprintf("      - %s", net))
		}
	}

	return lines
}
