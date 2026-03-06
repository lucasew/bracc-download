package portal_transparencia

import (
	"bracc/pkg/httpcontext"
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"context"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/davecgh/go-spew/spew"
)

const baseURL = "https://portaldatransparencia.gov.br/download-de-dados"

type Periodicity string

const (
	PeriodicityUnknown Periodicity = ""
	PeriodicityDaily               = "daily"
	PeriodicityMonthly Periodicity = "monthly"
	PeriodicityYearly  Periodicity = "yearly"
)

type Options struct {
	MonthsBack        int
	YearsBack         int
	ConsecutiveMisses int
}

type dataset struct {
	URL         *url.URL
	Slug        string
	Periodicity Periodicity
}

type Provider struct {
	pageURL           *url.URL
	monthsBack        int
	yearsBack         int
	consecutiveMisses int
}

func init() {
	p, err := NewProvider(Options{})
	if err != nil {
		panic(err)
	}
	provider.Providers = append(provider.Providers, p)
}

func NewProvider(opts Options) (*Provider, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	if opts.MonthsBack <= 0 {
		opts.MonthsBack = 60
	}
	if opts.YearsBack <= 0 {
		opts.YearsBack = 10
	}
	if opts.ConsecutiveMisses <= 0 {
		opts.ConsecutiveMisses = 6
	}

	return &Provider{
		pageURL:           u,
		monthsBack:        opts.MonthsBack,
		yearsBack:         opts.YearsBack,
		consecutiveMisses: opts.ConsecutiveMisses,
	}, nil
}

func (p *Provider) GetURL() *url.URL {
	return p.pageURL
}

func (p *Provider) Jobs(ctx context.Context) (iter.Seq[provider.Job], error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.pageURL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpcontext.Client(ctx).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, p.pageURL)
	}

	datasets, err := parseDatasets(p.pageURL, resp.Body)
	if err != nil {
		return nil, err
	}

	return func(yield func(provider.Job) bool) {
		for _, dataset := range datasets {
			switch dataset.Periodicity {
			case PeriodicityMonthly:
				if !p.generateMonthlyJobs(ctx, dataset, yield) {
					return
				}
			case PeriodicityYearly:
				if !p.generateYearlyJobs(ctx, dataset, yield) {
					return
				}
			default:
				spew.Dump("portal_transparencia_unknown_periodicity", dataset.Slug, dataset.URL.String())
			}
		}
	}, nil
}

func (p *Provider) generateMonthlyJobs(ctx context.Context, dataset dataset, yield func(provider.Job) bool) bool {
	now := time.Now().UTC()
	misses := 0
	foundAny := false

	for offset := 0; offset < p.monthsBack; offset++ {
		d := time.Date(now.Year(), now.Month()-time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
		u := *dataset.URL
		u.Path = path.Join(dataset.URL.Path, d.Format("200601"))

		ok := p.probe(ctx, &u)
		if !ok {
			if foundAny {
				misses++
				if misses >= p.consecutiveMisses {
					return true
				}
			}
			continue
		}

		foundAny = true
		misses = 0
		if !yield(simple.NewJob(u)) {
			return false
		}
	}

	return true
}

func (p *Provider) generateYearlyJobs(ctx context.Context, dataset dataset, yield func(provider.Job) bool) bool {
	now := time.Now().UTC()
	misses := 0
	foundAny := false

	for offset := 0; offset < p.yearsBack; offset++ {
		year := now.Year() - offset
		u := *dataset.URL
		u.Path = path.Join(dataset.URL.Path, fmt.Sprintf("%04d", year))

		ok := p.probe(ctx, &u)
		if !ok {
			if foundAny {
				misses++
				if misses >= p.consecutiveMisses {
					return true
				}
			}
			continue
		}

		foundAny = true
		misses = 0
		if !yield(simple.NewJob(u)) {
			return false
		}
	}

	return true
}

func (p *Provider) probe(ctx context.Context, u *url.URL) bool {
	headReq, err := http.NewRequest(http.MethodHead, u.String(), nil)
	if err == nil {
		headResp, err := httpcontext.Client(ctx).Do(headReq.WithContext(ctx))
		if err == nil {
			slog.Debug("portal_transparencia probe", "method", http.MethodHead, "url", u.String(), "status", headResp.StatusCode)
			headResp.Body.Close()
			if headResp.StatusCode >= 200 && headResp.StatusCode < 300 {
				return true
			}
		} else {
			slog.Debug("portal_transparencia probe error", "method", http.MethodHead, "url", u.String(), "error", err)
		}
	} else {
		slog.Debug("portal_transparencia probe request error", "method", http.MethodHead, "url", u.String(), "error", err)
	}

	getReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		slog.Debug("portal_transparencia probe request error", "method", http.MethodGet, "url", u.String(), "error", err)
		return false
	}
	getReq.Header.Set("Range", "bytes=0-0")
	getResp, err := httpcontext.Client(ctx).Do(getReq.WithContext(ctx))
	if err != nil {
		slog.Debug("portal_transparencia probe error", "method", http.MethodGet, "url", u.String(), "error", err)
		return false
	}
	defer getResp.Body.Close()
	slog.Debug("portal_transparencia probe", "method", http.MethodGet, "url", u.String(), "status", getResp.StatusCode)
	_, _ = io.Copy(io.Discard, io.LimitReader(getResp.Body, 1))
	return getResp.StatusCode == http.StatusOK || getResp.StatusCode == http.StatusPartialContent
}

func parseDatasets(base *url.URL, body io.Reader) ([]dataset, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	var datasets []dataset
	seen := map[string]struct{}{}

	for _, row := range doc.Find("tr").EachIter() {
		if row.Find("th").Length() > 0 {
			continue
		}

		tds := row.Find("td")
		if tds.Length() != 2 {
			continue
		}

		link, ok := tds.Eq(0).Find("a[href]").First().Attr("href")
		if !ok || strings.TrimSpace(link) == "" {
			continue
		}

		u, err := base.Parse(link)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(u.Path, base.Path) || u.Path == base.Path {
			continue
		}

		slug := path.Base(strings.TrimSuffix(u.Path, "/"))
		if slug == "." || slug == "/" || slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}

		periodicityText := normalizeText(tds.Eq(1).Text())
		d := dataset{
			URL:         u,
			Slug:        slug,
			Periodicity: detectPeriodicity(periodicityText),
		}
		spew.Dump(d)
		datasets = append(datasets, d)
		seen[slug] = struct{}{}
	}

	return datasets, nil
}

func detectPeriodicity(s string) Periodicity {
	s = normalizeText(s)
	switch {
	case strings.Contains(s, "mensal"):
		return PeriodicityMonthly
	case strings.Contains(s, "anual"):
		return PeriodicityYearly
	case strings.Contains(s, "diário"):
		return PeriodicityDaily
	default:
		return PeriodicityUnknown
	}
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}
