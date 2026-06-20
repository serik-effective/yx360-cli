package cli

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

const (
	mailProfile             = "mail"
	calendarTelemostProfile = "calendar-telemost"
	formsProfile            = "forms"
)

func newLoginCmd() *cobra.Command {
	var (
		noBrowser     bool
		device        bool
		mailScope     bool
		mailSendScope bool
		calendarScope bool
		telemostScope bool
		formsScope    bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Yandex 360 via OAuth",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := config.Default()
			selectedApps := 0
			if mailScope || mailSendScope {
				selectedApps++
			}
			if calendarScope || telemostScope {
				selectedApps++
			}
			if formsScope {
				selectedApps++
			}
			if selectedApps > 1 {
				return errors.New("mail, calendar/telemost, and forms scopes use different Yandex OAuth apps; run separate login commands")
			}
			profile := ""
			if mailScope || mailSendScope {
				profile = mailProfile
			}
			if calendarScope || telemostScope {
				profile = calendarTelemostProfile
				cfg.ClientID = config.CalendarClientID()
				if cfg.ClientID == "" {
					return errors.New("no Calendar/Telemost OAuth client_id: set YX360_CALENDAR_CLIENT_ID")
				}
			}
			if formsScope {
				profile = formsProfile
				cfg.ClientID = config.FormsClientID()
				if cfg.ClientID == "" {
					return errors.New("no Forms OAuth client_id: set YX360_FORMS_CLIENT_ID")
				}
			}
			scopes := append([]string(nil), cfg.Scopes...)
			if mailScope {
				scopes = append(scopes, config.MailReadScope)
			}
			if mailSendScope {
				scopes = append(scopes, config.MailSendScope)
			}
			if calendarScope {
				scopes = append(scopes, config.CalendarScope)
			}
			if telemostScope {
				scopes = append(scopes, config.TelemostCreateScope)
			}
			if formsScope {
				scopes = append(scopes, config.FormsReadScope, config.FormsWriteScope)
			}

			loopback := auth.NewLoopbackProvider(cfg)
			deviceProvider := auth.NewDeviceProvider(cfg, cmd.ErrOrStderr())
			ladder := auth.NewLadder(loopback, deviceProvider)

			cred, err := ladder.Authenticate(cmd.Context(), auth.AuthOptions{
				Scopes:       scopes,
				PreferDevice: device,
				NoBrowser:    noBrowser,
				Port:         cfg.LoopbackPort,
			})
			if err != nil {
				return err
			}

			store, err := selectStoreFor(profile)
			if err != nil {
				return err
			}
			if err := store.Save(cmd.Context(), cred); err != nil {
				return err
			}

			payload := loginPayload{
				Status:  "logged-in",
				Account: cred.Account,
				Profile: profile,
				Scopes:  cred.Scopes(),
			}
			if !cred.Expiry.IsZero() {
				payload.Expiry = cred.Expiry.Format(time.RFC3339)
			}
			return emit(cmd, humanLogin(payload), payload)
		},
	}
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "skip the loopback browser flow, use device flow")
	cmd.Flags().BoolVar(&device, "device", false, "force the device-authorization flow")
	cmd.Flags().BoolVar(&mailScope, "mail", false, "request read-only Mail IMAP access")
	cmd.Flags().BoolVar(&mailSendScope, "mail-send", false, "request Mail SMTP send access")
	cmd.Flags().BoolVar(&calendarScope, "calendar", false, "request Calendar access")
	cmd.Flags().BoolVar(&telemostScope, "telemost", false, "request Telemost conference creation access")
	cmd.Flags().BoolVar(&formsScope, "forms", false, "request Forms read and write access")
	return cmd
}

type loginPayload struct {
	Status  string   `json:"status"`
	Account string   `json:"account"`
	Profile string   `json:"profile,omitempty"`
	Scopes  []string `json:"scopes"`
	Expiry  string   `json:"expiry,omitempty"`
}

func humanLogin(p loginPayload) string {
	account := p.Account
	if account == "" {
		account = "(account label unavailable)"
	}
	msg := "Signed in as " + account
	if p.Profile != "" {
		msg += " (" + p.Profile + ")"
	}
	if p.Expiry != "" {
		msg += "; token expires " + p.Expiry
	}
	return msg
}
