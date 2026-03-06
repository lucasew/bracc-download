package portal_transparencia

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"fmt"
	"go/doc"
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
	PeriodicityMonthly Periodicity = "monthly"
	PeriodicityYearly  Periodicity = "yearly"
)

type Options struct {
	DefaultPeriodicity Periodicity
	MonthsBack         int
	YearsBack          int
	ConsecutiveMisses  int
}

type Metadata struct {
	Periodicity Periodicity
}

type Provider struct {
	pageURL            *url.URL
	defaultPeriodicity Periodicity
	monthsBack         int
	yearsBack          int
	consecutiveMisses  int
	client             *http.Client
}

func NewProvider(slug string, opts Options) (*Provider, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("slug is required")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, slug)

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
		pageURL:            u,
		defaultPeriodicity: opts.DefaultPeriodicity,
		monthsBack:         opts.MonthsBack,
		yearsBack:          opts.YearsBack,
		consecutiveMisses:  opts.ConsecutiveMisses,
		client:             http.DefaultClient,
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

	metadata, err := parseMetadata(resp.Body)
	if err != nil && p.defaultPeriodicity == PeriodicityUnknown {
		return nil, err
	}

	periodicity := metadata.Periodicity
	if periodicity == PeriodicityUnknown {
		periodicity = p.defaultPeriodicity
	}
	if periodicity == PeriodicityUnknown {
		return nil, fmt.Errorf("could not determine periodicity for %s", p.pageURL)
	}

	return func(yield func(provider.Job) bool) {
		switch periodicity {
		case PeriodicityMonthly:
			p.generateMonthlyJobs(yield)
		case PeriodicityYearly:
			p.generateYearlyJobs(yield)
		}
	}, nil
}

func (p *Provider) generateMonthlyJobs(yield func(provider.Job) bool) {
	now := time.Now().UTC()
	misses := 0
	foundAny := false

	for offset := 0; offset < p.monthsBack; offset++ {
		d := time.Date(now.Year(), now.Month()-time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
		u := *p.pageURL
		u.Path = path.Join(p.pageURL.Path, d.Format("200601"))

		ok := p.probe(&u)
		if !ok {
			if foundAny {
				misses++
				if misses >= p.consecutiveMisses {
					return
				}
			}
			continue
		}

		foundAny = true
		misses = 0
		if !yield(simple.NewJob(u)) {
			return
		}
	}
}

func (p *Provider) generateYearlyJobs(yield func(provider.Job) bool) {
	now := time.Now().UTC()
	misses := 0
	foundAny := false

	for offset := 0; offset < p.yearsBack; offset++ {
		year := now.Year() - offset
		u := *p.pageURL
		u.Path = path.Join(p.pageURL.Path, fmt.Sprintf("%04d", year))

		ok := p.probe(&u)
		if !ok {
			if foundAny {
				misses++
				if misses >= p.consecutiveMisses {
					return
				}
			}
			continue
		}

		foundAny = true
		misses = 0
		if !yield(simple.NewJob(u)) {
			return
		}
	}
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

func parseMetadata(body io.Reader) (*Metadata, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}
	for _, item := range doc.Find("tr").EachIter() {
		spew.Dump(item)
	}
	// Plug goquery here. The expected flow is:
	// 1. select the dataset information table rows (`tr`)
	// 2. skip header rows (`th`)
	// 3. normalize first-column labels and inspect values
	// 4. derive periodicity from cells like "Periodicidade"
	return &Metadata{}, fmt.Errorf("portal_transparencia metadata parser not implemented")
}
