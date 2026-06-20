package mail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
)

type UnsubscribeMethod string

const (
	UnsubscribeHTTPSPost UnsubscribeMethod = "https-post"
	UnsubscribeHTTPSGet  UnsubscribeMethod = "https-get"
	UnsubscribeMailto    UnsubscribeMethod = "mailto"
)

type UnsubscribeOption struct {
	Method       UnsubscribeMethod `json:"method"`
	URI          string            `json:"uri"`
	RequiresSMTP bool              `json:"requires_smtp"`
	OneClick     bool              `json:"one_click"`
}

type UnsubscribePreview struct {
	UID      uint32              `json:"uid"`
	Folder   string              `json:"folder"`
	Options  []UnsubscribeOption `json:"options"`
	Selected *UnsubscribeOption  `json:"selected,omitempty"`
}

type UnsubscribeResult struct {
	Status     string            `json:"status"`
	UID        uint32            `json:"uid"`
	Folder     string            `json:"folder"`
	Method     UnsubscribeMethod `json:"method"`
	URI        string            `json:"uri"`
	HTTPStatus int               `json:"http_status,omitempty"`
	Mail       *SendResult       `json:"mail,omitempty"`
}

func (s *Service) PreviewUnsubscribe(ctx context.Context, folder string, uid uint32, method UnsubscribeMethod) (*UnsubscribePreview, error) {
	options, err := s.unsubscribeOptions(ctx, folder, uid)
	if err != nil {
		return nil, err
	}
	preview := &UnsubscribePreview{UID: uid, Folder: folder, Options: options}
	selected, err := selectUnsubscribeOption(options, method)
	if err != nil {
		return nil, err
	}
	preview.Selected = selected
	return preview, nil
}

func (s *Service) ExecuteUnsubscribe(ctx context.Context, folder string, uid uint32, method UnsubscribeMethod) (*UnsubscribeResult, error) {
	preview, err := s.PreviewUnsubscribe(ctx, folder, uid, method)
	if err != nil {
		return nil, err
	}
	if preview.Selected == nil {
		return nil, errors.New("mail: no unsubscribe options found")
	}
	option := *preview.Selected
	result := &UnsubscribeResult{
		UID:    uid,
		Folder: folder,
		Method: option.Method,
		URI:    option.URI,
	}
	switch option.Method {
	case UnsubscribeHTTPSPost:
		status, err := executeUnsubscribePOST(ctx, option.URI)
		if err != nil {
			return nil, err
		}
		result.Status = "posted"
		result.HTTPStatus = status
	case UnsubscribeHTTPSGet:
		status, err := executeUnsubscribeGET(ctx, option.URI)
		if err != nil {
			return nil, err
		}
		result.Status = "requested"
		result.HTTPStatus = status
	case UnsubscribeMailto:
		sendResult, err := s.SendGeneratedUnsubscribe(ctx, option.URI)
		if err != nil {
			return nil, err
		}
		result.Status = "sent"
		result.Mail = sendResult
	default:
		return nil, fmt.Errorf("mail: unsupported unsubscribe method %q", option.Method)
	}
	return result, nil
}

func (s *Service) unsubscribeOptions(ctx context.Context, folder string, uid uint32) ([]UnsubscribeOption, error) {
	c, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if err := selectFolder(c, folder); err != nil {
		return nil, err
	}
	return fetchUnsubscribeOptions(c, imap.UID(uid))
}

func ParseUnsubscribeHeaders(listHeader string, postHeader string) []UnsubscribeOption {
	if strings.TrimSpace(listHeader) == "" {
		return nil
	}
	oneClick := isOneClickPost(postHeader)
	rawURIs := parseListUnsubscribeURIs(listHeader)
	options := make([]UnsubscribeOption, 0, len(rawURIs))
	for _, rawURI := range rawURIs {
		parsed, err := url.Parse(rawURI)
		if err != nil || parsed.Scheme == "" {
			continue
		}
		switch strings.ToLower(parsed.Scheme) {
		case "mailto":
			options = append(options, UnsubscribeOption{
				Method:       UnsubscribeMailto,
				URI:          rawURI,
				RequiresSMTP: true,
			})
		case "http", "https":
			option := UnsubscribeOption{
				Method: UnsubscribeHTTPSGet,
				URI:    rawURI,
			}
			if parsed.Scheme == "https" && oneClick {
				option.Method = UnsubscribeHTTPSPost
				option.OneClick = true
			}
			options = append(options, option)
		}
	}
	return options
}

func parseListUnsubscribeURIs(header string) []string {
	var out []string
	for {
		start := strings.IndexByte(header, '<')
		if start < 0 {
			return out
		}
		header = header[start+1:]
		end := strings.IndexByte(header, '>')
		if end < 0 {
			return out
		}
		raw := strings.Map(func(r rune) rune {
			switch r {
			case ' ', '\t', '\r', '\n':
				return -1
			default:
				return r
			}
		}, header[:end])
		if raw != "" {
			out = append(out, raw)
		}
		header = header[end+1:]
	}
}

func isOneClickPost(header string) bool {
	return strings.EqualFold(strings.TrimSpace(header), "List-Unsubscribe=One-Click")
}

func selectUnsubscribeOption(options []UnsubscribeOption, method UnsubscribeMethod) (*UnsubscribeOption, error) {
	if len(options) == 0 {
		return nil, nil
	}
	if method == "" {
		return &options[0], nil
	}
	for i := range options {
		if options[i].Method == method {
			return &options[i], nil
		}
	}
	return nil, fmt.Errorf("mail: no unsubscribe option for method %s", method)
}

func executeUnsubscribePOST(ctx context.Context, target string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader("List-Unsubscribe=One-Click"))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := unsubscribeHTTPClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 300 && res.StatusCode < 400 {
		return res.StatusCode, fmt.Errorf("mail: unsubscribe POST redirect blocked: %s", res.Status)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return res.StatusCode, fmt.Errorf("mail: unsubscribe POST failed: %s", res.Status)
	}
	return res.StatusCode, nil
}

func executeUnsubscribeGET(ctx context.Context, target string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	res, err := unsubscribeHTTPClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return res.StatusCode, fmt.Errorf("mail: unsubscribe GET failed: %s", res.Status)
	}
	return res.StatusCode, nil
}

func unsubscribeHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && via[0].Method == http.MethodPost {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}
