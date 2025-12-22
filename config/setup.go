package config

import (
	"context"
	"database/sql"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

func setupPostgres(connection string) *sql.DB {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func setupRedis(address, password string, db int) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal(err.Error())
	}

	return rdb
}
