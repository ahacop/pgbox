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

	// Determine which package manager to use
	packageManager := m.GetPackageManager()
	var packages []string

	switch packageManager {
	case "apt":
		packages = m.AptPackages
		if len(packages) > 0 {
			anchoredContent = generateAptInstall(m.BaseImage, packages)
		}
	case "apk":
		packages = m.ApkPackages
		if len(packages) > 0 {
			anchoredContent = generateApkInstall(packages)
		}
	case "yum":
		packages = m.YumPackages
		if len(packages) > 0 {
			anchoredContent = generateYumInstall(packages)
		}
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
	pgMajor := "17" // default
	if strings.Contains(baseImage, ":16") {
		pgMajor = "16"
	} else if strings.Contains(baseImage, ":17") {
		pgMajor = "17"
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

// generateApkInstall generates apk package installation commands
func generateApkInstall(packages []string) []string {
	if len(packages) == 0 {
		return []string{}
	}

	lines := []string{
		"# Install PostgreSQL extensions",
		"RUN set -eux; \\",
		"    apk add --no-cache \\",
	}

	for i, pkg := range packages {
		if i < len(packages)-1 {
			lines = append(lines, fmt.Sprintf("        %s \\", pkg))
		} else {
			lines = append(lines, fmt.Sprintf("        %s", pkg))
		}
	}

	return lines
}

// generateYumInstall generates yum package installation commands
func generateYumInstall(packages []string) []string {
	if len(packages) == 0 {
		return []string{}
	}

	lines := []string{
		"# Install PostgreSQL extensions",
		"RUN set -eux; \\",
		"    yum install -y \\",
	}

	for i, pkg := range packages {
		if i < len(packages)-1 {
			lines = append(lines, fmt.Sprintf("        %s \\", pkg))
		} else {
			lines = append(lines, fmt.Sprintf("        %s; \\", pkg))
		}
	}

	lines = append(lines,
		"    yum clean all",
	)

	return lines
}