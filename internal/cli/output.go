package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func emit(cmd *cobra.Command, human string, payload any) error {
	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	cmd.Println(human)
	return nil
}
