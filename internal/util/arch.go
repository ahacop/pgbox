// Package util provides shared utility functions for pgbox.
package util

import "runtime"

// GetDebArch returns the Debian architecture string for the current system.
// This is used when fetching .deb packages from apt repositories.
func GetDebArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return "amd64" // fallback
	}
}
