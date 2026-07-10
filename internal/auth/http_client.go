package auth

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/effective-dev-os/yx360-cli/internal/netutil"
)

func httpContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, netutil.IPv4Client())
}
