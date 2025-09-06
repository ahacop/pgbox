package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SavedConfig struct {
	Name        string    `json:"name"`
	Extensions  []string  `json:"extensions"`
	Port        string    `json:"port"`
	PgMajor     string    `json:"pg_major"`
	Database    string    `json:"database,omitempty"`
	User        string    `json:"user,omitempty"`
	Password    string    `json:"password,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	Description string    `json:"description,omitempty"`
}

type ConfigManager struct {
	configDir string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	stateHome := getEnvOrDefault("XDG_STATE_HOME", filepath.Join(os.Getenv("HOME"), ".local/state"))
	configDir := filepath.Join(stateHome, "pgbox", "configs")

	// Create configs directory if it doesn't exist
	_ = os.MkdirAll(configDir, 0755)

	return &ConfigManager{
		configDir: configDir,
	}
}

// SaveConfig saves a configuration to disk
func (cm *ConfigManager) SaveConfig(config *SavedConfig) error {
	if config.Name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}

	// Sanitize filename
	filename := sanitizeFilename(config.Name) + ".json"
	filepath := filepath.Join(cm.configDir, filename)

	// Update timestamps
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	config.LastUsed = time.Now()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfig loads a configuration from disk by name
func (cm *ConfigManager) LoadConfig(name string) (*SavedConfig, error) {
	filename := sanitizeFilename(name) + ".json"
	filepath := filepath.Join(cm.configDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SavedConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// ListConfigs returns all saved configurations sorted by last used (most recent first)
func (cm *ConfigManager) ListConfigs() ([]*SavedConfig, error) {
	files, err := filepath.Glob(filepath.Join(cm.configDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list config files: %w", err)
	}

	var configs []*SavedConfig
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}

		var config SavedConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue // Skip files we can't parse
		}

		configs = append(configs, &config)
	}

	// Sort by last used (most recent first)
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].LastUsed.After(configs[j].LastUsed)
	})

	return configs, nil
}

// DeleteConfig removes a configuration from disk
func (cm *ConfigManager) DeleteConfig(name string) error {
	filename := sanitizeFilename(name) + ".json"
	filepath := filepath.Join(cm.configDir, filename)

	err := os.Remove(filepath)
	if err != nil {
		return fmt.Errorf("failed to delete config file: %w", err)
	}

	return nil
}

// ConfigExists checks if a configuration with the given name exists
func (cm *ConfigManager) ConfigExists(name string) bool {
	filename := sanitizeFilename(name) + ".json"
	filepath := filepath.Join(cm.configDir, filename)

	_, err := os.Stat(filepath)
	return err == nil
}

// UpdateLastUsed updates the last used timestamp for a configuration
func (cm *ConfigManager) UpdateLastUsed(name string) error {
	config, err := cm.LoadConfig(name)
	if err != nil {
		return err
	}

	config.LastUsed = time.Now()
	return cm.SaveConfig(config)
}

// CreateConfigFromCurrent creates a SavedConfig from the current Config
func CreateConfigFromCurrent(current *Config, extensions []string, name, description string) *SavedConfig {
	return &SavedConfig{
		Name:        name,
		Extensions:  extensions,
		Port:        current.Port,
		PgMajor:     current.PgMajor,
		Database:    "appdb",    // Default database name
		User:        "appuser",  // Default user name
		Password:    "changeme", // Default password
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		Description: description,
	}
}

// ToConfig converts a SavedConfig back to a Config for use with the rest of the system
func (sc *SavedConfig) ToConfig() *Config {
	config := NewConfig()
	config.SetName(sc.Name)
	config.Port = sc.Port
	config.PgMajor = sc.PgMajor
	config.Extensions = strings.Join(sc.Extensions, ",")
	return config
}

// sanitizeFilename removes or replaces characters that are not safe for filenames
func sanitizeFilename(name string) string {
	// Replace spaces and other problematic characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")

	// Remove any leading/trailing whitespace and convert to lowercase
	name = strings.TrimSpace(strings.ToLower(name))

	// Ensure it's not empty after sanitization
	if name == "" {
		name = "unnamed_config"
	}

	return name
}
