package provider


const primaryURL = "https://data.brasil.io/dataset/socios-brasil/holding.csv.gz"

func init() {
	p, err := NewBrasilioHoldingsProvider()
	if err != nil {
		panic(err)
	}
	Providers = append(Providers, p)
}

func NewBrasilioHoldingsProvider() (*SimpleJobProvider, error) {
	return NewSimpleJobProvider(primaryURL)
}
