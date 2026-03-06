package simple

import (
	"bracc"
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type SimpleJobProvider struct {
	url *url.URL
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

func (s *SimpleJobProvider) Jobs() (iter.Seq[bracc.Job], error) {
	return func(yield func(j bracc.Job) bool) {
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
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()
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
	_, err = io.Copy(f, resp.Body)
	return err
}
