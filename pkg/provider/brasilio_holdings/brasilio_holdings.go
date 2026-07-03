package brasilio_holdings

import (
	"bracc/pkg/errorreporter"
	"bracc/pkg/provider"
	"bracc/pkg/provider/simple"
)

const primaryURL = "https://data.brasil.io/dataset/socios-brasil/holding.csv.gz"

func init() {
	p, err := NewProvider()
	if err != nil {
		errorreporter.ReportError(err)
		panic(err)
	}
	provider.Providers = append(provider.Providers, p)
}

func NewProvider() (*simple.SimpleJobProvider, error) {
	return simple.NewSimpleJobProvider(primaryURL)
}
