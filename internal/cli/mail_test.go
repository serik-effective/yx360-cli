package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/effective-dev-os/yx360-cli/internal/mail"
)

func TestParseUnsubscribeMethod(t *testing.T) {
	t.Parallel()

	tests := map[string]mail.UnsubscribeMethod{
		"":           "",
		"https-post": mail.UnsubscribeHTTPSPost,
		"https-get":  mail.UnsubscribeHTTPSGet,
		"mailto":     mail.UnsubscribeMailto,
	}
	for input, want := range tests {
		got, err := parseUnsubscribeMethod(input)
		if err != nil {
			t.Fatalf("parseUnsubscribeMethod(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("parseUnsubscribeMethod(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := parseUnsubscribeMethod("post"); err == nil {
		t.Fatalf("parseUnsubscribeMethod(post) error = nil, want error")
	}
}

func TestParseTimeValue(t *testing.T) {
	t.Parallel()

	got, err := parseTimeValue("2026-06-22T09:00:00+06:00")
	if err != nil {
		t.Fatalf("parseTimeValue RFC3339: %v", err)
	}
	if got.UTC().Format(time.RFC3339) != "2026-06-22T03:00:00Z" {
		t.Fatalf("UTC time = %s", got.UTC().Format(time.RFC3339))
	}

	got, err = parseTimeValue("2026-06-22")
	if err != nil {
		t.Fatalf("parseTimeValue date: %v", err)
	}
	if got.Format("2006-01-02") != "2026-06-22" {
		t.Fatalf("date = %s", got.Format("2006-01-02"))
	}
}

func TestHumanLoginIncludesProfile(t *testing.T) {
	t.Parallel()

	got := humanLogin(loginPayload{Account: "user@example.com", Profile: calendarTelemostProfile})
	if !strings.Contains(got, "(calendar-telemost)") {
		t.Fatalf("humanLogin = %q", got)
	}
}

func TestHumanUnsubscribePreviewMarksSelected(t *testing.T) {
	t.Parallel()

	preview := &mail.UnsubscribePreview{
		UID:    42,
		Folder: "INBOX",
		Options: []mail.UnsubscribeOption{
			{Method: mail.UnsubscribeMailto, URI: "mailto:list@example.com", RequiresSMTP: true},
			{Method: mail.UnsubscribeHTTPSPost, URI: "https://example.com/unsub", OneClick: true},
		},
	}
	preview.Selected = &preview.Options[1]

	got := humanUnsubscribePreview(preview)
	for _, want := range []string{
		"Unsubscribe options for INBOX/42:",
		"  1. mailto mailto:list@example.com requires_smtp=true",
		"* 2. https-post https://example.com/unsub one_click=true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview missing %q in:\n%s", want, got)
		}
	}
}
