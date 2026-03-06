package bracc

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const propfindBody = `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:resourcetype/>
    <d:getcontentlength/>
  </d:prop>
</d:propfind>`

type WebDAVJobProvider struct {
	url    *url.URL
	client *http.Client
}

func NewWebDAVJobProvider(rawURL string) (*WebDAVJobProvider, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid WebDAV URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("WebDAV URL must use http or https")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return &WebDAVJobProvider{
		url:    u,
		client: http.DefaultClient,
	}, nil
}

func (p *WebDAVJobProvider) Jobs() (iter.Seq[Job], error) {
	return func(yield func(Job) bool) {
		yield(&WebDAVMirrorJob{
			root:   p.url,
			client: p.client,
		})
	}, nil
}

type WebDAVMirrorJob struct {
	root   *url.URL
	client *http.Client
}

func (j *WebDAVMirrorJob) GetURL() *url.URL {
	return j.root
}

func (j *WebDAVMirrorJob) Download(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	pending := []*url.URL{j.root}
	seenCollections := map[string]struct{}{}

	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]

		normCollection := normalizeURLPath(current.Path)
		if _, ok := seenCollections[normCollection]; ok {
			continue
		}
		seenCollections[normCollection] = struct{}{}

		entries, err := j.list(ctx, current)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			rel, err := relativeURLPath(j.root.Path, entry.URL.Path)
			if err != nil {
				continue
			}
			if rel == "" {
				continue
			}

			localRel, err := urlPathToFSPath(rel)
			if err != nil {
				return err
			}
			localPath := filepath.Join(dir, localRel)

			if entry.IsCollection {
				if err := os.MkdirAll(localPath, os.ModePerm); err != nil {
					return err
				}
				pending = append(pending, entry.URL)
				continue
			}

			if err := os.MkdirAll(filepath.Dir(localPath), os.ModePerm); err != nil {
				return err
			}
			if err := j.downloadFile(ctx, entry.URL, localPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (j *WebDAVMirrorJob) list(ctx context.Context, collection *url.URL) ([]davEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", collection.String(), bytes.NewBufferString(propfindBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("PROPFIND %s failed: status=%d body=%q", collection, resp.StatusCode, string(b))
	}

	var ms multiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, err
	}

	items := make([]davEntry, 0, len(ms.Responses))
	currentPath := normalizeURLPath(collection.Path)

	for _, r := range ms.Responses {
		u, err := resolveHrefURL(collection, r.Href)
		if err != nil {
			continue
		}
		if normalizeURLPath(u.Path) == currentPath {
			continue
		}
		items = append(items, davEntry{
			URL:          u,
			IsCollection: r.IsCollection(),
		})
	}

	return items, nil
}

func (j *WebDAVMirrorJob) downloadFile(ctx context.Context, source *url.URL, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.String(), nil)
	if err != nil {
		return err
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("GET %s failed: status=%d body=%q", source, resp.StatusCode, string(b))
	}

	tmpPath := destination + ".part"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	return os.Rename(tmpPath, destination)
}

type davEntry struct {
	URL          *url.URL
	IsCollection bool
}

type multiStatus struct {
	Responses []response `xml:"response"`
}

type response struct {
	Href      string     `xml:"href"`
	PropStats []propStat `xml:"propstat"`
}

func (r response) IsCollection() bool {
	for _, ps := range r.PropStats {
		if ps.Prop.ResourceType.Collection != nil {
			return true
		}
	}
	return false
}

type propStat struct {
	Prop prop `xml:"prop"`
}

type prop struct {
	ResourceType resourceType `xml:"resourcetype"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

func resolveHrefURL(base *url.URL, href string) (*url.URL, error) {
	href = strings.TrimSpace(href)
	if href == "" {
		return nil, errors.New("empty href")
	}
	u, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	if u.IsAbs() {
		return u, nil
	}
	return base.ResolveReference(u), nil
}

func relativeURLPath(basePath, targetPath string) (string, error) {
	baseNorm := normalizeURLPath(basePath)
	targetNorm := normalizeURLPath(targetPath)

	if !strings.HasPrefix(targetNorm, baseNorm) {
		return "", fmt.Errorf("path %q is outside base %q", targetNorm, baseNorm)
	}
	rel := strings.TrimPrefix(targetNorm, baseNorm)
	rel = strings.TrimPrefix(rel, "/")
	return rel, nil
}

func normalizeURLPath(p string) string {
	if p == "" {
		return "/"
	}
	clean := path.Clean(p)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	if strings.HasSuffix(p, "/") && clean != "/" {
		clean += "/"
	}
	return clean
}

func urlPathToFSPath(rel string) (string, error) {
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return "", nil
	}
	parts := strings.Split(rel, "/")
	decoded := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		d, err := url.PathUnescape(part)
		if err != nil {
			return "", err
		}
		decoded = append(decoded, d)
	}
	if len(decoded) == 0 {
		return "", nil
	}
	return filepath.Join(decoded...), nil
}
