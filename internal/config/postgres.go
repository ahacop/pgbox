package config

import "fmt"

// PostgresConfig holds PostgreSQL-specific configuration
type PostgresConfig struct {
	Version  string // PostgreSQL version (e.g., "16", "17")
	Port     string
	Database string
	User     string
	Password string
}

// NewPostgresConfig returns a PostgresConfig with default values
func NewPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Version:  "17",
		Port:     "5432",
		Database: "postgres",
		User:     "postgres",
		Password: "postgres",
	}
}

// Image returns the Docker image name for this PostgreSQL version
func (c *PostgresConfig) Image() string {
	return fmt.Sprintf("postgres:%s", c.Version)
}
