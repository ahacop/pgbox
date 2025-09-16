package main

import (
	"context"
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
	// Use version directly as it already contains commit info from build
	ver := version

	if err := fang.Execute(context.Background(), cmd.RootCmd(), fang.WithVersion(ver)); err != nil {
		os.Exit(1)
	}
}
