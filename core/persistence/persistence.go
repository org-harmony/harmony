// Package persistence provides different persistence implementations.
// Part of this package is the database implementation and the session storage.
package persistence

// Cfg is the configuration for the persistence package.
type Cfg struct {
	DB *PostgresDBCfg `toml:"db"`
}
