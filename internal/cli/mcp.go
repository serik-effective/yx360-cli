package cli

import (
	"os"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	yx360mcp "github.com/effective-dev-os/yx360-cli/internal/mcp"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol integration",
	}
	cmd.AddCommand(newMCPServeCmd())
	return cmd
}

func newMCPServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start an MCP stdio server exposing yx360 tools",
		Long: `Start a JSON-RPC 2.0 MCP server on stdin/stdout.

Connect Claude Desktop or any MCP-compatible client to this process.
All cobra output is redirected to stderr to keep stdout clean for the
MCP protocol stream.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Keep stdout clean for JSON-RPC — redirect cobra output to stderr.
			cmd.Root().SetOut(os.Stderr)

			ctx := cmd.Context()
			srv := yx360mcp.NewServer(yx360mcp.Services{
				Disk:     diskService,
				Mail:     mailService,
				Calendar: calendarService,
				Telemost: telemostService,
				Forms:    formsService,
			})
			session, err := srv.Connect(ctx, &sdkmcp.StdioTransport{}, nil)
			if err != nil {
				return err
			}
			return session.Wait()
		},
	}
}
