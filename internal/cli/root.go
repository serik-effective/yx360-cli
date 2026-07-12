package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

var (
	jsonOutput        bool
	insecureFileStore bool
	dryRun            bool
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
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print what would happen without executing; overrides --yes")

	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	root.AddCommand(newMailCmd())
	root.AddCommand(newCalendarCmd())
	root.AddCommand(newTelemostCmd())
	root.AddCommand(newFormsCmd())
	root.AddCommand(newDiskCmd())
	root.AddCommand(newMCPCmd())
	return root
}

func selectStore() (tokenstore.TokenStore, error) {
	return selectStoreFor("")
}

func selectStoreFor(profile string) (tokenstore.TokenStore, error) {
	if insecureFileStore || os.Getenv("YX360_INSECURE_FILE_STORE") == "1" {
		return tokenstore.NewFileStoreFor(profile)
	}
	return tokenstore.NewKeyringStoreFor(profile), nil
}
