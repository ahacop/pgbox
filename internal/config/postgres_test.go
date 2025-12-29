package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPostgresConfig_Defaults(t *testing.T) {
	cfg := NewPostgresConfig()

	assert.Equal(t, "18", cfg.Version)
	assert.Equal(t, "5432", cfg.Port)
	assert.Equal(t, "postgres", cfg.Database)
	assert.Equal(t, "postgres", cfg.User)
	assert.Equal(t, "postgres", cfg.Password)
	assert.Empty(t, cfg.CustomImage)
}

func TestPostgresConfig_Image_Default(t *testing.T) {
	cfg := NewPostgresConfig()

	assert.Equal(t, "postgres:18", cfg.Image())
}

func TestPostgresConfig_Image_DifferentVersion(t *testing.T) {
	cfg := NewPostgresConfig()
	cfg.Version = "16"

	assert.Equal(t, "postgres:16", cfg.Image())
}

func TestPostgresConfig_Image_CustomImage(t *testing.T) {
	cfg := NewPostgresConfig()
	cfg.CustomImage = "pgbox-pg17-custom:abc123"

	assert.Equal(t, "pgbox-pg17-custom:abc123", cfg.Image())
}

func TestPostgresConfig_Image_CustomImageOverridesVersion(t *testing.T) {
	cfg := NewPostgresConfig()
	cfg.Version = "16"
	cfg.CustomImage = "myregistry/postgres:latest"

	// CustomImage should take precedence
	assert.Equal(t, "myregistry/postgres:latest", cfg.Image())
}
