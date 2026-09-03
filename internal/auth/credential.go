package auth

import (
	"strings"
	"time"
)

type GrantKind string

const (
	GrantLoopback    GrantKind = "loopback"
	GrantDevice      GrantKind = "device"
	GrantManual      GrantKind = "manual"
	GrantAppPassword GrantKind = "app-password"
)

const expirySkew = 60 * time.Second

type Credential struct {
	AccessToken  string    `json:"access_token,omitempty"`
	AppPassword  string    `json:"app_password,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope"`
	Account      string    `json:"account"`
	ObtainedVia  GrantKind `json:"obtained_via"`
}

func (c *Credential) IsAppPassword() bool {
	return c != nil && c.ObtainedVia == GrantAppPassword && c.AppPassword != "" && c.Account != ""
}

// UsableFor reports whether the credential can authenticate a surface. Yandex
// app passwords carry no scope list and no expiry — the service type is chosen
// when the password is created at id.yandex.ru/security/app-passwords — so
// possession is the only check the CLI can make; the server rejects a password
// issued for the wrong service at connect time.
func (c *Credential) UsableFor(scopes ...string) bool {
	if c.IsAppPassword() {
		return true
	}
	return c.Valid() && c.HasScopes(scopes...)
}

func (c *Credential) Valid() bool {
	if c == nil || c.AccessToken == "" {
		return false
	}
	if c.Expiry.IsZero() {
		return true
	}
	return time.Now().Add(expirySkew).Before(c.Expiry)
}

func (c *Credential) Scopes() []string {
	if c == nil || c.Scope == "" {
		return nil
	}
	return strings.Fields(c.Scope)
}

func (c *Credential) HasScopes(required ...string) bool {
	granted := make(map[string]bool, len(c.Scopes()))
	for _, scope := range c.Scopes() {
		granted[scope] = true
	}
	for _, scope := range required {
		if !granted[scope] {
			return false
		}
	}
	return true
}
