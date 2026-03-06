package simple

import (
	"bracc/pkg/provider"
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type SimpleJobProvider struct {
	url *url.URL
}

func (s *SimpleJobProvider) GetURL() *url.URL {
	return s.url
}

func NewSimpleJobProvider(rawURL string) (*SimpleJobProvider, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid URL %q: must be absolute", rawURL)
	}
	return &SimpleJobProvider{url: u}, nil
}

func (s *SimpleJobProvider) Jobs() (iter.Seq[provider.Job], error) {
	return func(yield func(j provider.Job) bool) {
		yield(&SimpleJob{s.url})
	}, nil
}

type SimpleJob struct {
	url *url.URL
}

func NewJob(u url.URL) *SimpleJob {
	return &SimpleJob{url: &u}
}

func (s *SimpleJob) GetURL() *url.URL {
	return s.url
}

func (s *SimpleJob) Download(ctx context.Context, dir string) error {
	filename := filepath.Base(s.url.Path)
	target := filepath.Join(dir, filename)
	req, err := http.NewRequest(http.MethodGet, s.url.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, s.url)
	}

	tmpPath := target + ".part"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	_, copyErr := provider.CopyWithProgress(ctx, s, f, resp.Body, resp.ContentLength)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	return os.Rename(tmpPath, target)
}
