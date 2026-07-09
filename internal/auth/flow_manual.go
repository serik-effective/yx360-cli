package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/effective-dev-os/yx360-cli/internal/config"
)

const manualLoginTTL = 10 * time.Minute
const maxManualInputLen = 4096

type ManualProvider struct {
	cfg config.OAuth
}

func NewManualProvider(cfg config.OAuth) *ManualProvider {
	return &ManualProvider{cfg: cfg}
}

type pendingManualLogin struct {
	Verifier  string    `json:"verifier"`
	State     string    `json:"state"`
	Profile   string    `json:"profile"`
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
}

func manualLoginPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "yx360", "manual-login.json"), nil
}

func writePendingManualLogin(p pendingManualLogin) error {
	path, err := manualLoginPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	blob, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0o600)
}

func loadPendingManualLogin() (pendingManualLogin, error) {
	path, err := manualLoginPath()
	if err != nil {
		return pendingManualLogin{}, err
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		return pendingManualLogin{}, err
	}
	var p pendingManualLogin
	if err := json.Unmarshal(blob, &p); err != nil {
		return pendingManualLogin{}, err
	}
	return p, nil
}

func deletePendingManualLogin() {
	path, err := manualLoginPath()
	if err != nil {
		return
	}
	os.Remove(path)
}

func LoadPendingProfile() (string, error) {
	pending, err := loadPendingManualLogin()
	if err != nil {
		return "", errors.New("no pending manual login; run `yx360 login --manual --begin` first")
	}
	return pending.Profile, nil
}

func (p *ManualProvider) Begin(ctx context.Context, profile string, opts AuthOptions) (string, error) {
	if err := missingClientID(p.cfg); err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()
	state, err := randomState()
	if err != nil {
		return "", err
	}
	conf := oauthConfig(p.cfg, opts)
	conf.RedirectURL = config.VerificationCodeRedirectURI
	authURL := conf.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
	if err := writePendingManualLogin(pendingManualLogin{
		Verifier:  verifier,
		State:     state,
		Profile:   profile,
		ClientID:  p.cfg.ClientID,
		Scopes:    conf.Scopes,
		CreatedAt: time.Now(),
	}); err != nil {
		return "", err
	}
	return authURL, nil
}

func (p *ManualProvider) Complete(ctx context.Context, input string) (*Credential, error) {
	pending, err := loadPendingManualLogin()
	if err != nil {
		return nil, errors.New("no pending manual login; run `yx360 login --manual --begin` first")
	}
	if time.Since(pending.CreatedAt) > manualLoginTTL {
		deletePendingManualLogin()
		return nil, errors.New("manual login expired (>10m); re-run `yx360 login --manual --begin`")
	}
	if len(input) > maxManualInputLen {
		return nil, errors.New("manual login input too long")
	}
	code, err := parseManualInput(input, pending.State)
	if err != nil {
		return nil, err
	}

	cfg := p.cfg
	cfg.ClientID = pending.ClientID
	conf := oauthConfig(cfg, AuthOptions{Scopes: pending.Scopes})
	conf.RedirectURL = config.VerificationCodeRedirectURI

	tok, err := exchangeCode(ctx, conf, code, pending.Verifier)
	deletePendingManualLogin()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			return nil, errors.New("authorization code expired or already used — re-run `yx360 login --manual --begin`")
		}
		return nil, err
	}
	cred := credentialFromToken(tok, GrantManual, conf.Scopes)
	populateAccount(ctx, cred)
	return cred, nil
}

func parseManualInput(input, expectedState string) (string, error) {
	if strings.Contains(input, "://") || strings.Contains(input, "code=") {
		u, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("could not parse manual login input as URL: %w", err)
		}
		q := u.Query()
		if oauthErr := q.Get("error"); oauthErr != "" {
			return "", fmt.Errorf("oauth authorize rejected: %s", oauthErr)
		}
		if gotState := q.Get("state"); gotState != "" {
			if subtle.ConstantTimeCompare([]byte(gotState), []byte(expectedState)) != 1 {
				return "", errors.New("oauth state mismatch")
			}
		}
		code := q.Get("code")
		if code == "" {
			return "", errors.New("no authorization code found in input")
		}
		return code, nil
	}
	code := strings.TrimSpace(input)
	if code == "" {
		return "", errors.New("no authorization code found in input")
	}
	return code, nil
}
