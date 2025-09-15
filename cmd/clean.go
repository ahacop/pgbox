package cmd

import (
	"fmt"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

func CleanCmd() *cobra.Command {
	var force bool
	var all bool

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove pgbox containers and images",
		Long: `Remove pgbox-related Docker containers and images to free up space and clear cache.

By default, this command will:
- Stop and remove all running pgbox containers
- Remove all pgbox Docker images

Use --all to also remove PostgreSQL base images.`,
		Example: `  # Clean pgbox containers and images
  pgbox clean

  # Clean without confirmation prompt
  pgbox clean --force

  # Clean everything including PostgreSQL base images
  pgbox clean --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanPgbox(force, all)
		},
	}

	cleanCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cleanCmd.Flags().BoolVarP(&all, "all", "a", false, "Also remove PostgreSQL base images")

	return cleanCmd
}

func cleanPgbox(force bool, all bool) error {
	client := docker.NewClient()

	// Find all pgbox containers (running and stopped)
	fmt.Println("Searching for pgbox containers...")
	containersOutput, err := client.RunCommandWithOutput("ps", "-a", "--filter", "name=pgbox", "--format", "{{.Names}}")
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

	// Find all pgbox images
	fmt.Println("Searching for pgbox images...")
	imagesOutput, err := client.RunCommandWithOutput("images", "--format", "{{.Repository}}:{{.Tag}}")
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	images := []string{}
	baseImages := []string{}
	for _, line := range strings.Split(strings.TrimSpace(imagesOutput), "\n") {
		if line != "" {
			if strings.HasPrefix(line, "pgbox-") {
				images = append(images, line)
			} else if all && (strings.HasPrefix(line, "postgres:") || strings.HasPrefix(line, "pgvector/pgvector:")) {
				baseImages = append(baseImages, line)
			}
		}
	}

	// Show what will be removed
	if len(containers) == 0 && len(images) == 0 && len(baseImages) == 0 {
		fmt.Println("No pgbox resources found to clean.")
		return nil
	}

	fmt.Println("\nThe following resources will be removed:")
	if len(containers) > 0 {
		fmt.Printf("\nContainers (%d):\n", len(containers))
		for _, c := range containers {
			fmt.Printf("  - %s\n", c)
		}
	}
	if len(images) > 0 {
		fmt.Printf("\nImages (%d):\n", len(images))
		for _, img := range images {
			fmt.Printf("  - %s\n", img)
		}
	}
	if len(baseImages) > 0 {
		fmt.Printf("\nBase Images (%d):\n", len(baseImages))
		for _, img := range baseImages {
			fmt.Printf("  - %s\n", img)
		}
	}

	// Confirm unless --force
	if !force {
		fmt.Print("\nAre you sure you want to remove these resources? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Clean cancelled.")
			return nil
		}
	}

	// Remove containers
	if len(containers) > 0 {
		fmt.Println("\nRemoving containers...")
		for _, container := range containers {
			fmt.Printf("  Removing %s...", container)
			if err := client.RemoveContainer(container); err != nil {
				fmt.Printf(" failed: %v\n", err)
			} else {
				fmt.Println(" done")
			}
		}
	}

	// Remove images
	allImages := append(images, baseImages...)
	if len(allImages) > 0 {
		fmt.Println("\nRemoving images...")
		for _, image := range allImages {
			fmt.Printf("  Removing %s...", image)
			if _, err := client.RunCommandWithOutput("rmi", image); err != nil {
				// Try force remove if normal remove fails
				if _, err := client.RunCommandWithOutput("rmi", "-f", image); err != nil {
					fmt.Printf(" failed: %v\n", err)
				} else {
					fmt.Println(" done (forced)")
				}
			} else {
				fmt.Println(" done")
			}
		}
	}

	// Also clean up any temp files
	fmt.Println("\nCleaning temporary files...")
	if output, err := client.RunCommandWithOutput("run", "--rm", "-v", "/tmp:/tmp", "alpine", "sh", "-c", "rm -f /tmp/pgbox-*.sql /tmp/pgbox-*.yml"); err != nil {
		// Non-critical error, just warn
		fmt.Printf("  Warning: Could not clean temp files: %v\n", err)
	} else if output != "" {
		fmt.Printf("  Cleaned: %s\n", output)
	}

	fmt.Println("\nClean completed successfully.")
	return nil
}
