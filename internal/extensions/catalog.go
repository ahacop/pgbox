// Package extensions provides the extension catalog for pgbox.
package extensions

import (
	"fmt"
	"sort"
	"strings"
)

// Extension represents a PostgreSQL extension configuration.
type Extension struct {
	// Package is the apt package pattern (e.g., "postgresql-{v}-pgvector").
	// Empty for built-in contrib extensions.
	Package string

	// DebURL is a URL template for downloading a .deb package directly.
	// Supports placeholders: {v} (PG version), {arch} (amd64/arm64).
	// If set, this is used instead of Package for installation.
	DebURL string

	// ZipURL is a URL template for downloading a .zip file containing a .deb package.
	// Supports placeholders: {v} (PG version), {arch} (amd64/arm64).
	// The zip is extracted and the .deb inside is installed.
	ZipURL string

	// BaseImage overrides the default postgres:{v} image.
	// Use this when a .deb requires a specific distro (e.g., "postgres:{v}-bookworm").
	BaseImage string

	// SQLName is the CREATE EXTENSION name if different from the catalog key.
	SQLName string

	// Preload lists shared_preload_libraries entries needed.
	Preload []string

	// GUCs contains PostgreSQL configuration parameters.
	GUCs map[string]string

	// InitSQL is custom initialization SQL. Empty means default CREATE EXTENSION.
	InitSQL string
}

// Catalog maps extension name to its configuration.
// The key is the name users specify (e.g., "pgvector", "pg_cron").
var Catalog = map[string]Extension{
	// ===== Built-in PostgreSQL contrib extensions (no apt package needed) =====
	"adminpack":          {},
	"amcheck":            {},
	"autoinc":            {},
	"bloom":              {},
	"btree_gin":          {},
	"btree_gist":         {},
	"citext":             {},
	"cube":               {},
	"dblink":             {},
	"dict_int":           {},
	"dict_xsyn":          {},
	"earthdistance":      {},
	"file_fdw":           {},
	"fuzzystrmatch":      {},
	"hstore":             {},
	"insert_username":    {},
	"intagg":             {},
	"intarray":           {},
	"isn":                {},
	"lo":                 {},
	"ltree":              {},
	"moddatetime":        {},
	"old_snapshot":       {},
	"pageinspect":        {},
	"pg_buffercache":     {},
	"pg_freespacemap":    {},
	"pg_prewarm":         {},
	"pg_stat_statements": {},
	"pg_surgery":         {},
	"pg_trgm":            {},
	"pg_visibility":      {},
	"pg_walinspect":      {},
	"pgcrypto":           {},
	"pgrowlocks":         {},
	"pgstattuple":        {},
	"plpgsql":            {},
	"postgres_fdw":       {},
	"refint":             {},
	"seg":                {},
	"sslinfo":            {},
	"tablefunc":          {},
	"tcn":                {},
	"tsm_system_rows":    {},
	"tsm_system_time":    {},
	"unaccent":           {},
	"uuid-ossp":          {},
	"xml2":               {},

	// ===== Third-party extensions (simple - just apt package) =====
	"age":                    {Package: "postgresql-{v}-age"},
	"asn1oid":                {Package: "postgresql-{v}-asn1oid"},
	"auto-failover":          {Package: "postgresql-{v}-auto-failover"},
	"bgw-replstatus":         {Package: "postgresql-{v}-bgw-replstatus"},
	"credcheck":              {Package: "postgresql-{v}-credcheck"},
	"debversion":             {Package: "postgresql-{v}-debversion"},
	"decoderbufs":            {Package: "postgresql-{v}-decoderbufs"},
	"dirtyread":              {Package: "postgresql-{v}-dirtyread"},
	"extra-window-functions": {Package: "postgresql-{v}-extra-window-functions"},
	"first-last-agg":         {Package: "postgresql-{v}-first-last-agg"},
	"h3":                     {Package: "postgresql-{v}-h3"},
	"hll":                    {Package: "postgresql-{v}-hll"},
	"http":                   {Package: "postgresql-{v}-http"},
	"hypopg":                 {Package: "postgresql-{v}-hypopg"},
	"icu-ext":                {Package: "postgresql-{v}-icu-ext"},
	"ip4r":                   {Package: "postgresql-{v}-ip4r"},
	"jsquery":                {Package: "postgresql-{v}-jsquery"},
	"londiste-sql":           {Package: "postgresql-{v}-londiste-sql"},
	"mimeo":                  {Package: "postgresql-{v}-mimeo"},
	"mobilitydb":             {Package: "postgresql-{v}-mobilitydb"},
	"mysql-fdw":              {Package: "postgresql-{v}-mysql-fdw"},
	"numeral":                {Package: "postgresql-{v}-numeral"},
	"ogr-fdw":                {Package: "postgresql-{v}-ogr-fdw"},
	"omnidb":                 {Package: "postgresql-{v}-omnidb"},
	"oracle-fdw":             {Package: "postgresql-{v}-oracle-fdw"},
	"orafce":                 {Package: "postgresql-{v}-orafce"},
	"partman":                {Package: "postgresql-{v}-partman"},
	"periods":                {Package: "postgresql-{v}-periods"},
	"pg-catcheck":            {Package: "postgresql-{v}-pg-catcheck"},
	"pg-checksums":           {Package: "postgresql-{v}-pg-checksums"},
	"pg-crash":               {Package: "postgresql-{v}-pg-crash"},
	"pg-fact-loader":         {Package: "postgresql-{v}-pg-fact-loader"},
	"pg-failover-slots":      {Package: "postgresql-{v}-pg-failover-slots"},
	"pg-gvm":                 {Package: "postgresql-{v}-pg-gvm"},
	"pg-hint-plan":           {Package: "postgresql-{v}-pg-hint-plan"},
	"pg-permissions":         {Package: "postgresql-{v}-pg-permissions"},
	"pg-qualstats":           {Package: "postgresql-{v}-pg-qualstats"},
	"pg-rewrite":             {Package: "postgresql-{v}-pg-rewrite"},
	"pg-rrule":               {Package: "postgresql-{v}-pg-rrule"},
	"pg-stat-kcache":         {Package: "postgresql-{v}-pg-stat-kcache"},
	"pg-track-settings":      {Package: "postgresql-{v}-pg-track-settings"},
	"pg-wait-sampling":       {Package: "postgresql-{v}-pg-wait-sampling"},
	"pgaudit":                {Package: "postgresql-{v}-pgaudit"},
	"pgauditlogtofile":       {Package: "postgresql-{v}-pgauditlogtofile"},
	"pgextwlist":             {Package: "postgresql-{v}-pgextwlist"},
	"pgfaceting":             {Package: "postgresql-{v}-pgfaceting"},
	"pgfincore":              {Package: "postgresql-{v}-pgfincore"},
	"pgl-ddl-deploy":         {Package: "postgresql-{v}-pgl-ddl-deploy"},
	"pglogical":              {Package: "postgresql-{v}-pglogical"},
	"pglogical-ticker":       {Package: "postgresql-{v}-pglogical-ticker"},
	"pgmemcache":             {Package: "postgresql-{v}-pgmemcache"},
	"pgmp":                   {Package: "postgresql-{v}-pgmp"},
	"pgnodemx":               {Package: "postgresql-{v}-pgnodemx"},
	"pgpcre":                 {Package: "postgresql-{v}-pgpcre"},
	"pgpool2":                {Package: "postgresql-{v}-pgpool2"},
	"pgq-node":               {Package: "postgresql-{v}-pgq-node"},
	"pgq3":                   {Package: "postgresql-{v}-pgq3"},
	"pgrouting":              {Package: "postgresql-{v}-pgrouting"},
	"pgrouting-doc":          {Package: "postgresql-{v}-pgrouting-doc"},
	"pgrouting-scripts":      {Package: "postgresql-{v}-pgrouting-scripts"},
	"pgsentinel":             {Package: "postgresql-{v}-pgsentinel"},
	"pgsphere":               {Package: "postgresql-{v}-pgsphere"},
	"pgtap":                  {Package: "postgresql-{v}-pgtap"},
	"pgtt":                   {Package: "postgresql-{v}-pgtt"},
	"pldebugger":             {Package: "postgresql-{v}-pldebugger"},
	"pljava":                 {Package: "postgresql-{v}-pljava"},
	"pljs":                   {Package: "postgresql-{v}-pljs"},
	"pllua":                  {Package: "postgresql-{v}-pllua"},
	"plpgsql-check":          {Package: "postgresql-{v}-plpgsql-check"},
	"plprofiler":             {Package: "postgresql-{v}-plprofiler"},
	"plproxy":                {Package: "postgresql-{v}-plproxy"},
	"plr":                    {Package: "postgresql-{v}-plr"},
	"plsh":                   {Package: "postgresql-{v}-plsh"},
	"pointcloud":             {Package: "postgresql-{v}-pointcloud"},
	"postgis-3": {
		Package: "postgresql-{v}-postgis-3",
		SQLName: "postgis",
		InitSQL: "-- Core PostGIS extension\n" +
			"CREATE EXTENSION IF NOT EXISTS postgis;\n\n" +
			"-- Grant usage on spatial_ref_sys to public\n" +
			"GRANT SELECT ON spatial_ref_sys TO PUBLIC;",
	},
	"postgis-3-scripts": {Package: "postgresql-{v}-postgis-3-scripts"},
	"powa":              {Package: "postgresql-{v}-powa"},
	"prefix":            {Package: "postgresql-{v}-prefix"},
	"preprepare":        {Package: "postgresql-{v}-preprepare"},
	"prioritize":        {Package: "postgresql-{v}-prioritize"},
	"q3c":               {Package: "postgresql-{v}-q3c"},
	"rational":          {Package: "postgresql-{v}-rational"},
	"rdkit":             {Package: "postgresql-{v}-rdkit"},
	"repack":            {Package: "postgresql-{v}-repack"},
	"repmgr":            {Package: "postgresql-{v}-repmgr"},
	"roaringbitmap":     {Package: "postgresql-{v}-roaringbitmap"},
	"rum":               {Package: "postgresql-{v}-rum"},
	"semver":            {Package: "postgresql-{v}-semver"},
	"set-user":          {Package: "postgresql-{v}-set-user"},
	"show-plans":        {Package: "postgresql-{v}-show-plans"},
	"similarity":        {Package: "postgresql-{v}-similarity"},
	"slony1-2":          {Package: "postgresql-{v}-slony1-2"},
	"snakeoil":          {Package: "postgresql-{v}-snakeoil"},
	"squeeze":           {Package: "postgresql-{v}-squeeze"},
	"statviz":           {Package: "postgresql-{v}-statviz"},
	"tablelog":          {Package: "postgresql-{v}-tablelog"},
	"tdigest":           {Package: "postgresql-{v}-tdigest"},
	"tds-fdw":           {Package: "postgresql-{v}-tds-fdw"},
	"timescaledb":       {Package: "postgresql-{v}-timescaledb"},
	"toastinfo":         {Package: "postgresql-{v}-toastinfo"},
	"unit":              {Package: "postgresql-{v}-unit"},

	// Extensions with different SQL names
	"pgvector": {Package: "postgresql-{v}-pgvector", SQLName: "vector"},

	// ===== Complex extensions (need shared_preload_libraries and/or GUCs) =====
	"pg_cron": {
		Package: "postgresql-{v}-cron",
		Preload: []string{"pg_cron"},
		GUCs: map[string]string{
			"cron.database_name":    "postgres",
			"cron.max_running_jobs": "5",
		},
		InitSQL: "CREATE EXTENSION IF NOT EXISTS pg_cron;\nGRANT USAGE ON SCHEMA cron TO postgres;",
	},
	"wal2json": {
		Package: "postgresql-{v}-wal2json",
		Preload: []string{"wal2json"},
		GUCs: map[string]string{
			"wal_level":             "logical",
			"max_replication_slots": "10",
			"max_wal_senders":       "10",
		},
		InitSQL: "-- wal2json logical decoding plugin is now available\n" +
			"-- To use it, create a replication slot with:\n" +
			"-- SELECT pg_create_logical_replication_slot('slot_name', 'wal2json');",
	},

	// ===== Extensions installed from .deb URLs (GitHub releases, etc.) =====
	"pg_search": {
		DebURL:    "https://github.com/paradedb/paradedb/releases/download/v0.20.5/postgresql-{v}-pg-search_0.20.5-1PARADEDB-bookworm_{arch}.deb",
		BaseImage: "postgres:{v}-bookworm",
		SQLName:   "pg_search",
		InitSQL:   "CREATE EXTENSION IF NOT EXISTS pg_search;",
	},

	// ===== Extensions installed from .zip files containing .deb packages =====
	// pg_textsearch: BM25 ranked text search (supports PostgreSQL 17 and 18 only)
	"pg_textsearch": {
		ZipURL:    "https://github.com/timescale/pg_textsearch/releases/download/v0.1.0/pg-textsearch-v0.1.0-pg{v}-{arch}.zip",
		BaseImage: "postgres:{v}-bookworm",
	},
}

// Get returns the extension configuration for the given name.
// Returns false if the extension is not found.
func Get(name string) (Extension, bool) {
	ext, ok := Catalog[name]
	return ext, ok
}

// GetPackage returns the apt package name for an extension and PostgreSQL version.
// Returns empty string if no package is needed (built-in extension).
func GetPackage(name, version string) string {
	ext, ok := Catalog[name]
	if !ok {
		return ""
	}
	return strings.ReplaceAll(ext.Package, "{v}", version)
}

// GetSQLName returns the SQL extension name for CREATE EXTENSION.
// Uses SQLName if set, otherwise uses the catalog key.
func GetSQLName(name string) string {
	ext, ok := Catalog[name]
	if !ok {
		return name
	}
	if ext.SQLName != "" {
		return ext.SQLName
	}
	return name
}

// GetInitSQL returns the initialization SQL for an extension.
// Returns default CREATE EXTENSION statement if no custom SQL is defined.
func GetInitSQL(name string) string {
	ext, ok := Catalog[name]
	if !ok {
		return ""
	}
	if ext.InitSQL != "" {
		return ext.InitSQL
	}
	sqlName := GetSQLName(name)
	return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", sqlName)
}

// ValidateExtensions checks that all extension names exist in the catalog.
func ValidateExtensions(names []string) error {
	var unknown []string
	for _, name := range names {
		if _, ok := Catalog[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown extensions: %s", strings.Join(unknown, ", "))
	}
	return nil
}

// ListExtensions returns all extension names sorted alphabetically.
func ListExtensions() []string {
	names := make([]string, 0, len(Catalog))
	for name := range Catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetExtensions returns the Extension configs for the given names.
func GetExtensions(names []string) []Extension {
	result := make([]Extension, 0, len(names))
	for _, name := range names {
		if ext, ok := Catalog[name]; ok {
			result = append(result, ext)
		}
	}
	return result
}

// NeedsPackages returns true if any of the given extensions require apt packages.
func NeedsPackages(names []string) bool {
	for _, name := range names {
		if ext, ok := Catalog[name]; ok && ext.Package != "" {
			return true
		}
	}
	return false
}

// GetPackages returns all apt packages needed for the given extensions and version.
func GetPackages(names []string, version string) []string {
	var packages []string
	seen := make(map[string]bool)
	for _, name := range names {
		pkg := GetPackage(name, version)
		if pkg != "" && !seen[pkg] {
			packages = append(packages, pkg)
			seen[pkg] = true
		}
	}
	return packages
}

// GetPreloadLibraries returns all shared_preload_libraries needed.
func GetPreloadLibraries(names []string) []string {
	var libs []string
	seen := make(map[string]bool)
	for _, name := range names {
		if ext, ok := Catalog[name]; ok {
			for _, lib := range ext.Preload {
				if !seen[lib] {
					libs = append(libs, lib)
					seen[lib] = true
				}
			}
		}
	}
	return libs
}

// GetGUCs returns all GUC settings needed, detecting conflicts.
func GetGUCs(names []string) (map[string]string, error) {
	gucs := make(map[string]string)
	sources := make(map[string]string) // Track which extension set each GUC

	for _, name := range names {
		if ext, ok := Catalog[name]; ok {
			for k, v := range ext.GUCs {
				if existing, hasKey := gucs[k]; hasKey && existing != v {
					return nil, fmt.Errorf("GUC conflict for '%s': %s sets '%s', %s sets '%s'",
						k, sources[k], existing, name, v)
				}
				gucs[k] = v
				sources[k] = name
			}
		}
	}
	return gucs, nil
}

// GetDebURL returns the resolved .deb URL for an extension.
// Returns empty string if the extension doesn't use .deb installation.
func GetDebURL(name, version, arch string) string {
	ext, ok := Catalog[name]
	if !ok || ext.DebURL == "" {
		return ""
	}
	url := strings.ReplaceAll(ext.DebURL, "{v}", version)
	url = strings.ReplaceAll(url, "{arch}", arch)
	return url
}

// GetDebURLs returns all .deb URLs needed for the given extensions.
func GetDebURLs(names []string, version, arch string) []string {
	var urls []string
	seen := make(map[string]bool)
	for _, name := range names {
		url := GetDebURL(name, version, arch)
		if url != "" && !seen[url] {
			urls = append(urls, url)
			seen[url] = true
		}
	}
	return urls
}

// NeedsDebPackages returns true if any of the given extensions require .deb downloads.
func NeedsDebPackages(names []string) bool {
	for _, name := range names {
		if ext, ok := Catalog[name]; ok && ext.DebURL != "" {
			return true
		}
	}
	return false
}

// HasDebURL returns true if the extension uses .deb installation.
func HasDebURL(name string) bool {
	ext, ok := Catalog[name]
	return ok && ext.DebURL != ""
}

// GetZipURL returns the resolved .zip URL for an extension.
// Returns empty string if the extension doesn't use .zip installation.
func GetZipURL(name, version, arch string) string {
	ext, ok := Catalog[name]
	if !ok || ext.ZipURL == "" {
		return ""
	}
	url := strings.ReplaceAll(ext.ZipURL, "{v}", version)
	url = strings.ReplaceAll(url, "{arch}", arch)
	return url
}

// GetZipURLs returns all .zip URLs needed for the given extensions.
func GetZipURLs(names []string, version, arch string) []string {
	var urls []string
	seen := make(map[string]bool)
	for _, name := range names {
		url := GetZipURL(name, version, arch)
		if url != "" && !seen[url] {
			urls = append(urls, url)
			seen[url] = true
		}
	}
	return urls
}

// NeedsZipPackages returns true if any of the given extensions require .zip downloads.
func NeedsZipPackages(names []string) bool {
	for _, name := range names {
		if ext, ok := Catalog[name]; ok && ext.ZipURL != "" {
			return true
		}
	}
	return false
}

// HasZipURL returns true if the extension uses .zip installation.
func HasZipURL(name string) bool {
	ext, ok := Catalog[name]
	return ok && ext.ZipURL != ""
}

// GetBaseImage returns the required base image for extensions.
// If any extension requires a specific base image, that takes precedence.
// Returns empty string if default postgres:{version} should be used.
func GetBaseImage(names []string, version string) string {
	for _, name := range names {
		if ext, ok := Catalog[name]; ok && ext.BaseImage != "" {
			return strings.ReplaceAll(ext.BaseImage, "{v}", version)
		}
	}
	return ""
}
