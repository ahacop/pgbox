// Package docker provides Docker container operations
package docker

import "github.com/ahacop/pgbox/internal/config"

// Docker defines the interface for Docker operations.
// This interface enables unit testing by allowing mock implementations.
type Docker interface {
	// RunCommand executes a docker command with the given arguments,
	// streaming output to stdout/stderr.
	RunCommand(args ...string) error

	// RunCommandWithOutput executes a docker command and returns its output.
	RunCommandWithOutput(args ...string) (string, error)

	// RunInteractive executes a docker command interactively with TTY support.
	RunInteractive(args ...string) error

	// IsContainerRunning checks if a container with the given name is running.
	IsContainerRunning(name string) (bool, error)

	// GetContainerEnv retrieves an environment variable from a running container.
	GetContainerEnv(containerName, envVar string) (string, error)

	// ListContainers returns a list of running container names matching a prefix.
	ListContainers(prefix string) ([]string, error)

	// StopContainer stops a running container.
	StopContainer(name string) error

	// RemoveContainer removes a container.
	RemoveContainer(name string) error

	// ExecCommand executes a command inside a container and returns the output.
	ExecCommand(containerName string, command ...string) (string, error)

	// RunPostgres runs a PostgreSQL container with the specified configuration.
	RunPostgres(pgConfig *config.PostgresConfig, opts ContainerOptions) error

	// FindPgboxContainer searches for running pgbox containers.
	// Returns the best matching container name or error if none found.
	FindPgboxContainer() (string, error)
}

// Verify that Client implements Docker interface at compile time
var _ Docker = (*Client)(nil)
