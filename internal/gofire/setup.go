package gofire

import (
	"context"
	"database/sql"
	"github.com/redis/go-redis/v9"
	"log"
	"net/url"
	"strconv"
	"strings"
)

func setupPostgres(connection string) *sql.DB {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func setupRedis(ctx context.Context, connectionString string) *redis.Client {
	// Parse connection string like: redis://:password@host:port/db
	u, err := url.Parse(connectionString)
	if err != nil {
		log.Fatalf("invalid redis connection string: %v", err)
	}

	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}

	db := 0
	if u.Path != "" && u.Path != "/" {
		dbStr := strings.TrimPrefix(u.Path, "/")
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			log.Fatalf("invalid db number in connection string: %v", err)
		}
	}

	options := &redis.Options{
		Addr:     u.Host,
		Password: password,
		DB:       db,
	}

	rdb := redis.NewClient(options)

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	return rdb
}
