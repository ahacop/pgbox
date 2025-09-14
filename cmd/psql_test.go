package cmd

import (
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
