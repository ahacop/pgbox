package container

import (
	"fmt"

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
