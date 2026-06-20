package mail

import (
	"bytes"
	"testing"
)

func TestBuildMessageOmitsBccHeader(t *testing.T) {
	t.Parallel()

	raw, err := buildMessage(SendOptions{
		From:    "sender@example.com",
		To:      []string{"to@example.com"},
		Bcc:     []string{"hidden@example.com"},
		Subject: "hello",
		Text:    "body",
	})
	if err != nil {
		t.Fatalf("buildMessage() error = %v", err)
	}
	if bytes.Contains(raw, []byte("Bcc:")) || bytes.Contains(raw, []byte("hidden@example.com")) {
		t.Fatalf("message leaked Bcc header: %s", raw)
	}
}

func TestAllRecipientsIncludesBcc(t *testing.T) {
	t.Parallel()

	got := allRecipients(SendOptions{
		To:  []string{"to@example.com"},
		Cc:  []string{"cc@example.com"},
		Bcc: []string{"hidden@example.com"},
	})
	want := []string{"to@example.com", "cc@example.com", "hidden@example.com"}
	if len(got) != len(want) {
		t.Fatalf("recipients = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("recipients = %v, want %v", got, want)
		}
	}
}
