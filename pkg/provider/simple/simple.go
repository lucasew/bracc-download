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

// SimpleJobProvider is a static, single-URL provider.
// It generates exactly one job pointing to the configured absolute URL.
type SimpleJobProvider struct {
	url *url.URL
}

// GetURL returns the absolute URL configured for this provider.
func (s *SimpleJobProvider) GetURL() *url.URL {
	return s.url
}

// NewSimpleJobProvider initializes a SimpleJobProvider with the given raw URL.
// It validates that the URL is absolute (has both scheme and host).
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

// Jobs returns an iterator containing exactly one SimpleJob for the configured URL.
func (s *SimpleJobProvider) Jobs(ctx context.Context) (iter.Seq[provider.Job], error) {
	_ = ctx
	return func(yield func(j provider.Job) bool) {
		yield(&SimpleJob{s.url})
	}, nil
}

// SimpleJob represents a download task for a single static URL.
type SimpleJob struct {
	url *url.URL
}

// NewJob creates a new SimpleJob from the given URL.
func NewJob(u url.URL) *SimpleJob {
	return &SimpleJob{url: &u}
}

// GetURL returns the target URL for this download job.
func (s *SimpleJob) GetURL() *url.URL {
	return s.url
}

// Download executes the HTTP GET request and streams the response body to a local file.
// It ensures atomic writes by downloading to a temporary `.part` file first,
// and renaming it to the final filename only upon a successful download.
// The target filename is derived from the response headers or URL.
// Progress is reported via the Context's ProgressBar, if configured.
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
	target := filepath.Join(dir, filename)

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

// filenameFromResponse derives the target filename for a downloaded file.
// It prioritizes the Content-Disposition header's filename* and filename parameters,
// falling back to the base name of the URL path, or the host name as a last resort.
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
