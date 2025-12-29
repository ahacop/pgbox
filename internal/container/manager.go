package container

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/extensions"
)

// Manager handles container lifecycle and naming
type Manager struct{}

// NewManager creates a new container manager
func NewManager() *Manager {
	return &Manager{}
}

// extensionHash generates a deterministic hash from sorted extension names AND their configs.
// This ensures the image is rebuilt when extension configurations change.
func extensionHash(extNames []string) string {
	if len(extNames) == 0 {
		return ""
	}

	sorted := make([]string, len(extNames))
	copy(sorted, extNames)
	sort.Strings(sorted)

	h := sha256.New()
	for _, name := range sorted {
		h.Write([]byte(name))
		if ext, ok := extensions.Get(name); ok {
			h.Write([]byte(ext.Package))
			h.Write([]byte(strings.Join(ext.Preload, ",")))
			var gucKeys []string
			for k := range ext.GUCs {
				gucKeys = append(gucKeys, k)
			}
			sort.Strings(gucKeys)
			for _, k := range gucKeys {
				h.Write([]byte(k + "=" + ext.GUCs[k]))
			}
			h.Write([]byte(ext.InitSQL))
		}
	}

	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:8])
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
		return fmt.Sprintf("postgres:%s", version)
	}
	hash := extensionHash(extensions)
	return fmt.Sprintf("pgbox-pg%s-custom:%s", version, hash)
}

// DefaultName returns the default container name for the default PostgreSQL version
func (m *Manager) DefaultName() string {
	return fmt.Sprintf("pgbox-pg%s", config.DefaultVersion)
}

// ErrNoContainerFound is returned when no suitable container is found
var ErrNoContainerFound = errors.New("no pgbox or postgres container found")

// SelectPgboxContainer selects the best pgbox container from docker ps output.
// Priority: 1) containers starting with "pgbox-", 2) any postgres container
func SelectPgboxContainer(dockerPsOutput string) (string, error) {
	if dockerPsOutput == "" {
		return "", ErrNoContainerFound
	}

	lines := strings.Split(dockerPsOutput, "\n")

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			if strings.HasPrefix(name, "pgbox-") {
				return name, nil
			}
		}
	}

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			image := strings.TrimSpace(parts[1])
			if strings.HasPrefix(image, "postgres:") || strings.HasPrefix(image, "pgbox-pg") {
				return name, nil
			}
		}
	}

	return "", ErrNoContainerFound
}
