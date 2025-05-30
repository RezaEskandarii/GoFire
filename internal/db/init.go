package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"gofire/internal/constants"
	"gofire/internal/lock"
	"log"
	"os"
	"path/filepath"
)

const (
	baseDir = "./migrations"
	schema  = "gofire_schema"
)

// Init establishes a connection to a database and runs schema initialization and migration scripts.
// It ensures that only one instance of the application runs the migration logic at a time by using a distributed lock.
//
// The function performs the following steps:
//  1. Opens a database connection using the given URL.
//  2. Acquires a distributed lock to prevent concurrent migrations.
//  3. Pings the database to verify the connection.
//  4. Creates the required schema if it does not exist.
//  5. Reads and executes SQL scripts from the predefined baseDir.
//
// If any step fails, the function returns an error. The lock is released and the database connection is closed automatically.
func Init(postgresURL string, distributedLock lock.DistributedLockManager) error {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return err
	}

	migrationLock := constants.MigrationLock

	if err = distributedLock.Acquire(migrationLock); err != nil {
		return err
	}
	defer distributedLock.Release(migrationLock)

	if err = db.Ping(); err != nil {
		return err
	}

	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
	if err != nil {
		return err
	}

	scripts, err := readSQLScripts()
	if err != nil {
		return err
	}
	for _, script := range scripts {
		log.Println(script)
		if _, err := db.Exec(script); err != nil {
			return err
		}
	}

	return err
}

func readSQLScripts() ([]string, error) {

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var scripts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(baseDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		scripts = append(scripts, string(content))
	}

	return scripts, nil
}
