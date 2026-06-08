// Package importxui imports a 3x-ui SQLite database into the active s-ui
// database.
//
// The importer reads the source database through a read-only SQLite handle,
// maps supported 3x-ui inbounds, WireGuard endpoints, Reality TLS settings
// and clients into s-ui models, and applies all destination mutations in one
// transaction. Dry-run mode executes the same mapping path and rolls the
// transaction back after building the report.
package importxui
