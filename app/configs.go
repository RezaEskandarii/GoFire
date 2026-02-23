package app

import (
	"database/sql"
	"log"
)

func getPostgresDB(connection string) *sql.DB {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		log.Fatal(err)
	}
	return db
}
