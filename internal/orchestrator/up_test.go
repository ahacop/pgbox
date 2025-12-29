package orchestrator

import (
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestUpOrchestrator_RestartExistingContainer(t *testing.T) {
	mock := docker.NewMockDocker()

	// Simulate existing container found
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 4 && args[0] == "ps" && args[1] == "-a" {
			return "pgbox-pg17\n", nil
		}
		return "", nil
	}

	orch := NewUpOrchestrator(mock)
	err := orch.Run(UpConfig{
		Version: "17",
	})

	assert.NoError(t, err)

	// Verify start was called
	assert.Len(t, mock.Calls.RunCommand, 1)
	assert.Equal(t, []string{"start", "pgbox-pg17"}, mock.Calls.RunCommand[0])
}

func TestUpOrchestrator_NewContainer(t *testing.T) {
	mock := docker.NewMockDocker()

	// Simulate no existing container
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		return "", nil
	}

	orch := NewUpOrchestrator(mock)
	err := orch.Run(UpConfig{
		Version:  "17",
		Port:     "5432",
		Database: "testdb",
		User:     "testuser",
		Password: "secret",
		Detach:   true,
	})

	assert.NoError(t, err)

	// Verify RunPostgres was called
	assert.Len(t, mock.Calls.RunPostgres, 1)
	assert.Equal(t, "17", mock.Calls.RunPostgres[0].Config.Version)
	assert.Equal(t, "testdb", mock.Calls.RunPostgres[0].Config.Database)
	assert.Equal(t, "testuser", mock.Calls.RunPostgres[0].Config.User)
}

func TestUpOrchestrator_CustomContainerName(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		return "", nil
	}

	orch := NewUpOrchestrator(mock)
	err := orch.Run(UpConfig{
		Version:       "17",
		ContainerName: "my-custom-pg",
		Detach:        true,
	})

	assert.NoError(t, err)

	// Verify container name was used
	assert.Len(t, mock.Calls.RunPostgres, 1)
	assert.Equal(t, "my-custom-pg", mock.Calls.RunPostgres[0].Opts.Name)
}
