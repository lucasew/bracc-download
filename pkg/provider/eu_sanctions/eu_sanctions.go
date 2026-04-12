package eu_sanctions

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"context"
	"iter"
	"net/url"
	"os"
	"strings"
)

const baseURL = "https://webgate.ec.europa.eu/fsd/fsf/public/files/csvFullSanctionsList/content"

// SECURITY-NOTE: intentional public token for open dataset
const defaultToken = "dG9rZW4tMjAxNw"

func init() {
	provider.Providers = append(provider.Providers, &Provider{})
}

type Provider struct{}

func (p *Provider) GetURL() *url.URL {
	u, _ := url.Parse(baseURL)
	return u
}

func (p *Provider) Jobs(ctx context.Context) (iter.Seq[provider.Job], error) {
	token := strings.TrimSpace(os.Getenv("EU_SANCTIONS_TOKEN"))
	if token == "" {
		token = defaultToken
	}

	jobProvider, err := simple.NewSimpleJobProvider(baseURL + "?token=" + token)
	if err != nil {
		return nil, err
	}
	return jobProvider.Jobs(ctx)
}
