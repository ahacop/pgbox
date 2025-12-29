package util

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDebArch(t *testing.T) {
	result := GetDebArch()

	// The result should be one of the valid architectures
	assert.Contains(t, []string{"amd64", "arm64"}, result)

	// Verify it matches the current runtime architecture
	switch runtime.GOARCH {
	case "amd64":
		assert.Equal(t, "amd64", result)
	case "arm64":
		assert.Equal(t, "arm64", result)
	default:
		// Fallback should be amd64
		assert.Equal(t, "amd64", result)
	}
}
