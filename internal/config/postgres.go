package config

import "fmt"

// DefaultVersion is the default PostgreSQL version used throughout pgbox.
// This is the single source of truth for the default version.
const DefaultVersion = "18"

// PostgresConfig holds PostgreSQL-specific configuration
type PostgresConfig struct {
	Version     string // PostgreSQL version (e.g., "16", "17")
	Port        string
	Database    string
	User        string
	Password    string
	CustomImage string // Custom Docker image name when using extensions
}

// NewPostgresConfig returns a PostgresConfig with default values
func NewPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Version:  DefaultVersion,
		Port:     "5432",
		Database: "postgres",
		User:     "postgres",
		Password: "postgres",
	}
}

// Image returns the Docker image name for this PostgreSQL version
func (c *PostgresConfig) Image() string {
	if c.CustomImage != "" {
		return c.CustomImage
	}
	return fmt.Sprintf("postgres:%s", c.Version)
}
