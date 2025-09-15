package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Catalog struct {
	GeneratedAt string      `json:"generated_at"`
	Source      string      `json:"source"`
	PgMajor     int         `json:"pg_major"`
	Entries     []Extension `json:"entries"`
}

type Extension struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Pkg         string `json:"pkg,omitempty"`
	Description string `json:"description"`
}

// Manager handles extension metadata and package resolution
type Manager struct {
	pgVersion   string
	extensions  map[string]Extension
	initialized bool
}

// NewManager creates a new extension manager
func NewManager(pgVersion string) *Manager {
	return &Manager{
		pgVersion:  pgVersion,
		extensions: make(map[string]Extension),
	}
}

// Initialize loads extension data from JSON files
func (m *Manager) Initialize() error {
	if m.initialized {
		return nil
	}

	// Load builtin extensions
	builtinPath := filepath.Join("pgbox-data", "builtin", fmt.Sprintf("pg%s.json", m.pgVersion))
	if err := m.loadCatalog(builtinPath); err != nil {
		return fmt.Errorf("failed to load builtin extensions: %w", err)
	}

	// Load apt extensions
	aptPath := filepath.Join("pgbox-data", "apt-pgdg", fmt.Sprintf("pg%s.json", m.pgVersion))
	if err := m.loadCatalog(aptPath); err != nil {
		// Not fatal - apt extensions are optional
		fmt.Fprintf(os.Stderr, "Warning: failed to load apt extensions: %v\n", err)
	}

	m.initialized = true
	return nil
}

func (m *Manager) loadCatalog(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	for _, ext := range catalog.Entries {
		// Builtin takes precedence
		if existing, exists := m.extensions[ext.Name]; !exists || ext.Kind == "builtin" {
			m.extensions[ext.Name] = ext
		} else if exists && existing.Kind != "builtin" && ext.Pkg != "" {
			// Update package info if more complete
			m.extensions[ext.Name] = ext
		}
	}

	return nil
}

// IsBuiltin checks if an extension is builtin
func (m *Manager) IsBuiltin(name string) bool {
	if err := m.Initialize(); err != nil {
		return false
	}
	ext, exists := m.extensions[name]
	return exists && ext.Kind == "builtin"
}

// GetPackage returns the package name for an extension
func (m *Manager) GetPackage(name string) string {
	if err := m.Initialize(); err != nil {
		return ""
	}
	ext, exists := m.extensions[name]
	if !exists {
		return ""
	}
	return ext.Pkg
}

// ValidateExtensions checks if all extensions in the list are valid
func (m *Manager) ValidateExtensions(extensions []string) error {
	if err := m.Initialize(); err != nil {
		return err
	}

	var unknown []string
	for _, name := range extensions {
		if _, exists := m.extensions[name]; !exists {
			unknown = append(unknown, name)
		}
	}

	if len(unknown) > 0 {
		return fmt.Errorf("unknown extensions: %s", strings.Join(unknown, ", "))
	}
	return nil
}

// GetRequiredPackages returns apt packages needed for the extensions
func (m *Manager) GetRequiredPackages(extensions []string) []string {
	if err := m.Initialize(); err != nil {
		return nil
	}

	packages := make(map[string]bool)
	for _, name := range extensions {
		if pkg := m.GetPackage(name); pkg != "" {
			packages[pkg] = true
		}
	}

	var result []string
	for pkg := range packages {
		result = append(result, pkg)
	}
	return result
}
