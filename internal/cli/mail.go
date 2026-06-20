package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/mail"
	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

func newMailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Read Yandex 360 Mail via IMAP",
	}
	cmd.AddCommand(newMailListCmd())
	cmd.AddCommand(newMailSearchCmd())
	cmd.AddCommand(newMailReadCmd())
	cmd.AddCommand(newMailAttachmentCmd())
	cmd.AddCommand(newMailSendCmd())
	return cmd
}

func newMailListCmd() *cobra.Command {
	var q mail.Query
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List mailbox messages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := mailService(cmd)
			if err != nil {
				return err
			}
			msgs, err := svc.List(cmd.Context(), q)
			if err != nil {
				return friendlyMailError(err)
			}
			return emit(cmd, humanMessages(msgs), msgs)
		},
	}
	addMailQueryFlags(cmd, &q, false)
	return cmd
}

func newMailSearchCmd() *cobra.Command {
	var (
		q     mail.Query
		since string
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search mailbox messages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if since != "" {
				parsed, err := time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("mail: --since must use YYYY-MM-DD: %w", err)
				}
				q.Since = parsed
			}
			svc, err := mailService(cmd)
			if err != nil {
				return err
			}
			msgs, err := svc.Search(cmd.Context(), q)
			if err != nil {
				return friendlyMailError(err)
			}
			return emit(cmd, humanMessages(msgs), msgs)
		},
	}
	addMailQueryFlags(cmd, &q, true)
	cmd.Flags().StringVar(&since, "since", "", "only messages since YYYY-MM-DD")
	return cmd
}

func newMailReadCmd() *cobra.Command {
	var folder string
	cmd := &cobra.Command{
		Use:   "read <uid>",
		Short: "Read one message by IMAP UID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := parseUID(args[0])
			if err != nil {
				return err
			}
			svc, err := mailService(cmd)
			if err != nil {
				return err
			}
			msg, err := svc.Read(cmd.Context(), folder, uid)
			if err != nil {
				return friendlyMailError(err)
			}
			return emit(cmd, humanMessage(*msg), msg)
		},
	}
	cmd.Flags().StringVar(&folder, "folder", "INBOX", "mail folder")
	return cmd
}

func newMailAttachmentCmd() *cobra.Command {
	var (
		folder string
		outDir string
	)
	cmd := &cobra.Command{
		Use:   "attachment <uid> <attachment-id>",
		Short: "Download one attachment by message UID and attachment ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := parseUID(args[0])
			if err != nil {
				return err
			}
			svc, err := mailService(cmd)
			if err != nil {
				return err
			}
			path, err := svc.DownloadAttachment(cmd.Context(), folder, uid, args[1], outDir)
			if err != nil {
				return friendlyMailError(err)
			}
			payload := struct {
				Path string `json:"path"`
			}{Path: path}
			return emit(cmd, "Downloaded "+path, payload)
		},
	}
	cmd.Flags().StringVar(&folder, "folder", "INBOX", "mail folder")
	cmd.Flags().StringVar(&outDir, "out", "", "directory for downloaded attachment")
	return cmd
}

func newMailSendCmd() *cobra.Command {
	var (
		opts     mail.SendOptions
		bodyFile string
		yes      bool
	)
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a Mail message via SMTP",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if bodyFile != "" {
				body, err := os.ReadFile(bodyFile)
				if err != nil {
					return err
				}
				opts.Text = string(body)
			}
			if !yes {
				if err := confirmSend(cmd, opts); err != nil {
					return err
				}
			}
			svc, err := mailService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Send(cmd.Context(), opts)
			if err != nil {
				return friendlyMailError(err)
			}
			return emit(cmd, "Sent mail to "+strings.Join(result.Recipients, ","), result)
		},
	}
	cmd.Flags().StringVar(&opts.From, "from", "", "sender address; defaults to the logged-in account")
	cmd.Flags().StringArrayVar(&opts.To, "to", nil, "recipient address; repeatable")
	cmd.Flags().StringArrayVar(&opts.Cc, "cc", nil, "cc recipient address; repeatable")
	cmd.Flags().StringArrayVar(&opts.Bcc, "bcc", nil, "bcc recipient address; repeatable")
	cmd.Flags().StringVar(&opts.Subject, "subject", "", "message subject")
	cmd.Flags().StringVar(&opts.Text, "body", "", "plain-text message body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "plain-text message body file")
	cmd.Flags().StringArrayVar(&opts.Attachments, "attach", nil, "attachment path; repeatable")
	cmd.Flags().BoolVar(&yes, "yes", false, "send without interactive confirmation")
	return cmd
}

func addMailQueryFlags(cmd *cobra.Command, q *mail.Query, search bool) {
	cmd.Flags().StringVar(&q.Folder, "folder", "INBOX", "mail folder")
	cmd.Flags().Uint32Var(&q.Limit, "limit", 20, "maximum messages to return")
	if search {
		cmd.Flags().StringVar(&q.From, "from", "", "match sender")
		cmd.Flags().StringVar(&q.Subject, "subject", "", "match subject")
		cmd.Flags().StringVar(&q.Text, "text", "", "match message text")
	}
}

func mailService(cmd *cobra.Command) (*mail.Service, error) {
	store, err := selectStore()
	if err != nil {
		return nil, err
	}
	cred, err := store.Load(cmd.Context())
	if err != nil {
		if errors.Is(err, tokenstore.ErrNoCredential) {
			return nil, mail.ErrReauthRequired
		}
		return nil, err
	}
	return mail.NewService(config.DefaultMail(), cred), nil
}

func friendlyMailError(err error) error {
	if errors.Is(err, mail.ErrReauthRequired) || errors.Is(err, mail.ErrMailboxSetup) || errors.Is(err, mail.ErrSendReauthRequired) {
		return err
	}
	return err
}

func parseUID(value string) (uint32, error) {
	uid, err := strconv.ParseUint(value, 10, 32)
	if err != nil || uid == 0 {
		return 0, fmt.Errorf("mail: uid must be a positive integer")
	}
	return uint32(uid), nil
}

func humanMessages(msgs []mail.Message) string {
	if len(msgs) == 0 {
		return "No messages"
	}
	var b strings.Builder
	for _, msg := range msgs {
		b.WriteString(humanMessage(msg))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func humanMessage(msg mail.Message) string {
	subject := msg.Subject
	if subject == "" {
		subject = "(no subject)"
	}
	line := fmt.Sprintf("%d %s", msg.UID, subject)
	if len(msg.From) > 0 {
		line += " from " + strings.Join(msg.From, ",")
	}
	if msg.Date != "" {
		line += " at " + msg.Date
	}
	if len(msg.Attachments) > 0 {
		line += fmt.Sprintf(" attachments=%d", len(msg.Attachments))
	}
	return line
}

func confirmSend(cmd *cobra.Command, opts mail.SendOptions) error {
	cmd.Println("Mail send preview:")
	cmd.Println("  From: " + valueOrDefault(opts.From, "(logged-in account)"))
	cmd.Println("  To: " + strings.Join(opts.To, ","))
	if len(opts.Cc) > 0 {
		cmd.Println("  Cc: " + strings.Join(opts.Cc, ","))
	}
	if len(opts.Bcc) > 0 {
		cmd.Println("  Bcc: " + strings.Join(opts.Bcc, ","))
	}
	cmd.Println("  Subject: " + opts.Subject)
	cmd.Println(fmt.Sprintf("  Body bytes: %d", len(opts.Text)))
	if len(opts.Attachments) > 0 {
		cmd.Println("  Attachments: " + strings.Join(opts.Attachments, ","))
	}
	cmd.Print("Send? [y/N] ")
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return errors.New("mail: send cancelled")
	}
	return nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
