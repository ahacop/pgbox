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
	name := cfg.ContainerName

	// Resolve container name (finds running container if not specified)
	if name == "" {
		foundName, err := o.docker.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		fmt.Fprintf(o.output, "Showing logs for container: %s\n", foundName)
		name = foundName
	}

	// Build docker logs command arguments
	args := []string{"logs"}
	if cfg.Follow {
		args = append(args, "-f")
	}
	args = append(args, name)

	// Show logs
	return o.docker.RunCommand(args...)
}
