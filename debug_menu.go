package main

import (
	"fmt"
	"log"

	"github.com/ahacop/pgbox/pkg/extensions"
	"github.com/ahacop/pgbox/pkg/ui"
)

func main() {
	// Create a mock extension manager
	extMgr := &extensions.Manager{
		ScriptDir: ".",
		PgMajor:   "17",
	}

	fmt.Println("Testing menu interface...")
	selectedConfig, selectedExts, err := ui.RunMainInterface(extMgr)

	if err != nil {
		log.Fatalf("Interface error: %v", err)
	}

	if selectedConfig != nil {
		fmt.Printf("✅ Selected existing config: %s\n", selectedConfig.Name)
	} else if selectedExts != nil {
		fmt.Printf("✅ Created new config with extensions: %v\n", selectedExts)
	} else {
		fmt.Println("❌ No result - user cancelled or error occurred")
	}
}
