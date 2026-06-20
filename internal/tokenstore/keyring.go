package tokenstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
)

const (
	keyringService = "yx360"
	keyringUser    = "credential"
)

type KeyringStore struct {
	user string
}

func NewKeyringStore() *KeyringStore {
	return NewKeyringStoreFor("")
}

func NewKeyringStoreFor(profile string) *KeyringStore {
	user := keyringUser
	if profile != "" {
		user = keyringUser + ":" + profile
	}
	return &KeyringStore{user: user}
}

func (s *KeyringStore) Save(_ context.Context, cred *auth.Credential) error {
	blob, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, s.user, string(blob)); err != nil {
		return wrapKeyringErr(err)
	}
	return nil
}

func (s *KeyringStore) Load(_ context.Context) (*auth.Credential, error) {
	blob, err := keyring.Get(keyringService, s.user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrNoCredential
		}
		return nil, wrapKeyringErr(err)
	}
	var cred auth.Credential
	if err := json.Unmarshal([]byte(blob), &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func (s *KeyringStore) Clear(_ context.Context) error {
	if err := keyring.Delete(keyringService, s.user); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNoCredential
		}
		return wrapKeyringErr(err)
	}
	return nil
}

func wrapKeyringErr(err error) error {
	return fmt.Errorf("OS keychain unavailable (%w): on headless/CI hosts re-run with --insecure-file-store", err)
}
