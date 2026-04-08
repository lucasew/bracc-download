package portal_transparencia

import (
	"bracc/pkg/httpcontext"
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const baseURL = "https://portaldatransparencia.gov.br/download-de-dados"

// Periodicity dictates how frequently a dataset updates, determining how its download URL keys are constructed.
type Periodicity string

const (
	PeriodicityUnknown Periodicity = ""
	PeriodicityDaily               = "daily"
	PeriodicityMonthly Periodicity = "monthly"
	PeriodicityYearly  Periodicity = "yearly"
)

// dataset represents a scraped dataset category from the portal, grouping its endpoint, slug, and expected update frequency.
type dataset struct {
	URL         *url.URL
	Slug        string
	Periodicity Periodicity
}

// arquivoEntry mirrors the structure of the JSON objects pushed to `arquivos` arrays within inline scripts on dataset pages.
type arquivoEntry struct {
	Ano    string `json:"ano"`
	Mes    string `json:"mes"`
	Dia    string `json:"dia"`
	Origem string `json:"origem"`
}

// Provider implements the JobProvider interface for 'Portal da Transparência', orchestrating HTML parsing to discover files dynamically.
type Provider struct {
}

func init() {
	p, err := NewProvider()
	if err != nil {
		panic(err)
	}
	provider.Providers = append(provider.Providers, p)
}

// NewProvider initializes the provider, ensuring the hardcoded baseURL is structurally valid before use.
func NewProvider() (*Provider, error) {
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	return &Provider{}, nil
}

// GetURL returns the root download URL from which all scraping begins.
func (p *Provider) GetURL() *url.URL {
	u, _ := url.Parse(baseURL)
	return u
}

// Jobs walks the base URL to collect datasets, then queries each dataset's page to yield discrete downloadable file Jobs.
func (p *Provider) Jobs(ctx context.Context) (iter.Seq[provider.Job], error) {
	base := p.GetURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpcontext.Client(ctx).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, base)
	}

	datasets, err := parseDatasets(base, resp.Body)
	if err != nil {
		return nil, err
	}

	return func(yield func(provider.Job) bool) {
		for _, dataset := range datasets {
			jobs, err := p.datasetJobs(ctx, dataset)
			if err != nil {
				slog.Error("portal_transparencia dataset parse error", "dataset", dataset.Slug, "url", dataset.URL.String(), "error", err)
				continue
			}
			for _, job := range jobs {
				if !yield(job) {
					return
				}
			}
		}
	}, nil
}

// datasetJobs queries an individual dataset's endpoint, scraping the inline Javascript to construct download endpoints for each entry.
func (p *Provider) datasetJobs(ctx context.Context, dataset dataset) ([]provider.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataset.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpcontext.Client(ctx).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, dataset.URL)
	}

	entries, err := parseArquivoEntries(resp.Body)
	if err != nil {
		return nil, err
	}

	jobs := make([]provider.Job, 0, len(entries))
	for _, entry := range entries {
		key, ok := datasetKey(dataset.Periodicity, entry)
		if !ok {
			continue
		}
		u := *dataset.URL
		u.Path = path.Join(dataset.URL.Path, key)
		jobs = append(jobs, simple.NewJob(u))
	}
	return jobs, nil
}

// parseDatasets reads the HTML body of the root dataset page, hunting for anchor links inside table rows while deduplicating by slug.
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
		datasets = append(datasets, d)
		seen[slug] = struct{}{}
	}

	return datasets, nil
}

var arquivoPushRE = regexp.MustCompile(`arquivos\.push\((\{.*?\})\);`)

// parseArquivoEntries hunts for `<script>` blocks within a dataset page, extracting Javascript object payloads into native structs using a RegExp pattern.
func parseArquivoEntries(body io.Reader) ([]arquivoEntry, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	var entries []arquivoEntry
	doc.Find("script").Each(func(_ int, script *goquery.Selection) {
		for _, match := range arquivoPushRE.FindAllStringSubmatch(script.Text(), -1) {
			if len(match) != 2 {
				continue
			}
			var entry arquivoEntry
			if err := json.Unmarshal([]byte(match[1]), &entry); err != nil {
				continue
			}
			entries = append(entries, entry)
		}
	})
	return entries, nil
}

// datasetKey synthesizes the download file path suffix based on how frequently the source data is cut (e.g. appending YYYYMM for monthly data).
func datasetKey(periodicity Periodicity, entry arquivoEntry) (string, bool) {
	switch periodicity {
	case PeriodicityMonthly:
		if entry.Ano == "" || entry.Mes == "" {
			return "", false
		}
		return entry.Ano + entry.Mes, true
	case PeriodicityYearly:
		if entry.Ano == "" {
			return "", false
		}
		return entry.Ano, true
	default:
		return "", false
	}
}

// detectPeriodicity infers the temporal distribution of a dataset by mapping scraped text keywords to defined enum constants.
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
