package orchestrator

import (
	"fmt"
	"io"

	"github.com/ahacop/pgbox/internal/docker"
)

// DownConfig holds configuration for the down command.
type DownConfig struct {
	ContainerName string
}

// DownOrchestrator handles stopping PostgreSQL containers.
type DownOrchestrator struct {
	docker docker.Docker
	output io.Writer
}

// NewDownOrchestrator creates a new DownOrchestrator.
func NewDownOrchestrator(d docker.Docker, w io.Writer) *DownOrchestrator {
	return &DownOrchestrator{docker: d, output: w}
}

// Run stops the PostgreSQL container.
func (o *DownOrchestrator) Run(cfg DownConfig) error {
	name, autoDetected, err := ResolveContainerName(o.docker, cfg.ContainerName)
	if err != nil {
		return fmt.Errorf("%w. Specify container name with -n flag", err)
	}
	if autoDetected {
		fmt.Fprintf(o.output, "Found running container: %s\n", name)
	}

	fmt.Fprintf(o.output, "Stopping container %s...\n", name)

	err = o.docker.StopContainer(name)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	fmt.Fprintf(o.output, "Container %s stopped successfully\n", name)
	return nil
}
