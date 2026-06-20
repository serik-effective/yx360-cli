package calendar

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Event struct {
	ID          string    `json:"id"`
	Href        string    `json:"href"`
	ETag        string    `json:"etag,omitempty"`
	UID         string    `json:"uid"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	URL         string    `json:"url,omitempty"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Attendees   []string  `json:"attendees,omitempty"`

	recurrenceRule string
}

func parseEvent(href, etag, data string) Event {
	lines := unfoldICS(data)
	event := Event{ID: href, Href: href, ETag: etag}
	for _, line := range lines {
		name, value, ok := splitICSLine(line)
		if !ok {
			continue
		}
		switch name {
		case "UID":
			event.UID = value
		case "SUMMARY":
			event.Title = unescapeText(value)
		case "DESCRIPTION":
			event.Description = unescapeText(value)
		case "LOCATION":
			event.Location = unescapeText(value)
		case "URL":
			event.URL = value
		case "DTSTART":
			event.StartsAt = parseICSTime(value)
		case "DTEND":
			event.EndsAt = parseICSTime(value)
		case "RRULE":
			event.recurrenceRule = value
		case "ATTENDEE":
			event.Attendees = append(event.Attendees, strings.TrimPrefix(value, "mailto:"))
		}
	}
	return event
}

func (event Event) occurrenceIn(from, to time.Time) Event {
	if event.StartsAt.IsZero() || event.EndsAt.IsZero() {
		return event
	}
	if event.StartsAt.Before(to) && event.EndsAt.After(from) {
		return event
	}
	if event.recurrenceRule == "" {
		if event.isRecurringInstance() {
			return event.recurringInstanceOccurrenceIn(from, to)
		}
		return event
	}

	nextStart, ok := nextRecurringStart(event.StartsAt, from, to, parseRRule(event.recurrenceRule))
	if !ok {
		if event.isRecurringInstance() {
			return event.recurringInstanceOccurrenceIn(from, to)
		}
		return event
	}
	duration := event.EndsAt.Sub(event.StartsAt)
	event.StartsAt = nextStart
	event.EndsAt = nextStart.Add(duration)
	return event
}

func (event Event) isRecurringInstance() bool {
	return strings.Contains(event.UID, "_R") || strings.Contains(event.Href, "_R")
}

func (event Event) recurringInstanceOccurrenceIn(from, to time.Time) Event {
	nextStart, ok := nextWeeklyStart(event.StartsAt, from, to, 1, map[time.Weekday]bool{event.StartsAt.Weekday(): true})
	if !ok {
		return event
	}
	duration := event.EndsAt.Sub(event.StartsAt)
	event.StartsAt = nextStart
	event.EndsAt = nextStart.Add(duration)
	return event
}

func parseRRule(rule string) map[string]string {
	parts := strings.Split(rule, ";")
	parsed := make(map[string]string, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if ok {
			parsed[strings.ToUpper(key)] = strings.ToUpper(value)
		}
	}
	return parsed
}

func nextRecurringStart(start, from, to time.Time, rule map[string]string) (time.Time, bool) {
	interval := recurrenceInterval(rule["INTERVAL"])
	switch rule["FREQ"] {
	case "DAILY":
		return nextDailyStart(start, from, to, interval)
	case "WEEKLY":
		return nextWeeklyStart(start, from, to, interval, recurrenceWeekdays(rule["BYDAY"], start.Weekday()))
	case "MONTHLY":
		return nextMonthlyStart(start, from, to, interval)
	default:
		return time.Time{}, false
	}
}

func recurrenceInterval(value string) int {
	var interval int
	if _, err := fmt.Sscanf(value, "%d", &interval); err != nil || interval < 1 {
		return 1
	}
	return interval
}

func recurrenceWeekdays(value string, fallback time.Weekday) map[time.Weekday]bool {
	if value == "" {
		return map[time.Weekday]bool{fallback: true}
	}
	days := map[string]time.Weekday{
		"SU": time.Sunday,
		"MO": time.Monday,
		"TU": time.Tuesday,
		"WE": time.Wednesday,
		"TH": time.Thursday,
		"FR": time.Friday,
		"SA": time.Saturday,
	}
	weekdays := make(map[time.Weekday]bool)
	for _, part := range strings.Split(value, ",") {
		if day, ok := days[part]; ok {
			weekdays[day] = true
		}
	}
	if len(weekdays) == 0 {
		weekdays[fallback] = true
	}
	return weekdays
}

func nextDailyStart(start, from, to time.Time, interval int) (time.Time, bool) {
	for candidate := start; candidate.Before(to); candidate = candidate.AddDate(0, 0, interval) {
		if !candidate.Before(from) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func nextWeeklyStart(start, from, to time.Time, interval int, weekdays map[time.Weekday]bool) (time.Time, bool) {
	for candidate := start; candidate.Before(to); candidate = candidate.AddDate(0, 0, 1) {
		weeks := int(candidate.Sub(start).Hours() / 24 / 7)
		if weeks%interval == 0 && weekdays[candidate.Weekday()] && !candidate.Before(from) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func nextMonthlyStart(start, from, to time.Time, interval int) (time.Time, bool) {
	for candidate := start; candidate.Before(to); candidate = candidate.AddDate(0, interval, 0) {
		if !candidate.Before(from) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func buildICS(event Event) string {
	var b strings.Builder
	writeICSLine(&b, "BEGIN:VCALENDAR")
	writeICSLine(&b, "VERSION:2.0")
	writeICSLine(&b, "PRODID:-//yx360//calendar//EN")
	writeICSLine(&b, "BEGIN:VEVENT")
	writeICSLine(&b, "UID:"+event.UID)
	writeICSLine(&b, "DTSTAMP:"+formatICSTime(time.Now().UTC()))
	writeICSLine(&b, "DTSTART:"+formatICSTime(event.StartsAt))
	writeICSLine(&b, "DTEND:"+formatICSTime(event.EndsAt))
	writeICSLine(&b, "SUMMARY:"+escapeText(event.Title))
	if event.Description != "" {
		writeICSLine(&b, "DESCRIPTION:"+escapeText(event.Description))
	}
	if event.Location != "" {
		writeICSLine(&b, "LOCATION:"+escapeText(event.Location))
	}
	if event.URL != "" {
		writeICSLine(&b, "URL:"+event.URL)
	}
	for _, attendee := range event.Attendees {
		writeICSLine(&b, "ATTENDEE:mailto:"+attendee)
	}
	writeICSLine(&b, "END:VEVENT")
	writeICSLine(&b, "END:VCALENDAR")
	return b.String()
}

func unfoldICS(data string) []string {
	raw := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if line == "" {
			continue
		}
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func splitICSLine(line string) (string, string, bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", "", false
	}
	name := strings.ToUpper(line[:idx])
	if semi := strings.IndexByte(name, ';'); semi >= 0 {
		name = name[:semi]
	}
	return name, line[idx+1:], true
}

func parseICSTime(value string) time.Time {
	for _, layout := range []string{"20060102T150405Z", "20060102T150405", "20060102"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

func formatICSTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

func escapeText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, ";", "\\;")
	return value
}

func unescapeText(value string) string {
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\\,", ",")
	value = strings.ReplaceAll(value, "\\;", ";")
	value = strings.ReplaceAll(value, "\\\\", "\\")
	return value
}

func writeICSLine(b *strings.Builder, line string) {
	b.WriteString(line)
	b.WriteString("\r\n")
}

func eventHref(calendarURL, uid string) string {
	return strings.TrimRight(calendarURL, "/") + "/" + url.PathEscape(uid) + ".ics"
}

func newUID() string {
	return fmt.Sprintf("yx360-%d@yandex360.local", time.Now().UnixNano())
}
