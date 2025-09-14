package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates and returns the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pgbox",
		Short: "PostgreSQL-in-Docker with selectable extensions",
		Long: `pgbox is a CLI tool that simplifies running PostgreSQL in Docker
with your choice of extensions.

It provides an easy way to spin up PostgreSQL instances with
specific extensions for development and testing purposes.`,
		// TODO: Add actual functionality here
		Run: func(cmd *cobra.Command, args []string) {
			// For now, just show help
			cmd.Help()
		},
	}

	// TODO: Add global flags here
	// Example:
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pgbox.yaml)")

	// TODO: Add configuration initialization here

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is kept for backward compatibility if needed
func Execute() error {
	return NewRootCmd().Execute()
}

// TODO: Add helper functions for common operations below
