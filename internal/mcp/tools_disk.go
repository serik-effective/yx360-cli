package yx360mcp

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/effective-dev-os/yx360-cli/internal/disk"
)

type diskListInput struct {
	Path   string `json:"path"             jsonschema:"remote path to list, e.g. /"`
	Limit  int    `json:"limit,omitempty"  jsonschema:"max items to return (0 = server default)"`
	Offset int    `json:"offset,omitempty" jsonschema:"pagination offset"`
}

type diskGetInput struct {
	RemotePath string `json:"remote_path" jsonschema:"source path on Yandex Disk"`
	LocalPath  string `json:"local_path"  jsonschema:"destination directory on local filesystem"`
}

type diskPutInput struct {
	LocalPath  string `json:"local_path"          jsonschema:"source path on local filesystem"`
	RemotePath string `json:"remote_path"         jsonschema:"destination path on Yandex Disk"`
	Overwrite  bool   `json:"overwrite,omitempty" jsonschema:"overwrite if destination exists"`
	Confirmed  bool   `json:"confirmed"           jsonschema:"set true to execute; omit for dry-run preview"`
}

type diskShareInput struct {
	Path      string `json:"path"      jsonschema:"path on Yandex Disk to share publicly"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

type diskUnshareInput struct {
	Path      string `json:"path"      jsonschema:"path on Yandex Disk to stop sharing"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

type diskRmInput struct {
	Path      string `json:"path"                jsonschema:"path on Yandex Disk to delete"`
	Permanent bool   `json:"permanent,omitempty" jsonschema:"skip trash and delete permanently"`
	Confirmed bool   `json:"confirmed"           jsonschema:"set true to execute; omit for dry-run preview"`
}

type diskMkdirInput struct {
	Path      string `json:"path"      jsonschema:"directory path to create on Yandex Disk"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

func registerDiskTools(srv *sdkmcp.Server, svcFn func(context.Context) (*disk.Service, error)) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_list",
		Description: "List files and folders at a path on Yandex Disk.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskListInput) (*sdkmcp.CallToolResult, any, error) {
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		result, err := svc.List(ctx, in.Path, in.Limit, in.Offset)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(result)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_get",
		Description: "Download a file from Yandex Disk to a local directory.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskGetInput) (*sdkmcp.CallToolResult, any, error) {
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		savedPath, err := svc.Get(ctx, in.RemotePath, in.LocalPath)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(map[string]string{"remote_path": in.RemotePath, "saved_to": savedPath})
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_put",
		Description: "Upload a local file to Yandex Disk. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskPutInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult(fmt.Sprintf("would upload %s → %s", in.LocalPath, in.RemotePath))
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		if err := svc.Put(ctx, in.LocalPath, in.RemotePath, in.Overwrite); err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(map[string]string{"uploaded": in.LocalPath, "remote_path": in.RemotePath})
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_share",
		Description: "Create a public link for a file or folder on Yandex Disk. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskShareInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would create public link for " + in.Path)
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		link, err := svc.Share(ctx, in.Path)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(map[string]string{"path": in.Path, "public_url": link})
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_unshare",
		Description: "Revoke the public link for a file or folder on Yandex Disk. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskUnshareInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would revoke public link for " + in.Path)
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		if err := svc.Unshare(ctx, in.Path); err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(map[string]string{"unshared": in.Path})
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_rm",
		Description: "Delete a file or folder from Yandex Disk. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskRmInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult(fmt.Sprintf("would delete %s (permanent=%v)", in.Path, in.Permanent))
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		if err := svc.Remove(ctx, in.Path, in.Permanent); err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(map[string]string{"deleted": in.Path})
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "disk_mkdir",
		Description: "Create a directory on Yandex Disk. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in diskMkdirInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would create directory " + in.Path)
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		if err := svc.Mkdir(ctx, in.Path); err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(map[string]string{"created": in.Path})
	})
}
