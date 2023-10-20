package persistence

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"sync"
)

var (
	// DBConfigError is returned when the database configuration is invalid.
	// This might be due to invalid PostgresDBCfg or invalid environment variables.
	DBConfigError = fmt.Errorf("error parsing database configuration")
	// DBConnError is the error returned when the database connection fails.
	DBConnError = fmt.Errorf("error connecting to database")
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

type PGRepositoryProvider struct {
	db           *pgxpool.Pool
	repositories map[string]Repository
	mu           sync.RWMutex
}

type Repository interface {
	RepositoryName() string
}

type RepositoryProvider interface {
	Repository(name string) (Repository, error)
	RegisterRepository(init func(db any) (Repository, error)) error
}

// NewDB creates a new database connection pool.
func NewDB(cfg *PostgresDBCfg) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", DBConfigError, err)
	}

	return newDBWithConfig(config)
}

func NewDBWithString(cfg string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", DBConfigError, err)
	}

	return newDBWithConfig(config)
}

func newDBWithConfig(config *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", DBConnError, err)
	}

	return pool, nil
}

// String returns the string representation of the database configuration.
func (cfg *PostgresDBCfg) String() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Pass, cfg.Name, cfg.SSLMode, cfg.MaxConns)
}

func (cfg *PostgresDBCfg) StringWoDbName() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s pool_max_conns=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Pass, cfg.SSLMode, cfg.MaxConns)
}

func NewPGRepositoryProvider(db *pgxpool.Pool) RepositoryProvider {
	return &PGRepositoryProvider{
		db:           db,
		repositories: make(map[string]Repository),
	}
}

func (rp *PGRepositoryProvider) Repository(name string) (Repository, error) {
	rp.mu.RLock()
	r, ok := rp.repositories[name]
	rp.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("repository %s not found", name)
	}

	return r, nil
}

func (rp *PGRepositoryProvider) RegisterRepository(init func(db any) (Repository, error)) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	repo, err := init(rp.db)
	if err != nil {
		return err
	}

	rp.repositories[repo.RepositoryName()] = repo

	return nil
}
