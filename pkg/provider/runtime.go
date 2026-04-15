package provider

import (
	"context"
	"log/slog"
	"os"
	"path"
	"strings"
)

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
			if err := r.processJob(ctx, job, destination); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *JobRuntime) processJob(ctx context.Context, job Job, destination string) error {
	u := job.GetURL()
	downloadDir := path.Join(destination, u.Host, strings.ReplaceAll(u.Path, "/", string(os.PathSeparator)), "_")
	slog.Info("downloading", "url", u, "download_dir", downloadDir, "job", job)

	var bar ProgressBar = nopProgressBar{}
	jobCtx := ctx
	if factory := progressFactoryFromContext(ctx); factory != nil {
		bar = factory.NewBar(job)
		jobCtx = WithProgressBar(ctx, bar)
	}

	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		return err
	}
	if err := job.Download(jobCtx, downloadDir); err != nil {
		bar.Complete(err)
		slog.Error("download error", "url", u, "download_dir", downloadDir)
		return nil // Continue processing other jobs even if one fails
	}

	bar.Complete(nil)
	return nil
}
