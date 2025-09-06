package main

import (
	"context"
	"os"

	"github.com/ahacop/pgbox/cmd"
	"github.com/charmbracelet/fang"
)

func main() {
	if err := fang.Execute(context.Background(), cmd.NewRootCmd()); err != nil {
		os.Exit(1)
	}
}
