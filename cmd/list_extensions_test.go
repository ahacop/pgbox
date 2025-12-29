package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListExtensionsCmd(t *testing.T) {
	cmd := ListExtensionsCmd()

	// Verify command configuration
	assert.Equal(t, "list-extensions", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestListExtensions_ReturnsExtensions(t *testing.T) {
	// Capture stdout
	var buf bytes.Buffer
	cmd := ListExtensionsCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should list extensions with count header
	assert.Contains(t, output, "PostgreSQL Extensions")
	assert.Contains(t, output, "available")

	// Should include some known extensions
	assert.Contains(t, output, "pgvector")
	assert.Contains(t, output, "hstore")
	assert.Contains(t, output, "pg_cron")
}

func TestListExtensions_SourceFlag(t *testing.T) {
	var buf bytes.Buffer
	cmd := ListExtensionsCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--source"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// With --source, should show source info
	assert.Contains(t, output, "builtin")
	assert.Contains(t, output, "apt")
}

func TestListExtensions_KindFilterBuiltin(t *testing.T) {
	var buf bytes.Buffer
	cmd := ListExtensionsCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--kind", "builtin"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should include builtin extensions
	assert.Contains(t, output, "hstore")
	assert.Contains(t, output, "ltree")

	// Should NOT include package extensions (pgvector requires apt)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// pgvector should not appear when filtering for builtin
		if strings.TrimSpace(line) == "pgvector" {
			t.Error("pgvector should not appear in builtin filter")
		}
	}
}

func TestListExtensions_KindFilterPackage(t *testing.T) {
	var buf bytes.Buffer
	cmd := ListExtensionsCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--kind", "package"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should include package extensions
	assert.Contains(t, output, "pgvector")
	assert.Contains(t, output, "hypopg")

	// Should NOT include builtin extensions
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// hstore is builtin, should not appear
		if trimmed == "hstore" {
			t.Error("hstore should not appear in package filter")
		}
	}
}
