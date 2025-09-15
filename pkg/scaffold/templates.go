package scaffold

import (
	"bytes"
	"fmt"
	"text/template"
)

// DockerfileData contains the data needed to generate a Dockerfile
type DockerfileData struct {
	PGMajor     string
	HasPackages bool
	Packages    []string
}

// DockerComposeData contains the data needed to generate a docker-compose.yml
type DockerComposeData struct {
	PGMajor      string
	ContainerName string
	Port         string
	User         string
	Password     string
	Database     string
	HasExtensions bool
}

// InitSQLData contains the data needed to generate init.sql
type InitSQLData struct {
	Extensions []ExtensionInfo
}

// ExtensionInfo contains information about an extension for SQL generation
type ExtensionInfo struct {
	Name   string
	SQLName string // The actual name to use in CREATE EXTENSION (e.g., "vector" for pgvector)
}

const dockerfileTemplate = `ARG PG_MAJOR={{.PGMajor}}
FROM postgres:${PG_MAJOR}
{{if .HasPackages}}
# Install PostgreSQL extensions
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends curl gnupg ca-certificates lsb-release; \
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql.gpg; \
    echo "deb [signed-by=/usr/share/keyrings/postgresql.gpg] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
{{- range $i, $pkg := .Packages}}
        {{$pkg}}{{if lt $i (sub (len $.Packages) 1)}} \{{else}}; \{{end}}
{{- end}}
    apt-get purge -y --auto-remove curl gnupg lsb-release; \
    rm -rf /var/lib/apt/lists/*
{{- else}}
# This Dockerfile can be customized to add extensions or other PostgreSQL configurations
{{- end}}
`

const dockerComposeTemplate = `version: '3.8'

services:
  postgres:
    {{- if .HasExtensions}}
    build:
      context: .
      dockerfile: Dockerfile
      args:
        PG_MAJOR: "{{.PGMajor}}"
    {{- else}}
    image: postgres:{{.PGMajor}}
    {{- end}}
    container_name: {{.ContainerName}}
    environment:
      POSTGRES_USER: {{.User}}
      POSTGRES_PASSWORD: {{.Password}}
      POSTGRES_DB: {{.Database}}
    ports:
      - "{{.Port}}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      {{- if .HasExtensions}}
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
      {{- end}}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U {{.User}} -d {{.Database}}"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
`

const initSQLTemplate = `-- Initialize PostgreSQL extensions

{{- range .Extensions}}
CREATE EXTENSION IF NOT EXISTS "{{.SQLName}}";
{{- end}}
`

// GenerateDockerfile generates a Dockerfile from template
func GenerateDockerfile(data DockerfileData) (string, error) {
	tmpl := template.New("dockerfile")

	// Add custom function for arithmetic
	tmpl.Funcs(template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	})

	tmpl, err := tmpl.Parse(dockerfileTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Dockerfile template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute Dockerfile template: %w", err)
	}

	return buf.String(), nil
}

// GenerateDockerCompose generates a docker-compose.yml from template
func GenerateDockerCompose(data DockerComposeData) (string, error) {
	tmpl, err := template.New("docker-compose").Parse(dockerComposeTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse docker-compose template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute docker-compose template: %w", err)
	}

	return buf.String(), nil
}

// GenerateInitSQL generates an init.sql file from template
func GenerateInitSQL(data InitSQLData) (string, error) {
	tmpl, err := template.New("init-sql").Parse(initSQLTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse init SQL template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute init SQL template: %w", err)
	}

	return buf.String(), nil
}

// MapExtensionToSQLName converts extension names to their PostgreSQL CREATE EXTENSION names
func MapExtensionToSQLName(extName string) string {
	// Map special cases where the package name differs from the extension name
	switch extName {
	case "pgvector":
		return "vector"
	default:
		return extName
	}
}