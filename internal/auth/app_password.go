package auth

import (
	"errors"
	"strings"
)

// AppPasswordURL is where a user creates a per-service password for a Yandex
// account. Printed in setup errors so the operator does not have to look it up.
const AppPasswordURL = "https://id.yandex.ru/security/app-passwords"

var (
	ErrAppPasswordAccount = errors.New("auth: app-password login needs the full account address, e.g. user@example.com")
	ErrAppPasswordEmpty   = errors.New("auth: app password is empty; create one at " + AppPasswordURL)
)

// NewAppPasswordCredential builds a credential for the protocol surfaces that
// accept a Yandex app password (IMAP, SMTP, CalDAV). It has no expiry and no
// scopes: Yandex binds the password to a service type at creation time.
func NewAppPasswordCredential(account, password string) (*Credential, error) {
	account = strings.TrimSpace(account)
	if !strings.Contains(account, "@") {
		return nil, ErrAppPasswordAccount
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return nil, ErrAppPasswordEmpty
	}
	return &Credential{
		AppPassword: password,
		Account:     account,
		ObtainedVia: GrantAppPassword,
	}, nil
}
