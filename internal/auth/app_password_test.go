package auth

import (
	"errors"
	"testing"
	"time"
)

func TestNewAppPasswordCredential(t *testing.T) {
	cred, err := NewAppPasswordCredential("  user@example.com ", " secret ")
	if err != nil {
		t.Fatalf("NewAppPasswordCredential: %v", err)
	}
	if cred.Account != "user@example.com" || cred.AppPassword != "secret" {
		t.Fatalf("credential = %+v", cred)
	}
	if cred.ObtainedVia != GrantAppPassword || !cred.IsAppPassword() {
		t.Fatalf("grant kind = %q", cred.ObtainedVia)
	}
	if _, err := NewAppPasswordCredential("user", "secret"); !errors.Is(err, ErrAppPasswordAccount) {
		t.Fatalf("bare login error = %v", err)
	}
	if _, err := NewAppPasswordCredential("user@example.com", "  "); !errors.Is(err, ErrAppPasswordEmpty) {
		t.Fatalf("empty password error = %v", err)
	}
}

func TestUsableFor(t *testing.T) {
	appPassword := &Credential{AppPassword: "secret", Account: "user@example.com", ObtainedVia: GrantAppPassword}
	if !appPassword.UsableFor("mail:imap_full") {
		t.Fatalf("app password rejected for a scope it carries no claim about")
	}

	expired := &Credential{AccessToken: "x", Scope: "mail:imap_full", Expiry: time.Now().Add(-time.Hour)}
	if expired.UsableFor("mail:imap_full") {
		t.Fatalf("expired OAuth credential accepted")
	}
	missingScope := &Credential{AccessToken: "x", Scope: "login:info"}
	if missingScope.UsableFor("mail:imap_full") {
		t.Fatalf("OAuth credential without the scope accepted")
	}
	// An app-password field without the matching grant kind must not bypass scopes.
	mislabelled := &Credential{AppPassword: "secret", Account: "user@example.com", Scope: "login:info"}
	if mislabelled.UsableFor("mail:imap_full") {
		t.Fatalf("credential without GrantAppPassword accepted")
	}
}
