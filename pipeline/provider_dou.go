package pipeline

import "bracc"

func init() {
	p, err := bracc.NewDOUJobProvider(24)
	if err != nil {
		panic(err)
	}
	Providers = append(Providers, p)
}
