package persistence

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/util"
	"path/filepath"
	"strings"
	"time"
)

// ReadTestDBCfg reads the test database configuration from the config directory.
func ReadTestDBCfg(configDir string) *Cfg {
	v := validator.New(validator.WithRequiredStructEnabled())
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("persistence"), config.FromDir(configDir)))
	util.Ok(config.C(cfg, config.From("persistence.test"), config.FromDir(configDir), config.Validate(v)))

	return cfg
}

// TODO do test init before all tests and remove db after tests (see: task or mage (alts to make-file)) for once

// InitTestDB initializes a database for tests. It returns the database connection. The database connection should be closed after use.
// InitTestDB creates a new database with a name [PREFIX_FROM_DB_CFG]_[RANDOM_UUID] and runs all migrations on it.
// The database connection is configured to drop the database after closing the connection.
//
// Example:
//
//	func TestMain(m *testing.M) {
//		db = InitTestDB("./../../../")
//		result := m.Run()
//		db.Close()
//		os.Exit(result)
//	}
func InitTestDB(baseDir string) *pgxpool.Pool {
	dbCfg := ReadTestDBCfg(filepath.Join(baseDir, "config"))
	db := util.Unwrap(NewDBWithString(dbCfg.DB.StringWoDBName()))

	id := strings.Replace(uuid.NewString(), "-", "", -1)
	dbCfg.DB.Name = fmt.Sprintf("%s_%s", dbCfg.DB.Name, id)
	_ = util.Unwrap(db.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbCfg.DB.Name)))

	db.Close()

	pgxConfig := util.Unwrap(pgxpool.ParseConfig(dbCfg.DB.String()))
	pgxConfig.BeforeClose = func(conn *pgx.Conn) {
		util.Ok(conn.Close(context.Background()))

		db := util.Unwrap(NewDBWithString(dbCfg.DB.StringWoDBName()))

		_, err := db.Exec(context.Background(), fmt.Sprintf("DROP DATABASE %s", dbCfg.DB.Name))
		if err != nil {
			panic(fmt.Errorf("unable to drop test database after use: %w", err))
		}

		db.Close()
	}
	db = util.Unwrap(newDBWithConfig(pgxConfig))

	util.Ok(Migrate(context.Background(), MigrateUp, filepath.Join(baseDir, dbCfg.DB.MigrationsDir), db))

	return db
}

// TruncateSessionDates truncates the session's dates to the millisecond.
// This is necessary because the database truncates the dates to the millisecond. (PostgreSQL)
func TruncateSessionDates[P, M any](session *Session[P, M]) {
	session.CreatedAt = session.CreatedAt.Truncate(time.Millisecond)
	session.ExpiresAt = session.ExpiresAt.Truncate(time.Millisecond)
	if session.UpdatedAt != nil {
		*session.UpdatedAt = session.UpdatedAt.Truncate(time.Millisecond)
	}
}
