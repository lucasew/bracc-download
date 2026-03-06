package rfb_nextcloud

import (
	"bracc"
	"bracc/webdav"
)

func init() {
	provider, err := NewProvider()
	if err != nil {
		panic(err)
	}
	bracc.Providers = append(bracc.Providers, provider)
}

func NewProvider() (*webdav.WebDAVJobProvider, error) {
	return webdav.NewWebDAVJobProvider("https://arquivos.receitafederal.gov.br/public.php/dav/files/gn672Ad4CF8N6TK/")
}
