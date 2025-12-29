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
	name := cfg.ContainerName

	// Try to find a running container if name not specified
	if name == "" {
		foundName, err := o.docker.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Specify container name with -n flag")
		}
		fmt.Fprintf(o.output, "Found running container: %s\n", foundName)
		name = foundName
	}

	fmt.Fprintf(o.output, "Stopping container %s...\n", name)

	err := o.docker.StopContainer(name)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	fmt.Fprintf(o.output, "Container %s stopped successfully\n", name)
	return nil
}
