package orchestrator

import (
	"fmt"

	"github.com/ahacop/pgbox/internal/docker"
)

// ErrNoContainer is returned when no pgbox container is found.
var ErrNoContainer = fmt.Errorf("no running pgbox container found")

// ResolveContainerName resolves the container name, finding a running pgbox container
// if name is empty. Returns the resolved name and whether it was auto-detected.
// Returns ErrNoContainer if name is empty and no container is found.
func ResolveContainerName(d docker.Docker, name string) (resolvedName string, autoDetected bool, err error) {
	if name != "" {
		return name, false, nil
	}

	foundName, err := d.FindPgboxContainer()
	if err != nil {
		return "", false, ErrNoContainer
	}

	return foundName, true, nil
}
