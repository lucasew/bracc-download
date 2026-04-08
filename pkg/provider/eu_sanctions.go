package provider

import (
	"context"
	"iter"
	"net/url"
	"os"
	"strings"
)

const baseURL = "https://webgate.ec.europa.eu/fsd/fsf/public/files/csvFullSanctionsList/content"
const defaultToken = "dG9rZW4tMjAxNw"

func init() {
	Providers = append(Providers, &EUSanctionsProvider{})
}

type EUSanctionsProvider struct{}

func (p *EUSanctionsProvider) GetURL() *url.URL {
	u, _ := url.Parse(baseURL)
	return u
}

func (p *EUSanctionsProvider) Jobs(ctx context.Context) (iter.Seq[Job], error) {
	token := strings.TrimSpace(os.Getenv("EU_SANCTIONS_TOKEN"))
	if token == "" {
		token = defaultToken
	}

	jobProvider, err := NewSimpleJobProvider(baseURL + "?token=" + token)
	if err != nil {
		return nil, err
	}
	return jobProvider.Jobs(ctx)
}
