package orchestrator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestRestartOrchestrator_RestartsNamedContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	var buf bytes.Buffer

	orch := NewRestartOrchestrator(mock, &buf)
	err := orch.Run(RestartConfig{
		ContainerName: "my-postgres",
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RunCommand, 1)
	assert.Equal(t, []string{"restart", "my-postgres"}, mock.Calls.RunCommand[0])
	assert.Contains(t, buf.String(), "Restarting container my-postgres")
	assert.Contains(t, buf.String(), "restarted successfully")
}

func TestRestartOrchestrator_FindsRunningContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "pgbox-pg17", nil
	}
	var buf bytes.Buffer

	orch := NewRestartOrchestrator(mock, &buf)
	err := orch.Run(RestartConfig{})

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.Calls.FindPgboxContainer)
	assert.Equal(t, []string{"restart", "pgbox-pg17"}, mock.Calls.RunCommand[0])
	assert.Contains(t, buf.String(), "Restarting container: pgbox-pg17")
}

func TestRestartOrchestrator_NoContainerFound(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "", errors.New("no container found")
	}
	var buf bytes.Buffer

	orch := NewRestartOrchestrator(mock, &buf)
	err := orch.Run(RestartConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running pgbox container found")
	assert.Len(t, mock.Calls.RunCommand, 0)
}

func TestRestartOrchestrator_RestartFails(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandFunc = func(args ...string) error {
		return errors.New("docker daemon not responding")
	}
	var buf bytes.Buffer

	orch := NewRestartOrchestrator(mock, &buf)
	err := orch.Run(RestartConfig{
		ContainerName: "my-postgres",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restart container")
}
