package timeutil

import (
	"testing"
	"time"
)

func TestParseLMSMoscowTime(t *testing.T) {
	// Audited quirk: "2026-10-24 04:00:00" in LMS is 04:00 Moscow (+03:00) == 08:00 WIB (+07:00).
	got, err := ParseLMSMoscowTime("2026-10-24 04:00:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "2026-10-24T08:00:00+07:00"
	if FormatWIB(got) != want {
		t.Fatalf("got %s, want %s", FormatWIB(got), want)
	}

	// Explicit offset should be respected without adding another +4h.
	gotExplicit, err := ParseLMSMoscowTime("2026-10-24T13:30:00+07:00")
	if err != nil {
		t.Fatalf("unexpected error on explicit offset: %v", err)
	}
	if FormatWIB(gotExplicit) != "2026-10-24T13:30:00+07:00" {
		t.Fatalf("got %s, want 2026-10-24T13:30:00+07:00", FormatWIB(gotExplicit))
	}
}

func TestCanRescheduleLessonMondayGrace(t *testing.T) {
	// Previous week lesson: Saturday 2026-10-24 10:00 WIB
	lesson := time.Date(2026, 10, 24, 10, 0, 0, 0, JakartaLoc)

	// Monday 2026-10-26 22:30 WIB -> still within Monday 23:59 WIB grace window
	nowMonday := time.Date(2026, 10, 26, 22, 30, 0, 0, JakartaLoc)
	if !CanRescheduleLesson(lesson, nowMonday) {
		t.Fatalf("expected lesson from previous week to be reschedulable on Monday before 23:59 WIB")
	}

	// Tuesday 2026-10-27 00:05 WIB -> grace window expired
	nowTuesday := time.Date(2026, 10, 27, 0, 5, 0, 0, JakartaLoc)
	if CanRescheduleLesson(lesson, nowTuesday) {
		t.Fatalf("expected lesson from previous week NOT to be reschedulable after Monday 23:59 WIB")
	}
}
