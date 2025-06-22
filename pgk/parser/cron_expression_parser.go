// Package parser provides a lightweight cron parser with efficient next-run calculation
package parser

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CronExpr represents a parsed cron expression with accepted values per field
// Each field contains a sorted slice of valid values
// Minute: 0–59
// Hour: 0–23
// DayOfMonth: 1–31
// Month: 1–12
// DayOfWeek: 0–6 (Sunday to Saturday)
type CronExpr struct {
	Minute     []int
	Hour       []int
	DayOfMonth []int
	Month      []int
	DayOfWeek  []int
}

// parsePart parses a cron field like "1,2,5-7" into a sorted slice of integers
func parsePart(part string, min, max int) ([]int, error) {
	set := make(map[int]struct{})

	if part == "*" || strings.HasPrefix(part, "*/") {
		step := 1
		if strings.HasPrefix(part, "*/") {
			s, err := strconv.Atoi(part[2:])
			if err != nil || s <= 0 {
				return nil, fmt.Errorf("invalid step: %s", part)
			}
			step = s
		}
		for i := min; i <= max; i += step {
			set[i] = struct{}{}
		}
	} else {
		for _, token := range strings.Split(part, ",") {
			if strings.Contains(token, "/") {
				rangeAndStep := strings.Split(token, "/")
				if len(rangeAndStep) != 2 {
					return nil, fmt.Errorf("invalid step expression: %s", token)
				}
				base := rangeAndStep[0]
				step, err := strconv.Atoi(rangeAndStep[1])
				if err != nil || step <= 0 {
					return nil, fmt.Errorf("invalid step value: %s", token)
				}

				var rStart, rEnd int
				if base == "*" {
					rStart, rEnd = min, max
				} else if strings.Contains(base, "-") {
					parts := strings.Split(base, "-")
					if len(parts) != 2 {
						return nil, fmt.Errorf("invalid range: %s", base)
					}
					rStart, err = strconv.Atoi(parts[0])
					rEnd, err2 := strconv.Atoi(parts[1])
					if err != nil || err2 != nil || rStart > rEnd || rStart < min || rEnd > max {
						return nil, fmt.Errorf("invalid range: %s", base)
					}
				} else {
					num, err := strconv.Atoi(base)
					if err != nil || num < min || num > max {
						return nil, fmt.Errorf("invalid value: %s", base)
					}
					rStart, rEnd = num, max
				}

				for i := rStart; i <= rEnd; i += step {
					set[i] = struct{}{}
				}
			} else if strings.Contains(token, "-") {
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
					set[i] = struct{}{}
				}
			} else {
				num, err := strconv.Atoi(token)
				if err != nil || num < min || num > max {
					return nil, fmt.Errorf("invalid value: %s", token)
				}
				set[num] = struct{}{}
			}
		}
	}

	result := make([]int, 0, len(set))
	for val := range set {
		result = append(result, val)
	}
	sort.Ints(result)
	return result, nil
}

// contains checks if value v exists in sorted slice vals
func contains(vals []int, v int) bool {
	i := sort.SearchInts(vals, v)
	return i < len(vals) && vals[i] == v
}

// ParseCron parses a cron string like "*/5 0 1-10 * 1,3" into a CronExpr structure
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

// CalculateNextRun finds the next run time based on a cron expression and starting time
func CalculateNextRun(expr string, from time.Time) time.Time {
	cronExpr, err := ParseCron(expr)
	if err != nil {
		log.Printf("invalid cron expression: %v", err)
		return from.Add(time.Hour)
	}

	t := from.Add(time.Minute).Truncate(time.Minute) // start from next minute

	// Loop for up to one year of minutes (worst-case fallback)
	for i := 0; i < 525600; i++ {
		// Fast-forward by skipping invalid months/days/hours
		if !contains(cronExpr.Month, int(t.Month())) {
			t = t.AddDate(0, 1, -t.Day()+1).Truncate(24 * time.Hour)
			continue
		}
		if !contains(cronExpr.DayOfMonth, t.Day()) || !contains(cronExpr.DayOfWeek, int(t.Weekday())) {
			t = t.AddDate(0, 0, 1).Truncate(24 * time.Hour)
			continue
		}
		if !contains(cronExpr.Hour, t.Hour()) {
			t = t.Add(time.Hour - time.Duration(t.Minute())*time.Minute)
			continue
		}
		if contains(cronExpr.Minute, t.Minute()) {
			return t // found!
		}
		t = t.Add(time.Minute)
	}

	// fallback if nothing matched
	return from.Add(time.Hour)
}
