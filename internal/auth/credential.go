package auth

import "time"

type GrantKind string

const (
	GrantLoopback GrantKind = "loopback"
	GrantDevice   GrantKind = "device"
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
