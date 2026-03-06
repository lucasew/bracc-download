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

func NewClient(userAgent string) *http.Client {
	base := http.DefaultTransport
	return &http.Client{
		Transport: userAgentTransport{
			base:      base,
			userAgent: userAgent,
		},
	}
}

func WithClient(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		return ctx
	}
	return context.WithValue(ctx, clientKey{}, client)
}

func Client(ctx context.Context) *http.Client {
	client, ok := ctx.Value(clientKey{}).(*http.Client)
	if !ok || client == nil {
		return http.DefaultClient
	}
	return client
}
