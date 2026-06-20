package tokenstore

import (
	"context"
	"errors"
	"testing"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
)

type memStore struct {
	cred *auth.Credential
}

func (m *memStore) Save(_ context.Context, cred *auth.Credential) error {
	m.cred = cred
	return nil
}

func (m *memStore) Load(_ context.Context) (*auth.Credential, error) {
	if m.cred == nil {
		return nil, ErrNoCredential
	}
	return m.cred, nil
}

func (m *memStore) Clear(_ context.Context) error {
	if m.cred == nil {
		return ErrNoCredential
	}
	m.cred = nil
	return nil
}

var _ TokenStore = (*memStore)(nil)

func TestMemStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}

	if _, err := store.Load(ctx); !errors.Is(err, ErrNoCredential) {
		t.Fatalf("empty Load = %v, want ErrNoCredential", err)
	}

	want := &auth.Credential{AccessToken: "tok", Account: "user@yandex.ru"}
	if err := store.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.Account != want.Account {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if err := store.Clear(ctx); !errors.Is(err, ErrNoCredential) {
		t.Fatalf("second Clear = %v, want ErrNoCredential", err)
	}
}
