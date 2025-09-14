package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPostgresArgs(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		config   PostgresConfig
		expected []string
	}{
		{
			name: "basic config with password",
			config: PostgresConfig{
				Name:     "test-pg",
				Image:    "postgres:17",
				Port:     "5432",
				Database: "testdb",
				User:     "testuser",
				Password: "secret",
			},
			expected: []string{
				"run", "-d",
				"--name", "test-pg",
				"-p", "5432:5432",
				"-e", "POSTGRES_DB=testdb",
				"-e", "POSTGRES_USER=testuser",
				"-e", "POSTGRES_PASSWORD=secret",
				"postgres:17",
			},
		},
		{
			name: "config without password uses trust auth",
			config: PostgresConfig{
				Name:     "test-pg",
				Image:    "postgres:16",
				Port:     "5433",
				Database: "mydb",
				User:     "myuser",
				Password: "",
			},
			expected: []string{
				"run", "-d",
				"--name", "test-pg",
				"-p", "5433:5432",
				"-e", "POSTGRES_DB=mydb",
				"-e", "POSTGRES_USER=myuser",
				"-e", "POSTGRES_HOST_AUTH_METHOD=trust",
				"postgres:16",
			},
		},
		{
			name: "config with extra args and env",
			config: PostgresConfig{
				Name:      "test-pg",
				Image:     "postgres:17",
				Port:      "5432",
				Database:  "testdb",
				User:      "testuser",
				Password:  "secret",
				ExtraEnv:  []string{"PGDATA=/var/lib/postgresql/data/pgdata"},
				ExtraArgs: []string{"--rm", "-v", "pgdata:/var/lib/postgresql/data"},
			},
			expected: []string{
				"run", "-d",
				"--name", "test-pg",
				"-p", "5432:5432",
				"-e", "POSTGRES_DB=testdb",
				"-e", "POSTGRES_USER=testuser",
				"-e", "POSTGRES_PASSWORD=secret",
				"-e", "PGDATA=/var/lib/postgresql/data/pgdata",
				"--rm", "-v", "pgdata:/var/lib/postgresql/data",
				"postgres:17",
			},
		},
		{
			name: "config with custom command",
			config: PostgresConfig{
				Name:     "test-pg",
				Image:    "postgres:17",
				Port:     "5432",
				Database: "testdb",
				User:     "testuser",
				Password: "secret",
				Command:  []string{"-c", "shared_buffers=256MB"},
			},
			expected: []string{
				"run", "-d",
				"--name", "test-pg",
				"-p", "5432:5432",
				"-e", "POSTGRES_DB=testdb",
				"-e", "POSTGRES_USER=testuser",
				"-e", "POSTGRES_PASSWORD=secret",
				"postgres:17",
				"-c", "shared_buffers=256MB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.buildPostgresArgs(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}
