package container

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
)

// Manager handles container lifecycle and naming
type Manager struct{}

// NewManager creates a new container manager
func NewManager() *Manager {
	return &Manager{}
}

// Name returns the container name for a PostgreSQL configuration
func (m *Manager) Name(cfg *config.PostgresConfig) string {
	return fmt.Sprintf("pgbox-pg%s", cfg.Version)
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
