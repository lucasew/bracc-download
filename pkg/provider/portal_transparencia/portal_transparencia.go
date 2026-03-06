package portal_transparencia

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"fmt"
	"io"
	"iter"
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
	client            *http.Client
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
		client:            http.DefaultClient,
	}, nil
}

func (p *Provider) GetURL() *url.URL {
	return p.pageURL
}

func (p *Provider) Jobs() (iter.Seq[provider.Job], error) {
	resp, err := p.client.Get(p.pageURL.String())
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
				if !p.generateMonthlyJobs(dataset, yield) {
					return
				}
			case PeriodicityYearly:
				if !p.generateYearlyJobs(dataset, yield) {
					return
				}
			default:
				spew.Dump("portal_transparencia_unknown_periodicity", dataset.Slug, dataset.URL.String())
			}
		}
	}, nil
}

func (p *Provider) generateMonthlyJobs(dataset dataset, yield func(provider.Job) bool) bool {
	now := time.Now().UTC()
	misses := 0
	foundAny := false

	for offset := 0; offset < p.monthsBack; offset++ {
		d := time.Date(now.Year(), now.Month()-time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
		u := *dataset.URL
		u.Path = path.Join(dataset.URL.Path, d.Format("200601"))

		ok := p.probe(&u)
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

func (p *Provider) generateYearlyJobs(dataset dataset, yield func(provider.Job) bool) bool {
	now := time.Now().UTC()
	misses := 0
	foundAny := false

	for offset := 0; offset < p.yearsBack; offset++ {
		year := now.Year() - offset
		u := *dataset.URL
		u.Path = path.Join(dataset.URL.Path, fmt.Sprintf("%04d", year))

		ok := p.probe(&u)
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

func (p *Provider) probe(u *url.URL) bool {
	headReq, err := http.NewRequest(http.MethodHead, u.String(), nil)
	if err == nil {
		headResp, err := p.client.Do(headReq)
		if err == nil {
			headResp.Body.Close()
			if headResp.StatusCode >= 200 && headResp.StatusCode < 300 {
				return true
			}
		}
	}

	getReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false
	}
	getReq.Header.Set("Range", "bytes=0-0")
	getResp, err := p.client.Do(getReq)
	if err != nil {
		return false
	}
	defer getResp.Body.Close()
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
