package auth

import (
	"strings"
	"time"
)

type GrantKind string

const (
	GrantLoopback GrantKind = "loopback"
	GrantDevice   GrantKind = "device"
	GrantManual   GrantKind = "manual"
)

const expirySkew = 60 * time.Second

type Credential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope"`
	Account      string    `json:"account"`
	ObtainedVia  GrantKind `json:"obtained_via"`
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
