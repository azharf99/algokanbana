package timeutil

import (
	"fmt"
	"strings"
	"time"
)

var (
	// MoscowLoc represents UTC+03:00 used by Algonova/Algoritmika LMS naive startTime strings.
	MoscowLoc = time.FixedZone("MSK", 3*3600)
	// JakartaLoc represents WIB (Asia/Jakarta, UTC+07:00).
	JakartaLoc = time.FixedZone("WIB", 7*3600)
)

// ParseLMSMoscowTime parses naive LMS datetime strings ("YYYY-MM-DD HH:MM:SS" or ISO strings)
// which lms.algonova.id emits in Moscow Time (UTC+03:00), returning the instant in Asia/Jakarta (UTC+07:00).
func ParseLMSMoscowTime(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty LMS timestamp")
	}

	// If explicit timezone offset or Z is already attached, parse RFC3339 directly.
	if strings.Contains(s, "Z") || (len(s) > 19 && (strings.Contains(s[19:], "+") || strings.Contains(s[19:], "-"))) {
		normalized := strings.Replace(s, " ", "T", 1)
		t, err := time.Parse(time.RFC3339, normalized)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse RFC3339 timestamp %q: %w", raw, err)
		}
		return t.In(JakartaLoc), nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, MoscowLoc); err == nil {
			return t.In(JakartaLoc), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported LMS timestamp format %q", raw)
}

// FormatWIB formats any time into RFC3339 in WIB (+07:00).
func FormatWIB(t time.Time) string {
	return t.In(JakartaLoc).Format(time.RFC3339)
}

// MondayOfWeekWIB returns the Monday 00:00:00 WIB for the week containing t.
func MondayOfWeekWIB(t time.Time) time.Time {
	wib := t.In(JakartaLoc)
	weekday := int(wib.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday -> 7
	}
	monday := wib.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, JakartaLoc)
}

// ParseDateWIB parses "YYYY-MM-DD" in WIB 00:00:00.
func ParseDateWIB(dateStr string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", strings.TrimSpace(dateStr), JakartaLoc)
}

// CanRescheduleLesson checks Algonova's Monday 23:59:59 WIB grace rule:
// Current/future week lessons are allowed; previous week lessons are allowed ONLY until Monday 23:59:59 WIB of the current week.
func CanRescheduleLesson(lessonStartsAt, now time.Time) bool {
	lessonWIB := lessonStartsAt.In(JakartaLoc)
	nowWIB := now.In(JakartaLoc)

	currentWeekMonday := MondayOfWeekWIB(nowWIB)
	if !lessonWIB.Before(currentWeekMonday) {
		return true
	}

	prevWeekMonday := currentWeekMonday.AddDate(0, 0, -7)
	mondayGraceDeadline := time.Date(
		currentWeekMonday.Year(),
		currentWeekMonday.Month(),
		currentWeekMonday.Day(),
		23, 59, 59, 0,
		JakartaLoc,
	)
	if !lessonWIB.Before(prevWeekMonday) && !nowWIB.After(mondayGraceDeadline) {
		return true
	}
	return false
}
