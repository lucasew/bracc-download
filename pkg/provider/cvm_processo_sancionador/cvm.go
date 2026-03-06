package cvm_processo_sancionador

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
	"iter"
	"net/url"
)

const zipURL = "https://dados.cvm.gov.br/dados/PROCESSO/SANCIONADOR/DADOS/processo_sancionador.zip"

func init() {
	provider.Providers = append(provider.Providers, &Provider{})
}

type Provider struct{}

func (p *Provider) GetURL() *url.URL {
	u, _ := url.Parse(zipURL)
	return u
}

func (p *Provider) Jobs() (iter.Seq[provider.Job], error) {
	jobProvider, err := simple.NewSimpleJobProvider(zipURL)
	if err != nil {
		return nil, err
	}
	return jobProvider.Jobs()
}
