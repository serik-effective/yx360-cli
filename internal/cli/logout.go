package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored Yandex 360 credential",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := selectStore()
			if err != nil {
				return err
			}
			if err := store.Clear(cmd.Context()); err != nil && !errors.Is(err, tokenstore.ErrNoCredential) {
				return err
			}
			payload := logoutPayload{Status: "logged-out"}
			return emit(cmd, "Logged out.", payload)
		},
	}
}

type logoutPayload struct {
	Status string `json:"status"`
}
