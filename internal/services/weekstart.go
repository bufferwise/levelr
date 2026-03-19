package services

import (
	"fmt"
	"time"
)

// WeekStart returns the Monday 00:00:00 UTC time of the week for a given time.
// This is used as the partition key for weekly stats.
func WeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday()) // Sunday=0, Monday=1, ...
	// ISO week start is Monday. In Go, Sunday is 0.
	// Map Sunday (0) to 7 so we can subtract weekday-1 safely.
	if weekday == 0 {
		weekday = 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// WeekStartString returns the ISO-formatted date string (YYYY-MM-DD) for use with SQLite.
func WeekStartString(t time.Time) string {
	ws := WeekStart(t)
	return fmt.Sprintf("%04d-%02d-%02d", ws.Year(), ws.Month(), ws.Day())
}
