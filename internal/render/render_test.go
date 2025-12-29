package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ahacop/pgbox/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create temp directory
func setupTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "pgbox-render-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// Helper to read file content
func readFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

// Dockerfile rendering tests

func TestRenderDockerfile_BasicAptPackages(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewDockerfileModel("postgres:17")
	m.AddPackages([]string{"postgresql-17-pgvector"}, "apt")

	err := RenderDockerfile(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assert.Contains(t, content, "FROM postgres:17")
	assert.Contains(t, content, "postgresql-17-pgvector")
	assert.Contains(t, content, "apt-get install")
}

func TestRenderDockerfile_DebURLs(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewDockerfileModel("postgres:17")
	m.AddDebURLs("https://example.com/ext.deb")

	err := RenderDockerfile(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assert.Contains(t, content, "https://example.com/ext.deb")
	assert.Contains(t, content, "dpkg -i")
}

func TestRenderDockerfile_ZipURLs(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewDockerfileModel("postgres:17")
	m.AddZipURLs("https://example.com/ext.zip")

	err := RenderDockerfile(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assert.Contains(t, content, "https://example.com/ext.zip")
	assert.Contains(t, content, "unzip")
}

func TestRenderDockerfile_NoPackages(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewDockerfileModel("postgres:17")

	err := RenderDockerfile(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assert.Contains(t, content, "FROM postgres:17")
	// Should not have apt-get commands when no packages
	assert.NotContains(t, content, "apt-get install")
}

// Compose rendering tests

func TestRenderCompose_Basic(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewComposeModel("db")
	m.Image = "postgres:17"
	m.AddPort("5432:5432")
	m.AddVolume("postgres_data:/var/lib/postgresql/data")
	m.SetEnv("POSTGRES_USER", "postgres")
	m.SetEnv("POSTGRES_PASSWORD", "postgres")

	pgConf := model.NewPGConfModel()

	err := RenderCompose(m, pgConf, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "docker-compose.yml"))
	assert.Contains(t, content, "services:")
	assert.Contains(t, content, "db:")
	assert.Contains(t, content, "5432:5432")
	assert.Contains(t, content, "postgres_data:/var/lib/postgresql/data")
	assert.Contains(t, content, "POSTGRES_USER: postgres")
}

func TestRenderCompose_WithBuildPath(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewComposeModel("db")
	m.BuildPath = "."
	m.Image = "postgres:17"

	pgConf := model.NewPGConfModel()

	err := RenderCompose(m, pgConf, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "docker-compose.yml"))
	assert.Contains(t, content, "build:")
	assert.Contains(t, content, "context: .")
	assert.Contains(t, content, "Dockerfile")
}

func TestRenderCompose_WithPGConf(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewComposeModel("db")
	m.Image = "postgres:17"

	pgConf := model.NewPGConfModel()
	pgConf.AddSharedPreload("pg_cron")
	pgConf.GUCs["cron.database_name"] = "postgres"

	err := RenderCompose(m, pgConf, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "docker-compose.yml"))
	assert.Contains(t, content, "command:")
	assert.Contains(t, content, "shared_preload_libraries=pg_cron")
	assert.Contains(t, content, "cron.database_name=postgres")
}

// Init SQL rendering tests

func TestRenderInitSQL_Basic(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewInitModel()
	m.AddFragment("pgvector", "CREATE EXTENSION IF NOT EXISTS vector;")

	err := RenderInitSQL(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "init.sql"))
	assert.Contains(t, content, "CREATE EXTENSION IF NOT EXISTS vector;")
	assert.Contains(t, content, "-- pgbox: begin pgvector")
	assert.Contains(t, content, "-- pgbox: end pgvector")
}

func TestRenderInitSQL_MultipleFragments(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewInitModel()
	m.AddFragment("pgvector", "CREATE EXTENSION IF NOT EXISTS vector;")
	m.AddFragment("hypopg", "CREATE EXTENSION IF NOT EXISTS hypopg;")

	err := RenderInitSQL(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "init.sql"))
	assert.Contains(t, content, "vector")
	assert.Contains(t, content, "hypopg")
}

func TestRenderInitSQL_Empty(t *testing.T) {
	dir := setupTempDir(t)
	m := model.NewInitModel()

	err := RenderInitSQL(m, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "init.sql"))
	// Should have header but no extension content
	assert.Contains(t, content, "Generated by pgbox")
}

// PostgreSQL conf rendering tests

func TestRenderPostgreSQLConf_WithSettings(t *testing.T) {
	dir := setupTempDir(t)
	pgConf := model.NewPGConfModel()
	pgConf.AddSharedPreload("pg_cron")
	pgConf.GUCs["cron.database_name"] = "postgres"

	err := RenderPostgreSQLConf(pgConf, dir)

	require.NoError(t, err)

	content := readFile(t, filepath.Join(dir, "postgresql.conf.pgbox"))
	assert.Contains(t, content, "shared_preload_libraries = 'pg_cron'")
	assert.Contains(t, content, "cron.database_name = postgres")
	assert.Contains(t, content, "ALTER SYSTEM")
}

func TestRenderPostgreSQLConf_Empty(t *testing.T) {
	dir := setupTempDir(t)
	pgConf := model.NewPGConfModel()

	err := RenderPostgreSQLConf(pgConf, dir)

	// Should not create file when nothing to configure
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "postgresql.conf.pgbox"))
	assert.True(t, os.IsNotExist(statErr))
}

// Common functions tests

func TestParseFileWithAnchors_NonExistent(t *testing.T) {
	parsed, err := ParseFileWithAnchors("/nonexistent/file", DockerfileAnchors)

	require.NoError(t, err)
	assert.False(t, parsed.HasAnchor)
	assert.Empty(t, parsed.PreAnchor)
	assert.Empty(t, parsed.Anchored)
	assert.Empty(t, parsed.PostAnchor)
}

func TestParseFileWithAnchors_WithAnchors(t *testing.T) {
	dir := setupTempDir(t)
	path := filepath.Join(dir, "test.txt")
	content := `before
# pgbox: BEGIN
anchored content
# pgbox: END
after`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	parsed, err := ParseFileWithAnchors(path, DockerfileAnchors)

	require.NoError(t, err)
	assert.True(t, parsed.HasAnchor)
	assert.Equal(t, []string{"before"}, parsed.PreAnchor)
	assert.Equal(t, []string{"anchored content"}, parsed.Anchored)
	assert.Equal(t, []string{"after"}, parsed.PostAnchor)
}

func TestReplaceAnchored(t *testing.T) {
	parsed := &ParsedFile{
		PreAnchor:  []string{"before"},
		Anchored:   []string{"old content"},
		PostAnchor: []string{"after"},
		HasAnchor:  true,
	}

	result := ReplaceAnchored(parsed, DockerfileAnchors, []string{"new content"})

	assert.Equal(t, []string{
		"before",
		"# pgbox: BEGIN",
		"new content",
		"# pgbox: END",
		"after",
	}, result)
}

func TestWriteLines(t *testing.T) {
	dir := setupTempDir(t)
	path := filepath.Join(dir, "test.txt")

	err := WriteLines(path, []string{"line1", "line2", "line3"})

	require.NoError(t, err)
	content := readFile(t, path)
	assert.Equal(t, "line1\nline2\nline3\n", content)
}

func TestIndentLines(t *testing.T) {
	lines := []string{"foo", "", "bar"}

	result := IndentLines(lines, 4)

	assert.Equal(t, []string{"    foo", "", "    bar"}, result)
}

func TestParseInitSQLAnchors_WithBlocks(t *testing.T) {
	dir := setupTempDir(t)
	path := filepath.Join(dir, "init.sql")
	content := `-- Header comment
-- pgbox: begin pgvector sha256=abc123
CREATE EXTENSION IF NOT EXISTS vector;
-- pgbox: end pgvector
-- pgbox: begin hypopg sha256=def456
CREATE EXTENSION IF NOT EXISTS hypopg;
-- pgbox: end hypopg`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	blocks, preContent, err := ParseInitSQLAnchors(path)

	require.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Contains(t, blocks, "pgvector")
	assert.Contains(t, blocks, "hypopg")
	assert.Contains(t, strings.Join(preContent, "\n"), "Header comment")
}

// generateAptInstall tests

func TestGenerateAptInstall_Empty(t *testing.T) {
	result := generateAptInstall("postgres:17", []string{})

	assert.Empty(t, result)
}

func TestGenerateAptInstall_WithPackages(t *testing.T) {
	result := generateAptInstall("postgres:17", []string{"postgresql-17-pgvector"})

	resultStr := strings.Join(result, "\n")
	assert.Contains(t, resultStr, "apt-get install")
	assert.Contains(t, resultStr, "postgresql-17-pgvector")
}

// generateDebInstall tests

func TestGenerateDebInstall_Empty(t *testing.T) {
	result := generateDebInstall([]string{})

	assert.Empty(t, result)
}

func TestGenerateDebInstall_WithURLs(t *testing.T) {
	result := generateDebInstall([]string{"https://example.com/ext.deb"})

	resultStr := strings.Join(result, "\n")
	assert.Contains(t, resultStr, "curl")
	assert.Contains(t, resultStr, "dpkg -i")
	assert.Contains(t, resultStr, "https://example.com/ext.deb")
}

// generateZipInstall tests

func TestGenerateZipInstall_Empty(t *testing.T) {
	result := generateZipInstall([]string{})

	assert.Empty(t, result)
}

func TestGenerateZipInstall_WithURLs(t *testing.T) {
	result := generateZipInstall([]string{"https://example.com/ext.zip"})

	resultStr := strings.Join(result, "\n")
	assert.Contains(t, resultStr, "unzip")
	assert.Contains(t, resultStr, "https://example.com/ext.zip")
}
