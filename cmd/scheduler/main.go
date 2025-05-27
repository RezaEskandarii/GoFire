package main

import (
	"context"
	"fmt"
	"gofire/internal/gofire"
	"gofire/internal/models/config"
)

func main() {
	const postgresURL = "host=localhost port=5432 user=postgres password=boofhichkas dbname=esim sslmode=disable"
	cfg := config.NewGofireConfig("west-canada").
		WithInterval(10).
		WithDashboardPort(8080).
		WithWorkerCount(15).
		WithBatchSize(500).
		WithPostgresConfig(config.PostgresConfig{ConnectionUrl: postgresURL})

	cfg.RegisterHandler(config.MethodHandler{
		MethodName: "send_sms",
		Func: func(args []interface{}) error {
			to := args[0].(string)
			return sendSms(to, args[1].(string))
		},
	})

	gofire.Run(context.Background(), *cfg)
}

func sendSms(to string, message string) error {
	fmt.Printf("Sending SMS to %s:\n%s\n", to, message)
	return nil
}
