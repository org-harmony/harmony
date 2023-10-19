package persistence

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/util"
	"path/filepath"
)

func ReadTestDBCfg(configDir string) *Cfg {
	v := validator.New(validator.WithRequiredStructEnabled())
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("persistence"), config.FromDir(configDir)))
	util.Ok(config.C(cfg, config.From("persistence.test"), config.FromDir(configDir), config.Validate(v)))

	return cfg
}

func InitTestDB(baseDir string) *pgxpool.Pool {
	dbCfg := ReadTestDBCfg(filepath.Join(baseDir, "config"))
	db := util.Unwrap(NewDBWithString(dbCfg.DB.StringWoDbName()))

	_ = util.Unwrap(db.Exec(context.Background(), "DROP DATABASE IF EXISTS "+dbCfg.DB.Name))
	_ = util.Unwrap(db.Exec(context.Background(), "CREATE DATABASE "+dbCfg.DB.Name))

	db.Close()

	db = util.Unwrap(NewDB(dbCfg.DB))

	util.Ok(Migrate(MigrateUp, filepath.Join(baseDir, dbCfg.DB.MigrationsDir), db, context.Background()))

	return db
}
