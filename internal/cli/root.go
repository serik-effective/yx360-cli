package cli

import (
	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

var (
	jsonOutput        bool
	insecureFileStore bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "yx360",
		Short:         "Yandex 360 command-line client",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON")
	root.PersistentFlags().BoolVar(&insecureFileStore, "insecure-file-store", false, "store the credential in a plaintext file instead of the OS keychain (headless/CI)")

	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	return root
}

func selectStore() (tokenstore.TokenStore, error) {
	if insecureFileStore {
		return tokenstore.NewFileStore()
	}
	return tokenstore.NewKeyringStore(), nil
}
