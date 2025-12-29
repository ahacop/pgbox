package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ahacop/pgbox/internal/model"
)

// RenderDockerfile renders a Dockerfile from the model
func RenderDockerfile(m *model.DockerfileModel, outputPath string) error {
	dockerfilePath := filepath.Join(outputPath, "Dockerfile")

	// Parse existing file if it exists
	parsed, err := ParseFileWithAnchors(dockerfilePath, DockerfileAnchors)
	if err != nil {
		return fmt.Errorf("failed to parse existing Dockerfile: %w", err)
	}

	// Generate new anchored content
	var anchoredContent []string

	// Generate apt packages installation
	if len(m.AptPackages) > 0 {
		anchoredContent = append(anchoredContent, generateAptInstall(m.BaseImage, m.AptPackages)...)
	}

	// Generate .deb downloads and installation
	if len(m.DebURLs) > 0 {
		anchoredContent = append(anchoredContent, generateDebInstall(m.DebURLs)...)
	}

	// If no existing file, create default structure
	if !parsed.HasAnchor && len(parsed.PreAnchor) == 0 {
		parsed.PreAnchor = generateDefaultDockerfileHeader(m.BaseImage)
	}

	// Replace anchored content
	lines := ReplaceAnchored(parsed, DockerfileAnchors, anchoredContent)

	// Write the file
	return WriteLines(dockerfilePath, lines)
}

// generateDefaultDockerfileHeader creates the default Dockerfile header
func generateDefaultDockerfileHeader(baseImage string) []string {
	// Extract major version from base image
	pgMajor := "18" // default
	if strings.Contains(baseImage, ":16") {
		pgMajor = "16"
	} else if strings.Contains(baseImage, ":17") {
		pgMajor = "17"
	} else if strings.Contains(baseImage, ":18") {
		pgMajor = "18"
	}

	return []string{
		fmt.Sprintf("ARG PG_MAJOR=%s", pgMajor),
		fmt.Sprintf("FROM %s", baseImage),
		"",
	}
}

// generateAptInstall generates apt package installation commands
func generateAptInstall(baseImage string, packages []string) []string {
	if len(packages) == 0 {
		return []string{}
	}

	lines := []string{
		"# Install PostgreSQL extensions",
		"RUN set -eux; \\",
		"    apt-get update; \\",
	}

	// Add PostgreSQL APT repository if we have extension packages
	hasExtensions := false
	for _, pkg := range packages {
		if strings.Contains(pkg, "postgresql-") {
			hasExtensions = true
			break
		}
	}

	if hasExtensions {
		lines = append(lines,
			"    apt-get install -y --no-install-recommends curl gnupg ca-certificates lsb-release; \\",
			"    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql.gpg; \\",
			"    echo \"deb [signed-by=/usr/share/keyrings/postgresql.gpg] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main\" > /etc/apt/sources.list.d/pgdg.list; \\",
			"    apt-get update; \\",
		)
	}

	// Add package installation
	lines = append(lines, "    apt-get install -y --no-install-recommends \\")
	for i, pkg := range packages {
		if i < len(packages)-1 {
			lines = append(lines, fmt.Sprintf("        %s \\", pkg))
		} else {
			lines = append(lines, fmt.Sprintf("        %s; \\", pkg))
		}
	}

	// Clean up
	if hasExtensions {
		lines = append(lines,
			"    apt-get purge -y --auto-remove curl gnupg lsb-release; \\",
		)
	}
	lines = append(lines,
		"    rm -rf /var/lib/apt/lists/*",
	)

	return lines
}

// generateDebInstall generates commands to download and install .deb packages
func generateDebInstall(debURLs []string) []string {
	if len(debURLs) == 0 {
		return []string{}
	}

	lines := []string{
		"",
		"# Install extensions from .deb packages",
		"RUN set -eux; \\",
		"    apt-get update; \\",
		"    apt-get install -y --no-install-recommends curl ca-certificates; \\",
	}

	// Download each .deb file
	for i, url := range debURLs {
		filename := fmt.Sprintf("/tmp/ext_%d.deb", i)
		lines = append(lines, fmt.Sprintf("    curl -fsSL -o %s '%s'; \\", filename, url))
	}

	// Install all .deb files
	var debFiles []string
	for i := range debURLs {
		debFiles = append(debFiles, fmt.Sprintf("/tmp/ext_%d.deb", i))
	}
	lines = append(lines, fmt.Sprintf("    dpkg -i %s || apt-get install -fy; \\", strings.Join(debFiles, " ")))

	// Cleanup
	lines = append(lines,
		"    rm -f /tmp/ext_*.deb; \\",
		"    apt-get purge -y --auto-remove curl ca-certificates; \\",
		"    rm -rf /var/lib/apt/lists/*",
	)

	return lines
}
