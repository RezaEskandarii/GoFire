package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"gofire/internal/constants"
	"gofire/internal/lock"
	"log"
)

const (
	schema = "gofire_schema"
)

// Init establishes a connection to a database and runs schema initialization and migration scripts.
// It ensures that only one instance of the application runs the migration logic at a time by using a distributed lock.
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
	defer func() {
		if err := distributedLock.Release(migrationLock); err != nil {
			log.Printf("Error releasing lock: %v", err)
		}
	}()

	if err = db.Ping(); err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing db: %v", err)
		}
	}()

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

	entries, err := MigrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var scripts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := MigrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, string(content))
	}

	return scripts, nil
}
