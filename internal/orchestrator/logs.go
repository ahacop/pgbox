package orchestrator

import (
	"fmt"
	"io"

	"github.com/ahacop/pgbox/internal/docker"
)

// LogsConfig holds configuration for the logs command.
type LogsConfig struct {
	ContainerName string
	Follow        bool
}

// LogsOrchestrator handles showing PostgreSQL container logs.
type LogsOrchestrator struct {
	docker docker.Docker
	output io.Writer
}

// NewLogsOrchestrator creates a new LogsOrchestrator.
func NewLogsOrchestrator(d docker.Docker, w io.Writer) *LogsOrchestrator {
	return &LogsOrchestrator{docker: d, output: w}
}

// Run shows logs from the PostgreSQL container.
func (o *LogsOrchestrator) Run(cfg LogsConfig) error {
	name, autoDetected, err := ResolveContainerName(o.docker, cfg.ContainerName)
	if err != nil {
		return fmt.Errorf("%w. Start one with: pgbox up", err)
	}
	if autoDetected {
		fmt.Fprintf(o.output, "Showing logs for container: %s\n", name)
	}

	args := []string{"logs"}
	if cfg.Follow {
		args = append(args, "-f")
	}
	args = append(args, name)

	return o.docker.RunCommand(args...)
}
