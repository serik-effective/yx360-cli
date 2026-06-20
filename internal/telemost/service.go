package telemost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

var ErrReauthRequired = errors.New("telemost: stored credential is missing, expired, or does not include telemost-api:conferences.create; run yx360 login --telemost")

type Service struct {
	cfg    config.Telemost
	cred   *auth.Credential
	client *http.Client
}

type CreateOptions struct {
	WaitingRoomLevel string
}

type Conference struct {
	ID      string `json:"id"`
	JoinURL string `json:"join_url"`
}

func NewService(cfg config.Telemost, cred *auth.Credential) *Service {
	return &Service{cfg: cfg, cred: cred, client: ipv4Client()}
}

func (s *Service) Create(ctx context.Context, opts CreateOptions) (*Conference, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(s.cfg.CreateScope) {
		return nil, ErrReauthRequired
	}
	waitingRoom := opts.WaitingRoomLevel
	if waitingRoom == "" {
		waitingRoom = "PUBLIC"
	}
	body, err := json.Marshal(struct {
		WaitingRoomLevel string `json:"waiting_room_level"`
	}{WaitingRoomLevel: waitingRoom})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.BaseURL, "/")+"/conferences", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+s.cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("telemost: create failed: HTTP %d", resp.StatusCode)
	}
	var conference Conference
	if err := json.NewDecoder(resp.Body).Decode(&conference); err != nil {
		return nil, err
	}
	if conference.ID == "" || conference.JoinURL == "" {
		return nil, errors.New("telemost: create response missing id or join_url")
	}
	return &conference, nil
}

func ipv4Client() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext(ctx, "tcp4", address)
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: transport}
}
