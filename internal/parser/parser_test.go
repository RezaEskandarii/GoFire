package parser

import (
	"testing"
	"time"
)

func TestParsePart(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		min    int
		max    int
		valid  bool
		expect []int
	}{
		{"wildcard full range", "*", 0, 5, true, []int{0, 1, 2, 3, 4, 5}},
		{"single value", "3", 0, 5, true, []int{3}},
		{"range", "2-4", 0, 5, true, []int{2, 3, 4}},
		{"mixed", "1,3-4,5", 0, 5, true, []int{1, 3, 4, 5}},
		{"invalid number", "a", 0, 5, false, nil},
		{"out of bounds", "7", 0, 5, false, nil},
		{"bad range", "4-2", 0, 5, false, nil},
		{"bad range format", "1-2-3", 0, 5, false, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parsePart(test.input, test.min, test.max)
			if test.valid && err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if !test.valid && err == nil {
				t.Errorf("expected error but got none")
			} else if test.valid {
				if len(result) != len(test.expect) {
					t.Errorf("expected length %d, got %d", len(test.expect), len(result))
				}
				for i := range result {
					if result[i] != test.expect[i] {
						t.Errorf("expected %v, got %v", test.expect, result)
						break
					}
				}
			}
		})
	}
}

func TestCalculateNextRun(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		from    time.Time
		expects time.Time
	}{
		{
			name:    "next 15-min mark in same hour",
			expr:    "*/15 14 * * *",
			from:    time.Date(2025, 6, 21, 14, 0, 0, 0, time.UTC),
			expects: time.Date(2025, 6, 21, 14, 15, 0, 0, time.UTC),
		},
		{
			name:    "next day midnight",
			expr:    "0 0 1 1 *",
			from:    time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC),
			expects: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "next weekday",
			expr:    "0 9 * * 1",
			from:    time.Date(2025, 6, 20, 8, 59, 0, 0, time.UTC),
			expects: time.Date(2025, 6, 23, 9, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid cron fallback",
			expr:    "invalid expression",
			from:    time.Date(2025, 6, 21, 10, 0, 0, 0, time.UTC),
			expects: time.Date(2025, 6, 21, 11, 0, 0, 0, time.UTC), // fallback 1h
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			next := CalculateNextRun(test.expr, test.from)
			if !next.Equal(test.expects) {
				t.Errorf("CalculateNextRun(%q, %v) = %v; want %v", test.expr, test.from, next, test.expects)
			}
		})
	}
}
