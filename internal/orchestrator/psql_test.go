package orchestrator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestPsqlOrchestrator_ConnectsToNamedContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return true, nil
	}
	mock.GetContainerEnvFunc = func(containerName, envVar string) (string, error) {
		return "", nil // No env vars set
	}
	var buf bytes.Buffer
	notTerminal := false

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{
		ContainerName:   "my-postgres",
		User:            "testuser",
		Database:        "testdb",
		StdinIsTerminal: &notTerminal,
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RunInteractive, 1)
	// Non-interactive (not terminal), so should use -i flag only
	assert.Equal(t, []string{"exec", "-i", "my-postgres", "psql", "-U", "testuser", "-d", "testdb"}, mock.Calls.RunInteractive[0])
}

func TestPsqlOrchestrator_InteractiveSession(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return true, nil
	}
	var buf bytes.Buffer
	isTerminal := true

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{
		ContainerName:   "my-postgres",
		User:            "postgres",
		Database:        "postgres",
		StdinIsTerminal: &isTerminal,
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RunInteractive, 1)
	// Interactive session should use -it flags
	assert.Equal(t, []string{"exec", "-it", "my-postgres", "psql", "-U", "postgres", "-d", "postgres"}, mock.Calls.RunInteractive[0])
	assert.Contains(t, buf.String(), "Connecting to my-postgres")
	assert.Contains(t, buf.String(), "Type \\q to exit")
}

func TestPsqlOrchestrator_NonInteractiveWithCommand(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return true, nil
	}
	var buf bytes.Buffer
	isTerminal := true // Even with terminal, -c makes it non-interactive

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{
		ContainerName:   "my-postgres",
		User:            "postgres",
		Database:        "postgres",
		ExtraArgs:       []string{"-c", "SELECT 1;"},
		StdinIsTerminal: &isTerminal,
	})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RunInteractive, 1)
	// -c flag makes it non-interactive, no -it flags
	args := mock.Calls.RunInteractive[0]
	assert.Equal(t, "exec", args[0])
	assert.Equal(t, "my-postgres", args[1])
	assert.NotContains(t, buf.String(), "Connecting to")
}

func TestPsqlOrchestrator_FindsRunningContainer(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "pgbox-pg17", nil
	}
	mock.GetContainerEnvFunc = func(containerName, envVar string) (string, error) {
		switch envVar {
		case "POSTGRES_USER":
			return "myuser", nil
		case "POSTGRES_DB":
			return "mydb", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	notTerminal := false

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{
		StdinIsTerminal: &notTerminal,
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.Calls.FindPgboxContainer)
	// Should use env vars for user and database
	args := mock.Calls.RunInteractive[0]
	assert.Contains(t, args, "myuser")
	assert.Contains(t, args, "mydb")
}

func TestPsqlOrchestrator_NoContainerFound(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "", errors.New("no container found")
	}
	var buf bytes.Buffer

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no running pgbox container found")
}

func TestPsqlOrchestrator_ContainerNotRunning(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return false, nil
	}
	var buf bytes.Buffer

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{
		ContainerName: "my-postgres",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container my-postgres is not running")
}

func TestPsqlOrchestrator_ExtraArgs(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.IsContainerRunningFunc = func(name string) (bool, error) {
		return true, nil
	}
	var buf bytes.Buffer
	notTerminal := false

	orch := NewPsqlOrchestrator(mock, &buf)
	err := orch.Run(PsqlConfig{
		ContainerName:   "my-postgres",
		User:            "postgres",
		Database:        "postgres",
		ExtraArgs:       []string{"-t", "-A", "-c", "SELECT 1;"},
		StdinIsTerminal: &notTerminal,
	})

	assert.NoError(t, err)
	args := mock.Calls.RunInteractive[0]
	assert.Contains(t, args, "-t")
	assert.Contains(t, args, "-A")
	assert.Contains(t, args, "-c")
	assert.Contains(t, args, "SELECT 1;")
}
