package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mailmessage "github.com/emersion/go-message/mail"
)

const maxSendAttachmentBytes int64 = 25 << 20

var ErrSendReauthRequired = errors.New("mail: stored credential is missing, expired, or does not include mail:smtp; run yx360 login --mail --mail-send")

type SendOptions struct {
	From        string   `json:"from"`
	To          []string `json:"to"`
	Cc          []string `json:"cc,omitempty"`
	Bcc         []string `json:"bcc,omitempty"`
	Subject     string   `json:"subject"`
	Text        string   `json:"text,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type SendResult struct {
	Status     string   `json:"status"`
	From       string   `json:"from"`
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
}

func (s *Service) Send(ctx context.Context, opts SendOptions) (*SendResult, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(s.cfg.SendScope) {
		return nil, ErrSendReauthRequired
	}
	if opts.From == "" {
		opts.From = s.cred.Account
	}
	recipients := allRecipients(opts)
	if opts.From == "" {
		return nil, errors.New("mail: from address is required")
	}
	if len(recipients) == 0 {
		return nil, errors.New("mail: at least one recipient is required")
	}
	if opts.Subject == "" {
		return nil, errors.New("mail: subject is required")
	}
	if opts.Text == "" && len(opts.Attachments) == 0 {
		return nil, errors.New("mail: body or attachment is required")
	}
	raw, err := buildMessage(opts)
	if err != nil {
		return nil, err
	}
	if err := s.sendSMTP(ctx, opts.From, recipients, raw); err != nil {
		return nil, err
	}
	return &SendResult{
		Status:     "sent",
		From:       opts.From,
		Recipients: recipients,
		Subject:    opts.Subject,
	}, nil
}

func (s *Service) SendGeneratedUnsubscribe(ctx context.Context, mailtoURI string) (*SendResult, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(s.cfg.SendScope) {
		return nil, ErrSendReauthRequired
	}
	parsed, err := parseMailtoUnsubscribe(mailtoURI)
	if err != nil {
		return nil, err
	}
	opts := SendOptions{
		From:    s.cred.Account,
		To:      parsed.To,
		Subject: parsed.Subject,
		Text:    parsed.Body,
	}
	if opts.From == "" {
		return nil, errors.New("mail: from address is required")
	}
	if len(opts.To) == 0 {
		return nil, errors.New("mail: mailto unsubscribe has no recipient")
	}
	if opts.Subject == "" {
		opts.Subject = "Unsubscribe"
	}
	if opts.Text == "" {
		opts.Text = "Unsubscribe"
	}
	raw, err := buildMessage(opts)
	if err != nil {
		return nil, err
	}
	recipients := allRecipients(opts)
	if err := s.sendSMTP(ctx, opts.From, recipients, raw); err != nil {
		return nil, err
	}
	return &SendResult{
		Status:     "sent",
		From:       opts.From,
		Recipients: recipients,
		Subject:    opts.Subject,
	}, nil
}

type mailtoUnsubscribe struct {
	To      []string
	Subject string
	Body    string
}

func parseMailtoUnsubscribe(raw string) (*mailtoUnsubscribe, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(parsed.Scheme) != "mailto" {
		return nil, fmt.Errorf("mail: unsubscribe URI is not mailto: %s", raw)
	}
	recipients := parsed.Opaque
	if recipients == "" {
		recipients = strings.TrimPrefix(parsed.Path, "/")
	}
	to, err := url.PathUnescape(recipients)
	if err != nil {
		return nil, err
	}
	values := parsed.Query()
	out := &mailtoUnsubscribe{
		Subject: values.Get("subject"),
		Body:    values.Get("body"),
	}
	for _, recipient := range strings.Split(to, ",") {
		recipient = strings.TrimSpace(recipient)
		if recipient != "" {
			out.To = append(out.To, recipient)
		}
	}
	return out, nil
}

func buildMessage(opts SendOptions) ([]byte, error) {
	from, err := parseAddressList([]string{opts.From})
	if err != nil {
		return nil, err
	}
	to, err := parseAddressList(opts.To)
	if err != nil {
		return nil, err
	}
	cc, err := parseAddressList(opts.Cc)
	if err != nil {
		return nil, err
	}
	if _, err := parseAddressList(opts.Bcc); err != nil {
		return nil, err
	}

	var header mailmessage.Header
	header.SetDate(time.Now())
	header.SetAddressList("From", from)
	header.SetAddressList("To", to)
	header.SetAddressList("Cc", cc)
	header.SetSubject(opts.Subject)
	if err := header.GenerateMessageID(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if len(opts.Attachments) == 0 {
		part, err := mailmessage.CreateSingleInlineWriter(&buf, header)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(part, opts.Text); err != nil {
			return nil, err
		}
		if err := part.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	writer, err := mailmessage.CreateWriter(&buf, header)
	if err != nil {
		return nil, err
	}
	inline := mailmessage.InlineHeader{}
	inline.Set("Content-Type", "text/plain; charset=utf-8")
	body, err := writer.CreateSingleInline(inline)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(body, opts.Text); err != nil {
		return nil, err
	}
	if err := body.Close(); err != nil {
		return nil, err
	}
	for _, path := range opts.Attachments {
		if err := addAttachment(writer, path); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addAttachment(writer *mailmessage.Writer, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > maxSendAttachmentBytes {
		return fmt.Errorf("mail: attachment %s exceeds %d bytes", path, maxSendAttachmentBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	attachment := mailmessage.AttachmentHeader{}
	attachment.SetFilename(safeFilename(filepath.Base(path)))
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	attachment.Set("Content-Type", contentType)
	part, err := writer.CreateAttachment(attachment)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, io.LimitReader(file, maxSendAttachmentBytes)); err != nil {
		return err
	}
	return part.Close()
}

func (s *Service) sendSMTP(ctx context.Context, from string, recipients []string, raw []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	address := net.JoinHostPort(s.cfg.SMTPHost, strconv.Itoa(s.cfg.SMTPPort))
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp4", address, &tls.Config{
		ServerName: s.cfg.SMTPHost,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if err := client.Auth(newSMTPOAuth2Auth("XOAUTH2", s.cred.Account, s.cred.AccessToken)); err != nil {
		if fallbackErr := client.Auth(newSMTPOAuth2Auth("OAUTHBEARER", s.cred.Account, s.cred.AccessToken)); fallbackErr != nil {
			return fmt.Errorf("mail: SMTP OAuth authentication failed: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

type smtpOAuth2Auth struct {
	mechanism string
	username  string
	token     string
}

func newSMTPOAuth2Auth(mechanism, username, token string) smtp.Auth {
	return &smtpOAuth2Auth{mechanism: mechanism, username: username, token: token}
}

func (a *smtpOAuth2Auth) Start(*smtp.ServerInfo) (string, []byte, error) {
	var response string
	if a.mechanism == "OAUTHBEARER" {
		response = "n,a=" + a.username + ",\x01auth=Bearer " + a.token + "\x01\x01"
	} else {
		response = "user=" + a.username + "\x01auth=Bearer " + a.token + "\x01\x01"
	}
	return a.mechanism, []byte(response), nil
}

func (a *smtpOAuth2Auth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("mail: unexpected SMTP OAuth challenge")
	}
	return nil, nil
}

func parseAddressList(values []string) ([]*mailmessage.Address, error) {
	out := make([]*mailmessage.Address, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := mailmessage.ParseAddressList(value)
		if err != nil {
			return nil, fmt.Errorf("mail: invalid address %q: %w", value, err)
		}
		out = append(out, parsed...)
	}
	return out, nil
}

func allRecipients(opts SendOptions) []string {
	values := append([]string{}, opts.To...)
	values = append(values, opts.Cc...)
	values = append(values, opts.Bcc...)
	out := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := mailmessage.ParseAddressList(value)
		if err != nil {
			continue
		}
		for _, addr := range parsed {
			out = append(out, addr.Address)
		}
	}
	return out
}
