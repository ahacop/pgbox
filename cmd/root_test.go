package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	// TODO: Add more comprehensive tests as commands are added

	t.Run("Execute without error", func(t *testing.T) {
		err := Execute()
		assert.NoError(t, err)
	})

	t.Run("Shows help by default", func(t *testing.T) {
		// TODO: Capture output and verify help is shown
		rootCmd := RootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)

		err := rootCmd.Execute()
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "pgbox")
		assert.Contains(t, output, "PostgreSQL")
	})
}

// TODO: Add test helpers below
