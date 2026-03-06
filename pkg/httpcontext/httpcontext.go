package httpcontext

import (
	"context"
	"net/http"
)

type clientKey struct{}

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
