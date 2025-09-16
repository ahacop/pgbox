package container

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
)

// Manager handles container lifecycle and naming
type Manager struct{}

// NewManager creates a new container manager
func NewManager() *Manager {
	return &Manager{}
}

// extensionHash generates a deterministic hash from sorted extension names
func extensionHash(extensions []string) string {
	if len(extensions) == 0 {
		return ""
	}

	// Sort extensions to ensure deterministic hash
	sorted := make([]string, len(extensions))
	copy(sorted, extensions)
	sort.Strings(sorted)

	// Create hash from sorted, comma-separated extensions
	h := sha256.Sum256([]byte(strings.Join(sorted, ",")))
	// Use first 8 bytes (16 hex chars) for readability
	return hex.EncodeToString(h[:8])
}

// Name returns the container name for a PostgreSQL configuration with optional extensions
func (m *Manager) Name(cfg *config.PostgresConfig, extensions []string) string {
	base := fmt.Sprintf("pgbox-pg%s", cfg.Version)
	if hash := extensionHash(extensions); hash != "" {
		return fmt.Sprintf("%s-%s", base, hash)
	}
	return base
}

// ImageName returns the Docker image name for the given version and extensions
func (m *Manager) ImageName(version string, extensions []string) string {
	if len(extensions) == 0 {
		// No extensions, use standard postgres image
		return fmt.Sprintf("postgres:%s", version)
	}
	// Extensions require custom image with deterministic tag
	hash := extensionHash(extensions)
	return fmt.Sprintf("pgbox-pg%s-custom:%s", version, hash)
}

// DefaultName returns the default container name (for PostgreSQL 17)
func (m *Manager) DefaultName() string {
	return "pgbox-pg17"
}

// ErrNoContainerFound is returned when no suitable container is found
var ErrNoContainerFound = errors.New("no pgbox or postgres container found")

// SelectPgboxContainer selects the best pgbox container from docker ps output
// This is pure business logic with no side effects
// Priority: 1) containers starting with "pgbox-", 2) any postgres container
func SelectPgboxContainer(dockerPsOutput string) (string, error) {
	if dockerPsOutput == "" {
		return "", ErrNoContainerFound
	}

	lines := strings.Split(dockerPsOutput, "\n")

	// First priority: containers starting with "pgbox-"
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			if strings.HasPrefix(name, "pgbox-") {
				return name, nil
			}
		}
	}

	// Second priority: any container with postgres or pgbox custom image
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			image := strings.TrimSpace(parts[1])
			// Match both standard postgres images and our custom pgbox images
			if strings.HasPrefix(image, "postgres:") || strings.HasPrefix(image, "pgbox-pg") {
				return name, nil
			}
		}
	}

	return "", ErrNoContainerFound
}
