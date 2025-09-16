// Package extspec provides TOML schema and loading for PostgreSQL extensions
package extspec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// ExtensionSpec represents a PostgreSQL extension specification
type ExtensionSpec struct {
	// Required identity
	Extension   string `toml:"extension"`    // SQL name for CREATE EXTENSION
	DisplayName string `toml:"display_name"` // Human-friendly name (optional)
	Package     string `toml:"package"`      // Full apt package name (optional)
	Description string `toml:"description"`  // Brief description (optional)
	MinPG       string `toml:"min_pg"`       // Minimum PostgreSQL version (optional)
	MaxPG       string `toml:"max_pg"`       // Maximum PostgreSQL version (optional)

	// Image mutations
	Image ImageSpec `toml:"image"`

	// PostgreSQL configuration - nested structure
	PostgreSQL struct {
		Conf PostgresConfSpec `toml:"conf"`
	} `toml:"postgresql"`

	// Alias for easier access
	PostgresConf PostgresConfSpec `toml:"-"`

	// SQL initialization
	SQL SQLSpec `toml:"sql"`

	// pgbox hints
	PGBox PGBoxSpec `toml:"pgbox"`
}

// ImageSpec defines packages to install in the Docker image
type ImageSpec struct {
	AptPackages []string `toml:"apt_packages"` // Debian packages for standard PostgreSQL images
}

// PostgresConfSpec defines PostgreSQL configuration
type PostgresConfSpec struct {
	SharedPreloadLibraries []string          `toml:"shared_preload_libraries"`
	Extra                  map[string]string `toml:"-"` // Will be populated manually
}

// UnmarshalTOML implements custom TOML unmarshaling to capture extra fields
func (p *PostgresConfSpec) UnmarshalTOML(data interface{}) error {
	// Create a temporary struct for known fields
	type postgresConfAlias struct {
		SharedPreloadLibraries []string `toml:"shared_preload_libraries"`
	}

	var known postgresConfAlias

	// First, decode into the known struct
	if m, ok := data.(map[string]interface{}); ok {
		// Handle shared_preload_libraries
		if v, ok := m["shared_preload_libraries"]; ok {
			if arr, ok := v.([]interface{}); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						known.SharedPreloadLibraries = append(known.SharedPreloadLibraries, s)
					}
				}
			}
			delete(m, "shared_preload_libraries")
		}

		// Copy known fields
		p.SharedPreloadLibraries = known.SharedPreloadLibraries

		// Everything else goes into Extra
		p.Extra = make(map[string]string)
		for k, v := range m {
			if s, ok := v.(string); ok {
				p.Extra[k] = s
			}
		}
	}

	return nil
}

// SQLSpec defines SQL initialization commands
type SQLSpec struct {
	InitDB    []SQLFragment `toml:"initdb"`    // Run during initialization
	PostStart []SQLFragment `toml:"poststart"` // Run after server start
}

// SQLFragment represents a SQL command
type SQLFragment struct {
	Text string `toml:"text"`
}

// PGBoxSpec provides hints to the pgbox engine
type PGBoxSpec struct {
	NeedsRestart bool              `toml:"needs_restart"` // Requires server restart
	ComposeEnv   map[string]string `toml:"compose_env"`   // Docker Compose env vars
	Ports        []string          `toml:"ports"`         // Additional ports to expose
}

// Loader handles loading and validation of extension specs
type Loader struct {
	baseDir string // Base directory for extensions
}

// NewLoader creates a new spec loader
func NewLoader(baseDir string) *Loader {
	return &Loader{
		baseDir: baseDir,
	}
}

// ParseAndValidate parses TOML data and validates the spec
func (l *Loader) ParseAndValidate(data []byte, spec *ExtensionSpec) error {
	// Parse TOML
	if err := toml.Unmarshal(data, spec); err != nil {
		return fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Validate and normalize
	if err := l.validate(spec); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	l.normalize(spec)
	return nil
}

// Load loads an extension spec from a TOML file
func (l *Loader) Load(path string) (*ExtensionSpec, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	// Parse TOML
	var spec ExtensionSpec
	if err := toml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Validate and normalize
	if err := l.validate(&spec); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	l.normalize(&spec)

	return &spec, nil
}

// LoadExtension loads a spec for a specific extension and PostgreSQL version
func (l *Loader) LoadExtension(name string, pgMajor string) (*ExtensionSpec, error) {
	// Try version-specific file first
	path := filepath.Join(l.baseDir, name, fmt.Sprintf("%s.toml", pgMajor))
	if _, err := os.Stat(path); err == nil {
		return l.Load(path)
	}

	// Fall back to default.toml
	path = filepath.Join(l.baseDir, name, "default.toml")
	if _, err := os.Stat(path); err == nil {
		return l.Load(path)
	}

	return nil, fmt.Errorf("no spec found for extension %s (PostgreSQL %s)", name, pgMajor)
}

// LoadMultiple loads multiple extension specs
func (l *Loader) LoadMultiple(extensions []string, pgMajor string) ([]*ExtensionSpec, error) {
	specs := make([]*ExtensionSpec, 0, len(extensions))

	for _, ext := range extensions {
		spec, err := l.LoadExtension(ext, pgMajor)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", ext, err)
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// validate checks that a spec is valid
func (l *Loader) validate(spec *ExtensionSpec) error {
	// Extension name is required
	if spec.Extension == "" {
		return fmt.Errorf("extension name is required")
	}

	// Validate extension name (PostgreSQL identifier rules)
	if !isValidIdentifier(spec.Extension) {
		return fmt.Errorf("invalid extension name: %s", spec.Extension)
	}

	// Validate GUC keys
	for key := range spec.PostgresConf.Extra {
		if !isValidGUCKey(key) {
			return fmt.Errorf("invalid GUC key: %s", key)
		}
	}

	// Validate port format
	for _, port := range spec.PGBox.Ports {
		if !isValidPort(port) {
			return fmt.Errorf("invalid port format: %s", port)
		}
	}

	// Validate SQL fragments are non-empty
	for _, frag := range spec.SQL.InitDB {
		if strings.TrimSpace(frag.Text) == "" {
			return fmt.Errorf("empty SQL fragment in initdb")
		}
	}
	for _, frag := range spec.SQL.PostStart {
		if strings.TrimSpace(frag.Text) == "" {
			return fmt.Errorf("empty SQL fragment in poststart")
		}
	}

	return nil
}

// normalize cleans up and standardizes the spec
func (l *Loader) normalize(spec *ExtensionSpec) {
	// Copy PostgreSQL.Conf to PostgresConf for easier access
	spec.PostgresConf = spec.PostgreSQL.Conf

	// Set display name if not provided
	if spec.DisplayName == "" {
		spec.DisplayName = spec.Extension
	}

	// Sort and dedupe package list
	spec.Image.AptPackages = dedupeSort(spec.Image.AptPackages)

	// Sort and dedupe shared preload libraries
	spec.PostgresConf.SharedPreloadLibraries = dedupeSort(spec.PostgresConf.SharedPreloadLibraries)

	// Infer needs_restart if shared_preload_libraries is set
	if len(spec.PostgresConf.SharedPreloadLibraries) > 0 && !spec.PGBox.NeedsRestart {
		spec.PGBox.NeedsRestart = true
	}

	// Initialize maps if nil
	if spec.PostgresConf.Extra == nil {
		spec.PostgresConf.Extra = make(map[string]string)
	}
	if spec.PGBox.ComposeEnv == nil {
		spec.PGBox.ComposeEnv = make(map[string]string)
	}

	// Trim whitespace from SQL fragments
	for i := range spec.SQL.InitDB {
		spec.SQL.InitDB[i].Text = strings.TrimSpace(spec.SQL.InitDB[i].Text)
	}
	for i := range spec.SQL.PostStart {
		spec.SQL.PostStart[i].Text = strings.TrimSpace(spec.SQL.PostStart[i].Text)
	}
}

// Helper functions

func isValidIdentifier(s string) bool {
	// PostgreSQL identifier: letters, digits, underscore, hyphen
	// Must start with letter or underscore
	// Hyphens are allowed in extension names like uuid-ossp
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, s)
	return match
}

func isValidGUCKey(s string) bool {
	// GUC keys: letters, digits, underscore, dot
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_.]*$`, s)
	return match
}

func isValidPort(s string) bool {
	// Port format: host:container or host:container/proto
	match, _ := regexp.MatchString(`^\d+:\d+(/\w+)?$`, s)
	return match
}

func dedupeSort(items []string) []string {
	if len(items) == 0 {
		return items
	}

	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	sort.Strings(result)
	return result
}
