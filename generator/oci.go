package generator

import (
	"context"
	"errors"
	"net/url"
)

const (
	ociScheme = "oci"
)

type ociLoader struct {
}

func newOciLoader() (*ociLoader, error) {
	return &ociLoader{}, nil
}

func (o *ociLoader) Load(ctx context.Context, url *url.URL) (*Generator, error) {
	return nil, errors.New("implement meeeee")
}
