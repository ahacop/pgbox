package cmd

import (
	"fmt"
	"strings"
)

// ValidPostgresVersions contains the supported PostgreSQL versions.
var ValidPostgresVersions = []string{"16", "17", "18"}

// ValidatePostgresVersion checks if the given version is a supported PostgreSQL version.
func ValidatePostgresVersion(version string) error {
	for _, v := range ValidPostgresVersions {
		if version == v {
			return nil
		}
	}
	return fmt.Errorf("invalid PostgreSQL version: %s (must be 16, 17, or 18)", version)
}

// ParseExtensionList parses a comma-separated list of extensions and returns a slice.
// Returns nil if the input is empty.
func ParseExtensionList(extList string) []string {
	if extList == "" {
		return nil
	}
	parts := strings.Split(extList, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}
