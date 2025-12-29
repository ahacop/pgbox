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
)

func main() {
	if err := fang.Execute(context.Background(), cmd.RootCmd(), fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}
