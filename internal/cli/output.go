package cli

import (
	"encoding/json"
	"fmt"

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

func isDryRun() bool { return dryRun }

func emitDryRun(cmd *cobra.Command, msg string) error {
	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"dry_run": "true", "would": msg})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] %s\n", msg)
	return nil
}
