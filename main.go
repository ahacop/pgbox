package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ahacop/pgbox/cmd"
	"github.com/charmbracelet/fang"
)

var (
	// These are set at build time via -ldflags
	version = "dev"
	commit  = "unknown"
)

func main() {
	// Compose version string with all metadata
	ver := version
	if commit != "" && commit != "unknown" {
		if len(commit) > 7 {
			commit = commit[:7]
		}
		ver = fmt.Sprintf("%s (%s)", version, commit)
	}

	if err := fang.Execute(context.Background(), cmd.RootCmd(), fang.WithVersion(ver)); err != nil {
		os.Exit(1)
	}
}
