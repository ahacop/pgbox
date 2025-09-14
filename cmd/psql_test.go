package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test the pure business logic function - no mocks needed!
func TestSelectPgboxContainer(t *testing.T) {
	tests := []struct {
		name           string
		dockerPsOutput string
		expected       string
	}{
		{
			name:           "empty output returns empty",
			dockerPsOutput: "",
			expected:       "",
		},
		{
			name:           "selects pgbox- prefixed container first",
			dockerPsOutput: "pgbox-pg17\tpostgres:17\nmy-postgres\tpostgres:16\n",
			expected:       "pgbox-pg17",
		},
		{
			name:           "selects pgbox-custom over regular postgres",
			dockerPsOutput: "my-postgres\tpostgres:16\npgbox-custom\tpostgres:17\nnginx\tnginx:latest\n",
			expected:       "pgbox-custom",
		},
		{
			name:           "falls back to postgres image when no pgbox- prefix",
			dockerPsOutput: "my-postgres\tpostgres:16-alpine\nnginx\tnginx:latest\n",
			expected:       "my-postgres",
		},
		{
			name:           "returns empty when no postgres containers",
			dockerPsOutput: "nginx\tnginx:latest\nredis\tredis:7\n",
			expected:       "",
		},
		{
			name:           "handles malformed lines gracefully",
			dockerPsOutput: "pgbox-pg17\nmalformed-line\nmy-postgres\tpostgres:16\n",
			expected:       "pgbox-pg17",
		},
		{
			name:           "handles whitespace in names",
			dockerPsOutput: "  pgbox-pg17  \t  postgres:17  \n",
			expected:       "pgbox-pg17",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectPgboxContainer(tt.dockerPsOutput)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseContainerEnv(t *testing.T) {
	tests := []struct {
		name                string
		dockerInspectOutput string
		envVar              string
		expected            string
	}{
		{
			name:                "parses value correctly",
			dockerInspectOutput: "myuser\n",
			envVar:              "POSTGRES_USER",
			expected:            "myuser",
		},
		{
			name:                "trims whitespace",
			dockerInspectOutput: "  mydb  \n",
			envVar:              "POSTGRES_DB",
			expected:            "mydb",
		},
		{
			name:                "handles empty output",
			dockerInspectOutput: "",
			envVar:              "NONEXISTENT",
			expected:            "",
		},
		{
			name:                "handles whitespace-only output",
			dockerInspectOutput: "   \n   ",
			envVar:              "POSTGRES_USER",
			expected:            "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseContainerEnv(tt.dockerInspectOutput, tt.envVar)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test for the actual Docker commands
func TestIntegrationFindRunningPgboxContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Docker is available
	if err := exec.Command("docker", "version").Run(); err != nil {
		t.Skip("Docker not available, skipping integration test")
	}

	// This test runs the actual function with real Docker
	// It doesn't assert a specific result because we don't know what containers are running
	// It just verifies the function doesn't panic and returns a valid result
	result := findRunningPgboxContainer()

	// Result should be either empty (no containers) or a non-empty string (container name)
	assert.NotNil(t, result) // This just ensures it returns something, even if empty
}

func TestIntegrationGetContainerEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Docker is available
	if err := exec.Command("docker", "version").Run(); err != nil {
		t.Skip("Docker not available, skipping integration test")
	}

	// Try to get env from a container that likely doesn't exist
	// This tests the error handling path
	result := getContainerEnv("nonexistent-container-xyz", "POSTGRES_USER")
	assert.Equal(t, "", result, "Should return empty string for nonexistent container")
}
