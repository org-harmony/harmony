package persistence

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrDBConn is the error returned when the database connection fails.
	ErrDBConn = fmt.Errorf("error connecting to database")
)

// PostgresDBCfg is the configuration for the Postgres database.
type PostgresDBCfg struct {
	Host          string `toml:"host" env:"DB_HOST" validate:"required"`
	Port          string `toml:"port" env:"DB_PORT" validate:"required"`
	User          string `toml:"user" env:"DB_USER" validate:"required"`
	Pass          string `toml:"pass" env:"DB_PASS" validate:"required"`
	Name          string `toml:"name" env:"DB_NAME" validate:"required"`
	SSLMode       string `toml:"ssl_mode" env:"DB_SSL_MODE" validate:"required"`
	MaxConns      string `toml:"max_conns" env:"DB_MAX_CONNS" validate:"required"`
	MigrationsDir string `toml:"migrations_dir" env:"DB_MIGRATIONS_DIR"`
}

// NewDB creates a new database connection pool.
func NewDB(cfg *PostgresDBCfg) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDBConn, err)
	}

	return pool, nil
}

// String returns the string representation of the database configuration.
func (cfg *PostgresDBCfg) String() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Pass, cfg.Name, cfg.SSLMode, cfg.MaxConns)
}
