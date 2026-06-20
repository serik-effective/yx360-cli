package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

func newLoginCmd() *cobra.Command {
	var (
		noBrowser bool
		device    bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Yandex 360 via OAuth",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := config.Default()

			loopback := auth.NewLoopbackProvider(cfg)
			deviceProvider := auth.NewDeviceProvider(cfg, cmd.ErrOrStderr())
			ladder := auth.NewLadder(loopback, deviceProvider)

			cred, err := ladder.Authenticate(cmd.Context(), auth.AuthOptions{
				Scopes:       cfg.Scopes,
				PreferDevice: device,
				NoBrowser:    noBrowser,
				Port:         cfg.LoopbackPort,
			})
			if err != nil {
				return err
			}

			store, err := selectStore()
			if err != nil {
				return err
			}
			if err := store.Save(cmd.Context(), cred); err != nil {
				return err
			}

			payload := loginPayload{
				Status:  "logged-in",
				Account: cred.Account,
				Scopes:  cfg.Scopes,
			}
			if !cred.Expiry.IsZero() {
				payload.Expiry = cred.Expiry.Format(time.RFC3339)
			}
			return emit(cmd, humanLogin(payload), payload)
		},
	}
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "skip the loopback browser flow, use device flow")
	cmd.Flags().BoolVar(&device, "device", false, "force the device-authorization flow")
	return cmd
}

type loginPayload struct {
	Status  string   `json:"status"`
	Account string   `json:"account"`
	Scopes  []string `json:"scopes"`
	Expiry  string   `json:"expiry,omitempty"`
}

func humanLogin(p loginPayload) string {
	account := p.Account
	if account == "" {
		account = "(account label unavailable)"
	}
	msg := "Signed in as " + account
	if p.Expiry != "" {
		msg += "; token expires " + p.Expiry
	}
	return msg
}
