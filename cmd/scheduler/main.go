package main

import (
	"context"
	"fmt"
	"gofire/internal/gofire"
	"gofire/internal/models/config"
	"log"
	"strings"
)

func main() {

	const postgresURL = "host=localhost port=5432 user=postgres password=boofhichkas dbname=esim sslmode=disable"
	cfg := config.NewGofireConfig("west-canada").
		WithInterval(int(700000)).
		WithDashboardPort(8080).
		WithWorkerCount(15).
		WithBatchSize(500).
		WithPostgresConfig(config.PostgresConfig{ConnectionUrl: postgresURL}).
		WithAdminDashboardConfig("reza", "1234", "my-secret-key-1234-5")

	for i := 0; i < 20; i++ {
		jobName := fmt.Sprintf("job-%d", i)
		cfg.RegisterHandler(config.MethodHandler{
			MethodName: jobName,
			Func: func(args ...any) error {
				to := args[0].(string)
				message := args[1].(string)
				return sendSms(to, message)
			},
		})
	}
	cfg.RegisterHandler(config.MethodHandler{
		MethodName: "send_sms",
		Func: func(args ...any) error {
			to := args[0].(string)
			message := args[1].(string)
			return sendSms(to, message)
		},
	})

	jobManager, err := gofire.SetUp(context.Background(), *cfg)
	if err != nil {
		log.Println(err.Error())
	}

	jobManager.Schedule(context.Background(), "GenerateDailySalesReportExcel", "0 0 * * *")
	jobManager.Schedule(context.Background(), "MonthlyProfitReportJob", "* * 1 * *", "2025-05")
	jobManager.Schedule(context.Background(), "MonthlyProfitReportJob", "0 0 1 * *", "2025-05")

	jobManager.Schedule(context.Background(), "DailySalesReportJob", "0 0 * * *")

	jobManager.Schedule(context.Background(), "WeeklyUserCleanupJob", "0 3 * * 0", 30)

	jobManager.Schedule(context.Background(), "HourlyCacheRefreshJob", "0 * * * *", "us-east-1")

	jobManager.Schedule(context.Background(), "QuarterlyTaxReportJob", "0 0 1 */3 *", "Q2")

	jobManager.Schedule(context.Background(), "DailyEmailNotificationJob", "30 8 * * *", "welcome")

	jobManager.Schedule(context.Background(), "NightlyBackupJob", "0 2 * * *", "full")

	jobManager.Schedule(context.Background(), "MonthlyInvoiceGenerationJob", "0 4 1 * *", "2025-05")

	jobManager.Schedule(context.Background(), "WeeklyDataSyncJob", "15 1 * * 1", "CRM", true)

	jobManager.Schedule(context.Background(), "DailyReportArchivalJob", "45 23 * * *", "/archive/", 1024)
	jobManager.Schedule(context.Background(), "OverdueUsersReportJob", "45 23 * * *", "3b8656f7-9072-4a5b-a3e1-36db42692c13", 5547, false)

	select {}
}

func sendSms(to string, message string) error {
	log.Println("===================================")
	fmt.Printf("Sending SMS to %s:\n%s\n", to, message)
	if strings.HasPrefix(to, "phone-1") {
		//return errors.New("has -1")
	}
	return nil
}
