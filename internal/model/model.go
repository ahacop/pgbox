// Package model provides in-memory representations of Docker artifacts
package model

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// DockerfileModel represents a concrete Dockerfile with anchored, mergeable blocks
type DockerfileModel struct {
	BaseImage   string              // Base Docker image (e.g., "postgres:17")
	AptPackages []string            // Debian/Ubuntu packages to install
	DebURLs     []string            // Direct .deb URLs to download and install
	ZipURLs     []string            // .zip URLs containing .deb packages to download and install
	Blocks      map[string][]string // Named blocks for custom content
}

// NewDockerfileModel creates a new Dockerfile model with defaults
func NewDockerfileModel(baseImage string) *DockerfileModel {
	return &DockerfileModel{
		BaseImage:   baseImage,
		AptPackages: []string{},
		DebURLs:     []string{},
		ZipURLs:     []string{},
		Blocks:      make(map[string][]string),
	}
}

// AddDebURLs adds .deb URLs to download and install
func (d *DockerfileModel) AddDebURLs(urls ...string) {
	d.DebURLs = appendUnique(d.DebURLs, urls...)
}

// AddZipURLs adds .zip URLs (containing .deb packages) to download and install
func (d *DockerfileModel) AddZipURLs(urls ...string) {
	d.ZipURLs = appendUnique(d.ZipURLs, urls...)
}

// AddPackages adds packages to install via apt
func (d *DockerfileModel) AddPackages(packages []string, packageType string) {
	// We only support apt for standard PostgreSQL images
	if packageType == "apt" {
		d.AptPackages = appendUnique(d.AptPackages, packages...)
	}
}

// ComposeModel represents docker-compose.yml configuration
type ComposeModel struct {
	ServiceName string            // Service name (usually "db")
	Image       string            // Docker image or build config
	BuildPath   string            // Path to Dockerfile if building
	Env         map[string]string // Environment variables
	Ports       []string          // Port mappings "host:container"
	Volumes     []string          // Volume mounts
	Networks    []string          // Networks to join
	Anchored    map[string]any    // Anchored blocks for preservation
}

// NewComposeModel creates a new Compose model with defaults
func NewComposeModel(serviceName string) *ComposeModel {
	return &ComposeModel{
		ServiceName: serviceName,
		Env:         make(map[string]string),
		Ports:       []string{},
		Volumes:     []string{},
		Networks:    []string{},
		Anchored:    make(map[string]any),
	}
}

// AddPort adds a port mapping, avoiding duplicates
func (c *ComposeModel) AddPort(port string) {
	for _, p := range c.Ports {
		if p == port {
			return
		}
	}
	c.Ports = append(c.Ports, port)
	sort.Strings(c.Ports)
}

// AddVolume adds a volume mount, avoiding duplicates
func (c *ComposeModel) AddVolume(volume string) {
	for _, v := range c.Volumes {
		if v == volume {
			return
		}
	}
	c.Volumes = append(c.Volumes, volume)
	sort.Strings(c.Volumes)
}

// SetEnv sets an environment variable
func (c *ComposeModel) SetEnv(key, value string) {
	c.Env[key] = value
}

// PGConfModel holds PostgreSQL server configuration
type PGConfModel struct {
	SharedPreload  []string          // shared_preload_libraries values
	GUCs           map[string]string // Generic GUC key-value pairs
	RequireRestart bool              // Whether changes require restart
}

// NewPGConfModel creates a new PostgreSQL config model
func NewPGConfModel() *PGConfModel {
	return &PGConfModel{
		SharedPreload: []string{},
		GUCs:          make(map[string]string),
	}
}

// AddSharedPreload adds libraries to shared_preload_libraries
func (p *PGConfModel) AddSharedPreload(libs ...string) {
	p.SharedPreload = appendUnique(p.SharedPreload, libs...)
	if len(p.SharedPreload) > 0 {
		p.RequireRestart = true
	}
}

// SetGUC sets a PostgreSQL configuration parameter
func (p *PGConfModel) SetGUC(key, value string) error {
	if existing, ok := p.GUCs[key]; ok && existing != value {
		return fmt.Errorf("conflicting values for GUC %s: %q vs %q", key, existing, value)
	}
	p.GUCs[key] = value
	return nil
}

// GetSharedPreloadString returns the shared_preload_libraries value as a string
func (p *PGConfModel) GetSharedPreloadString() string {
	if len(p.SharedPreload) == 0 {
		return ""
	}
	return strings.Join(p.SharedPreload, ",")
}

// InitModel holds ordered SQL initialization fragments
type InitModel struct {
	Fragments []InitFragment
}

// InitFragment represents a SQL initialization fragment
type InitFragment struct {
	Name    string // Fragment identifier (e.g., "pgvector", "pg_cron")
	SHA256  string // SHA256 hash of normalized content
	Content string // SQL content
}

// NewInitModel creates a new init SQL model
func NewInitModel() *InitModel {
	return &InitModel{
		Fragments: []InitFragment{},
	}
}

// AddFragment adds a SQL fragment, avoiding duplicates by hash
func (i *InitModel) AddFragment(name, content string) {
	// Normalize content for consistent hashing
	normalized := strings.TrimSpace(content)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))

	// Check for duplicate hash
	for _, f := range i.Fragments {
		if f.SHA256 == hash {
			return // Skip duplicate
		}
	}

	i.Fragments = append(i.Fragments, InitFragment{
		Name:    name,
		SHA256:  hash,
		Content: content,
	})
}

// GetOrderedFragments returns fragments in a stable order
func (i *InitModel) GetOrderedFragments() []InitFragment {
	// Sort by name for deterministic output
	sorted := make([]InitFragment, len(i.Fragments))
	copy(sorted, i.Fragments)
	sort.Slice(sorted, func(a, b int) bool {
		return sorted[a].Name < sorted[b].Name
	})
	return sorted
}

// Helper function to append unique strings to a slice
func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool)
	for _, s := range slice {
		seen[s] = true
	}

	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}

	sort.Strings(slice)
	return slice
}
