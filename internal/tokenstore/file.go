package tokenstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
)

type FileStore struct {
	path string
}

func NewFileStore() (*FileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{path: filepath.Join(dir, "yx360", "credential.json")}, nil
}

func (s *FileStore) Save(_ context.Context, cred *auth.Credential) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	blob, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, blob, 0o600)
}

func (s *FileStore) Load(_ context.Context) (*auth.Credential, error) {
	blob, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoCredential
		}
		return nil, err
	}
	var cred auth.Credential
	if err := json.Unmarshal(blob, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func (s *FileStore) Clear(_ context.Context) error {
	if err := os.Remove(s.path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNoCredential
		}
		return err
	}
	return nil
}
