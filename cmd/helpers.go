package cmd

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
)

// ValidPostgresVersions contains the supported PostgreSQL versions.
var ValidPostgresVersions = []string{"16", "17"}

// ValidatePostgresVersion checks if the given version is a supported PostgreSQL version.
func ValidatePostgresVersion(version string) error {
	for _, v := range ValidPostgresVersions {
		if version == v {
			return nil
		}
	}
	return fmt.Errorf("invalid PostgreSQL version: %s (must be 16 or 17)", version)
}

// ParseExtensionList parses a comma-separated list of extensions and returns a slice.
// Returns nil if the input is empty.
func ParseExtensionList(extList string) []string {
	if extList == "" {
		return nil
	}
	parts := strings.Split(extList, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}

// ResolveRunningContainer resolves the container name to use.
// If containerName is provided, it validates that the container is running.
// If containerName is empty, it finds a running pgbox container.
// Returns the resolved container name or an error.
func ResolveRunningContainer(client *docker.Client, containerName string) (string, error) {
	if containerName == "" {
		foundName, err := client.FindPgboxContainer()
		if err != nil {
			return "", fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		return foundName, nil
	}

	// Container name was provided, verify it's running
	running, err := client.IsContainerRunning(containerName)
	if err != nil {
		return "", fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		return "", fmt.Errorf("container %s is not running. Start it with: pgbox up", containerName)
	}
	return containerName, nil
}

// FindContainer finds a running pgbox container without validating if it's running.
// This is useful for commands like 'down' that work on stopped containers too.
func FindContainer(client *docker.Client, containerName string) (string, error) {
	if containerName == "" {
		foundName, err := client.FindPgboxContainer()
		if err != nil {
			return "", fmt.Errorf("no running pgbox container found. Specify container name with -n flag")
		}
		return foundName, nil
	}
	return containerName, nil
}
