package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ahacop/pgbox/internal/config"
)

type Manager struct {
	Config *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{Config: cfg}
}

func (m *Manager) EnsureTools() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is required but not found in PATH")
	}

	// Check if docker compose plugin is available
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose plugin is required")
	}

	return nil
}

func (m *Manager) ComposeUp(scaffoldPath string) error {
	cmd := exec.Command("docker", "compose", "-p", m.Config.Name, "up", "-d", "--build")
	cmd.Dir = scaffoldPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) ComposeDown(scaffoldPath string) error {
	cmd := exec.Command("docker", "compose", "-p", m.Config.Name, "down")
	cmd.Dir = scaffoldPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) ComposeRestart(scaffoldPath string) error {
	cmd := exec.Command("docker", "compose", "-p", m.Config.Name, "restart")
	cmd.Dir = scaffoldPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) ComposeLogs(scaffoldPath string, follow bool) error {
	args := []string{"compose", "-p", m.Config.Name, "logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, "db")

	cmd := exec.Command("docker", args...)
	cmd.Dir = scaffoldPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) Status() error {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=^%s$", m.Config.Name))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) RemoveVolume() error {
	cmd := exec.Command("docker", "volume", "rm", "-f", m.Config.DataVol)
	return cmd.Run()
}

func (m *Manager) ContainerExists() bool {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	containers := strings.Split(string(output), "\n")
	for _, container := range containers {
		if strings.TrimSpace(container) == m.Config.Name {
			return true
		}
	}
	return false
}

func (m *Manager) Psql(db, user string) error {
	if _, err := exec.LookPath("psql"); err != nil {
		return fmt.Errorf("psql not found in PATH")
	}

	connStr := fmt.Sprintf("postgres://%s:changeme@localhost:%s/%s", user, m.Config.Port, db)
	cmd := exec.Command("psql", connStr)
	cmd.Env = append(os.Environ(), "PGPASSWORD=changeme")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
