package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Extension struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Package     string `json:"pkg,omitempty"`
}

type ExtensionFile struct {
	GeneratedAt string      `json:"generated_at"`
	Source      string      `json:"source"`
	PgMajor     int         `json:"pg_major"`
	Entries     []Extension `json:"entries"`
}

type Manager struct {
	ScriptDir string
	PgMajor   string
}

func NewManager(scriptDir, pgMajor string) *Manager {
	return &Manager{
		ScriptDir: scriptDir,
		PgMajor:   pgMajor,
	}
}

func (m *Manager) IsBuiltinExtension(name string) bool {
	builtinFile := filepath.Join(m.ScriptDir, "pgbox-data", "builtin", fmt.Sprintf("pg%s.json", m.PgMajor))

	data, err := os.ReadFile(builtinFile)
	if err != nil {
		return false
	}

	var extFile ExtensionFile
	if err := json.Unmarshal(data, &extFile); err != nil {
		return false
	}

	for _, ext := range extFile.Entries {
		if ext.Name == name {
			return true
		}
	}
	return false
}

func (m *Manager) GetExtensionPackage(name string) string {
	aptFile := filepath.Join(m.ScriptDir, "pgbox-data", "apt-bookworm-pgdg", fmt.Sprintf("pg%s.json", m.PgMajor))

	data, err := os.ReadFile(aptFile)
	if err != nil {
		return ""
	}

	var extFile ExtensionFile
	if err := json.Unmarshal(data, &extFile); err != nil {
		return ""
	}

	for _, ext := range extFile.Entries {
		if ext.Name == name {
			return ext.Package
		}
	}
	return ""
}

func (m *Manager) GetAllExtensions() ([]Extension, error) {
	var allExtensions []Extension

	// Load builtin extensions
	builtinFile := filepath.Join(m.ScriptDir, "pgbox-data", "builtin", fmt.Sprintf("pg%s.json", m.PgMajor))
	if data, err := os.ReadFile(builtinFile); err == nil {
		var extFile ExtensionFile
		if err := json.Unmarshal(data, &extFile); err == nil {
			for _, ext := range extFile.Entries {
				ext.Kind = "builtin"
				allExtensions = append(allExtensions, ext)
			}
		}
	}

	// Load apt extensions
	aptFile := filepath.Join(m.ScriptDir, "pgbox-data", "apt-bookworm-pgdg", fmt.Sprintf("pg%s.json", m.PgMajor))
	if data, err := os.ReadFile(aptFile); err == nil {
		var extFile ExtensionFile
		if err := json.Unmarshal(data, &extFile); err == nil {
			for _, ext := range extFile.Entries {
				ext.Kind = "apt package"
				allExtensions = append(allExtensions, ext)
			}
		}
	}

	// Sort by name and remove duplicates
	sort.Slice(allExtensions, func(i, j int) bool {
		return allExtensions[i].Name < allExtensions[j].Name
	})

	// Remove duplicates (prioritize builtin over apt)
	uniqueExtensions := make([]Extension, 0, len(allExtensions))
	seen := make(map[string]bool)

	for _, ext := range allExtensions {
		if !seen[ext.Name] {
			uniqueExtensions = append(uniqueExtensions, ext)
			seen[ext.Name] = true
		}
	}

	return uniqueExtensions, nil
}

func (m *Manager) ParseExtensionList(extensionStr string) []string {
	if extensionStr == "" {
		return nil
	}

	extensions := strings.Split(extensionStr, ",")
	for i, ext := range extensions {
		extensions[i] = strings.TrimSpace(ext)
	}

	return extensions
}

func (m *Manager) GetExtensionInfo(name string) (Extension, bool) {
	allExts, err := m.GetAllExtensions()
	if err != nil {
		return Extension{}, false
	}

	for _, ext := range allExts {
		if ext.Name == name {
			return ext, true
		}
	}
	return Extension{}, false
}
