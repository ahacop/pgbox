// Package applier applies extension specifications to in-memory models
package applier

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/extspec"
	"github.com/ahacop/pgbox/internal/model"
)

// Applier handles applying extension specs to models
type Applier struct {
	conflicts []Conflict // Track conflicts encountered
}

// Conflict represents a configuration conflict between extensions
type Conflict struct {
	Type       string   // Type of conflict (e.g., "GUC")
	Key        string   // Conflicting key
	Extensions []string // Extensions involved
	Values     []string // Conflicting values
}

// New creates a new applier
func New() *Applier {
	return &Applier{
		conflicts: []Conflict{},
	}
}

// Apply applies multiple extension specs to the models
func (a *Applier) Apply(specs []*extspec.ExtensionSpec, dockerfile *model.DockerfileModel, compose *model.ComposeModel, pgconf *model.PGConfModel, initSQL *model.InitModel) error {
	// Track GUC values by extension for conflict detection
	gucSources := make(map[string]map[string]string) // guc -> extension -> value

	for _, spec := range specs {
		// Apply image packages
		if err := a.applyImagePackages(spec, dockerfile); err != nil {
			return fmt.Errorf("failed to apply packages for %s: %w", spec.Extension, err)
		}

		// Apply PostgreSQL configuration
		if err := a.applyPGConf(spec, pgconf, gucSources); err != nil {
			return fmt.Errorf("failed to apply pgconf for %s: %w", spec.Extension, err)
		}

		// Apply init SQL
		if err := a.applyInitSQL(spec, initSQL); err != nil {
			return fmt.Errorf("failed to apply init SQL for %s: %w", spec.Extension, err)
		}

		// Apply compose hints
		if err := a.applyComposeHints(spec, compose); err != nil {
			return fmt.Errorf("failed to apply compose hints for %s: %w", spec.Extension, err)
		}
	}

	// Check for conflicts
	if len(a.conflicts) > 0 {
		return a.formatConflictError()
	}

	return nil
}

// applyImagePackages applies package requirements to the Dockerfile model
func (a *Applier) applyImagePackages(spec *extspec.ExtensionSpec, dockerfile *model.DockerfileModel) error {
	// Determine which packages to use based on the base image
	packageManager := dockerfile.GetPackageManager()

	switch packageManager {
	case "apt":
		dockerfile.AddPackages(spec.Image.AptPackages, "apt")
	case "apk":
		dockerfile.AddPackages(spec.Image.ApkPackages, "apk")
	case "yum":
		dockerfile.AddPackages(spec.Image.YumPackages, "yum")
	default:
		// If we can't determine, use apt as default
		if len(spec.Image.AptPackages) > 0 {
			dockerfile.AddPackages(spec.Image.AptPackages, "apt")
		}
	}

	return nil
}

// applyPGConf applies PostgreSQL configuration
func (a *Applier) applyPGConf(spec *extspec.ExtensionSpec, pgconf *model.PGConfModel, gucSources map[string]map[string]string) error {
	// Add shared preload libraries
	if len(spec.PostgresConf.SharedPreloadLibraries) > 0 {
		pgconf.AddSharedPreload(spec.PostgresConf.SharedPreloadLibraries...)
	}

	// Apply GUCs with conflict detection
	for key, value := range spec.PostgresConf.Extra {
		// Skip shared_preload_libraries as it's handled separately
		if key == "shared_preload_libraries" {
			continue
		}

		// Track the source of this GUC value
		if gucSources[key] == nil {
			gucSources[key] = make(map[string]string)
		}
		gucSources[key][spec.Extension] = value

		// Check for conflicts
		if existing, ok := pgconf.GUCs[key]; ok && existing != value {
			// Collect all extensions that set this GUC
			var extensions []string
			var values []string
			for ext, val := range gucSources[key] {
				extensions = append(extensions, ext)
				values = append(values, val)
			}

			a.conflicts = append(a.conflicts, Conflict{
				Type:       "GUC",
				Key:        key,
				Extensions: extensions,
				Values:     values,
			})
			continue
		}

		// Set the GUC
		pgconf.GUCs[key] = value
	}

	// Update restart requirement
	if spec.PGBox.NeedsRestart {
		pgconf.RequireRestart = true
	}

	return nil
}

// applyInitSQL applies SQL initialization fragments
func (a *Applier) applyInitSQL(spec *extspec.ExtensionSpec, initSQL *model.InitModel) error {
	// Add initdb fragments
	for _, frag := range spec.SQL.InitDB {
		if frag.Text != "" {
			fragmentName := fmt.Sprintf("%s-init", spec.Extension)
			initSQL.AddFragment(fragmentName, frag.Text)
		}
	}

	// Add poststart fragments (tagged differently)
	for i, frag := range spec.SQL.PostStart {
		if frag.Text != "" {
			fragmentName := fmt.Sprintf("%s-post-%d", spec.Extension, i)
			initSQL.AddFragment(fragmentName, frag.Text)
		}
	}

	return nil
}

// applyComposeHints applies Docker Compose configuration hints
func (a *Applier) applyComposeHints(spec *extspec.ExtensionSpec, compose *model.ComposeModel) error {
	// Apply environment variables (last-writer-wins with warning)
	for key, value := range spec.PGBox.ComposeEnv {
		if existing, ok := compose.Env[key]; ok && existing != value {
			// Log warning but don't fail
			fmt.Printf("Warning: Environment variable %s redefined by %s (was: %s, now: %s)\n",
				key, spec.Extension, existing, value)
		}
		compose.SetEnv(key, value)
	}

	// Add ports
	for _, port := range spec.PGBox.Ports {
		compose.AddPort(port)
	}

	return nil
}

// formatConflictError formats conflicts into a user-friendly error message
func (a *Applier) formatConflictError() error {
	var messages []string

	for _, conflict := range a.conflicts {
		switch conflict.Type {
		case "GUC":
			msg := fmt.Sprintf("GUC '%s' has conflicting values:", conflict.Key)
			for i, ext := range conflict.Extensions {
				msg += fmt.Sprintf("\n  - %s: %s", ext, conflict.Values[i])
			}
			messages = append(messages, msg)
		default:
			msg := fmt.Sprintf("%s conflict on '%s' between: %s",
				conflict.Type, conflict.Key, strings.Join(conflict.Extensions, ", "))
			messages = append(messages, msg)
		}
	}

	return fmt.Errorf("configuration conflicts detected:\n%s", strings.Join(messages, "\n"))
}

// GetConflicts returns any conflicts encountered during application
func (a *Applier) GetConflicts() []Conflict {
	return a.conflicts
}
