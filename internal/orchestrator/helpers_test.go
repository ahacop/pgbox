package orchestrator

import (
	"errors"
	"testing"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestResolveContainerName_WithExplicitName(t *testing.T) {
	mock := docker.NewMockDocker()

	name, autoDetected, err := ResolveContainerName(mock, "my-container")

	assert.NoError(t, err)
	assert.Equal(t, "my-container", name)
	assert.False(t, autoDetected)
	// FindPgboxContainer should not be called when name is provided
	assert.Empty(t, mock.Calls.FindPgboxContainer)
}

func TestResolveContainerName_AutoDetect(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "pgbox-pg17", nil
	}

	name, autoDetected, err := ResolveContainerName(mock, "")

	assert.NoError(t, err)
	assert.Equal(t, "pgbox-pg17", name)
	assert.True(t, autoDetected)
}

func TestResolveContainerName_NoContainerFound(t *testing.T) {
	mock := docker.NewMockDocker()
	mock.FindPgboxContainerFunc = func() (string, error) {
		return "", errors.New("no container found")
	}

	name, autoDetected, err := ResolveContainerName(mock, "")

	assert.ErrorIs(t, err, ErrNoContainer)
	assert.Empty(t, name)
	assert.False(t, autoDetected)
}
