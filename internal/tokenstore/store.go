package tokenstore

import (
	"context"
	"errors"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
)

var ErrNoCredential = errors.New("tokenstore: no credential stored")

type TokenStore interface {
	Save(ctx context.Context, cred *auth.Credential) error
	Load(ctx context.Context) (*auth.Credential, error)
	Clear(ctx context.Context) error
}
