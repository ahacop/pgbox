package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		extName string
		wantOK  bool
		wantPkg string
		wantSQL string
	}{
		{"built-in extension", "hstore", true, "", "hstore"},
		{"simple third-party", "hypopg", true, "postgresql-{v}-hypopg", "hypopg"},
		{"extension with SQL name", "pgvector", true, "postgresql-{v}-pgvector", "vector"},
		{"complex extension", "pg_cron", true, "postgresql-{v}-cron", "pg_cron"},
		{"unknown extension", "nonexistent", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, ok := Get(tt.extName)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantPkg, ext.Package)
				if tt.wantSQL != "" && ext.SQLName != "" {
					assert.Equal(t, tt.wantSQL, ext.SQLName)
				}
			}
		})
	}
}

func TestGetPackage(t *testing.T) {
	tests := []struct {
		name    string
		extName string
		version string
		want    string
	}{
		{"built-in", "hstore", "17", ""},
		{"third-party v17", "hypopg", "17", "postgresql-17-hypopg"},
		{"third-party v16", "hypopg", "16", "postgresql-16-hypopg"},
		{"pgvector v17", "pgvector", "17", "postgresql-17-pgvector"},
		{"pg_cron v17", "pg_cron", "17", "postgresql-17-cron"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPackage(tt.extName, tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetSQLName(t *testing.T) {
	assert.Equal(t, "hstore", GetSQLName("hstore"))
	assert.Equal(t, "vector", GetSQLName("pgvector"))
	assert.Equal(t, "postgis", GetSQLName("postgis-3"))
	assert.Equal(t, "pg_cron", GetSQLName("pg_cron"))
}

func TestGetInitSQL(t *testing.T) {
	// Built-in extension
	sql := GetInitSQL("hstore")
	assert.Equal(t, "CREATE EXTENSION IF NOT EXISTS hstore;", sql)

	// Extension with different SQL name
	sql = GetInitSQL("pgvector")
	assert.Equal(t, "CREATE EXTENSION IF NOT EXISTS vector;", sql)

	// Custom init SQL
	sql = GetInitSQL("pg_cron")
	assert.Contains(t, sql, "CREATE EXTENSION IF NOT EXISTS pg_cron")
	assert.Contains(t, sql, "GRANT USAGE ON SCHEMA cron")
}

func TestValidateExtensions(t *testing.T) {
	// Valid extensions
	err := ValidateExtensions([]string{"hstore", "pgvector", "pg_cron"})
	assert.NoError(t, err)

	// Invalid extension
	err = ValidateExtensions([]string{"hstore", "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestListExtensions(t *testing.T) {
	list := ListExtensions()
	assert.Greater(t, len(list), 100) // Should have 150+ extensions

	// Check sorted
	for i := 1; i < len(list); i++ {
		assert.Less(t, list[i-1], list[i], "extensions should be sorted")
	}

	// Check some known extensions
	assert.Contains(t, list, "hstore")
	assert.Contains(t, list, "pgvector")
	assert.Contains(t, list, "pg_cron")
}

func TestGetPackages(t *testing.T) {
	packages := GetPackages([]string{"hstore", "pgvector", "hypopg"}, "17")
	assert.Len(t, packages, 2) // hstore is built-in
	assert.Contains(t, packages, "postgresql-17-pgvector")
	assert.Contains(t, packages, "postgresql-17-hypopg")
}

func TestGetPreloadLibraries(t *testing.T) {
	// No preload needed
	libs := GetPreloadLibraries([]string{"hstore", "pgvector"})
	assert.Len(t, libs, 0)

	// With preload
	libs = GetPreloadLibraries([]string{"pg_cron", "wal2json"})
	assert.Len(t, libs, 2)
	assert.Contains(t, libs, "pg_cron")
	assert.Contains(t, libs, "wal2json")
}

func TestGetGUCs(t *testing.T) {
	// No GUCs
	gucs, err := GetGUCs([]string{"hstore", "pgvector"})
	assert.NoError(t, err)
	assert.Len(t, gucs, 0)

	// With GUCs
	gucs, err = GetGUCs([]string{"pg_cron"})
	assert.NoError(t, err)
	assert.Equal(t, "postgres", gucs["cron.database_name"])
	assert.Equal(t, "5", gucs["cron.max_running_jobs"])

	// wal2json GUCs
	gucs, err = GetGUCs([]string{"wal2json"})
	assert.NoError(t, err)
	assert.Equal(t, "logical", gucs["wal_level"])
}

func TestNeedsPackages(t *testing.T) {
	assert.False(t, NeedsPackages([]string{"hstore", "ltree"}))
	assert.True(t, NeedsPackages([]string{"hstore", "pgvector"}))
	assert.True(t, NeedsPackages([]string{"pg_cron"}))
}
