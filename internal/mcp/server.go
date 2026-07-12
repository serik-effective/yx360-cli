// Package yx360mcp implements the MCP (Model Context Protocol) stdio server
// exposing Yandex 360 service operations as typed MCP tools.
package yx360mcp

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/effective-dev-os/yx360-cli/internal/calendar"
	"github.com/effective-dev-os/yx360-cli/internal/disk"
	"github.com/effective-dev-os/yx360-cli/internal/forms"
	"github.com/effective-dev-os/yx360-cli/internal/mail"
	"github.com/effective-dev-os/yx360-cli/internal/telemost"
)

const serverVersion = "0.1.0"

// Services holds per-request service factory functions. Each factory is called
// once per tool invocation so credentials are always fresh.
type Services struct {
	Disk     func(context.Context) (*disk.Service, error)
	Mail     func(context.Context) (*mail.Service, error)
	Calendar func(context.Context) (*calendar.Service, error)
	Telemost func(context.Context) (*telemost.Service, error)
	Forms    func(context.Context) (*forms.Service, error)
}

// NewServer creates and returns a configured MCP server with all available
// yx360 tools registered. Service factories that are nil are silently omitted.
func NewServer(svcs Services) *sdkmcp.Server {
	srv := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "yx360-mcp", Version: serverVersion},
		nil,
	)
	if svcs.Disk != nil {
		registerDiskTools(srv, svcs.Disk)
	}
	if svcs.Mail != nil {
		registerMailTools(srv, svcs.Mail)
	}
	if svcs.Calendar != nil {
		registerCalendarTools(srv, svcs.Calendar, svcs.Telemost)
	}
	if svcs.Forms != nil {
		registerFormsTools(srv, svcs.Forms)
	}
	return srv
}

// textResult serialises v as JSON and wraps it in an MCP text content result.
func textResult(v any) (*sdkmcp.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
	}, nil, nil
}

// dryRunResult returns a preview result without executing the operation.
func dryRunResult(msg string) (*sdkmcp.CallToolResult, any, error) {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: "[dry-run] " + msg}},
	}, nil, nil
}

// toolErr sanitises service errors by stripping OAuth tokens before they reach
// the MCP client (INVARIANT-§12).
func toolErr(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(tokenRe.ReplaceAllString(err.Error(), "[REDACTED]"))
}

var tokenRe = regexp.MustCompile(`(?i)(Bearer|OAuth|Token)[[:space:]]+[A-Za-z0-9._~+/\-]+=*`)
