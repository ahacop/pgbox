package orchestrator

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportOrchestrator_BasicExport(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir: dir,
		Version:   "17",
		Port:      "5432",
	})

	require.NoError(t, err)

	// Check files were created
	assert.FileExists(t, filepath.Join(dir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(dir, "docker-compose.yml"))
	assert.FileExists(t, filepath.Join(dir, "init.sql"))

	// Check output message
	assert.Contains(t, buf.String(), "Exported Docker configuration")
	assert.Contains(t, buf.String(), dir)
}

func TestExportOrchestrator_WithExtensions(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir:  dir,
		Version:    "17",
		Port:       "5432",
		Extensions: []string{"pgvector", "hypopg"},
	})

	require.NoError(t, err)

	// Check Dockerfile has packages
	dockerfileContent, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	require.NoError(t, err)
	assert.Contains(t, string(dockerfileContent), "postgresql-17-pgvector")
	assert.Contains(t, string(dockerfileContent), "postgresql-17-hypopg")

	// Check output message mentions extensions
	assert.Contains(t, buf.String(), "With extensions")
}

func TestExportOrchestrator_CustomPort(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir: dir,
		Version:   "17",
		Port:      "5433",
	})

	require.NoError(t, err)

	// Check docker-compose.yml has custom port
	composeContent, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(composeContent), "5433:5432")
}

func TestExportOrchestrator_CustomBaseImage(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir: dir,
		Version:   "17",
		Port:      "5432",
		BaseImage: "postgres:17-alpine",
	})

	require.NoError(t, err)

	// Check Dockerfile uses custom base image
	dockerfileContent, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	require.NoError(t, err)
	assert.Contains(t, string(dockerfileContent), "FROM postgres:17-alpine")
}

func TestExportOrchestrator_WithPreloadExtensions(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir:  dir,
		Version:    "17",
		Port:       "5432",
		Extensions: []string{"pg_cron"},
	})

	require.NoError(t, err)

	// Check postgresql.conf.pgbox was created (pg_cron requires shared_preload_libraries)
	assert.FileExists(t, filepath.Join(dir, "postgresql.conf.pgbox"))

	// Check docker-compose.yml has shared_preload_libraries in command
	composeContent, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(composeContent), "shared_preload_libraries")
}

func TestExportOrchestrator_InvalidExtension(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir:  dir,
		Version:    "17",
		Port:       "5432",
		Extensions: []string{"nonexistent_extension"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_extension")
}

func TestExportOrchestrator_CustomCredentials(t *testing.T) {
	dir, err := os.MkdirTemp("", "pgbox-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	orch := NewExportOrchestrator(&buf)

	err = orch.Run(ExportConfig{
		TargetDir: dir,
		Version:   "17",
		Port:      "5432",
		User:      "myuser",
		Password:  "mypassword",
		Database:  "mydb",
	})

	require.NoError(t, err)

	// Check docker-compose.yml has custom credentials
	composeContent, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(composeContent), "POSTGRES_USER: myuser")
	assert.Contains(t, string(composeContent), "POSTGRES_PASSWORD: mypassword")
	assert.Contains(t, string(composeContent), "POSTGRES_DB: mydb")
}
