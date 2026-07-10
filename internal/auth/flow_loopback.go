package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/oauth2"

	"github.com/effective-dev-os/yx360-cli/internal/config"
)

const loopbackTimeout = 3 * time.Minute

type LoopbackProvider struct {
	cfg config.OAuth
}

func NewLoopbackProvider(cfg config.OAuth) *LoopbackProvider {
	return &LoopbackProvider{cfg: cfg}
}

func (p *LoopbackProvider) Authenticate(ctx context.Context, opts AuthOptions) (*Credential, error) {
	if opts.NoBrowser {
		return nil, errRungUnavailable
	}
	if err := missingClientID(p.cfg); err != nil {
		return nil, err
	}

	port := opts.Port
	if port == 0 {
		port = p.cfg.LoopbackPort
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return nil, errRungUnavailable
		}
		return nil, err
	}
	defer listener.Close()

	conf := oauthConfig(p.cfg, opts)
	verifier := oauth2.GenerateVerifier()
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	authURL := conf.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	type result struct {
		code string
		err  error
	}
	results := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if oauthErr := q.Get("error"); oauthErr != "" {
			http.Error(w, "authorization failed", http.StatusBadRequest)
			results <- result{err: fmt.Errorf("oauth authorize rejected: %s", oauthErr)}
			return
		}
		gotState := q.Get("state")
		if subtle.ConstantTimeCompare([]byte(gotState), []byte(state)) != 1 {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			results <- result{err: errors.New("oauth state mismatch")}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			results <- result{err: errors.New("oauth response missing code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body><h2>yx360: signed in.</h2><p>You may close this tab.</p></body></html>")
		results <- result{code: code}
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	if err := openBrowser(authURL); err != nil {
		return nil, errRungUnavailable
	}

	waitCtx, cancel := context.WithTimeout(ctx, loopbackTimeout)
	defer cancel()

	select {
	case <-waitCtx.Done():
		return nil, errRungUnavailable
	case res := <-results:
		if res.err != nil {
			return nil, res.err
		}
		tok, err := exchangeCode(ctx, conf, res.code, verifier)
		if err != nil {
			return nil, err
		}
		cred := credentialFromToken(tok, GrantLoopback, conf.Scopes)
		populateAccount(ctx, cred)
		return cred, nil
	}
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
