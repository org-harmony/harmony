package persistence

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/util"
	"path/filepath"
	"strings"
)

func ReadTestDBCfg(configDir string) *Cfg {
	v := validator.New(validator.WithRequiredStructEnabled())
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("persistence"), config.FromDir(configDir)))
	util.Ok(config.C(cfg, config.From("persistence.test"), config.FromDir(configDir), config.Validate(v)))

	return cfg
}

// TODO do test init before all tests and remove db after tests (see: task or mage (alts to make-file)) for once

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

	util.Ok(Migrate(MigrateUp, filepath.Join(baseDir, dbCfg.DB.MigrationsDir), db, context.Background()))

	return db
}
