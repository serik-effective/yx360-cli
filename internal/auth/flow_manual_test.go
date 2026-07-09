package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/effective-dev-os/yx360-cli/internal/config"
)

func redirectConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	return dir
}

func TestParseManualInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		state    string
		wantCode string
		wantErr  string
	}{
		{"bare code", "abc123", "st", "abc123", ""},
		{"bare code trimmed", "  abc123\n", "st", "abc123", ""},
		{"url with matching state", "https://oauth.yandex.ru/verification_code?code=xyz&state=st", "st", "xyz", ""},
		{"url with mismatched state", "https://oauth.yandex.ru/verification_code?code=xyz&state=bad", "st", "", "state mismatch"},
		{"url with error param", "https://oauth.yandex.ru/verification_code?error=access_denied", "st", "", "authorize rejected"},
		{"url without code", "https://oauth.yandex.ru/verification_code?state=st", "st", "", "no authorization code"},
		{"empty input", "", "st", "", "no authorization code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := parseManualInput(tt.input, tt.state)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tt.wantCode {
				t.Fatalf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestBeginWritesPendingAndURL(t *testing.T) {
	dir := redirectConfigDir(t)
	cfg := config.Default()
	cfg.ClientID = "test-client"
	mp := NewManualProvider(cfg)

	authURL, err := mp.Begin(context.Background(), "mail", AuthOptions{Scopes: []string{"login:info", config.MailReadScope}})
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if !strings.Contains(authURL, "verification_code") {
		t.Fatalf("auth URL missing verification_code redirect: %s", authURL)
	}
	if !strings.Contains(authURL, "code_challenge=") {
		t.Fatalf("auth URL missing PKCE challenge: %s", authURL)
	}

	path := filepath.Join(dir, "config", "yx360", "manual-login.json")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(dir, "Library", "Application Support", "yx360", "manual-login.json")
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("pending file not written: %v", err)
	}
	var pending pendingManualLogin
	if err := json.Unmarshal(blob, &pending); err != nil {
		t.Fatalf("pending unmarshal: %v", err)
	}
	if pending.Verifier == "" || pending.State == "" {
		t.Fatalf("pending missing verifier/state: %+v", pending)
	}
	if pending.Profile != "mail" || pending.ClientID != "test-client" {
		t.Fatalf("pending profile/client wrong: %+v", pending)
	}

	profile, err := LoadPendingProfile()
	if err != nil {
		t.Fatalf("LoadPendingProfile: %v", err)
	}
	if profile != "mail" {
		t.Fatalf("profile = %q, want mail", profile)
	}
}

func TestCompleteRejectsMissingPending(t *testing.T) {
	redirectConfigDir(t)
	mp := NewManualProvider(config.Default())
	_, err := mp.Complete(context.Background(), "code")
	if err == nil || !strings.Contains(err.Error(), "no pending manual login") {
		t.Fatalf("want no-pending error, got %v", err)
	}
}

func TestCompleteRejectsExpired(t *testing.T) {
	redirectConfigDir(t)
	if err := writePendingManualLogin(pendingManualLogin{
		Verifier:  "v",
		State:     "s",
		Profile:   "mail",
		ClientID:  "c",
		CreatedAt: time.Now().Add(-11 * time.Minute),
	}); err != nil {
		t.Fatalf("write pending: %v", err)
	}
	mp := NewManualProvider(config.Default())
	_, err := mp.Complete(context.Background(), "code")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("want expired error, got %v", err)
	}
	if _, err := loadPendingManualLogin(); err == nil {
		t.Fatal("stale pending file should have been deleted")
	}
}

func TestCompleteRejectsOversizedInput(t *testing.T) {
	redirectConfigDir(t)
	if err := writePendingManualLogin(pendingManualLogin{
		Verifier:  "v",
		State:     "s",
		Profile:   "mail",
		ClientID:  "c",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("write pending: %v", err)
	}
	mp := NewManualProvider(config.Default())
	_, err := mp.Complete(context.Background(), strings.Repeat("a", maxManualInputLen+1))
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Fatalf("want too-long error, got %v", err)
	}
}

func TestCompleteRejectsURLErrorParam(t *testing.T) {
	redirectConfigDir(t)
	if err := writePendingManualLogin(pendingManualLogin{
		Verifier:  "v",
		State:     "s",
		Profile:   "mail",
		ClientID:  "c",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("write pending: %v", err)
	}
	mp := NewManualProvider(config.Default())
	_, err := mp.Complete(context.Background(), "https://oauth.yandex.ru/verification_code?error=access_denied")
	if err == nil || !strings.Contains(err.Error(), "authorize rejected") {
		t.Fatalf("want authorize-rejected error, got %v", err)
	}
}

func TestCompleteRejectsMismatchedState(t *testing.T) {
	redirectConfigDir(t)
	if err := writePendingManualLogin(pendingManualLogin{
		Verifier:  "v",
		State:     "expected-state",
		Profile:   "mail",
		ClientID:  "c",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("write pending: %v", err)
	}
	mp := NewManualProvider(config.Default())
	_, err := mp.Complete(context.Background(), "https://oauth.yandex.ru/verification_code?code=x&state=wrong")
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("want state-mismatch error, got %v", err)
	}
}
