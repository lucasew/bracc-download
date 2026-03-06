package rfb_nextcloud

import (
	"bracc/pkg/provider"
	"bracc/pkg/provider/webdav"
)

func init() {
	p, err := NewProvider()
	if err != nil {
		panic(err)
	}
	provider.Providers = append(provider.Providers, p)
}

func NewProvider() (*webdav.WebDAVJobProvider, error) {
	return webdav.NewWebDAVJobProvider("https://arquivos.receitafederal.gov.br/public.php/dav/files/gn672Ad4CF8N6TK/")
}
