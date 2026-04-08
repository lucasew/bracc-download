package provider

import (
	"context"
	"iter"
	"net/url"
)

const zipURL = "https://dados.cvm.gov.br/dados/PROCESSO/SANCIONADOR/DADOS/processo_sancionador.zip"

func init() {
	Providers = append(Providers, &CVMProvider{})
}

type CVMProvider struct{}

func (p *CVMProvider) GetURL() *url.URL {
	u, _ := url.Parse(zipURL)
	return u
}

func (p *CVMProvider) Jobs(ctx context.Context) (iter.Seq[Job], error) {
	jobProvider, err := NewSimpleJobProvider(zipURL)
	if err != nil {
		return nil, err
	}
	return jobProvider.Jobs(ctx)
}
