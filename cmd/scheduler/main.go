package main

import (
	"context"
	"database/sql"
	"fmt"
	"gofire/internal/app"
	"gofire/internal/db"
	"time"
)

func main() {

	postgresURL := "host=localhost user=postgres password=boofhichkas dbname=esim port=5432 sslmode=disable"

	db.Init(postgresURL)
	psdb, _ := sql.Open("postgres", postgresURL)
	r := db.NewJobRepository(psdb)

	let := app.NewScheduler(r, "n")

	ctx := context.Background()

	jobID, err := let.Enqueue(ctx, "send_sms", time.Now(), "09374052874", "your ticket is ready")

	if err == nil {
		fmt.Println(jobID)
	}
	select {}
}
