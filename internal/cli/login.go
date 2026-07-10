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
	diskProfile             = "disk"
)

func newLoginCmd() *cobra.Command {
	var (
		noBrowser      bool
		device         bool
		mailScope      bool
		mailSendScope  bool
		calendarScope  bool
		telemostScope  bool
		formsScope     bool
		diskScope      bool
		manual         bool
		manualBegin    bool
		manualComplete bool
		manualCode     string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Yandex 360 via OAuth",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if manual {
				if device {
					return errors.New("--manual cannot be combined with --device")
				}
				if manualBegin == manualComplete {
					return errors.New("--manual requires exactly one of --begin or --complete")
				}
				if manualBegin {
					profile, cfg, scopes, err := resolveManualTarget(mailScope, mailSendScope, calendarScope, telemostScope, formsScope, diskScope)
					if err != nil {
						return err
					}
					mp := auth.NewManualProvider(cfg)
					authURL, err := mp.Begin(cmd.Context(), profile, auth.AuthOptions{Scopes: scopes})
					if err != nil {
						return err
					}
					human := "Open this URL, approve access, then run `yx360 login --manual --complete --code <code>`:\n" + authURL
					return emit(cmd, human, manualBeginPayload{Status: "manual-begin", AuthURL: authURL})
				}
				if manualCode == "" {
					return errors.New("--manual --complete requires --code")
				}
				profile, err := auth.LoadPendingProfile()
				if err != nil {
					return err
				}
				mp := auth.NewManualProvider(config.Default())
				cred, err := mp.Complete(cmd.Context(), manualCode)
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
			}
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
			if diskScope {
				selectedApps++
			}
			if selectedApps > 1 {
				return errors.New("mail, calendar/telemost, forms, and disk scopes use different Yandex OAuth apps; run separate login commands")
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
			if diskScope {
				profile = diskProfile
				cfg.ClientID = config.DiskClientID()
				if cfg.ClientID == "" {
					return errors.New("no Disk OAuth client_id: set YX360_DISK_CLIENT_ID")
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
			if diskScope {
				scopes = append(scopes, config.DiskReadScope, config.DiskWriteScope)
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
	cmd.Flags().BoolVar(&diskScope, "disk", false, "request Disk read and write access")
	cmd.Flags().BoolVar(&manual, "manual", false, "headless two-step login via manual code paste")
	cmd.Flags().BoolVar(&manualBegin, "begin", false, "start manual login and print the auth URL")
	cmd.Flags().BoolVar(&manualComplete, "complete", false, "finish manual login with the pasted code")
	cmd.Flags().StringVar(&manualCode, "code", "", "authorization code or full redirect URL for --manual --complete")
	return cmd
}

func resolveManualTarget(mailScope, mailSendScope, calendarScope, telemostScope, formsScope, diskScope bool) (string, config.OAuth, []string, error) {
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
	if diskScope {
		selectedApps++
	}
	if selectedApps > 1 {
		return "", cfg, nil, errors.New("mail, calendar/telemost, forms, and disk scopes use different Yandex OAuth apps; run separate login commands")
	}
	profile := ""
	if mailScope || mailSendScope {
		profile = mailProfile
	}
	if calendarScope || telemostScope {
		profile = calendarTelemostProfile
		cfg.ClientID = config.CalendarClientID()
		if cfg.ClientID == "" {
			return "", cfg, nil, errors.New("no Calendar/Telemost OAuth client_id: set YX360_CALENDAR_CLIENT_ID")
		}
	}
	if formsScope {
		profile = formsProfile
		cfg.ClientID = config.FormsClientID()
		if cfg.ClientID == "" {
			return "", cfg, nil, errors.New("no Forms OAuth client_id: set YX360_FORMS_CLIENT_ID")
		}
	}
	if diskScope {
		profile = diskProfile
		cfg.ClientID = config.DiskClientID()
		if cfg.ClientID == "" {
			return "", cfg, nil, errors.New("no Disk OAuth client_id: set YX360_DISK_CLIENT_ID")
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
	if diskScope {
		scopes = append(scopes, config.DiskReadScope, config.DiskWriteScope)
	}
	return profile, cfg, scopes, nil
}

type manualBeginPayload struct {
	Status  string `json:"status"`
	AuthURL string `json:"auth_url"`
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
