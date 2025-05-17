package main

import (
	"gofire/internal/db"
)

func main() {

	dsn := "host=localhost user=postgres password=boofhichkas dbname=esim port=5432 sslmode=disable"

	db.Init(dsn)
}
