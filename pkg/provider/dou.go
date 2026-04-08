package provider

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"path"
	"time"
)

const (
	douBaseURL = "https://dadosabertos-download.cgu.gov.br/inlabs"
	defaultDOUMonthsBack = 24
)

func init() {
	p, err := NewDOUJobProvider(defaultDOUMonthsBack)
	if err != nil {
		panic(err)
	}
	Providers = append(Providers, p)
}

type DOUJobProvider struct {
	baseURL    *url.URL
	monthsBack int
	sections   []int
}

func (p *DOUJobProvider) GetURL() *url.URL {
	return p.baseURL
}

func NewDOUJobProvider(monthsBack int) (*DOUJobProvider, error) {
	if monthsBack <= 0 {
		return nil, fmt.Errorf("monthsBack must be > 0")
	}
	baseURL, err := url.Parse(douBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DOU base URL: %w", err)
	}
	return &DOUJobProvider{
		baseURL:    baseURL,
		monthsBack: monthsBack,
		sections:   []int{1, 2, 3},
	}, nil
}

func (p *DOUJobProvider) Jobs(ctx context.Context) (iter.Seq[Job], error) {
	_ = ctx
	now := time.Now().UTC()
	// Monthly DOU dumps are not reliably available for the current open month.
	// Start from the previous closed month and walk backwards.
	start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	return func(yield func(Job) bool) {
		for offset := 0; offset < p.monthsBack; offset++ {
			d := time.Date(start.Year(), start.Month()-time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
			aamm := fmt.Sprintf("%02d%02d", d.Year()%100, int(d.Month()))

			for _, section := range p.sections {
				filename := fmt.Sprintf("S0%d%s.zip", section, aamm)
				u := *p.baseURL
				u.Path = path.Join(p.baseURL.Path, aamm, filename)
				if !yield(NewJob(u)) {
					return
				}
			}
		}
	}, nil
}
