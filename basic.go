package bracc

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type SimpleJobProvider struct {
	url *url.URL
}

func (s *SimpleJobProvider) Jobs() (iter.Seq[Job], error) {
	return func(yield func(j Job) bool) {
		yield(&SimpleJob{s.url})
	}, nil
}

type SimpleJob struct {
	url *url.URL
}

func (s *SimpleJob) GetURL() *url.URL {
	return s.url
}

func (s *SimpleJob) Download(ctx context.Context, dir string) error {
	filename := filepath.Base(s.url.Path)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	req, err := http.NewRequest(http.MethodGet, s.url.String(), f)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	_, err = http.DefaultClient.Do(req)
	return err
}
