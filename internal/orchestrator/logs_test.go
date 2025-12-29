package orchestrator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestLogsOrchestrator_ShowsLogsForNamedContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	var buf bytes.Buffer

	orch := NewLogsOrchestrator(mock, &buf)
	err := orch.Run(LogsConfig{
		ContainerName: "my-postgres",
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RunCommand, 1)
	assert.Equal(t, []string{"logs", "my-postgres"}, mock.Calls.RunCommand[0])
}

func TestLogsOrchestrator_FollowFlag(t *testing.T) {
	mock := docker.NewMockDocker()
	var buf bytes.Buffer

	orch := NewLogsOrchestrator(mock, &buf)
	err := orch.Run(LogsConfig{
		ContainerName: "my-postgres",
		Follow:        true,
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RunCommand, 1)
	assert.Equal(t, []string{"logs", "-f", "my-postgres"}, mock.Calls.RunCommand[0])
}

func TestLogsOrchestrator_FindsRunningContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "pgbox-pg17", nil
	}
	var buf bytes.Buffer

	orch := NewLogsOrchestrator(mock, &buf)
	err := orch.Run(LogsConfig{})

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.Calls.FindPgboxContainer)
	assert.Contains(t, buf.String(), "Showing logs for container: pgbox-pg17")
	assert.Equal(t, []string{"logs", "pgbox-pg17"}, mock.Calls.RunCommand[0])
}

func TestLogsOrchestrator_NoContainerFound(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "", errors.New("no container found")
	}
	var buf bytes.Buffer

	orch := NewLogsOrchestrator(mock, &buf)
	err := orch.Run(LogsConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running pgbox container found")
}
