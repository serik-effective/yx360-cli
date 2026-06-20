package auth

import (
	"context"
	"fmt"
	"io"

	"github.com/effective-dev-os/yx360-cli/internal/config"
)

type DeviceProvider struct {
	cfg    config.OAuth
	prompt io.Writer
}

func NewDeviceProvider(cfg config.OAuth, prompt io.Writer) *DeviceProvider {
	return &DeviceProvider{cfg: cfg, prompt: prompt}
}

func (p *DeviceProvider) Authenticate(ctx context.Context, opts AuthOptions) (*Credential, error) {
	if err := missingClientID(p.cfg); err != nil {
		return nil, err
	}

	conf := oauthConfig(p.cfg, opts)
	da, err := conf.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}

	fmt.Fprintf(p.prompt, "Open %s and enter code: %s\n", da.VerificationURI, da.UserCode)

	tok, err := conf.DeviceAccessToken(ctx, da)
	if err != nil {
		return nil, fmt.Errorf("device token exchange failed: %w", err)
	}

	cred := credentialFromToken(tok, GrantDevice, conf.Scopes)
	populateAccount(ctx, cred)
	return cred, nil
}
