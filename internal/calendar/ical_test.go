package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestBuildAndParseICS(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 6, 22, 3, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	ics := buildICS(Event{
		UID:         "uid-1",
		Title:       "Smoke, test",
		Description: "line 1\nline 2",
		Location:    "https://telemost.360.yandex.ru/j/1",
		URL:         "https://telemost.360.yandex.ru/j/1",
		StartsAt:    start,
		EndsAt:      end,
		Attendees:   []string{"user@example.com"},
	})
	if !strings.Contains(ics, "SUMMARY:Smoke\\, test") {
		t.Fatalf("ICS did not escape summary:\n%s", ics)
	}

	event := parseEvent("/event.ics", `"etag"`, ics)
	if event.UID != "uid-1" || event.Title != "Smoke, test" {
		t.Fatalf("parsed event = %+v", event)
	}
	if event.Description != "line 1\nline 2" {
		t.Fatalf("description = %q", event.Description)
	}
	if !event.StartsAt.Equal(start) || !event.EndsAt.Equal(end) {
		t.Fatalf("time range = %s-%s", event.StartsAt, event.EndsAt)
	}
	if len(event.Attendees) != 1 || event.Attendees[0] != "user@example.com" {
		t.Fatalf("attendees = %+v", event.Attendees)
	}
}

func TestUnfoldICS(t *testing.T) {
	t.Parallel()

	lines := unfoldICS("SUMMARY:hello\r\n world\r\nUID:1\r\n")
	if len(lines) != 2 || lines[0] != "SUMMARY:helloworld" || lines[1] != "UID:1" {
		t.Fatalf("unfoldICS = %#v", lines)
	}
}

func TestRecurringWeeklyOccurrenceInRange(t *testing.T) {
	t.Parallel()

	event := parseEvent("/event.ics", `"etag"`, strings.Join([]string{
		"BEGIN:VCALENDAR",
		"BEGIN:VEVENT",
		"UID:weekly-1",
		"DTSTART:20260608T160000Z",
		"DTEND:20260608T163000Z",
		"RRULE:FREQ=WEEKLY;INTERVAL=1;BYDAY=MO",
		"SUMMARY:Traction Stand-up",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n"))

	from := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC)
	got := event.occurrenceIn(from, to)
	wantStart := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 6, 22, 16, 30, 0, 0, time.UTC)

	if !got.StartsAt.Equal(wantStart) || !got.EndsAt.Equal(wantEnd) {
		t.Fatalf("occurrence = %s-%s, want %s-%s", got.StartsAt, got.EndsAt, wantStart, wantEnd)
	}
}

func TestRecurringGoogleInstanceWithoutRuleInRange(t *testing.T) {
	t.Parallel()

	event := Event{
		Href:     "/events/weekly_R20260504T100000@google.com.ics",
		UID:      "weekly_R20260504T100000@google.com",
		Title:    "Traction Stand-up",
		StartsAt: time.Date(2026, 6, 8, 16, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 6, 8, 16, 30, 0, 0, time.UTC),
	}

	from := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC)
	got := event.occurrenceIn(from, to)
	wantStart := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 6, 22, 16, 30, 0, 0, time.UTC)

	if !got.StartsAt.Equal(wantStart) || !got.EndsAt.Equal(wantEnd) {
		t.Fatalf("occurrence = %s-%s, want %s-%s", got.StartsAt, got.EndsAt, wantStart, wantEnd)
	}
}
