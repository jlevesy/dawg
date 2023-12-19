package generator

import (
	"context"
	"net/url"
	"os"
)

const fileScheme = "file"

type fileLoader struct{}

func (f *fileLoader) Load(_ context.Context, url *url.URL) (*Generator, error) {
	bin, err := os.ReadFile(url.Path)
	if err != nil {
		return nil, err
	}

	return &Generator{Bin: bin}, nil
}
