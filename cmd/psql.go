package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ahacop/pgbox/internal/docker"
	"github.com/spf13/cobra"
)

func PsqlCmd() *cobra.Command {
	var psqlDatabase string
	var psqlUser string
	var psqlName string

	psqlCmd := &cobra.Command{
		Use:   "psql [flags] [-- psql-args...]",
		Short: "Connect to PostgreSQL with psql",
		Long: `Connect to a running PostgreSQL container using psql client.

This command executes psql inside the container, so no local PostgreSQL client is needed.

You can pass additional arguments to psql after a '--' separator.`,
		Example: `  # Connect to default container with default database and user
  pgbox psql

  # Connect to a specific database
  pgbox psql --database mydb

  # Connect with a specific user
  pgbox psql --user myuser

  # Connect to a container with custom name
  pgbox psql -n my-postgres

  # Pass additional arguments to psql (e.g., execute a command)
  pgbox psql -- -c "SELECT version();"

  # Run psql with specific options
  pgbox psql -- -t -A -c "SELECT current_database();"

  # Execute a SQL file
  pgbox psql -- -f /path/to/file.sql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPsql(cmd, args, &psqlDatabase, &psqlUser, &psqlName)
		},
		DisableFlagParsing: false,
		Args:               cobra.ArbitraryArgs,
	}

	psqlCmd.Flags().StringVarP(&psqlDatabase, "database", "d", "postgres", "Database name to connect to")
	psqlCmd.Flags().StringVarP(&psqlUser, "user", "u", "postgres", "Username for connection")
	psqlCmd.Flags().StringVarP(&psqlName, "name", "n", "", "Container name (default: pgbox-pg17)")

	return psqlCmd
}

func runPsql(cmd *cobra.Command, args []string, psqlDatabase, psqlUser, psqlName *string) error {
	client := docker.NewClient()

	// Resolve container name (finds running container if not specified)
	resolvedName, err := ResolveRunningContainer(client, *psqlName)
	if err != nil {
		return err
	}
	*psqlName = resolvedName

	// If user and database weren't specified, try to get them from container env vars
	if !cmd.Flags().Changed("user") {
		if envUser, err := client.GetContainerEnv(*psqlName, "POSTGRES_USER"); err == nil && envUser != "" {
			*psqlUser = envUser
		}
	}
	if !cmd.Flags().Changed("database") {
		if envDB, err := client.GetContainerEnv(*psqlName, "POSTGRES_DB"); err == nil && envDB != "" {
			*psqlDatabase = envDB
		}
	}

	// Build the psql command arguments
	psqlArgs := []string{"psql", "-U", *psqlUser, "-d", *psqlDatabase}

	// Check if there are additional arguments after --
	dashPos := cmd.ArgsLenAtDash()
	if dashPos > -1 {
		// There's a -- separator, append everything after it
		psqlArgs = append(psqlArgs, args[dashPos:]...)
	}

	// Check if we're running an interactive session or a one-off command
	// First check if stdin is a terminal
	stdinIsTerminal := false
	if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		stdinIsTerminal = true
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
		fmt.Printf("Connecting to %s as user '%s' to database '%s'...\n", *psqlName, *psqlUser, *psqlDatabase)
		fmt.Println("Type \\q to exit")
		fmt.Println(strings.Repeat("-", 40))
	}

	// Build the full docker command
	dockerArgs := []string{"exec"}
	if isInteractive {
		// Use -it for fully interactive sessions
		dockerArgs = append(dockerArgs, "-it")
	} else if !stdinIsTerminal {
		// Use -i for piped input (stdin needs to be connected but not a tty)
		dockerArgs = append(dockerArgs, "-i")
	}
	dockerArgs = append(dockerArgs, *psqlName)
	dockerArgs = append(dockerArgs, psqlArgs...)

	// Execute psql inside the container
	return client.RunInteractive(dockerArgs...)
}
