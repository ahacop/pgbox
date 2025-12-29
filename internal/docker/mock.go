package docker

import "github.com/ahacop/pgbox/internal/config"

// MockDocker is a mock implementation of the Docker interface for testing.
type MockDocker struct {
	// RunCommandFunc is called when RunCommand is invoked.
	RunCommandFunc func(args ...string) error
	// RunCommandWithOutputFunc is called when RunCommandWithOutput is invoked.
	RunCommandWithOutputFunc func(args ...string) (string, error)
	// RunInteractiveFunc is called when RunInteractive is invoked.
	RunInteractiveFunc func(args ...string) error
	// IsContainerRunningFunc is called when IsContainerRunning is invoked.
	IsContainerRunningFunc func(name string) (bool, error)
	// GetContainerEnvFunc is called when GetContainerEnv is invoked.
	GetContainerEnvFunc func(containerName, envVar string) (string, error)
	// ListContainersFunc is called when ListContainers is invoked.
	ListContainersFunc func(prefix string) ([]string, error)
	// StopContainerFunc is called when StopContainer is invoked.
	StopContainerFunc func(name string) error
	// RemoveContainerFunc is called when RemoveContainer is invoked.
	RemoveContainerFunc func(name string) error
	// ExecCommandFunc is called when ExecCommand is invoked.
	ExecCommandFunc func(containerName string, command ...string) (string, error)
	// RunPostgresFunc is called when RunPostgres is invoked.
	RunPostgresFunc func(pgConfig *config.PostgresConfig, opts ContainerOptions) error
	// FindPgboxContainerFunc is called when FindPgboxContainer is invoked.
	FindPgboxContainerFunc func() (string, error)

	// Calls records the arguments passed to each method for assertions.
	Calls struct {
		RunCommand           [][]string
		RunCommandWithOutput [][]string
		RunInteractive       [][]string
		IsContainerRunning   []string
		GetContainerEnv      []struct{ Container, EnvVar string }
		ListContainers       []string
		StopContainer        []string
		RemoveContainer      []string
		ExecCommand          []struct {
			Container string
			Command   []string
		}
		RunPostgres []struct {
			Config *config.PostgresConfig
			Opts   ContainerOptions
		}
		FindPgboxContainer int
	}
}

// NewMockDocker creates a new MockDocker with default no-op implementations.
func NewMockDocker() *MockDocker {
	m := &MockDocker{}
	m.RunCommandFunc = func(args ...string) error { return nil }
	m.RunCommandWithOutputFunc = func(args ...string) (string, error) { return "", nil }
	m.RunInteractiveFunc = func(args ...string) error { return nil }
	m.IsContainerRunningFunc = func(name string) (bool, error) { return false, nil }
	m.GetContainerEnvFunc = func(containerName, envVar string) (string, error) { return "", nil }
	m.ListContainersFunc = func(prefix string) ([]string, error) { return nil, nil }
	m.StopContainerFunc = func(name string) error { return nil }
	m.RemoveContainerFunc = func(name string) error { return nil }
	m.ExecCommandFunc = func(containerName string, command ...string) (string, error) { return "", nil }
	m.RunPostgresFunc = func(pgConfig *config.PostgresConfig, opts ContainerOptions) error { return nil }
	m.FindPgboxContainerFunc = func() (string, error) { return "", nil }
	return m
}

func (m *MockDocker) RunCommand(args ...string) error {
	m.Calls.RunCommand = append(m.Calls.RunCommand, args)
	return m.RunCommandFunc(args...)
}

func (m *MockDocker) RunCommandWithOutput(args ...string) (string, error) {
	m.Calls.RunCommandWithOutput = append(m.Calls.RunCommandWithOutput, args)
	return m.RunCommandWithOutputFunc(args...)
}

func (m *MockDocker) RunInteractive(args ...string) error {
	m.Calls.RunInteractive = append(m.Calls.RunInteractive, args)
	return m.RunInteractiveFunc(args...)
}

func (m *MockDocker) IsContainerRunning(name string) (bool, error) {
	m.Calls.IsContainerRunning = append(m.Calls.IsContainerRunning, name)
	return m.IsContainerRunningFunc(name)
}

func (m *MockDocker) GetContainerEnv(containerName, envVar string) (string, error) {
	m.Calls.GetContainerEnv = append(m.Calls.GetContainerEnv, struct{ Container, EnvVar string }{containerName, envVar})
	return m.GetContainerEnvFunc(containerName, envVar)
}

func (m *MockDocker) ListContainers(prefix string) ([]string, error) {
	m.Calls.ListContainers = append(m.Calls.ListContainers, prefix)
	return m.ListContainersFunc(prefix)
}

func (m *MockDocker) StopContainer(name string) error {
	m.Calls.StopContainer = append(m.Calls.StopContainer, name)
	return m.StopContainerFunc(name)
}

func (m *MockDocker) RemoveContainer(name string) error {
	m.Calls.RemoveContainer = append(m.Calls.RemoveContainer, name)
	return m.RemoveContainerFunc(name)
}

func (m *MockDocker) ExecCommand(containerName string, command ...string) (string, error) {
	m.Calls.ExecCommand = append(m.Calls.ExecCommand, struct {
		Container string
		Command   []string
	}{containerName, command})
	return m.ExecCommandFunc(containerName, command...)
}

func (m *MockDocker) RunPostgres(pgConfig *config.PostgresConfig, opts ContainerOptions) error {
	m.Calls.RunPostgres = append(m.Calls.RunPostgres, struct {
		Config *config.PostgresConfig
		Opts   ContainerOptions
	}{pgConfig, opts})
	return m.RunPostgresFunc(pgConfig, opts)
}

func (m *MockDocker) FindPgboxContainer() (string, error) {
	m.Calls.FindPgboxContainer++
	return m.FindPgboxContainerFunc()
}

// Verify MockDocker implements Docker interface
var _ Docker = (*MockDocker)(nil)
