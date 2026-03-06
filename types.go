package bracc

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
	Jobs() (iter.Seq[Job], error)
}

type Job interface {
	GetURL() *url.URL
	Download(ctx context.Context, dir string) error
}

var Providers []JobProvider

type JobRuntime struct {
	providers []JobProvider
}

func NewJobRuntime(providers []JobProvider) *JobRuntime {
	return &JobRuntime{providers}
}

func (r *JobRuntime) Run(ctx context.Context, destination string) error {
	for _, provider := range r.providers {
		js, err := provider.Jobs()
		if err != nil {
			slog.Error("bad provider", "error", err)
		}
		for job := range js {
			u := job.GetURL()
			download_dir := path.Join(destination, u.Host, strings.ReplaceAll(u.Path, "/", string(os.PathSeparator)), "_")
			slog.Info("downloading", "url", u, "download_dir", download_dir, "job", job)
			if err := os.MkdirAll(download_dir, os.ModePerm); err != nil {
				return err
			}
			if err := job.Download(ctx, download_dir); err != nil {
				slog.Error("download error", "url", u, "download_dir", download_dir)
			}
		}
	}
	return nil
}
