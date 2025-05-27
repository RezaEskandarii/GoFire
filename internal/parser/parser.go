package parser

import (
	"github.com/robfig/cron/v3"
	"log"
	"time"
)

func CalculateNextRun(expr string, from time.Time) time.Time {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expr)
	if err != nil {
		log.Printf("cronJobManager: invalid cron expression '%s': %v", expr, err)
		return from.Add(1 * time.Hour)
	}
	return schedule.Next(from)
}
