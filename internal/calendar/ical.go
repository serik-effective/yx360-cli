package calendar

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Event struct {
	ID           string        `json:"id"`
	Href         string        `json:"href"`
	ETag         string        `json:"etag,omitempty"`
	UID          string        `json:"uid"`
	Title        string        `json:"title"`
	Description  string        `json:"description,omitempty"`
	Location     string        `json:"location,omitempty"`
	URL          string        `json:"url,omitempty"`
	StartsAt     time.Time     `json:"starts_at"`
	EndsAt       time.Time     `json:"ends_at"`
	Attendees    []string      `json:"attendees,omitempty"`
	Organizer    *Participant  `json:"organizer,omitempty"`
	Participants []Participant `json:"participants,omitempty"`
	Rooms        []Room        `json:"rooms,omitempty"`
	Resources    []Room        `json:"resources,omitempty"`

	recurrenceRule string
}

type Participant struct {
	Email          string `json:"email,omitempty"`
	URI            string `json:"uri,omitempty"`
	Name           string `json:"name,omitempty"`
	Kind           string `json:"kind,omitempty"`
	Role           string `json:"role,omitempty"`
	Partstat       string `json:"partstat,omitempty"`
	RSVP           string `json:"rsvp,omitempty"`
	ScheduleStatus string `json:"schedule_status,omitempty"`
}

type Room struct {
	Alias          string `json:"alias,omitempty"`
	Name           string `json:"name,omitempty"`
	Email          string `json:"email,omitempty"`
	URI            string `json:"uri,omitempty"`
	Status         string `json:"status,omitempty"`
	Role           string `json:"role,omitempty"`
	RSVP           string `json:"rsvp,omitempty"`
	ScheduleStatus string `json:"schedule_status,omitempty"`
	Kind           string `json:"kind,omitempty"`
}

func parseEvent(href, etag, data string) Event {
	lines := unfoldICS(data)
	event := Event{ID: href, Href: href, ETag: etag}
	for _, line := range lines {
		name, params, value, ok := splitICSProperty(line)
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
		case "ORGANIZER":
			organizer := participantFromICS(value, params)
			event.Organizer = &organizer
		case "ATTENDEE":
			participant := participantFromICS(value, params)
			event.Participants = append(event.Participants, participant)
			switch strings.ToUpper(participant.Kind) {
			case "ROOM":
				event.Rooms = append(event.Rooms, roomFromParticipant(participant, "ROOM"))
			case "RESOURCE":
				event.Resources = append(event.Resources, roomFromParticipant(participant, "RESOURCE"))
			default:
				if participant.Email != "" {
					event.Attendees = append(event.Attendees, participant.Email)
				}
			}
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
	if event.Organizer != nil {
		writeICSLine(&b, organizerICSLine(*event.Organizer))
	}
	for _, attendee := range event.Attendees {
		writeICSLine(&b, "ATTENDEE:mailto:"+attendee)
	}
	for _, room := range event.Rooms {
		writeICSLine(&b, roomAttendeeICSLine(room, "ROOM"))
	}
	for _, resource := range event.Resources {
		writeICSLine(&b, roomAttendeeICSLine(resource, "RESOURCE"))
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
	name, _, value, ok := splitICSProperty(line)
	return name, value, ok
}

func splitICSProperty(line string) (string, map[string]string, string, bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", nil, "", false
	}
	left := line[:idx]
	value := line[idx+1:]
	parts := strings.Split(left, ";")
	name := strings.ToUpper(parts[0])
	params := make(map[string]string)
	for _, part := range parts[1:] {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		params[strings.ToUpper(key)] = unquoteParam(val)
	}
	return name, params, value, true
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

func participantFromICS(value string, params map[string]string) Participant {
	uri := value
	email := strings.TrimPrefix(value, "mailto:")
	if strings.EqualFold(email, value) {
		email = ""
	}
	return Participant{
		Email:          email,
		URI:            uri,
		Name:           unescapeText(params["CN"]),
		Kind:           strings.ToUpper(params["CUTYPE"]),
		Role:           strings.ToUpper(params["ROLE"]),
		Partstat:       strings.ToUpper(params["PARTSTAT"]),
		RSVP:           strings.ToUpper(params["RSVP"]),
		ScheduleStatus: params["SCHEDULE-STATUS"],
	}
}

func roomFromParticipant(participant Participant, kind string) Room {
	name := participant.Name
	if name == "" {
		name = participant.Email
	}
	return Room{
		Alias:          name,
		Name:           name,
		Email:          participant.Email,
		URI:            participant.URI,
		Status:         participant.Partstat,
		Role:           participant.Role,
		RSVP:           participant.RSVP,
		ScheduleStatus: participant.ScheduleStatus,
		Kind:           kind,
	}
}

func roomAttendeeICSLine(room Room, fallbackKind string) string {
	kind := strings.ToUpper(firstNonEmpty(room.Kind, fallbackKind))
	partstat := firstNonEmpty(room.Status, "ACCEPTED")
	name := firstNonEmpty(room.Name, room.Alias, room.Email)
	role := firstNonEmpty(room.Role, "REQ-PARTICIPANT")
	uri := room.URI
	if uri == "" && room.Email != "" {
		uri = "mailto:" + room.Email
	}

	var b strings.Builder
	b.WriteString("ATTENDEE")
	b.WriteString(";CUTYPE=")
	b.WriteString(kind)
	b.WriteString(";PARTSTAT=")
	b.WriteString(partstat)
	if name != "" {
		b.WriteString(";CN=")
		b.WriteString(escapeParam(name))
	}
	b.WriteString(";ROLE=")
	b.WriteString(role)
	if room.RSVP != "" {
		b.WriteString(";RSVP=")
		b.WriteString(room.RSVP)
	}
	if room.ScheduleStatus != "" {
		b.WriteString(";SCHEDULE-STATUS=")
		b.WriteString(escapeParam(room.ScheduleStatus))
	}
	b.WriteByte(':')
	b.WriteString(uri)
	return b.String()
}

func organizerICSLine(participant Participant) string {
	uri := participant.URI
	if uri == "" && participant.Email != "" {
		uri = "mailto:" + participant.Email
	}
	var b strings.Builder
	b.WriteString("ORGANIZER")
	if participant.Name != "" {
		b.WriteString(";CN=")
		b.WriteString(escapeParam(participant.Name))
	}
	b.WriteByte(':')
	b.WriteString(uri)
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func unquoteParam(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		value = strings.TrimPrefix(strings.TrimSuffix(value, `"`), `"`)
	}
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}

func escapeParam(value string) string {
	if strings.ContainsAny(value, `;:,"`) {
		value = strings.ReplaceAll(value, `\`, `\\`)
		value = strings.ReplaceAll(value, `"`, `\"`)
		return `"` + value + `"`
	}
	return value
}

func eventHref(calendarURL, uid string) string {
	return strings.TrimRight(calendarURL, "/") + "/" + url.PathEscape(uid) + ".ics"
}

func newUID() string {
	return fmt.Sprintf("yx360-%d@yandex360.local", time.Now().UnixNano())
}
