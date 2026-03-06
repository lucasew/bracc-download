package provider

import (
	"context"
	"iter"
	"log/slog"
	"net/url"
	"os"
	"path"
	"strings"
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

type ProgressBar interface {
	SetTotal(total int64)
	SetCurrent(current int64)
	Complete(err error)
}

type ProgressFactory interface {
	NewBar(job Job) ProgressBar
}

type progressFactoryKey struct{}
type progressBarKey struct{}

type nopProgressBar struct{}

func (nopProgressBar) SetTotal(int64)   {}
func (nopProgressBar) SetCurrent(int64) {}
func (nopProgressBar) Complete(error)   {}

func WithProgressFactory(ctx context.Context, factory ProgressFactory) context.Context {
	return context.WithValue(ctx, progressFactoryKey{}, factory)
}

func WithProgressBar(ctx context.Context, bar ProgressBar) context.Context {
	return context.WithValue(ctx, progressBarKey{}, bar)
}

func progressBarFromContext(ctx context.Context) ProgressBar {
	bar, ok := ctx.Value(progressBarKey{}).(ProgressBar)
	if !ok || bar == nil {
		return nopProgressBar{}
	}
	return bar
}

func progressFactoryFromContext(ctx context.Context) ProgressFactory {
	factory, ok := ctx.Value(progressFactoryKey{}).(ProgressFactory)
	if !ok {
		return nil
	}
	return factory
}

type JobRuntime struct {
	providers  []JobProvider
	urlFilters []string
}

func NewJobRuntime(providers []JobProvider) *JobRuntime {
	return &JobRuntime{providers: providers}
}

func (r *JobRuntime) WithURLFilters(filters []string) *JobRuntime {
	r.urlFilters = append([]string(nil), filters...)
	return r
}

func (r *JobRuntime) Match(job Job) bool {
	if len(r.urlFilters) == 0 {
		return true
	}
	u := job.GetURL().String()
	return matchURLFilters(u, r.urlFilters)
}

func (r *JobRuntime) MatchProvider(p JobProvider) bool {
	if len(r.urlFilters) == 0 {
		return true
	}
	return matchURLFilters(p.GetURL().String(), r.urlFilters)
}

func matchURLFilters(u string, filters []string) bool {
	for _, filter := range filters {
		if strings.Contains(u, filter) {
			return true
		}
	}
	return false
}

func (r *JobRuntime) Run(ctx context.Context, destination string) error {
	for _, provider := range r.providers {
		if !r.MatchProvider(provider) {
			continue
		}
		js, err := provider.Jobs(ctx)
		if err != nil {
			slog.Error("bad provider", "provider", provider, "error", err)
			continue
		}
		for job := range js {
			if !r.Match(job) {
				continue
			}
			u := job.GetURL()
			download_dir := path.Join(destination, u.Host, strings.ReplaceAll(u.Path, "/", string(os.PathSeparator)), "_")
			slog.Info("downloading", "url", u, "download_dir", download_dir, "job", job)
			var bar ProgressBar = nopProgressBar{}
			jobCtx := ctx
			if factory := progressFactoryFromContext(ctx); factory != nil {
				bar = factory.NewBar(job)
				jobCtx = WithProgressBar(ctx, bar)
			}
			if err := os.MkdirAll(download_dir, os.ModePerm); err != nil {
				return err
			}
			if err := job.Download(jobCtx, download_dir); err != nil {
				bar.Complete(err)
				slog.Error("download error", "url", u, "download_dir", download_dir)
				continue
			}
			bar.Complete(nil)
		}
	}
	return nil
}
