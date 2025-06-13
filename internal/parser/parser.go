package parser

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type CronExpr struct {
	Minute     map[int]bool
	Hour       map[int]bool
	DayOfMonth map[int]bool
	Month      map[int]bool
	DayOfWeek  map[int]bool
}

func parsePart(part string, min, max int) (map[int]bool, error) {
	values := make(map[int]bool)

	if part == "*" {
		for i := min; i <= max; i++ {
			values[i] = true
		}
		return values, nil
	}

	for _, token := range strings.Split(part, ",") {
		if strings.Contains(token, "-") {
			parts := strings.Split(token, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", token)
			}
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || start > end || start < min || end > max {
				return nil, fmt.Errorf("invalid range: %s", token)
			}
			for i := start; i <= end; i++ {
				values[i] = true
			}
		} else {
			num, err := strconv.Atoi(token)
			if err != nil || num < min || num > max {
				return nil, fmt.Errorf("invalid value: %s", token)
			}
			values[num] = true
		}
	}
	return values, nil
}

func ParseCron(expr string) (*CronExpr, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid cron expression: %s", expr)
	}

	minute, err := parsePart(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute error: %v", err)
	}
	hour, err := parsePart(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour error: %v", err)
	}
	dayOfMonth, err := parsePart(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("dayOfMonth error: %v", err)
	}
	month, err := parsePart(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month error: %v", err)
	}
	dayOfWeek, err := parsePart(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("dayOfWeek error: %v", err)
	}

	return &CronExpr{
		Minute:     minute,
		Hour:       hour,
		DayOfMonth: dayOfMonth,
		Month:      month,
		DayOfWeek:  dayOfWeek,
	}, nil
}

func CalculateNextRun(expr string, from time.Time) time.Time {
	cronExpr, err := ParseCron(expr)
	if err != nil {
		log.Printf("invalid cron expression: %v", err)
		return from.Add(1 * time.Hour)
	}

	for i := 1; i <= 525600; i++ { // Max one year
		t := from.Add(time.Duration(i) * time.Minute)
		if cronExpr.Month[int(t.Month())] &&
			cronExpr.DayOfMonth[t.Day()] &&
			cronExpr.Hour[t.Hour()] &&
			cronExpr.Minute[t.Minute()] &&
			cronExpr.DayOfWeek[int(t.Weekday())] {
			return t
		}
	}

	return from.Add(1 * time.Hour) // fallback
}
