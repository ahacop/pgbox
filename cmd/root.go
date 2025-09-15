package cmd

import (
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pgbox",
		Short: "PostgreSQL-in-Docker with selectable extensions",
		Long: `pgbox is a CLI tool that simplifies running PostgreSQL in Docker
with your choice of extensions.

It provides an easy way to spin up PostgreSQL instances with
specific extensions for development and testing purposes.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	rootCmd.AddCommand(UpCmd())
	rootCmd.AddCommand(DownCmd())
	rootCmd.AddCommand(RestartCmd())
	rootCmd.AddCommand(StatusCmd())
	rootCmd.AddCommand(LogsCmd())
	rootCmd.AddCommand(PsqlCmd())
	rootCmd.AddCommand(ExportCmd())

	return rootCmd
}

func Execute() error {
	return RootCmd().Execute()
}
