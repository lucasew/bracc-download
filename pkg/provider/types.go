package provider

import (
	"context"
	"iter"
	"net/url"
)

type JobProvider interface {
	GetURL() *url.URL
	Jobs(ctx context.Context) (iter.Seq[Job], error)
}

type Job interface {
	GetURL() *url.URL
	Download(ctx context.Context, dir string) error
}

var Providers []JobProvider
