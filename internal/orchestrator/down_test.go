package orchestrator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestDownOrchestrator_StopsNamedContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	var buf bytes.Buffer

	orch := NewDownOrchestrator(mock, &buf)
	err := orch.Run(DownConfig{
		ContainerName: "my-postgres",
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.StopContainer, 1)
	assert.Equal(t, "my-postgres", mock.Calls.StopContainer[0])
	assert.Contains(t, buf.String(), "Stopping container my-postgres")
	assert.Contains(t, buf.String(), "stopped successfully")
}

func TestDownOrchestrator_FindsRunningContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "pgbox-pg17", nil
	}
	var buf bytes.Buffer

	orch := NewDownOrchestrator(mock, &buf)
	err := orch.Run(DownConfig{})

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.Calls.FindPgboxContainer)
	assert.Len(t, mock.Calls.StopContainer, 1)
	assert.Equal(t, "pgbox-pg17", mock.Calls.StopContainer[0])
	assert.Contains(t, buf.String(), "Found running container: pgbox-pg17")
}

func TestDownOrchestrator_NoContainerFound(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "", errors.New("no container found")
	}
	var buf bytes.Buffer

	orch := NewDownOrchestrator(mock, &buf)
	err := orch.Run(DownConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running pgbox container found")
	assert.Len(t, mock.Calls.StopContainer, 0)
}

func TestDownOrchestrator_StopFails(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.StopContainerFunc = func(name string) error {
		return errors.New("docker daemon not responding")
	}
	var buf bytes.Buffer

	orch := NewDownOrchestrator(mock, &buf)
	err := orch.Run(DownConfig{
		ContainerName: "my-postgres",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop container")
	assert.Contains(t, err.Error(), "docker daemon not responding")
}
