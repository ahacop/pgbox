package orchestrator

import (
	"fmt"
	"io"

	"github.com/ahacop/pgbox/internal/docker"
)

// RestartConfig holds configuration for the restart command.
type RestartConfig struct {
	ContainerName string
}

// RestartOrchestrator handles restarting PostgreSQL containers.
type RestartOrchestrator struct {
	docker docker.Docker
	output io.Writer
}

// NewRestartOrchestrator creates a new RestartOrchestrator.
func NewRestartOrchestrator(d docker.Docker, w io.Writer) *RestartOrchestrator {
	return &RestartOrchestrator{docker: d, output: w}
}

// Run restarts the PostgreSQL container.
func (o *RestartOrchestrator) Run(cfg RestartConfig) error {
	name := cfg.ContainerName

	// Resolve container name (finds running container if not specified)
	if name == "" {
		foundName, err := o.docker.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		fmt.Fprintf(o.output, "Restarting container: %s\n", foundName)
		name = foundName
	}

	// Restart the container
	fmt.Fprintf(o.output, "Restarting container %s...\n", name)
	err := o.docker.RunCommand("restart", name)
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	fmt.Fprintf(o.output, "Container %s restarted successfully\n", name)
	return nil
}
