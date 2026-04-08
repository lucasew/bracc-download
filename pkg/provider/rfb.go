package provider

func init() {
	p, err := NewRFBProvider()
	if err != nil {
		panic(err)
	}
	Providers = append(Providers, p)
}

func NewRFBProvider() (*WebDAVJobProvider, error) {
	return NewWebDAVJobProvider("https://arquivos.receitafederal.gov.br/public.php/dav/files/gn672Ad4CF8N6TK/")
}
