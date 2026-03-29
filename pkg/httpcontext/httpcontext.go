package httpcontext

import (
	"context"
	"net/http"
)

type clientKey struct{}

type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if clone.Header.Get("User-Agent") == "" && t.userAgent != "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(clone)
}

// NewClient instantiates a standard HTTP Client configured with a custom
// RoundTripper which injects a default User-Agent on all outgoing requests.
func NewClient(userAgent string) *http.Client {
	base := http.DefaultTransport
	return &http.Client{
		Transport: userAgentTransport{
			base:      base,
			userAgent: userAgent,
		},
	}
}

// WithClient returns a child context populated with an active http.Client.
// Used at entrypoint to make custom HTTP configurations (like User-Agent interception)
// available to the downstream provider logic.
func WithClient(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		return ctx
	}
	return context.WithValue(ctx, clientKey{}, client)
}

// Client attempts to extract an http.Client embedded within the request context.
// If no specific HTTP client configuration was injected via WithClient,
// it gracefully defaults to the standard http.DefaultClient.
func Client(ctx context.Context) *http.Client {
	client, ok := ctx.Value(clientKey{}).(*http.Client)
	if !ok || client == nil {
		return http.DefaultClient
	}
	return client
}
