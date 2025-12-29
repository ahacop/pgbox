package orchestrator

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
)

// PsqlConfig holds configuration for the psql command.
type PsqlConfig struct {
	ContainerName string
	Database      string
	User          string
	ExtraArgs     []string // Additional psql arguments after --
	// For testing: allows overriding stdin terminal detection
	StdinIsTerminal *bool
}

// PsqlOrchestrator handles connecting to PostgreSQL via psql.
type PsqlOrchestrator struct {
	docker docker.Docker
	output io.Writer
}

// NewPsqlOrchestrator creates a new PsqlOrchestrator.
func NewPsqlOrchestrator(d docker.Docker, w io.Writer) *PsqlOrchestrator {
	return &PsqlOrchestrator{docker: d, output: w}
}

// Run connects to PostgreSQL via psql.
func (o *PsqlOrchestrator) Run(cfg PsqlConfig) error {
	name := cfg.ContainerName

	// Resolve container name (finds running container if not specified)
	if name == "" {
		foundName, err := o.docker.FindPgboxContainer()
		if err != nil {
			return fmt.Errorf("no running pgbox container found. Start one with: pgbox up")
		}
		name = foundName
	} else {
		// Verify container is running
		running, err := o.docker.IsContainerRunning(name)
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}
		if !running {
			return fmt.Errorf("container %s is not running. Start it with: pgbox up", name)
		}
	}

	// Get user/database from container env if not specified
	user := cfg.User
	database := cfg.Database

	if user == "" {
		if envUser, err := o.docker.GetContainerEnv(name, "POSTGRES_USER"); err == nil && envUser != "" {
			user = envUser
		} else {
			user = "postgres"
		}
	}
	if database == "" {
		if envDB, err := o.docker.GetContainerEnv(name, "POSTGRES_DB"); err == nil && envDB != "" {
			database = envDB
		} else {
			database = "postgres"
		}
	}

	// Build the psql command arguments
	psqlArgs := []string{"psql", "-U", user, "-d", database}
	psqlArgs = append(psqlArgs, cfg.ExtraArgs...)

	// Determine if stdin is a terminal
	stdinIsTerminal := false
	if cfg.StdinIsTerminal != nil {
		stdinIsTerminal = *cfg.StdinIsTerminal
	} else {
		if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			stdinIsTerminal = true
		}
	}

	// Determine if this is an interactive session
	isInteractive := stdinIsTerminal
	for _, arg := range psqlArgs {
		if arg == "-c" || arg == "--command" ||
			arg == "-f" || arg == "--file" ||
			arg == "-l" || arg == "--list" ||
			arg == "--help" || arg == "--version" {
			isInteractive = false
			break
		}
	}

	if isInteractive {
		fmt.Fprintf(o.output, "Connecting to %s as user '%s' to database '%s'...\n", name, user, database)
		fmt.Fprintln(o.output, "Type \\q to exit")
		fmt.Fprintln(o.output, strings.Repeat("-", 40))
	}

	// Build the full docker command
	dockerArgs := []string{"exec"}
	if isInteractive {
		dockerArgs = append(dockerArgs, "-it")
	} else if !stdinIsTerminal {
		dockerArgs = append(dockerArgs, "-i")
	}
	dockerArgs = append(dockerArgs, name)
	dockerArgs = append(dockerArgs, psqlArgs...)

	// Execute psql inside the container
	return o.docker.RunInteractive(dockerArgs...)
}
