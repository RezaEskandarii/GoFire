package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"gofire/internal/constants"
	"gofire/internal/lock"
	"os"
	"path/filepath"
)

const (
	baseDir = "./migrations"
	schema  = "gofire_schema"
)

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

	if scripts, err := readSQLScripts(); err == nil {
		for _, script := range scripts {
			fmt.Println(script)
			_, err = db.Exec(script)
			if err != nil {
				return err
			}
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
