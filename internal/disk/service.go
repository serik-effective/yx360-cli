package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/netutil"
)

// ErrReauthRequired is returned when the stored credential is missing, expired,
// or does not include the required Disk scopes.
var ErrReauthRequired = errors.New("disk: stored credential is missing, expired, or does not include disk scopes; run yx360 login --disk")

// ErrConflict is returned by Put when the target path already exists and
// overwrite is false. Callers should prompt the user and retry with overwrite=true.
var ErrConflict = errors.New("disk: target path already exists; re-run with --yes to overwrite")

// Service provides access to the Yandex Disk REST API.
type Service struct {
	cfg    config.Disk
	cred   *auth.Credential
	client *http.Client
}

// NewService creates a Service backed by the given credential and config.
func NewService(cfg config.Disk, cred *auth.Credential) *Service {
	return &Service{cfg: cfg, cred: cred, client: netutil.IPv4Client()}
}

// diskPath ensures path carries the disk: scheme prefix required by the API.
func diskPath(p string) string {
	if strings.HasPrefix(p, "disk:") || strings.HasPrefix(p, "trash:") || strings.HasPrefix(p, "app:") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "disk:" + p
}

func (s *Service) base() string { return strings.TrimRight(s.cfg.BaseURL, "/") }

func (s *Service) do(ctx context.Context, method, endpoint, scope string, body io.Reader) (*http.Response, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(scope) {
		return nil, ErrReauthRequired
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+s.cred.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, ErrReauthRequired
	}
	return resp, nil
}

// List returns the contents of a directory at path.
func (s *Service) List(ctx context.Context, path string, limit, offset int) (*ResourceList, error) {
	if limit <= 0 {
		limit = 20
	}
	ep := fmt.Sprintf("%s/resources?path=%s&limit=%d&offset=%d",
		s.base(), url.QueryEscape(diskPath(path)), limit, offset)
	resp, err := s.do(ctx, http.MethodGet, ep, s.cfg.ReadScope, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp, "list")
	}
	var raw resourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Type == "file" {
		return nil, errors.New("disk: path is a file; use 'disk get' to download it")
	}
	if raw.Embedded == nil {
		return &ResourceList{}, nil
	}
	return raw.Embedded, nil
}

// Get downloads the file at remotePath into outDir.
// v1: returns an error if remotePath resolves to a directory.
func (s *Service) Get(ctx context.Context, remotePath, outDir string) (string, error) {
	ep := fmt.Sprintf("%s/resources/download?path=%s",
		s.base(), url.QueryEscape(diskPath(remotePath)))
	resp, err := s.do(ctx, http.MethodGet, ep, s.cfg.ReadScope, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", errors.New("disk: path is a directory; recursive download not supported in v1")
	}
	if resp.StatusCode != http.StatusOK {
		return "", httpError(resp, "download-link")
	}
	var link Link
	if err := json.NewDecoder(resp.Body).Decode(&link); err != nil {
		return "", err
	}
	if link.Href == "" {
		return "", errors.New("disk: download response missing href")
	}

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, link.Href, nil)
	if err != nil {
		return "", err
	}
	dlResp, err := s.client.Do(dlReq)
	if err != nil {
		return "", err
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("disk: download failed: HTTP %d", dlResp.StatusCode)
	}

	// Path-traversal protection: strip directory components, reject unsafe names.
	base := filepath.Base(remotePath)
	if base == "." || base == ".." {
		return "", fmt.Errorf("disk: unsafe remote filename: %q", base)
	}
	outPath := filepath.Join(outDir, base)
	if !strings.HasPrefix(filepath.Clean(outPath)+string(filepath.Separator), filepath.Clean(outDir)+string(filepath.Separator)) {
		return "", fmt.Errorf("disk: filename would escape output directory: %q", base)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, dlResp.Body); err != nil {
		return "", err
	}
	return outPath, nil
}

// Put uploads localPath to remotePath. If the target exists and overwrite is
// false, returns ErrConflict. The upload URL expires in 30 minutes.
func (s *Service) Put(ctx context.Context, localPath, remotePath string, overwrite bool) error {
	ov := "false"
	if overwrite {
		ov = "true"
	}
	ep := fmt.Sprintf("%s/resources/upload?path=%s&overwrite=%s",
		s.base(), url.QueryEscape(diskPath(remotePath)), ov)
	resp, err := s.do(ctx, http.MethodGet, ep, s.cfg.WriteScope, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusConflict:
		return ErrConflict
	case http.StatusRequestEntityTooLarge:
		return errors.New("disk: file exceeds maximum upload size (1 GB standard / 50 GB Yandex 360)")
	case http.StatusInsufficientStorage:
		return errors.New("disk: Yandex Disk quota exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return httpError(resp, "upload-link")
	}
	var link Link
	if err := json.NewDecoder(resp.Body).Decode(&link); err != nil {
		return err
	}
	if link.Href == "" {
		return errors.New("disk: upload response missing href")
	}

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// The PUT to the upload URL does not require an OAuth header (per Yandex docs).
	ulReq, err := http.NewRequestWithContext(ctx, http.MethodPut, link.Href, f)
	if err != nil {
		return err
	}
	ulReq.ContentLength = fi.Size()
	ulResp, err := s.client.Do(ulReq)
	if err != nil {
		return err
	}
	defer ulResp.Body.Close()
	if ulResp.StatusCode != http.StatusCreated && ulResp.StatusCode != http.StatusOK {
		return fmt.Errorf("disk: upload failed: HTTP %d", ulResp.StatusCode)
	}
	return nil
}

// Share makes a resource publicly accessible and returns the public URL.
func (s *Service) Share(ctx context.Context, path string) (string, error) {
	ep := fmt.Sprintf("%s/resources/publish?path=%s",
		s.base(), url.QueryEscape(diskPath(path)))
	resp, err := s.do(ctx, http.MethodPut, ep, s.cfg.WriteScope, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return "", httpError(resp, "share")
	}
	return s.publicURL(ctx, path)
}

func (s *Service) publicURL(ctx context.Context, path string) (string, error) {
	ep := fmt.Sprintf("%s/resources?path=%s&fields=public_url",
		s.base(), url.QueryEscape(diskPath(path)))
	resp, err := s.do(ctx, http.MethodGet, ep, s.cfg.ReadScope, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", httpError(resp, "get-public-url")
	}
	var raw struct {
		PublicURL string `json:"public_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}
	return raw.PublicURL, nil
}

// Unshare removes public access from a resource.
func (s *Service) Unshare(ctx context.Context, path string) error {
	ep := fmt.Sprintf("%s/resources/unpublish?path=%s",
		s.base(), url.QueryEscape(diskPath(path)))
	resp, err := s.do(ctx, http.MethodPut, ep, s.cfg.WriteScope, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return httpError(resp, "unshare")
	}
	return nil
}

// Remove moves a resource to Trash (permanent=false) or permanently deletes it.
// Async operations (202 Accepted) are polled until completion.
func (s *Service) Remove(ctx context.Context, path string, permanent bool) error {
	perm := "false"
	if permanent {
		perm = "true"
	}
	ep := fmt.Sprintf("%s/resources?path=%s&permanently=%s",
		s.base(), url.QueryEscape(diskPath(path)), perm)
	resp, err := s.do(ctx, http.MethodDelete, ep, s.cfg.WriteScope, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode == http.StatusAccepted {
		opURL := resp.Header.Get("Location")
		if opURL == "" {
			var link Link
			_ = json.NewDecoder(resp.Body).Decode(&link)
			opURL = link.Href
		}
		return s.pollOperation(ctx, opURL)
	}
	return httpError(resp, "remove")
}

func (s *Service) pollOperation(ctx context.Context, opURL string) error {
	if opURL == "" {
		return nil
	}
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, opURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "OAuth "+s.cred.AccessToken)
		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		var status OperationStatus
		_ = json.NewDecoder(resp.Body).Decode(&status)
		resp.Body.Close()
		switch status.Status {
		case "success":
			return nil
		case "failed":
			return errors.New("disk: remote operation failed")
		}
	}
	return nil // operation may still complete; caller can check
}

// Mkdir creates a directory at path. Parents must exist (API does not create
// them recursively).
func (s *Service) Mkdir(ctx context.Context, path string) error {
	ep := fmt.Sprintf("%s/resources?path=%s",
		s.base(), url.QueryEscape(diskPath(path)))
	resp, err := s.do(ctx, http.MethodPut, ep, s.cfg.WriteScope, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return nil
	}
	if resp.StatusCode == http.StatusConflict {
		return errors.New("disk: directory already exists")
	}
	return httpError(resp, "mkdir")
}

func httpError(resp *http.Response, action string) error {
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	msg := strings.TrimSpace(string(snippet))
	if msg == "" {
		return fmt.Errorf("disk: %s failed: HTTP %d", action, resp.StatusCode)
	}
	return fmt.Errorf("disk: %s failed: HTTP %d: %s", action, resp.StatusCode, msg)
}
