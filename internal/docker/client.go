package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
	"github.com/ahacop/pgbox/internal/container"
)

// Client provides an interface to Docker operations
type Client struct{}

// NewClient creates a new Docker client
func NewClient() *Client {
	return &Client{}
}

// RunCommand executes a docker command with the given arguments
func (c *Client) RunCommand(args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunCommandWithOutput executes a docker command and returns its output
func (c *Client) RunCommandWithOutput(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunInteractive executes a docker command interactively with TTY support
func (c *Client) RunInteractive(args ...string) error {
	return c.RunCommand(args...)
}

// IsContainerRunning checks if a container with the given name is running
func (c *Client) IsContainerRunning(name string) (bool, error) {
	output, err := c.RunCommandWithOutput("ps", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}

	containers := strings.Split(strings.TrimSpace(output), "\n")
	for _, container := range containers {
		if container == name {
			return true, nil
		}
	}
	return false, nil
}

// GetContainerEnv retrieves an environment variable from a running container
func (c *Client) GetContainerEnv(containerName, envVar string) (string, error) {
	output, err := c.RunCommandWithOutput("exec", containerName, "printenv", envVar)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// ListContainers returns a list of running container names matching a prefix
func (c *Client) ListContainers(prefix string) ([]string, error) {
	output, err := c.RunCommandWithOutput("ps", "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}

	var matching []string
	containers := strings.Split(strings.TrimSpace(output), "\n")
	for _, container := range containers {
		if strings.HasPrefix(container, prefix) {
			matching = append(matching, container)
		}
	}
	return matching, nil
}

// StopContainer stops a running container
func (c *Client) StopContainer(name string) error {
	return c.RunCommand("stop", name)
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(name string) error {
	return c.RunCommand("rm", "-f", name)
}

// ExecCommand executes a command inside a container and returns the output
func (c *Client) ExecCommand(containerName string, command ...string) (string, error) {
	args := append([]string{"exec", containerName}, command...)
	var out bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// ContainerOptions holds Docker-specific options for running a container
type ContainerOptions struct {
	Name      string
	ExtraEnv  []string
	ExtraArgs []string
	Command   []string
}

// RunPostgres runs a PostgreSQL container with the specified configuration
func (c *Client) RunPostgres(pgConfig *config.PostgresConfig, opts ContainerOptions) error {
	args := c.buildPostgresArgs(pgConfig, opts)
	// Debug: Print the command being executed
	// fmt.Printf("DEBUG: docker %s\n", strings.Join(args, " "))
	return c.RunCommand(args...)
}

// buildPostgresArgs builds the docker run arguments for PostgreSQL
func (c *Client) buildPostgresArgs(pgConfig *config.PostgresConfig, opts ContainerOptions) []string {
	args := []string{"run"}
	args = append(args, "--name", opts.Name)
	args = append(args, "-p", fmt.Sprintf("%s:5432", pgConfig.Port))

	args = append(args, "-e", fmt.Sprintf("POSTGRES_DB=%s", pgConfig.Database))
	args = append(args, "-e", fmt.Sprintf("POSTGRES_USER=%s", pgConfig.User))

	if pgConfig.Password != "" {
		args = append(args, "-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", pgConfig.Password))
	} else {
		args = append(args, "-e", "POSTGRES_HOST_AUTH_METHOD=trust")
	}

	for _, env := range opts.ExtraEnv {
		args = append(args, "-e", env)
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, pgConfig.Image())
	args = append(args, opts.Command...)

	return args
}

// FindPgboxContainer searches for running pgbox containers
// Returns the best matching container name or error if none found
func (c *Client) FindPgboxContainer() (string, error) {
	// Get list of running containers
	output, err := c.RunCommandWithOutput("ps", "--format", "{{.Names}}\t{{.Image}}")
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// Use the container package's selection logic
	containerName, err := container.SelectPgboxContainer(output)
	if err != nil {
		return "", err
	}

	return containerName, nil
}
