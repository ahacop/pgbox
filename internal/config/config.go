package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Name       string
	Port       string
	PgMajor    string
	Extensions string
	ExportDir  string
	StateDir   string
	DataVol    string
	RunDirBase string
	ScriptDir  string
}

func NewConfig() *Config {
	c := &Config{
		Name:    "pgbox",
		Port:    "5432",
		PgMajor: getEnvOrDefault("PG_MAJOR", "17"),
	}
	c.updateDerivedValues()
	return c
}

func (c *Config) updateDerivedValues() {
	c.DataVol = fmt.Sprintf("pgbox_%s_data", c.Name)

	// XDG directories
	stateHome := getEnvOrDefault("XDG_STATE_HOME", filepath.Join(os.Getenv("HOME"), ".local/state"))
	c.StateDir = filepath.Join(stateHome, "pgbox", c.Name)

	runtimeDir := getEnvOrDefault("XDG_RUNTIME_DIR", "/tmp")
	c.RunDirBase = filepath.Join(runtimeDir, "pgbox")

	// Create necessary directories
	_ = os.MkdirAll(c.StateDir, 0755)
	_ = os.MkdirAll(c.RunDirBase, 0755)

	// Get script directory (where the executable is)
	ex, err := os.Executable()
	if err == nil {
		c.ScriptDir = filepath.Dir(ex)
	} else {
		// Fallback to current directory
		c.ScriptDir, _ = os.Getwd()
	}
}

func (c *Config) SetName(name string) {
	c.Name = name
	c.updateDerivedValues()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
