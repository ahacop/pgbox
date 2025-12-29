package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ahacop/pgbox/internal/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Change to project root directory so extension TOML files can be found
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filename))
	if err := os.Chdir(projectRoot); err != nil {
		panic(err)
	}
}

// runExport is a helper to run the export orchestrator with the given parameters
func runExport(t *testing.T, dir, version, port, extList, baseImage string) error {
	var buf bytes.Buffer
	orch := orchestrator.NewExportOrchestrator(&buf)

	extensions := ParseExtensionList(extList)

	return orch.Run(orchestrator.ExportConfig{
		TargetDir:  dir,
		Version:    version,
		Port:       port,
		Extensions: extensions,
		BaseImage:  baseImage,
	})
}

func TestExportGeneratesCorrectFiles(t *testing.T) {
	tests := []struct {
		name       string
		pgVersion  string
		extensions string
		wantFiles  []string
		assertions func(t *testing.T, dir string)
	}{
		{
			name:       "no extensions - base postgres",
			pgVersion:  "17",
			extensions: "",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql"},
			assertions: func(t *testing.T, dir string) {
				// Dockerfile should use base postgres image
				dockerfile := readFile(t, filepath.Join(dir, "Dockerfile"))
				assert.Contains(t, dockerfile, "FROM postgres:17")

				// docker-compose.yml should have basic setup
				compose := readFile(t, filepath.Join(dir, "docker-compose.yml"))
				assert.Contains(t, compose, "postgres_data")
				assert.Contains(t, compose, "5432:5432")

				// init.sql should exist but be minimal
				initSQL := readFile(t, filepath.Join(dir, "init.sql"))
				assert.NotEmpty(t, initSQL)

				// No postgresql.conf.pgbox should be generated
				_, err := os.Stat(filepath.Join(dir, "postgresql.conf.pgbox"))
				assert.True(t, os.IsNotExist(err), "postgresql.conf.pgbox should not exist for base postgres")
			},
		},
		{
			name:       "builtin extension - xml2 (no apt packages)",
			pgVersion:  "17",
			extensions: "xml2",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql"},
			assertions: func(t *testing.T, dir string) {
				// init.sql should have CREATE EXTENSION for xml2
				initSQL := readFile(t, filepath.Join(dir, "init.sql"))
				assert.Contains(t, initSQL, "CREATE EXTENSION")
				assert.Contains(t, initSQL, "xml2")

				// No postgresql.conf.pgbox needed (xml2 doesn't require shared_preload)
				_, err := os.Stat(filepath.Join(dir, "postgresql.conf.pgbox"))
				assert.True(t, os.IsNotExist(err), "postgresql.conf.pgbox should not exist for xml2")
			},
		},
		{
			name:       "extension with apt packages - hypopg",
			pgVersion:  "17",
			extensions: "hypopg",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql"},
			assertions: func(t *testing.T, dir string) {
				// Dockerfile should have apt-get install
				dockerfile := readFile(t, filepath.Join(dir, "Dockerfile"))
				assert.Contains(t, dockerfile, "apt-get")
				assert.Contains(t, dockerfile, "hypopg")

				// init.sql should have CREATE EXTENSION
				initSQL := readFile(t, filepath.Join(dir, "init.sql"))
				assert.Contains(t, initSQL, "CREATE EXTENSION")
				assert.Contains(t, initSQL, "hypopg")
			},
		},
		{
			name:       "extension with shared_preload_libraries - pg_cron",
			pgVersion:  "17",
			extensions: "pg_cron",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql", "postgresql.conf.pgbox"},
			assertions: func(t *testing.T, dir string) {
				// Dockerfile should have apt package
				dockerfile := readFile(t, filepath.Join(dir, "Dockerfile"))
				assert.Contains(t, dockerfile, "cron")

				// postgresql.conf.pgbox should have shared_preload_libraries
				pgConf := readFile(t, filepath.Join(dir, "postgresql.conf.pgbox"))
				assert.Contains(t, pgConf, "shared_preload_libraries")
				assert.Contains(t, pgConf, "pg_cron")

				// docker-compose.yml should have command with shared_preload_libraries
				compose := readFile(t, filepath.Join(dir, "docker-compose.yml"))
				assert.Contains(t, compose, "shared_preload_libraries")

				// init.sql should have CREATE EXTENSION
				initSQL := readFile(t, filepath.Join(dir, "init.sql"))
				assert.Contains(t, initSQL, "pg_cron")
			},
		},
		{
			name:       "extension with custom GUCs - wal2json",
			pgVersion:  "17",
			extensions: "wal2json",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql", "postgresql.conf.pgbox"},
			assertions: func(t *testing.T, dir string) {
				// postgresql.conf.pgbox should have wal_level = logical
				pgConf := readFile(t, filepath.Join(dir, "postgresql.conf.pgbox"))
				assert.Contains(t, pgConf, "wal_level")
				assert.Contains(t, pgConf, "logical")

				// Should also have shared_preload_libraries
				assert.Contains(t, pgConf, "shared_preload_libraries")
				assert.Contains(t, pgConf, "wal2json")
			},
		},
		{
			name:       "extension with complex SQL - postgis",
			pgVersion:  "17",
			extensions: "postgis-3",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql"},
			assertions: func(t *testing.T, dir string) {
				// Dockerfile should have postgis packages
				dockerfile := readFile(t, filepath.Join(dir, "Dockerfile"))
				assert.Contains(t, dockerfile, "postgis")

				// init.sql should have CREATE EXTENSION postgis and grants
				initSQL := readFile(t, filepath.Join(dir, "init.sql"))
				assert.Contains(t, initSQL, "postgis")
				assert.Contains(t, initSQL, "GRANT")
			},
		},
		{
			name:       "multiple extensions combined",
			pgVersion:  "17",
			extensions: "hypopg,pg_cron",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql", "postgresql.conf.pgbox"},
			assertions: func(t *testing.T, dir string) {
				// Dockerfile should have both packages
				dockerfile := readFile(t, filepath.Join(dir, "Dockerfile"))
				assert.Contains(t, dockerfile, "hypopg")
				assert.Contains(t, dockerfile, "cron")

				// init.sql should have both extensions
				initSQL := readFile(t, filepath.Join(dir, "init.sql"))
				assert.Contains(t, initSQL, "hypopg")
				assert.Contains(t, initSQL, "pg_cron")

				// postgresql.conf.pgbox should have pg_cron in shared_preload
				pgConf := readFile(t, filepath.Join(dir, "postgresql.conf.pgbox"))
				assert.Contains(t, pgConf, "pg_cron")
			},
		},
		{
			name:       "PostgreSQL 16 version",
			pgVersion:  "16",
			extensions: "hypopg",
			wantFiles:  []string{"Dockerfile", "docker-compose.yml", "init.sql"},
			assertions: func(t *testing.T, dir string) {
				// Dockerfile should use postgres:16
				dockerfile := readFile(t, filepath.Join(dir, "Dockerfile"))
				assert.Contains(t, dockerfile, "FROM postgres:16")

				// Package should be version-specific
				assert.Contains(t, dockerfile, "postgresql-16")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for output
			tmpDir, err := os.MkdirTemp("", "pgbox-export-test-*")
			require.NoError(t, err)
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Run export
			err = runExport(t, tmpDir, tt.pgVersion, "5432", tt.extensions, "")
			require.NoError(t, err)

			// Check expected files exist
			for _, wantFile := range tt.wantFiles {
				path := filepath.Join(tmpDir, wantFile)
				_, err := os.Stat(path)
				assert.NoError(t, err, "expected file %s to exist", wantFile)
			}

			// Run custom assertions
			if tt.assertions != nil {
				tt.assertions(t, tmpDir)
			}
		})
	}
}

func TestExportUnknownExtension(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pgbox-export-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	err = runExport(t, tmpDir, "17", "5432", "nonexistent_extension", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown extensions")
}

func TestExportCustomPort(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pgbox-export-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	err = runExport(t, tmpDir, "17", "5433", "", "")
	require.NoError(t, err)

	compose := readFile(t, filepath.Join(tmpDir, "docker-compose.yml"))
	assert.Contains(t, compose, "5433:5432")
}

func TestExportCustomBaseImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pgbox-export-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	err = runExport(t, tmpDir, "17", "5432", "", "postgres:17-alpine")
	require.NoError(t, err)

	dockerfile := readFile(t, filepath.Join(tmpDir, "Dockerfile"))
	assert.Contains(t, dockerfile, "FROM postgres:17-alpine")
}

// readFile is a helper to read file contents, failing the test on error
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read %s", path)
	return strings.TrimSpace(string(data))
}
