package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// DockerfileModel tests

func TestNewDockerfileModel(t *testing.T) {
	m := NewDockerfileModel("postgres:17")

	assert.Equal(t, "postgres:17", m.BaseImage)
	assert.Empty(t, m.AptPackages)
	assert.Empty(t, m.DebURLs)
	assert.Empty(t, m.ZipURLs)
	assert.Empty(t, m.Blocks)
}

func TestDockerfileModel_AddPackages(t *testing.T) {
	m := NewDockerfileModel("postgres:17")

	m.AddPackages([]string{"postgresql-17-pgvector", "postgresql-17-hypopg"}, "apt")

	assert.Equal(t, []string{"postgresql-17-hypopg", "postgresql-17-pgvector"}, m.AptPackages)
}

func TestDockerfileModel_AddPackages_Deduplication(t *testing.T) {
	m := NewDockerfileModel("postgres:17")

	m.AddPackages([]string{"postgresql-17-pgvector"}, "apt")
	m.AddPackages([]string{"postgresql-17-pgvector", "postgresql-17-hypopg"}, "apt")

	assert.Equal(t, []string{"postgresql-17-hypopg", "postgresql-17-pgvector"}, m.AptPackages)
}

func TestDockerfileModel_AddPackages_NonApt(t *testing.T) {
	m := NewDockerfileModel("postgres:17")

	m.AddPackages([]string{"some-package"}, "yum")

	// Non-apt packages should be ignored
	assert.Empty(t, m.AptPackages)
}

func TestDockerfileModel_AddDebURLs(t *testing.T) {
	m := NewDockerfileModel("postgres:17")

	m.AddDebURLs("https://example.com/ext1.deb", "https://example.com/ext2.deb")

	assert.Equal(t, []string{"https://example.com/ext1.deb", "https://example.com/ext2.deb"}, m.DebURLs)
}

func TestDockerfileModel_AddZipURLs(t *testing.T) {
	m := NewDockerfileModel("postgres:17")

	m.AddZipURLs("https://example.com/ext1.zip")

	assert.Equal(t, []string{"https://example.com/ext1.zip"}, m.ZipURLs)
}

// ComposeModel tests

func TestNewComposeModel(t *testing.T) {
	m := NewComposeModel("db")

	assert.Equal(t, "db", m.ServiceName)
	assert.Empty(t, m.Image)
	assert.Empty(t, m.Env)
	assert.Empty(t, m.Ports)
	assert.Empty(t, m.Volumes)
}

func TestComposeModel_AddPort(t *testing.T) {
	m := NewComposeModel("db")

	m.AddPort("5432:5432")

	assert.Equal(t, []string{"5432:5432"}, m.Ports)
}

func TestComposeModel_AddPort_Deduplication(t *testing.T) {
	m := NewComposeModel("db")

	m.AddPort("5432:5432")
	m.AddPort("5432:5432")
	m.AddPort("5433:5432")

	assert.Equal(t, []string{"5432:5432", "5433:5432"}, m.Ports)
}

func TestComposeModel_AddVolume(t *testing.T) {
	m := NewComposeModel("db")

	m.AddVolume("postgres_data:/var/lib/postgresql/data")

	assert.Equal(t, []string{"postgres_data:/var/lib/postgresql/data"}, m.Volumes)
}

func TestComposeModel_SetEnv(t *testing.T) {
	m := NewComposeModel("db")

	m.SetEnv("POSTGRES_USER", "myuser")
	m.SetEnv("POSTGRES_PASSWORD", "secret")

	assert.Equal(t, "myuser", m.Env["POSTGRES_USER"])
	assert.Equal(t, "secret", m.Env["POSTGRES_PASSWORD"])
}

// PGConfModel tests

func TestNewPGConfModel(t *testing.T) {
	m := NewPGConfModel()

	assert.Empty(t, m.SharedPreload)
	assert.Empty(t, m.GUCs)
	assert.False(t, m.RequireRestart)
}

func TestPGConfModel_AddSharedPreload(t *testing.T) {
	m := NewPGConfModel()

	m.AddSharedPreload("pg_cron", "wal2json")

	assert.Equal(t, []string{"pg_cron", "wal2json"}, m.SharedPreload)
	assert.True(t, m.RequireRestart)
}

func TestPGConfModel_AddSharedPreload_Deduplication(t *testing.T) {
	m := NewPGConfModel()

	m.AddSharedPreload("pg_cron")
	m.AddSharedPreload("pg_cron", "wal2json")

	assert.Equal(t, []string{"pg_cron", "wal2json"}, m.SharedPreload)
}

func TestPGConfModel_SetGUC(t *testing.T) {
	m := NewPGConfModel()

	err := m.SetGUC("cron.database_name", "postgres")

	assert.NoError(t, err)
	assert.Equal(t, "postgres", m.GUCs["cron.database_name"])
}

func TestPGConfModel_SetGUC_Conflict(t *testing.T) {
	m := NewPGConfModel()

	_ = m.SetGUC("cron.database_name", "postgres")
	err := m.SetGUC("cron.database_name", "mydb")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting values")
}

func TestPGConfModel_SetGUC_SameValue(t *testing.T) {
	m := NewPGConfModel()

	_ = m.SetGUC("cron.database_name", "postgres")
	err := m.SetGUC("cron.database_name", "postgres")

	// Same value should not error
	assert.NoError(t, err)
}

func TestPGConfModel_GetSharedPreloadString(t *testing.T) {
	m := NewPGConfModel()

	m.AddSharedPreload("pg_cron", "wal2json")

	assert.Equal(t, "pg_cron,wal2json", m.GetSharedPreloadString())
}

func TestPGConfModel_GetSharedPreloadString_Empty(t *testing.T) {
	m := NewPGConfModel()

	assert.Empty(t, m.GetSharedPreloadString())
}

// InitModel tests

func TestNewInitModel(t *testing.T) {
	m := NewInitModel()

	assert.Empty(t, m.Fragments)
}

func TestInitModel_AddFragment(t *testing.T) {
	m := NewInitModel()

	m.AddFragment("pgvector", "CREATE EXTENSION IF NOT EXISTS vector;")

	assert.Len(t, m.Fragments, 1)
	assert.Equal(t, "pgvector", m.Fragments[0].Name)
	assert.Equal(t, "CREATE EXTENSION IF NOT EXISTS vector;", m.Fragments[0].Content)
	assert.NotEmpty(t, m.Fragments[0].SHA256)
}

func TestInitModel_AddFragment_Deduplication(t *testing.T) {
	m := NewInitModel()

	m.AddFragment("pgvector", "CREATE EXTENSION IF NOT EXISTS vector;")
	m.AddFragment("pgvector2", "CREATE EXTENSION IF NOT EXISTS vector;") // Same content, different name

	// Should only have one fragment since content is identical
	assert.Len(t, m.Fragments, 1)
}

func TestInitModel_AddFragment_DifferentContent(t *testing.T) {
	m := NewInitModel()

	m.AddFragment("pgvector", "CREATE EXTENSION IF NOT EXISTS vector;")
	m.AddFragment("hypopg", "CREATE EXTENSION IF NOT EXISTS hypopg;")

	assert.Len(t, m.Fragments, 2)
}

func TestInitModel_GetOrderedFragments(t *testing.T) {
	m := NewInitModel()

	m.AddFragment("zebra", "CREATE EXTENSION IF NOT EXISTS zebra;")
	m.AddFragment("alpha", "CREATE EXTENSION IF NOT EXISTS alpha;")
	m.AddFragment("beta", "CREATE EXTENSION IF NOT EXISTS beta;")

	ordered := m.GetOrderedFragments()

	assert.Len(t, ordered, 3)
	assert.Equal(t, "alpha", ordered[0].Name)
	assert.Equal(t, "beta", ordered[1].Name)
	assert.Equal(t, "zebra", ordered[2].Name)
}

// appendUnique helper tests

func TestAppendUnique(t *testing.T) {
	result := appendUnique([]string{"a", "b"}, "c", "d")

	assert.Equal(t, []string{"a", "b", "c", "d"}, result)
}

func TestAppendUnique_WithDuplicates(t *testing.T) {
	result := appendUnique([]string{"a", "b"}, "b", "c", "a")

	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestAppendUnique_SortsResult(t *testing.T) {
	result := appendUnique([]string{}, "z", "a", "m")

	assert.Equal(t, []string{"a", "m", "z"}, result)
}
