package yx360mcp

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/effective-dev-os/yx360-cli/internal/mail"
)

type mailListInput struct {
	Folder  string `json:"folder,omitempty"  jsonschema:"IMAP folder name (default: INBOX)"`
	Limit   uint32 `json:"limit,omitempty"   jsonschema:"max messages to return"`
	From    string `json:"from,omitempty"    jsonschema:"filter by sender address"`
	Subject string `json:"subject,omitempty" jsonschema:"filter by subject substring"`
}

type mailReadInput struct {
	Folder string `json:"folder" jsonschema:"IMAP folder name"`
	UID    uint32 `json:"uid"    jsonschema:"message UID from mail_list"`
}

type mailSendInput struct {
	From        string   `json:"from"                  jsonschema:"sender address"`
	To          []string `json:"to"                    jsonschema:"recipient addresses"`
	Cc          []string `json:"cc,omitempty"          jsonschema:"CC addresses"`
	Bcc         []string `json:"bcc,omitempty"         jsonschema:"BCC addresses"`
	Subject     string   `json:"subject"               jsonschema:"email subject"`
	Text        string   `json:"text,omitempty"        jsonschema:"plain-text body"`
	Attachments []string `json:"attachments,omitempty" jsonschema:"local file paths to attach"`
	Confirmed   bool     `json:"confirmed"             jsonschema:"set true to send; omit for dry-run preview"`
}

func registerMailTools(srv *sdkmcp.Server, svcFn func(context.Context) (*mail.Service, error)) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "mail_list",
		Description: "List recent messages in an IMAP folder.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in mailListInput) (*sdkmcp.CallToolResult, any, error) {
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		msgs, err := svc.List(ctx, mail.Query{
			Folder:  in.Folder,
			Limit:   in.Limit,
			From:    in.From,
			Subject: in.Subject,
		})
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(msgs)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "mail_read",
		Description: "Read the full content of a message by UID.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in mailReadInput) (*sdkmcp.CallToolResult, any, error) {
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		msg, err := svc.Read(ctx, in.Folder, in.UID)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(msg)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "mail_send",
		Description: "Send an email via SMTP. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in mailSendInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult(fmt.Sprintf("would send email from %s to %v subject=%q", in.From, in.To, in.Subject))
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		result, err := svc.Send(ctx, mail.SendOptions{
			From:        in.From,
			To:          in.To,
			Cc:          in.Cc,
			Bcc:         in.Bcc,
			Subject:     in.Subject,
			Text:        in.Text,
			Attachments: in.Attachments,
		})
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(result)
	})
}
