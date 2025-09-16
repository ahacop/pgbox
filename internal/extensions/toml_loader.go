package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahacop/pgbox/internal/extspec"
)

// For now, we'll use runtime loading from the filesystem
// TODO: Switch to embedding once the build process is finalized

// TOMLManager manages extensions loaded from embedded TOML files
type TOMLManager struct {
	pgVersion string
	specs     map[string]*extspec.ExtensionSpec
	loader    *extspec.Loader
}

// NewTOMLManager creates a new TOML-based extension manager
func NewTOMLManager(pgVersion string) *TOMLManager {
	return &TOMLManager{
		pgVersion: pgVersion,
		specs:     make(map[string]*extspec.ExtensionSpec),
		loader:    &extspec.Loader{}, // Will use embedded FS
	}
}

// Initialize loads all available extensions for the PostgreSQL version
func (m *TOMLManager) Initialize() error {
	// Use filesystem path for extensions
	extensionsDir := "extensions"

	// Walk through extensions directory
	err := filepath.Walk(extensionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If extensions directory doesn't exist, that's okay for now
			if os.IsNotExist(err) && path == extensionsDir {
				return nil
			}
			return err
		}

		// Skip if not a TOML file
		if info.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		// Check if this is for our PostgreSQL version
		base := filepath.Base(path)
		if !strings.HasSuffix(base, fmt.Sprintf("%s.toml", m.pgVersion)) &&
			!strings.HasSuffix(base, "default.toml") {
			return nil
		}

		// Skip if we already have a version-specific file and this is default.toml
		if strings.HasSuffix(base, "default.toml") {
			extName := filepath.Base(filepath.Dir(path))
			versionSpecific := fmt.Sprintf("%s.toml", m.pgVersion)
			if _, exists := m.specs[extName]; exists {
				return nil // Already loaded version-specific
			}

			// Check if version-specific exists
			versionPath := filepath.Join(filepath.Dir(path), versionSpecific)
			if _, err := os.Stat(versionPath); err == nil {
				return nil // Version-specific exists, skip default
			}
		}

		// Load the spec
		spec, err := m.loadSpec(path)
		if err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to load %s: %v\n", path, err)
			return nil
		}

		// Extract extension name from path
		extName := filepath.Base(filepath.Dir(path))
		m.specs[extName] = spec

		return nil
	})

	return err
}

// loadSpec loads a spec from the filesystem
func (m *TOMLManager) loadSpec(path string) (*extspec.ExtensionSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse and validate using the spec loader
	spec := &extspec.ExtensionSpec{}
	if err := m.loader.ParseAndValidate(data, spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// GetSpec returns the spec for a given extension
func (m *TOMLManager) GetSpec(name string) (*extspec.ExtensionSpec, error) {
	if err := m.Initialize(); err != nil {
		return nil, err
	}

	spec, ok := m.specs[name]
	if !ok {
		return nil, fmt.Errorf("extension %s not found", name)
	}

	return spec, nil
}

// GetSpecs returns specs for multiple extensions
func (m *TOMLManager) GetSpecs(names []string) ([]*extspec.ExtensionSpec, error) {
	if err := m.Initialize(); err != nil {
		return nil, err
	}

	specs := make([]*extspec.ExtensionSpec, 0, len(names))
	for _, name := range names {
		spec, err := m.GetSpec(name)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// ListAvailable returns all available extension names
func (m *TOMLManager) ListAvailable() ([]string, error) {
	if err := m.Initialize(); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(m.specs))
	for name := range m.specs {
		names = append(names, name)
	}

	return names, nil
}

// ValidateExtensions checks if all extensions exist
func (m *TOMLManager) ValidateExtensions(names []string) error {
	if err := m.Initialize(); err != nil {
		return err
	}

	var missing []string
	for _, name := range names {
		if _, ok := m.specs[name]; !ok {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("unknown extensions: %s", strings.Join(missing, ", "))
	}

	return nil
}

// GetRequiredPackages returns all apt packages needed for the extensions
func (m *TOMLManager) GetRequiredPackages(names []string) ([]string, error) {
	specs, err := m.GetSpecs(names)
	if err != nil {
		return nil, err
	}

	packageMap := make(map[string]bool)
	for _, spec := range specs {
		for _, pkg := range spec.Image.AptPackages {
			packageMap[pkg] = true
		}
	}

	packages := make([]string, 0, len(packageMap))
	for pkg := range packageMap {
		packages = append(packages, pkg)
	}

	return packages, nil
}
