package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/effective-dev-os/yx360-cli/internal/config"
)

const accountInfoURL = "https://login.yandex.ru/info?format=json"

func oauthConfig(cfg config.OAuth, opts AuthOptions) *oauth2.Config {
	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = cfg.Scopes
	}
	return &oauth2.Config{
		ClientID:    cfg.ClientID,
		RedirectURL: cfg.RedirectURI,
		Scopes:      scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:       cfg.AuthURL,
			TokenURL:      cfg.TokenURL,
			DeviceAuthURL: cfg.DeviceAuthURL,
		},
	}
}

func credentialFromToken(tok *oauth2.Token, via GrantKind, scopes []string) *Credential {
	return &Credential{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Expiry:       tok.Expiry,
		Scope:        strings.Join(scopes, " "),
		ObtainedVia:  via,
	}
}

func populateAccount(ctx context.Context, cred *Credential) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpContext(ctx), http.MethodGet, accountInfoURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "OAuth "+cred.AccessToken)

	client := httpContext(ctx).Value(oauth2.HTTPClient).(*http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var info struct {
		Login        string `json:"login"`
		DefaultEmail string `json:"default_email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return
	}
	if info.DefaultEmail != "" {
		cred.Account = info.DefaultEmail
	} else {
		cred.Account = info.Login
	}
}

func missingClientID(cfg config.OAuth) error {
	if cfg.ClientID == "" {
		return fmt.Errorf("no OAuth client_id: register a Yandex OAuth app and set YX360_CLIENT_ID (see B1 in the plan)")
	}
	return nil
}
