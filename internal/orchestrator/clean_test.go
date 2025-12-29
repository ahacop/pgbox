package orchestrator

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestCleanOrchestrator_NoResources(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		return "", nil // No containers, volumes, or images
	}
	var buf bytes.Buffer
	input := strings.NewReader("")

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: true})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No pgbox resources found to clean")
}

func TestCleanOrchestrator_RemovesContainers(t *testing.T) {
	mock := docker.NewMockDocker()
	callCount := 0
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		callCount++
		if len(args) >= 2 && args[0] == "ps" {
			return "pgbox-pg17\npgbox-pg16", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("")

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: true})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RemoveContainer, 2)
	assert.Contains(t, mock.Calls.RemoveContainer, "pgbox-pg17")
	assert.Contains(t, mock.Calls.RemoveContainer, "pgbox-pg16")
	assert.Contains(t, buf.String(), "Removing containers")
}

func TestCleanOrchestrator_RemovesVolumes(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "volume" && args[1] == "ls" {
			return "pgbox-pg17-data\npgbox-pg16-data\nother-volume", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("")

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: true})

	assert.NoError(t, err)
	// Should have called volume rm for the two pgbox volumes
	volumeRmCalls := 0
	for _, call := range mock.Calls.RunCommandWithOutput {
		if len(call) >= 3 && call[0] == "volume" && call[1] == "rm" {
			volumeRmCalls++
		}
	}
	assert.Equal(t, 2, volumeRmCalls)
	assert.Contains(t, buf.String(), "Removing volumes")
}

func TestCleanOrchestrator_RemovesImages(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 1 && args[0] == "images" {
			return "pgbox-pg17:latest\npgbox-pg16:latest\nalpine:latest", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("")

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: true})

	assert.NoError(t, err)
	// Should have called rmi for the two pgbox images
	rmiCalls := 0
	for _, call := range mock.Calls.RunCommandWithOutput {
		if len(call) >= 2 && call[0] == "rmi" {
			rmiCalls++
		}
	}
	assert.Equal(t, 2, rmiCalls)
	assert.Contains(t, buf.String(), "Removing images")
}

func TestCleanOrchestrator_AllFlag_IncludesBaseImages(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 1 && args[0] == "images" {
			return "pgbox-pg17:latest\npostgres:17\npostgres:16", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("")

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: true, All: true})

	assert.NoError(t, err)
	// Should have called rmi for pgbox image AND postgres base images
	rmiCalls := 0
	for _, call := range mock.Calls.RunCommandWithOutput {
		if len(call) >= 2 && call[0] == "rmi" {
			rmiCalls++
		}
	}
	assert.Equal(t, 3, rmiCalls)
	assert.Contains(t, buf.String(), "Base Images")
}

func TestCleanOrchestrator_ConfirmationRequired(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "ps" {
			return "pgbox-pg17", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("n\n") // User says no

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: false})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Are you sure")
	assert.Contains(t, buf.String(), "Clean cancelled")
	assert.Len(t, mock.Calls.RemoveContainer, 0) // Nothing should be removed
}

func TestCleanOrchestrator_ConfirmationAccepted(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "ps" {
			return "pgbox-pg17", nil
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("y\n") // User says yes

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: false})

	assert.NoError(t, err)
	assert.Len(t, mock.Calls.RemoveContainer, 1)
}

func TestCleanOrchestrator_ListContainersFails(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.RunCommandWithOutputFunc = func(args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "ps" {
			return "", errors.New("docker not available")
		}
		return "", nil
	}
	var buf bytes.Buffer
	input := strings.NewReader("")

	orch := NewCleanOrchestrator(mock, &buf, input)
	err := orch.Run(CleanConfig{Force: true})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list containers")
}
