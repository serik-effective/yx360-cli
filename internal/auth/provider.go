package auth

import (
	"context"
	"errors"
)

type AuthOptions struct {
	Scopes       []string
	PreferDevice bool
	NoBrowser    bool
	Port         int
}

type Provider interface {
	Authenticate(ctx context.Context, opts AuthOptions) (*Credential, error)
}

// Refresher is intentionally unimplemented in PR-1: Yandex's refresh-token docs
// list client_secret as required, which contradicts the public-client (no-secret)
// exchange. Until the secretless-refresh round-trip is empirically verified (B2),
// no Refresher exists — at expiry the user re-runs `yx360 login`. Declaring the
// seam here keeps the open question isolated behind a type assertion.
type Refresher interface {
	Refresh(ctx context.Context, cred *Credential) (*Credential, error)
}

var errRungUnavailable = errors.New("auth: rung unavailable")
