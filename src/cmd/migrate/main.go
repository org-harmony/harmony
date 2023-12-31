package main

import (
	"context"
	"fmt"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
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

	v := validation.New()

	dbCfg := &persistence.Cfg{}
	util.Ok(config.C(dbCfg, config.From("persistence"), config.Validate(v)))
	db := util.Unwrap(persistence.NewDB(dbCfg.DB))
	defer db.Close()

	fmt.Println("migrating database...")

	err := persistence.Migrate(context.Background(), persistence.MigrateDirection(direction), dbCfg.DB.MigrationsDir, db)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("database migrated successfully")
	os.Exit(0)
}
