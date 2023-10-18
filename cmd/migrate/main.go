package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/ctx"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/util"
	"os"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("please specify a migration direction: 'migrate <up|down>'")
		os.Exit(1)
	}

	direction := args[1]
	if direction != string(persistence.MigrateUp) && direction != string(persistence.MigrateDown) {
		fmt.Printf("invalid direction, expected '%s' or '%s'", persistence.MigrateUp, persistence.MigrateDown)
		os.Exit(1)
	}

	l := trace.NewLogger()
	v := validator.New(validator.WithRequiredStructEnabled())

	dbCfg := &persistence.Cfg{}
	util.Ok(config.C(dbCfg, config.From("persistence"), config.Validate(v)))
	db := util.Unwrap(persistence.NewDB(dbCfg.DB))
	defer db.Close()

	appCtx := ctx.NewAppContext(l, v, db)

	fmt.Println("migrating database...")

	err := persistence.Migrate(persistence.MigrateDirection(direction), dbCfg.DB.MigrationsDir, appCtx, context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("database migrated successfully")
	os.Exit(0)
}
