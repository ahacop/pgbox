package extensions

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"strconv"
)

// ExtensionData represents a PostgreSQL extension with all its metadata from the JSONL file
type ExtensionData struct {
	PgMajor     int    `json:"pg_major"`
	Name        string `json:"name"`     // Package/extension name
	Type        string `json:"type"`     // "builtin" or "package"
	Package     string `json:"package"`  // Full package name (e.g., postgresql-17-pgvector)
	SQLName     string `json:"sql_name"` // Name used in CREATE EXTENSION
	Description string `json:"description"`
}

//go:embed extensions.jsonl
var extensionsJSONL []byte

// extensionMap holds all extensions indexed by PG version and name
var extensionMap map[int]map[string][]ExtensionData

func init() {
	extensionMap = make(map[int]map[string][]ExtensionData)

	// Parse the JSONL file
	scanner := bufio.NewScanner(bytes.NewReader(extensionsJSONL))
	for scanner.Scan() {
		var ext ExtensionData
		if err := json.Unmarshal(scanner.Bytes(), &ext); err != nil {
			continue // Skip invalid lines
		}

		// Initialize maps if needed
		if extensionMap[ext.PgMajor] == nil {
			extensionMap[ext.PgMajor] = make(map[string][]ExtensionData)
		}

		// Add to map indexed by name
		extensionMap[ext.PgMajor][ext.Name] = append(extensionMap[ext.PgMajor][ext.Name], ext)
	}
}

// GetSQLName returns the SQL name for CREATE EXTENSION for a given package/extension name
func GetSQLName(name string, pgVersion string) string {
	pgMajor, _ := strconv.Atoi(pgVersion)
	if pgMajor == 0 {
		pgMajor = 17 // Default
	}

	if exts, ok := extensionMap[pgMajor][name]; ok && len(exts) > 0 {
		return exts[0].SQLName
	}
	// Fallback to the name itself
	return name
}
