package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/ahacop/pgbox/internal/extensions"
	"github.com/spf13/cobra"
)

func ListExtensionsCmd() *cobra.Command {
	var showSource bool
	var filterKind string

	listExtCmd := &cobra.Command{
		Use:   "list-extensions",
		Short: "List available PostgreSQL extensions",
		Long: `List all available PostgreSQL extensions from the catalog.

Extensions include both built-in PostgreSQL contrib modules and third-party
extensions installable from apt.postgresql.org.`,
		Example: `  # List all extensions
  pgbox list-extensions

  # Show source information for each extension
  pgbox list-extensions --source

  # Filter by kind (builtin or package)
  pgbox list-extensions --kind builtin
  pgbox list-extensions --kind package`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listExtensions(cmd.OutOrStdout(), showSource, filterKind)
		},
	}

	listExtCmd.Flags().BoolVarP(&showSource, "source", "s", false, "Show source information for each extension")
	listExtCmd.Flags().StringVarP(&filterKind, "kind", "k", "", "Filter by kind (builtin or package)")

	return listExtCmd
}

func listExtensions(w io.Writer, showSource bool, filterKind string) error {
	allExtensions := extensions.ListExtensions()

	var displayed []string
	for _, name := range allExtensions {
		ext, _ := extensions.Get(name)

		if filterKind != "" {
			isBuiltin := ext.Package == ""
			if filterKind == "builtin" && !isBuiltin {
				continue
			}
			if filterKind == "package" && isBuiltin {
				continue
			}
		}
		displayed = append(displayed, name)
	}

	_, _ = fmt.Fprintf(w, "PostgreSQL Extensions (%d available):\n\n", len(displayed))

	for _, name := range displayed {
		ext, _ := extensions.Get(name)
		if showSource {
			source := "builtin"
			if ext.Package != "" {
				source = fmt.Sprintf("apt (%s)", strings.ReplaceAll(ext.Package, "{v}", "<version>"))
			}
			_, _ = fmt.Fprintf(w, "%-30s %s\n", name, source)
		} else {
			_, _ = fmt.Fprintf(w, "%s\n", name)
		}
	}

	return nil
}
