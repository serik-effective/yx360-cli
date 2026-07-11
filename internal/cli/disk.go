package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/disk"
	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

func newDiskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disk",
		Short: "Manage files on Yandex Disk",
	}
	cmd.AddCommand(newDiskListCmd())
	cmd.AddCommand(newDiskGetCmd())
	cmd.AddCommand(newDiskPutCmd())
	cmd.AddCommand(newDiskShareCmd())
	cmd.AddCommand(newDiskUnshareCmd())
	cmd.AddCommand(newDiskRmCmd())
	cmd.AddCommand(newDiskMkdirCmd())
	return cmd
}

func newDiskListCmd() *cobra.Command {
	var (
		path   string
		limit  int
		offset int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List directory contents on Yandex Disk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.List(cmd.Context(), path, limit, offset)
			if err != nil {
				return friendlyDiskError(err)
			}
			return emit(cmd, humanDiskList(result), result)
		},
	}
	cmd.Flags().StringVar(&path, "path", "/", "remote path to list")
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum items to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	return cmd
}

func newDiskGetCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "get <remote-path>",
		Short: "Download a file from Yandex Disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if outDir == "" {
				outDir = "."
			}
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			outPath, err := svc.Get(cmd.Context(), args[0], outDir)
			if err != nil {
				return friendlyDiskError(err)
			}
			return emit(cmd, "Downloaded to "+outPath, map[string]string{"path": outPath})
		},
	}
	cmd.Flags().StringVar(&outDir, "out", ".", "local directory to save the file")
	return cmd
}

func newDiskPutCmd() *cobra.Command {
	var (
		remotePath string
		yes        bool
	)
	cmd := &cobra.Command{
		Use:   "put <local-file>",
		Short: "Upload a file to Yandex Disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if remotePath == "" {
				return errors.New("disk: --to is required")
			}
			if isDryRun() {
				return emitDryRun(cmd, fmt.Sprintf("would upload %s to disk:%s", args[0], remotePath))
			}
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			err = svc.Put(cmd.Context(), args[0], remotePath, false)
			if errors.Is(err, disk.ErrConflict) {
				if !yes {
					cmd.Println("disk put preview:")
					cmd.Printf("  Local:   %s\n  Remote:  %s\n  Warning: target exists and will be overwritten\n", args[0], remotePath)
					cmd.Println("Re-run with --yes to overwrite.")
					return nil
				}
				err = svc.Put(cmd.Context(), args[0], remotePath, true)
			}
			if err != nil {
				return friendlyDiskError(err)
			}
			return emit(cmd, fmt.Sprintf("Uploaded %s → %s", args[0], remotePath),
				map[string]string{"local": args[0], "remote": remotePath})
		},
	}
	cmd.Flags().StringVar(&remotePath, "to", "", "remote destination path")
	cmd.Flags().BoolVar(&yes, "yes", false, "overwrite existing file without confirmation")
	return cmd
}

func newDiskShareCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "share <remote-path>",
		Short: "Create a public link for a file or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isDryRun() {
				return emitDryRun(cmd, fmt.Sprintf("would make disk:%s publicly accessible", args[0]))
			}
			if !yes {
				cmd.Println("disk share preview:")
				cmd.Printf("  Path: %s\n  Note: this will create a publicly accessible URL\n", args[0])
				cmd.Println("Re-run with --yes to proceed.")
				return nil
			}
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			publicURL, err := svc.Share(cmd.Context(), args[0])
			if err != nil {
				return friendlyDiskError(err)
			}
			return emit(cmd, "Public link: "+publicURL,
				map[string]string{"path": args[0], "public_url": publicURL})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "create public link without confirmation")
	return cmd
}

func newDiskUnshareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unshare <remote-path>",
		Short: "Remove public access from a file or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isDryRun() {
				return emitDryRun(cmd, fmt.Sprintf("would revoke public access for disk:%s", args[0]))
			}
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			if err := svc.Unshare(cmd.Context(), args[0]); err != nil {
				return friendlyDiskError(err)
			}
			return emit(cmd, "Removed public access for "+args[0],
				map[string]string{"path": args[0], "status": "unpublished"})
		},
	}
	return cmd
}

func newDiskRmCmd() *cobra.Command {
	var (
		yes       bool
		permanent bool
	)
	cmd := &cobra.Command{
		Use:   "rm <remote-path>",
		Short: "Move a file or directory to Trash (use --permanent to delete immediately)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isDryRun() {
				action := "move to Trash"
				if permanent {
					action = "permanently delete"
				}
				return emitDryRun(cmd, fmt.Sprintf("would %s disk:%s", action, args[0]))
			}
			if !yes {
				action := "move to Trash (reversible)"
				if permanent {
					action = "permanently delete (irreversible)"
				}
				cmd.Println("disk rm preview:")
				cmd.Printf("  Path:   %s\n  Action: %s\n", args[0], action)
				cmd.Println("Re-run with --yes to proceed.")
				return nil
			}
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			if err := svc.Remove(cmd.Context(), args[0], permanent); err != nil {
				return friendlyDiskError(err)
			}
			status := "moved-to-trash"
			if permanent {
				status = "deleted"
			}
			return emit(cmd, fmt.Sprintf("Removed %s (%s)", args[0], status),
				map[string]string{"path": args[0], "status": status})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "proceed without confirmation")
	cmd.Flags().BoolVar(&permanent, "permanent", false, "permanently delete instead of moving to Trash")
	return cmd
}

func newDiskMkdirCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mkdir <remote-path>",
		Short: "Create a directory on Yandex Disk (parent must exist)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isDryRun() {
				return emitDryRun(cmd, fmt.Sprintf("would create directory disk:%s", args[0]))
			}
			svc, err := diskService(cmd)
			if err != nil {
				return err
			}
			if err := svc.Mkdir(cmd.Context(), args[0]); err != nil {
				return friendlyDiskError(err)
			}
			return emit(cmd, "Created directory "+args[0],
				map[string]string{"path": args[0], "status": "created"})
		},
	}
	return cmd
}

func diskService(cmd *cobra.Command) (*disk.Service, error) {
	if config.DiskClientID() == "" {
		return nil, errors.New("disk: no Disk OAuth client_id: set YX360_DISK_CLIENT_ID")
	}
	store, err := selectStoreFor(diskProfile)
	if err != nil {
		return nil, err
	}
	cred, err := store.Load(cmd.Context())
	if err != nil {
		if errors.Is(err, tokenstore.ErrNoCredential) {
			return nil, disk.ErrReauthRequired
		}
		return nil, err
	}
	return disk.NewService(config.DefaultDisk(), cred), nil
}

func friendlyDiskError(err error) error { return err }

func humanDiskList(result *disk.ResourceList) string {
	if result == nil || len(result.Items) == 0 {
		return "No items"
	}
	var b strings.Builder
	for _, item := range result.Items {
		sizeStr := ""
		if item.Size > 0 {
			sizeStr = fmt.Sprintf(" (%d bytes)", item.Size)
		}
		b.WriteString(fmt.Sprintf("[%s] %s%s\n", item.Type, item.Name, sizeStr))
	}
	return strings.TrimRight(b.String(), "\n")
}
