// generate-extension-toml.go - Generate TOML files from JSON extension data
//
// This script reads the intermediate JSON files (builtin/pg*.json, apt-pgdg/pg*.json)
// and generates TOML files for each extension.
//
// Usage: go run scripts/generate-extension-toml.go [--force]

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Extension represents an extension from the JSON files
type Extension struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Package     string `json:"pkg"`
	Description string `json:"description"`
}

// Catalog represents a catalog of extensions
type Catalog struct {
	GeneratedAt string      `json:"generated_at"`
	Source      string      `json:"source"`
	PGMajor     int         `json:"pg_major"`
	Entries     []Extension `json:"entries"`
}

// ExtensionMapping holds SQL name mappings
type ExtensionMapping struct {
	Mappings map[string][]string `json:"mappings"`
}

func main() {
	var force bool
	flag.BoolVar(&force, "force", false, "Overwrite existing TOML files")
	flag.Parse()

	dataDir := "pgbox-data"
	extensionsDir := "extensions"

	// Create extensions directory if it doesn't exist
	if err := os.MkdirAll(extensionsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating extensions directory: %v\n", err)
		os.Exit(1)
	}

	// Process both PostgreSQL versions
	for _, pgMajor := range []string{"16", "17"} {
		fmt.Printf("Processing PostgreSQL %s extensions...\n", pgMajor)

		// Load builtin extensions
		builtinPath := filepath.Join(dataDir, "builtin", fmt.Sprintf("pg%s.json", pgMajor))
		builtinCatalog, err := loadCatalog(builtinPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load builtin catalog: %v\n", err)
		}

		// Load apt extensions
		aptPath := filepath.Join(dataDir, "apt-pgdg", fmt.Sprintf("pg%s.json", pgMajor))
		aptCatalog, err := loadCatalog(aptPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load apt catalog: %v\n", err)
		}

		// Load extension mappings
		mappingsPath := filepath.Join(dataDir, fmt.Sprintf("extension-mappings-pg%s.json", pgMajor))
		mappings, err := loadMappings(mappingsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load mappings: %v\n", err)
			mappings = &ExtensionMapping{Mappings: make(map[string][]string)}
		}

		// Process builtin extensions
		if builtinCatalog != nil {
			for _, ext := range builtinCatalog.Entries {
				if err := generateTOML(extensionsDir, ext, pgMajor, "", mappings, force); err != nil {
					fmt.Fprintf(os.Stderr, "Error generating TOML for %s: %v\n", ext.Name, err)
				}
			}
		}

		// Process apt extensions
		if aptCatalog != nil {
			for _, ext := range aptCatalog.Entries {
				if err := generateTOML(extensionsDir, ext, pgMajor, ext.Package, mappings, force); err != nil {
					fmt.Fprintf(os.Stderr, "Error generating TOML for %s: %v\n", ext.Name, err)
				}
			}
		}
	}

	fmt.Println("\nTOML generation complete!")
	fmt.Printf("Files created in: %s/\n", extensionsDir)
}

func loadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}

	return &catalog, nil
}

func loadMappings(path string) (*ExtensionMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var mapping ExtensionMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}

	return &mapping, nil
}

func generateTOML(baseDir string, ext Extension, pgMajor string, packageName string, mappings *ExtensionMapping, force bool) error {
	// Determine the extension directory name
	// For apt packages, use the package name without postgresql-XX- prefix
	dirName := ext.Name
	if packageName != "" {
		// Remove postgresql-XX- prefix if present
		prefix := fmt.Sprintf("postgresql-%s-", pgMajor)
		if strings.HasPrefix(packageName, prefix) {
			dirName = strings.TrimPrefix(packageName, prefix)
		}
	}

	// Create extension directory
	extDir := filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	tomlPath := filepath.Join(extDir, fmt.Sprintf("%s.toml", pgMajor))
	if _, err := os.Stat(tomlPath); err == nil && !force {
		// File exists and --force not specified, skip
		return nil
	}

	// Determine SQL name(s) for the extension
	sqlNames := getSQLNames(ext.Name, mappings)

	// Generate TOML content
	content := generateTOMLContent(ext, pgMajor, packageName, sqlNames)

	// Write TOML file
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write TOML: %w", err)
	}

	fmt.Printf("  Created: %s\n", tomlPath)
	return nil
}

func getSQLNames(extName string, mappings *ExtensionMapping) []string {
	// Check if there's a mapping for this extension
	if sqlNames, ok := mappings.Mappings[extName]; ok && len(sqlNames) > 0 {
		return sqlNames
	}

	// Special cases for known extensions
	switch extName {
	case "pgvector":
		return []string{"vector"}
	case "cron":
		return []string{"pg_cron"}
	case "postgis":
		return []string{"postgis", "postgis_topology", "postgis_raster"}
	default:
		// Default to the extension name itself
		return []string{extName}
	}
}

func generateTOMLContent(ext Extension, pgMajor string, packageName string, sqlNames []string) string {
	var lines []string

	// Header comment
	lines = append(lines, fmt.Sprintf("# Auto-generated from pgbox-data - PostgreSQL %s", pgMajor))
	lines = append(lines, "")

	// Primary SQL name (first one in the list)
	primarySQL := sqlNames[0]

	// Basic metadata
	lines = append(lines, fmt.Sprintf("extension = %q", primarySQL))

	// Display name (use the directory/package name if different from SQL name)
	if ext.Name != primarySQL {
		lines = append(lines, fmt.Sprintf("display_name = %q", ext.Name))
	}

	// Package name for apt extensions
	if packageName != "" {
		lines = append(lines, fmt.Sprintf("package = %q", packageName))
	}

	// Description
	if ext.Description != "" {
		// Clean up description (remove leading dash and whitespace)
		desc := strings.TrimSpace(ext.Description)
		desc = strings.TrimPrefix(desc, "- ")
		lines = append(lines, fmt.Sprintf("description = %q", desc))
	}

	lines = append(lines, "")

	// Image section for apt packages
	if packageName != "" {
		lines = append(lines, "[image]")
		lines = append(lines, fmt.Sprintf("apt_packages = [%q]", packageName))
		lines = append(lines, "")
	}

	// SQL initialization
	lines = append(lines, "# SQL initialization")
	for _, sqlName := range sqlNames {
		lines = append(lines, "[[sql.initdb]]")
		lines = append(lines, fmt.Sprintf("text = \"CREATE EXTENSION IF NOT EXISTS %s;\"", sqlName))
		if sqlName != sqlNames[len(sqlNames)-1] {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n") + "\n"
}
