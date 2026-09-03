package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

func newLogoutCmd() *cobra.Command {
	var (
		appPassword   bool
		mailScope     bool
		calendarScope bool
	)
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored Yandex 360 credential",
		RunE: func(cmd *cobra.Command, _ []string) error {
			profiles := []string{""}
			if appPassword {
				var err error
				profiles, err = logoutAppPasswordProfiles(mailScope, calendarScope)
				if err != nil {
					return err
				}
			}
			for _, profile := range profiles {
				store, err := selectStoreFor(profile)
				if err != nil {
					return err
				}
				if err := store.Clear(cmd.Context()); err != nil && !errors.Is(err, tokenstore.ErrNoCredential) {
					return err
				}
			}
			payload := logoutPayload{Status: "logged-out"}
			if appPassword {
				payload.Profiles = profiles
			}
			return emit(cmd, "Logged out.", payload)
		},
	}
	cmd.Flags().BoolVar(&appPassword, "app-password", false, "clear stored app passwords instead of the OAuth credential")
	cmd.Flags().BoolVar(&mailScope, "mail", false, "with --app-password, clear the Mail app password only")
	cmd.Flags().BoolVar(&calendarScope, "calendar", false, "with --app-password, clear the Calendar app password only")
	return cmd
}

func logoutAppPasswordProfiles(mailScope, calendarScope bool) ([]string, error) {
	switch {
	case mailScope && calendarScope:
		return nil, errors.New("--mail and --calendar are mutually exclusive; omit both to clear every stored app password")
	case mailScope:
		return []string{mailAppPasswordProfile}, nil
	case calendarScope:
		return []string{calendarAppPasswordProfile}, nil
	default:
		return []string{mailAppPasswordProfile, calendarAppPasswordProfile}, nil
	}
}

type logoutPayload struct {
	Status   string   `json:"status"`
	Profiles []string `json:"profiles,omitempty"`
}
