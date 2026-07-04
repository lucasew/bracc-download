package simple

import (
	"bracc/pkg/httpcontext"
	"bracc/pkg/provider"
	"context"
	"fmt"
	"iter"
	"mime"
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

func (s *SimpleJobProvider) Jobs(ctx context.Context) (iter.Seq[provider.Job], error) {
	_ = ctx
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
	req, err := http.NewRequest(http.MethodGet, s.url.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := httpcontext.Client(ctx).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, s.url)
	}
	filename := filenameFromResponse(resp)
	provider.ProgressBarFromContext(ctx).SetName(filename)

	targetDir := filepath.Clean(dir)
	target := filepath.Join(targetDir, filepath.Clean(filepath.Join("/", filename)))
	tmpPath := target + ".part"

	// #nosec G304 -- We construct paths using filepath.Join safely.
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

func filenameFromResponse(resp *http.Response) string {
	filename := filepath.Base(resp.Request.URL.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = filepath.Base(resp.Request.URL.Host)
	}

	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return filename
	}

	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return filename
	}
	if value := params["filename*"]; value != "" {
		return value
	}
	if value := params["filename"]; value != "" {
		return value
	}
	return filename
}
