package netutil

import (
	"context"
	"net"
	"net/http"
	"time"
)

// IPv4Client returns an *http.Client that forces IPv4 (tcp4) for all
// connections. Use for Yandex API endpoints where IPv6 routing is unreliable
// in the deployment network (D-006).
func IPv4Client() *http.Client {
	transport := &http.Transport{
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
