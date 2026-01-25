package di

import (
	"database/sql"
	"log"
)

func getPG(connection string) *sql.DB {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		log.Fatal(err)
	}
	return db
}
