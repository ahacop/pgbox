package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type ExtensionCatalog struct {
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

func ListExtensionsCmd() *cobra.Command {
	var pgVersion string
	var showSource bool
	var filterKind string

	listExtCmd := &cobra.Command{
		Use:   "list-extensions",
		Short: "List available PostgreSQL extensions",
		Long: `List all available PostgreSQL extensions from both builtin and apt sources.

Extensions are uniqued by name and sorted alphabetically. When the same extension
appears in multiple sources, the builtin version is preferred.`,
		Example: `  # List all extensions for PostgreSQL 17
  pgbox list-extensions

  # List extensions for PostgreSQL 16
  pgbox list-extensions -v 16

  # Show source information for each extension
  pgbox list-extensions --source

  # Filter by kind (builtin or package)
  pgbox list-extensions --kind builtin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listExtensions(pgVersion, showSource, filterKind)
		},
	}

	listExtCmd.Flags().StringVarP(&pgVersion, "version", "v", "17", "PostgreSQL version (16 or 17)")
	listExtCmd.Flags().BoolVarP(&showSource, "source", "s", false, "Show source information for each extension")
	listExtCmd.Flags().StringVarP(&filterKind, "kind", "k", "", "Filter by kind (builtin or package)")

	return listExtCmd
}

func listExtensions(pgVersion string, showSource bool, filterKind string) error {
	// Validate version
	if err := ValidatePostgresVersion(pgVersion); err != nil {
		return err
	}

	// Map to store unique extensions by name
	extensionMap := make(map[string]Extension)

	// Load builtin extensions
	builtinPath := filepath.Join("pgbox-data", "builtin", fmt.Sprintf("pg%s.json", pgVersion))
	if err := loadExtensions(builtinPath, extensionMap); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load builtin extensions: %v\n", err)
	}

	// Load apt extensions
	aptPath := filepath.Join("pgbox-data", "apt-pgdg", fmt.Sprintf("pg%s.json", pgVersion))
	if err := loadExtensions(aptPath, extensionMap); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load apt extensions: %v\n", err)
	}

	// Convert map to slice and sort
	var extensions []Extension
	for _, ext := range extensionMap {
		// Apply filter if specified
		if filterKind != "" {
			if filterKind == "builtin" && ext.Kind != "builtin" {
				continue
			}
			if filterKind == "package" && ext.Pkg == "" {
				continue
			}
		}
		extensions = append(extensions, ext)
	}

	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Name < extensions[j].Name
	})

	// Display extensions
	fmt.Printf("PostgreSQL %s Extensions (%d available):\n\n", pgVersion, len(extensions))

	for _, ext := range extensions {
		if showSource {
			source := "builtin"
			if ext.Pkg != "" {
				source = fmt.Sprintf("package (%s)", ext.Pkg)
			}
			fmt.Printf("%-30s %-25s %s\n", ext.Name, source, cleanDescription(ext.Description))
		} else {
			fmt.Printf("%-30s %s\n", ext.Name, cleanDescription(ext.Description))
		}
	}

	return nil
}

func loadExtensions(path string, extensionMap map[string]Extension) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog ExtensionCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	for _, ext := range catalog.Entries {
		// Only add if not already present (builtin takes precedence)
		if _, exists := extensionMap[ext.Name]; !exists {
			extensionMap[ext.Name] = ext
		} else if ext.Kind == "builtin" {
			// Builtin always overrides package version
			extensionMap[ext.Name] = ext
		}
	}

	return nil
}

func cleanDescription(desc string) string {
	// Remove leading "- " if present
	desc = strings.TrimPrefix(desc, "- ")
	// Truncate if too long
	if len(desc) > 80 {
		desc = desc[:77] + "..."
	}
	return desc
}
