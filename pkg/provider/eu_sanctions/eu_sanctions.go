package eu_sanctions

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"iter"
	"net/url"
	"os"
	"strings"
)

const baseURL = "https://webgate.ec.europa.eu/fsd/fsf/public/files/csvFullSanctionsList/content"
const defaultToken = "dG9rZW4tMjAxNw"

func init() {
	provider.Providers = append(provider.Providers, &Provider{})
}

type Provider struct{}

func (p *Provider) GetURL() *url.URL {
	u, _ := url.Parse(baseURL)
	return u
}

func (p *Provider) Jobs() (iter.Seq[provider.Job], error) {
	token := strings.TrimSpace(os.Getenv("EU_SANCTIONS_TOKEN"))
	if token == "" {
		token = defaultToken
	}

	jobProvider, err := simple.NewSimpleJobProvider(baseURL + "?token=" + token)
	if err != nil {
		return nil, err
	}
	return jobProvider.Jobs()
}
