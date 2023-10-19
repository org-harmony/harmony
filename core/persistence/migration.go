package persistence

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrateDirection is the direction of the migration.
type MigrateDirection string

const (
	// MigrateUp is a migration that adds new things to the database.
	MigrateUp MigrateDirection = "up"
	// MigrateDown is a migration that removes things from the database.
	MigrateDown MigrateDirection = "down"
)

// Migration is a migration entity as stored in the database.
type Migration struct {
	Name       string
	Timestamp  time.Time
	ExecutedAt time.Time
}

// Migrate takes a direction and a directory of migrations and executes them in the given direction.
func Migrate(direction MigrateDirection, migrationsDir string, db *pgxpool.Pool, c context.Context) error {
	migDir, err := os.ReadDir(migrationsDir) // read all migrations from directory
	if err != nil {
		return err
	}

	migrations := make(map[string]string) // list of available migrations
	for _, f := range migDir {
		filename := f.Name()
		if !strings.HasSuffix(filename, string("_"+direction+".sql")) {
			continue
		}

		key := trimMigrationSuffix(filename)

		migrations[key] = filename
	}

	err = ensureMigrationsTablesPresence(db, c)
	if err != nil {
		return err
	}

	rows, err := db.Query(c, "SELECT * FROM database_migrations")
	if err != nil {
		return err
	}

	executedMigrations := make(map[string]Migration)
	for rows.Next() {
		var m Migration
		err = rows.Scan(&m.Name, &m.Timestamp, &m.ExecutedAt)
		if err != nil {
			return err
		}

		executedMigrations[m.Name] = m // add found migration to list of executed migrations
	}

	err = migrateMigrations(direction, migrations, executedMigrations, migrationsDir, db, c)
	if err != nil {
		return err
	}

	return nil
}

// migrateMigrations migrates a list of migrations up/down depending on the direction and the status of the migration.
func migrateMigrations(direction MigrateDirection, migrations map[string]string, executedMigrations map[string]Migration, migrationsDir string, db *pgxpool.Pool, c context.Context) error {
	for name, migration := range migrations {
		_, isMigrationExecuted := executedMigrations[name]
		if direction == MigrateUp && isMigrationExecuted {
			fmt.Printf("skipping migration %s on %s: already executed\n", name, MigrateUp)
			continue
		}

		if direction == MigrateDown && !isMigrationExecuted {
			fmt.Printf("skipping migration %s on %s: migration not present in databse\n", name, MigrateDown)
			continue
		}

		fmt.Printf("executing migration %s\n", name)
		err := migrate(name, direction, filepath.Join(migrationsDir, migration), db, c)
		if err != nil {
			return err
		}
	}

	return nil
}

// migrate executes a single migration.
func migrate(name string, direction MigrateDirection, migrationsPath string, db *pgxpool.Pool, c context.Context) error {
	f, err := os.ReadFile(migrationsPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(c, string(f))
	if err != nil {
		return err
	}

	if direction == MigrateUp {
		_, err = db.Exec(c, "INSERT INTO database_migrations (name, timestamp) VALUES ($1, $2)", name, time.Now())
		if err != nil {
			return err
		}

		return nil
	}

	if direction == MigrateDown {
		_, err = db.Exec(c, "DELETE FROM database_migrations WHERE name = $1", name)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("invalid migration direction '%s'", direction)
}

// ensureMigrationsTablesPresence ensures that the migrations table is present in the database.
func ensureMigrationsTablesPresence(db *pgxpool.Pool, c context.Context) error {
	dbMigrationsTable, err := db.Query(c, "SELECT * FROM information_schema.tables WHERE table_name = 'database_migrations'")
	if err != nil {
		return err
	}

	dbMigrationsTableExists := dbMigrationsTable.Next()
	if !dbMigrationsTableExists {
		_, err = db.Exec(c, "CREATE TABLE database_migrations (name VARCHAR(255) NOT NULL PRIMARY KEY, timestamp TIMESTAMP NOT NULL, executed_at TIMESTAMP NOT NULL DEFAULT NOW())")
		if err != nil {
			return err
		}
	}

	return nil
}

// trimMigrationSuffix trims the migration suffix from the migration name.
// Schema: <name>_<direction>.sql => <name>
func trimMigrationSuffix(name string) string {
	return name[:strings.LastIndexByte(name, '_')]
}
