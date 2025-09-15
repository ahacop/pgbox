package docker

import (
	"testing"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildPostgresArgs(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		pgConfig *config.PostgresConfig
		opts     ContainerOptions
		expected []string
	}{
		{
			name: "basic config with password",
			pgConfig: &config.PostgresConfig{
				Version:  "17",
				Port:     "5432",
				Database: "testdb",
				User:     "testuser",
				Password: "secret",
			},
			opts: ContainerOptions{
				Name: "test-pg",
			},
			expected: []string{
				"run", "--name", "test-pg",
				"-p", "5432:5432",
				"-e", "POSTGRES_DB=testdb",
				"-e", "POSTGRES_USER=testuser",
				"-e", "POSTGRES_PASSWORD=secret",
				"postgres:17",
			},
		},
		{
			name: "config without password uses trust auth",
			pgConfig: &config.PostgresConfig{
				Version:  "16",
				Port:     "5433",
				Database: "mydb",
				User:     "myuser",
				Password: "",
			},
			opts: ContainerOptions{
				Name: "test-pg",
			},
			expected: []string{
				"run", "--name", "test-pg",
				"-p", "5433:5432",
				"-e", "POSTGRES_DB=mydb",
				"-e", "POSTGRES_USER=myuser",
				"-e", "POSTGRES_HOST_AUTH_METHOD=trust",
				"postgres:16",
			},
		},
		{
			name: "config with extra args and env",
			pgConfig: &config.PostgresConfig{
				Version:  "17",
				Port:     "5432",
				Database: "testdb",
				User:     "testuser",
				Password: "secret",
			},
			opts: ContainerOptions{
				Name:      "test-pg",
				ExtraEnv:  []string{"PGDATA=/var/lib/postgresql/data/pgdata"},
				ExtraArgs: []string{"--rm", "-v", "pgdata:/var/lib/postgresql/data"},
			},
			expected: []string{
				"run", "--name", "test-pg",
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
			pgConfig: &config.PostgresConfig{
				Version:  "17",
				Port:     "5432",
				Database: "testdb",
				User:     "testuser",
				Password: "secret",
			},
			opts: ContainerOptions{
				Name:    "test-pg",
				Command: []string{"-c", "shared_buffers=256MB"},
			},
			expected: []string{
				"run", "--name", "test-pg",
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
			result := client.buildPostgresArgs(tt.pgConfig, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
