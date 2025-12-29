package orchestrator

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
)

// CleanConfig holds configuration for the clean command.
type CleanConfig struct {
	Force bool // Skip confirmation prompt
	All   bool // Also remove PostgreSQL base images
}

// CleanOrchestrator handles cleaning up pgbox resources.
type CleanOrchestrator struct {
	docker docker.Docker
	output io.Writer
	input  io.Reader
}

// NewCleanOrchestrator creates a new CleanOrchestrator.
func NewCleanOrchestrator(d docker.Docker, w io.Writer, r io.Reader) *CleanOrchestrator {
	return &CleanOrchestrator{docker: d, output: w, input: r}
}

// Run cleans up pgbox containers, volumes, and images.
func (o *CleanOrchestrator) Run(cfg CleanConfig) error {
	// Find all pgbox containers (running and stopped)
	fmt.Fprintln(o.output, "Searching for pgbox containers...")
	containersOutput, err := o.docker.RunCommandWithOutput("ps", "-a", "--filter", "name=pgbox", "--format", "{{.Names}}")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	containers := []string{}
	if containersOutput != "" {
		for _, line := range strings.Split(strings.TrimSpace(containersOutput), "\n") {
			if line != "" {
				containers = append(containers, line)
			}
		}
	}

	// Find all pgbox volumes
	fmt.Fprintln(o.output, "Searching for pgbox volumes...")
	volumesOutput, err := o.docker.RunCommandWithOutput("volume", "ls", "--format", "{{.Name}}")
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	volumes := []string{}
	if volumesOutput != "" {
		for _, line := range strings.Split(strings.TrimSpace(volumesOutput), "\n") {
			if line != "" && strings.HasPrefix(line, "pgbox-") && strings.HasSuffix(line, "-data") {
				volumes = append(volumes, line)
			}
		}
	}

	// Find all pgbox images
	fmt.Fprintln(o.output, "Searching for pgbox images...")
	imagesOutput, err := o.docker.RunCommandWithOutput("images", "--format", "{{.Repository}}:{{.Tag}}")
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	images := []string{}
	baseImages := []string{}
	for _, line := range strings.Split(strings.TrimSpace(imagesOutput), "\n") {
		if line != "" {
			if strings.HasPrefix(line, "pgbox-") {
				images = append(images, line)
			} else if cfg.All && (strings.HasPrefix(line, "postgres:") || strings.HasPrefix(line, "pgvector/pgvector:")) {
				baseImages = append(baseImages, line)
			}
		}
	}

	// Show what will be removed
	if len(containers) == 0 && len(volumes) == 0 && len(images) == 0 && len(baseImages) == 0 {
		fmt.Fprintln(o.output, "No pgbox resources found to clean.")
		return nil
	}

	fmt.Fprintln(o.output, "\nThe following resources will be removed:")
	if len(containers) > 0 {
		fmt.Fprintf(o.output, "\nContainers (%d):\n", len(containers))
		for _, c := range containers {
			fmt.Fprintf(o.output, "  - %s\n", c)
		}
	}
	if len(volumes) > 0 {
		fmt.Fprintf(o.output, "\nVolumes (%d):\n", len(volumes))
		for _, v := range volumes {
			fmt.Fprintf(o.output, "  - %s\n", v)
		}
	}
	if len(images) > 0 {
		fmt.Fprintf(o.output, "\nImages (%d):\n", len(images))
		for _, img := range images {
			fmt.Fprintf(o.output, "  - %s\n", img)
		}
	}
	if len(baseImages) > 0 {
		fmt.Fprintf(o.output, "\nBase Images (%d):\n", len(baseImages))
		for _, img := range baseImages {
			fmt.Fprintf(o.output, "  - %s\n", img)
		}
	}

	// Confirm unless --force
	if !cfg.Force {
		fmt.Fprint(o.output, "\nAre you sure you want to remove these resources? (y/N): ")
		reader := bufio.NewReader(o.input)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(o.output, "Clean cancelled.")
			return nil
		}
	}

	// Remove containers
	if len(containers) > 0 {
		fmt.Fprintln(o.output, "\nRemoving containers...")
		for _, container := range containers {
			fmt.Fprintf(o.output, "  Removing %s...", container)
			if err := o.docker.RemoveContainer(container); err != nil {
				fmt.Fprintf(o.output, " failed: %v\n", err)
			} else {
				fmt.Fprintln(o.output, " done")
			}
		}
	}

	// Remove volumes
	if len(volumes) > 0 {
		fmt.Fprintln(o.output, "\nRemoving volumes...")
		for _, volume := range volumes {
			fmt.Fprintf(o.output, "  Removing %s...", volume)
			if _, err := o.docker.RunCommandWithOutput("volume", "rm", volume); err != nil {
				fmt.Fprintf(o.output, " failed: %v\n", err)
			} else {
				fmt.Fprintln(o.output, " done")
			}
		}
	}

	// Remove images
	allImages := append(images, baseImages...)
	if len(allImages) > 0 {
		fmt.Fprintln(o.output, "\nRemoving images...")
		for _, image := range allImages {
			fmt.Fprintf(o.output, "  Removing %s...", image)
			if _, err := o.docker.RunCommandWithOutput("rmi", image); err != nil {
				// Try force remove if normal remove fails
				if _, err := o.docker.RunCommandWithOutput("rmi", "-f", image); err != nil {
					fmt.Fprintf(o.output, " failed: %v\n", err)
				} else {
					fmt.Fprintln(o.output, " done (forced)")
				}
			} else {
				fmt.Fprintln(o.output, " done")
			}
		}
	}

	// Also clean up any temp files
	fmt.Fprintln(o.output, "\nCleaning temporary files...")
	if output, err := o.docker.RunCommandWithOutput("run", "--rm", "-v", "/tmp:/tmp", "alpine", "sh", "-c", "rm -f /tmp/pgbox-*.sql /tmp/pgbox-*.yml"); err != nil {
		// Non-critical error, just warn
		fmt.Fprintf(o.output, "  Warning: Could not clean temp files: %v\n", err)
	} else if output != "" {
		fmt.Fprintf(o.output, "  Cleaned: %s\n", output)
	}

	fmt.Fprintln(o.output, "\nClean completed successfully.")
	return nil
}
