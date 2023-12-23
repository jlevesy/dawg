package generator

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
)

const fileScheme = "file"

type fileStore struct{}

func (f *fileStore) Load(_ context.Context, url *url.URL) (*Generator, error) {
	bin, err := os.ReadFile(url.Path)
	if err != nil {
		return nil, err
	}

	return &Generator{Bin: bin}, nil
}

func (f *fileStore) Store(_ context.Context, url *url.URL, g *Generator) error {
	if err := os.MkdirAll(filepath.Join(url.Host, filepath.Dir(url.Path)), 0600); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(url.Host, url.Path), g.Bin, 0600)
}
