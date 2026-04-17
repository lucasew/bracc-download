package httpcontext

import (
	"context"
	"fmt"
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

func Get(ctx context.Context, urlStr string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	resp, err := Client(ctx).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, urlStr)
	}
	return resp, nil
}
