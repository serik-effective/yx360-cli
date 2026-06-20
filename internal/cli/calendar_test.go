package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/effective-dev-os/yx360-cli/internal/calendar"
)

func TestCalendarRoomsAddAndList(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	jsonOutput = false
	insecureFileStore = false

	var out bytes.Buffer
	add := NewRootCmd()
	add.SetArgs([]string{"calendar", "rooms", "add", "Sun", "sun@example.com"})
	add.SetOut(&out)
	add.SetErr(&out)
	if err := add.Execute(); err != nil {
		t.Fatalf("rooms add: %v\n%s", err, out.String())
	}

	out.Reset()
	list := NewRootCmd()
	list.SetArgs([]string{"calendar", "rooms", "list"})
	list.SetOut(&out)
	list.SetErr(&out)
	if err := list.Execute(); err != nil {
		t.Fatalf("rooms list: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Sun sun@example.com") {
		t.Fatalf("rooms list output = %q", out.String())
	}
}

func TestCalendarCreateUnknownRoomFailsBeforeAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	jsonOutput = false
	insecureFileStore = false

	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"calendar", "create",
		"--title", "Meeting",
		"--starts-at", "2026-06-22T09:00:00+06:00",
		"--ends-at", "2026-06-22T09:30:00+06:00",
		"--room", "Sun",
		"--yes",
	})
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("create error = nil")
	}
	if !strings.Contains(err.Error(), `calendar: unknown room "Sun"`) {
		t.Fatalf("create error = %v", err)
	}
}

func TestConfirmCalendarMutationShowsRoomsLocationAndURL(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader("y\n"))

	err := confirmCalendarMutation(cmd, "Create calendar event?", calendar.Event{
		Location: "Sun",
		URL:      "https://telemost.example/j/1",
		Rooms: []calendar.Room{{
			Name:  "Sun",
			Email: "sun@example.com",
		}},
	}, true)
	if err != nil {
		t.Fatalf("confirmCalendarMutation: %v", err)
	}
	for _, want := range []string{
		"Rooms: Sun <sun@example.com>",
		"Location: Sun",
		"URL: https://telemost.example/j/1",
		"Telemost: create link",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("preview missing %q in:\n%s", want, out.String())
		}
	}
}

func TestFillRoomLocationUsesRoomNames(t *testing.T) {
	event := calendar.Event{
		Rooms: []calendar.Room{{
			Name:  "Sun",
			Email: "sun@example.com",
		}},
	}
	fillRoomLocation(&event)
	if event.Location != "Sun" {
		t.Fatalf("location = %q, want Sun", event.Location)
	}

	event.Location = "Office"
	fillRoomLocation(&event)
	if event.Location != "Office" {
		t.Fatalf("explicit location overwritten: %q", event.Location)
	}
}
