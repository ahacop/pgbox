package orchestrator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestStatusOrchestrator_NoContainersRunning(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.ListContainersFunc = func(prefix string) ([]string, error) {
		return nil, nil
	}
	var buf bytes.Buffer

	orch := NewStatusOrchestrator(mock, &buf)
	err := orch.Run(StatusConfig{})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No pgbox containers are running")
	assert.Contains(t, buf.String(), "pgbox up")
}

func TestStatusOrchestrator_ListsAllContainers(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.ListContainersFunc = func(prefix string) ([]string, error) {
		return []string{"pgbox-pg17", "pgbox-pg16"}, nil
	}
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		return "NAMES\tIMAGE\tSTATUS\tPORTS\npgbox-pg17\tpostgres:17\tUp 2 hours\t0.0.0.0:5432->5432/tcp", nil
	}
	var buf bytes.Buffer

	orch := NewStatusOrchestrator(mock, &buf)
	err := orch.Run(StatusConfig{})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "PostgreSQL containers:")
	assert.Contains(t, buf.String(), "pgbox-pg17")
}

func TestStatusOrchestrator_SpecificContainerNotRunning(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return false, nil
	}
	var buf bytes.Buffer

	orch := NewStatusOrchestrator(mock, &buf)
	err := orch.Run(StatusConfig{ContainerName: "my-postgres"})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Container 'my-postgres' is not running")
}

func TestStatusOrchestrator_SpecificContainerRunning(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return true, nil
	}
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		return "NAMES\tIMAGE\tSTATUS\tPORTS\nmy-postgres\tpostgres:17\tUp 2 hours\t0.0.0.0:5432->5432/tcp", nil
	}
	mock.GetContainerEnvFunc = func(containerName, envVar string) (string, error) {
		switch envVar {
		case "POSTGRES_DB":
			return "mydb", nil
		case "POSTGRES_USER":
			return "myuser", nil
		}
		return "", nil
	}
	var buf bytes.Buffer

	orch := NewStatusOrchestrator(mock, &buf)
	err := orch.Run(StatusConfig{ContainerName: "my-postgres"})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Container status:")
	assert.Contains(t, buf.String(), "my-postgres")
	assert.Contains(t, buf.String(), "Database: mydb")
	assert.Contains(t, buf.String(), "User: myuser")
}

func TestStatusOrchestrator_ListContainersFails(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.ListContainersFunc = func(prefix string) ([]string, error) {
		return nil, errors.New("docker not available")
	}
	var buf bytes.Buffer

	orch := NewStatusOrchestrator(mock, &buf)
	err := orch.Run(StatusConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list containers")
}
