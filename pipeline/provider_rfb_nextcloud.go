package pipeline

import "bracc"

func init() {
	p, err := bracc.NewWebDAVJobProvider("https://arquivos.receitafederal.gov.br/public.php/dav/files/gn672Ad4CF8N6TK/")
	if err != nil {
		panic(err)
	}
	Providers = append(Providers, p)
}
