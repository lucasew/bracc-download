package webdav

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"path"
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

func (p *WebDAVJobProvider) Jobs() (iter.Seq[provider.Job], error) {
	return func(yield func(provider.Job) bool) {
		ctx := context.Background()
		pending := []*url.URL{p.url}
		seenCollections := map[string]struct{}{}

		for len(pending) > 0 {
			current := pending[0]
			pending = pending[1:]

			normCollection := normalizeURLPath(current.Path)
			if _, ok := seenCollections[normCollection]; ok {
				continue
			}
			seenCollections[normCollection] = struct{}{}

			entries, err := p.list(ctx, current)
			if err != nil {
				slog.Error("webdav list failed", "collection", current.String(), "error", err)
				return
			}

			for _, entry := range entries {
				if entry.IsCollection {
					pending = append(pending, entry.URL)
					continue
				}
				if !yield(simple.NewJob(*entry.URL)) {
					return
				}
			}
		}
	}, nil
}

func (p *WebDAVJobProvider) list(ctx context.Context, collection *url.URL) ([]davEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", collection.String(), bytes.NewBufferString(propfindBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := p.client.Do(req)
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
		items = append(items, davEntry{URL: u, IsCollection: r.IsCollection()})
	}

	return items, nil
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
