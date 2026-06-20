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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	mailmessage "github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

const maxReadBytes = 10 << 20

var (
	ErrReauthRequired = errors.New("mail: stored credential is missing, expired, or does not include mail:imap_full; run yx360 login --mail")
	ErrMailboxSetup   = errors.New("mail: IMAP OAuth authentication failed; enable mail-client access and app passwords/OAuth tokens in Yandex 360 Mail settings, then run yx360 login --mail")
)

type Service struct {
	cfg  config.Mail
	cred *auth.Credential
}

type Query struct {
	Folder  string
	Limit   uint32
	Since   time.Time
	From    string
	Subject string
	Text    string
}

type Message struct {
	UID         uint32       `json:"uid"`
	Folder      string       `json:"folder"`
	Subject     string       `json:"subject"`
	From        []string     `json:"from,omitempty"`
	To          []string     `json:"to,omitempty"`
	Date        string       `json:"date,omitempty"`
	Size        int64        `json:"size,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Body        *Body        `json:"body,omitempty"`
}

type Body struct {
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
}

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MIMEType string `json:"mime_type"`
	Size     uint32 `json:"size,omitempty"`
}

func NewService(cfg config.Mail, cred *auth.Credential) *Service {
	return &Service{cfg: cfg, cred: cred}
}

func (s *Service) List(ctx context.Context, q Query) ([]Message, error) {
	return s.search(ctx, q, false)
}

func (s *Service) Search(ctx context.Context, q Query) ([]Message, error) {
	return s.search(ctx, q, true)
}

func (s *Service) Read(ctx context.Context, folder string, uid uint32) (*Message, error) {
	c, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if err := selectFolder(c, folder); err != nil {
		return nil, err
	}
	msgs, err := fetchMessages(c, folder, []imap.UID{imap.UID(uid)}, true)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("mail: message UID %d not found in %s", uid, folder)
	}
	return &msgs[0], nil
}

func (s *Service) DownloadAttachment(ctx context.Context, folder string, uid uint32, attachmentID string, outDir string) (string, error) {
	if outDir == "" {
		return "", errors.New("mail: --out is required for attachment downloads")
	}
	c, err := s.connect(ctx)
	if err != nil {
		return "", err
	}
	defer c.Close()
	if err := selectFolder(c, folder); err != nil {
		return "", err
	}
	msgs, err := fetchMessages(c, folder, []imap.UID{imap.UID(uid)}, true)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "", fmt.Errorf("mail: message UID %d not found in %s", uid, folder)
	}
	body, err := fetchWholeMessage(c, imap.UID(uid))
	if err != nil {
		return "", err
	}
	return writeAttachment(body, msgs[0].Attachments, attachmentID, outDir)
}

func (s *Service) search(ctx context.Context, q Query, filtered bool) ([]Message, error) {
	if q.Folder == "" {
		q.Folder = "INBOX"
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	c, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if err := selectFolder(c, q.Folder); err != nil {
		return nil, err
	}
	criteria := &imap.SearchCriteria{}
	if filtered {
		criteria.Since = q.Since
		if q.From != "" {
			criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{Key: "FROM", Value: q.From})
		}
		if q.Subject != "" {
			criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{Key: "SUBJECT", Value: q.Subject})
		}
		if q.Text != "" {
			criteria.Text = append(criteria.Text, q.Text)
		}
	}
	data, err := c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, err
	}
	uids := limitNewest(data.AllUIDs(), q.Limit)
	return fetchMessages(c, q.Folder, uids, false)
}

func (s *Service) connect(ctx context.Context) (*imapclient.Client, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(s.cfg.ReadScope) {
		return nil, ErrReauthRequired
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	address := net.JoinHostPort(s.cfg.IMAPHost, strconv.Itoa(s.cfg.IMAPPort))
	options := &imapclient.Options{
		TLSConfig:   &tls.Config{ServerName: s.cfg.IMAPHost, MinVersion: tls.VersionTLS12},
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
	c, err := dialTLS4(address, options)
	if err != nil {
		return nil, err
	}
	if err := c.Authenticate(newXOAUTH2Client(s.cred.Account, s.cred.AccessToken)); err != nil {
		if fallbackErr := c.Authenticate(sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
			Username: s.cred.Account,
			Token:    s.cred.AccessToken,
			Host:     s.cfg.IMAPHost,
			Port:     s.cfg.IMAPPort,
		})); fallbackErr != nil {
			c.Close()
			return nil, fmt.Errorf("%w: %v", ErrMailboxSetup, err)
		}
	}
	return c, nil
}

func dialTLS4(address string, options *imapclient.Options) (*imapclient.Client, error) {
	tlsConfig := options.TLSConfig.Clone()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp4", address, tlsConfig)
	if err != nil {
		return nil, err
	}
	return imapclient.New(conn, options), nil
}

func selectFolder(c *imapclient.Client, folder string) error {
	_, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	return err
}

func fetchMessages(c *imapclient.Client, folder string, uids []imap.UID, includeBody bool) ([]Message, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	set := imap.UIDSetNum(uids...)
	options := &imap.FetchOptions{
		UID:           true,
		Envelope:      true,
		InternalDate:  true,
		RFC822Size:    true,
		BodyStructure: &imap.FetchItemBodyStructure{Extended: true},
	}
	if includeBody {
		options.BodySection = []*imap.FetchItemBodySection{{Peek: true, Partial: &imap.SectionPartial{Size: maxReadBytes}}}
	}
	buffers, err := c.Fetch(set, options).Collect()
	if err != nil {
		return nil, err
	}
	out := make([]Message, 0, len(buffers))
	for _, buf := range buffers {
		msg := messageFromBuffer(folder, buf)
		if includeBody {
			raw := buf.FindBodySection(options.BodySection[0])
			msg.Body = bodyFromRaw(raw)
		}
		out = append(out, msg)
	}
	return out, nil
}

func fetchWholeMessage(c *imapclient.Client, uid imap.UID) ([]byte, error) {
	section := &imap.FetchItemBodySection{Peek: true, Partial: &imap.SectionPartial{Size: maxReadBytes}}
	buffers, err := c.Fetch(imap.UIDSetNum(uid), &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{section}}).Collect()
	if err != nil {
		return nil, err
	}
	if len(buffers) == 0 {
		return nil, fmt.Errorf("mail: message UID %d not found", uid)
	}
	return buffers[0].FindBodySection(section), nil
}

func messageFromBuffer(folder string, buf *imapclient.FetchMessageBuffer) Message {
	msg := Message{UID: uint32(buf.UID), Folder: folder, Size: buf.RFC822Size}
	if !buf.InternalDate.IsZero() {
		msg.Date = buf.InternalDate.Format(time.RFC3339)
	}
	if buf.Envelope != nil {
		msg.Subject = buf.Envelope.Subject
		msg.From = addresses(buf.Envelope.From)
		msg.To = addresses(buf.Envelope.To)
		if !buf.Envelope.Date.IsZero() {
			msg.Date = buf.Envelope.Date.Format(time.RFC3339)
		}
	}
	msg.Attachments = attachmentsFromStructure(buf.BodyStructure)
	return msg
}

func addresses(addrs []imap.Address) []string {
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if email := addr.Addr(); email != "" {
			out = append(out, email)
		}
	}
	return out
}

func attachmentsFromStructure(bs imap.BodyStructure) []Attachment {
	if bs == nil {
		return nil
	}
	var out []Attachment
	bs.Walk(func(path []int, part imap.BodyStructure) bool {
		single, ok := part.(*imap.BodyStructureSinglePart)
		if !ok {
			return true
		}
		filename := single.Filename()
		disp := single.Disposition()
		if filename == "" && (disp == nil || strings.ToLower(disp.Value) != "attachment") {
			return true
		}
		out = append(out, Attachment{
			ID:       partID(path),
			Filename: safeFilename(filename),
			MIMEType: single.MediaType(),
			Size:     single.Size,
		})
		return true
	})
	return out
}

func bodyFromRaw(raw []byte) *Body {
	reader, err := mailmessage.CreateReader(bytes.NewReader(raw))
	if err != nil && reader == nil {
		return nil
	}
	defer reader.Close()
	body := &Body{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil || part == nil {
			break
		}
		inline, ok := part.Header.(*mailmessage.InlineHeader)
		if !ok {
			continue
		}
		mediaType, _, _ := inline.ContentType()
		data, _ := io.ReadAll(io.LimitReader(part.Body, maxReadBytes))
		switch mediaType {
		case "text/plain":
			if body.Text == "" {
				body.Text = string(data)
			}
		case "text/html":
			if body.HTML == "" {
				body.HTML = string(data)
			}
		}
	}
	if body.Text == "" && body.HTML == "" {
		return nil
	}
	return body
}

func writeAttachment(raw []byte, manifest []Attachment, id string, outDir string) (string, error) {
	var wanted *Attachment
	for i := range manifest {
		if manifest[i].ID == id {
			wanted = &manifest[i]
			break
		}
	}
	if wanted == nil {
		return "", fmt.Errorf("mail: attachment %s not found", id)
	}
	reader, err := mailmessage.CreateReader(bytes.NewReader(raw))
	if err != nil && reader == nil {
		return "", err
	}
	defer reader.Close()
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil || part == nil {
			return "", err
		}
		attachment, ok := part.Header.(*mailmessage.AttachmentHeader)
		if !ok {
			continue
		}
		filename, _ := attachment.Filename()
		if safeFilename(filename) != wanted.Filename {
			continue
		}
		path := filepath.Join(outDir, wanted.Filename)
		if err := writeFile0600(path, part.Body); err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("mail: attachment %s not found in message body", id)
}

func limitNewest(uids []imap.UID, limit uint32) []imap.UID {
	if len(uids) == 0 {
		return nil
	}
	if limit == 0 || int(limit) >= len(uids) {
		return reverseUIDs(uids)
	}
	return reverseUIDs(uids[len(uids)-int(limit):])
}

func reverseUIDs(uids []imap.UID) []imap.UID {
	out := make([]imap.UID, len(uids))
	for i := range uids {
		out[i] = uids[len(uids)-1-i]
	}
	return out
}

func partID(path []int) string {
	parts := make([]string, len(path))
	for i, num := range path {
		parts[i] = strconv.Itoa(num)
	}
	return strings.Join(parts, ".")
}

func safeFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "attachment"
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', 0:
			return '-'
		default:
			return r
		}
	}, name)
}

func writeFile0600(path string, src io.Reader) error {
	dst, err := openAttachmentFile(path)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, io.LimitReader(src, maxReadBytes))
	return err
}
