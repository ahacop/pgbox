package orchestrator

import (
	"fmt"
	"io"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
)

// StatusConfig holds configuration for the status command.
type StatusConfig struct {
	ContainerName string
}

// StatusOrchestrator handles showing PostgreSQL container status.
type StatusOrchestrator struct {
	docker docker.Docker
	output io.Writer
}

// NewStatusOrchestrator creates a new StatusOrchestrator.
func NewStatusOrchestrator(d docker.Docker, w io.Writer) *StatusOrchestrator {
	return &StatusOrchestrator{docker: d, output: w}
}

// Run shows the status of PostgreSQL containers.
func (o *StatusOrchestrator) Run(cfg StatusConfig) error {
	if cfg.ContainerName == "" {
		containers, err := o.docker.ListContainers("pgbox")
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers) == 0 {
			fmt.Fprintln(o.output, "No pgbox containers are running.")
			fmt.Fprintln(o.output, "\nStart a container with: pgbox up")
			return nil
		}

		fmt.Fprintln(o.output, "PostgreSQL containers:")
		output, err := o.docker.RunCommandWithOutput("ps", "--filter", "name=pgbox", "--format", "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
		if err != nil {
			return fmt.Errorf("failed to get container status: %w", err)
		}
		fmt.Fprintln(o.output, output)
		return nil
	}

	running, err := o.docker.IsContainerRunning(cfg.ContainerName)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		fmt.Fprintf(o.output, "Container '%s' is not running.\n", cfg.ContainerName)
		return nil
	}

	output, err := o.docker.RunCommandWithOutput("ps", "--filter", fmt.Sprintf("name=%s", cfg.ContainerName), "--format", "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
	if err != nil {
		return fmt.Errorf("failed to get container details: %w", err)
	}

	fmt.Fprintln(o.output, "Container status:")
	fmt.Fprintln(o.output, output)

	dbName, _ := o.docker.GetContainerEnv(cfg.ContainerName, "POSTGRES_DB")
	userName, _ := o.docker.GetContainerEnv(cfg.ContainerName, "POSTGRES_USER")

	if dbName != "" || userName != "" {
		fmt.Fprintln(o.output, "\nDatabase configuration:")
		if dbName != "" {
			fmt.Fprintf(o.output, "  Database: %s\n", dbName)
		}
		if userName != "" {
			fmt.Fprintf(o.output, "  User: %s\n", userName)
		}

		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				ports := fields[3]
				if strings.Contains(ports, "->") {
					portMapping := strings.Split(ports, "->")[0]
					port := strings.TrimPrefix(portMapping, "0.0.0.0:")
					port = strings.TrimPrefix(port, ":")

					fmt.Fprintln(o.output, "\nConnection string:")
					fmt.Fprintf(o.output, "  postgres://%s@localhost:%s/%s\n", userName, port, dbName)
				}
			}
		}
	}

	return nil
}
