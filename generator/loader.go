package generator

import (
	"context"
	"fmt"
	"net/url"
)

// Loader allows to retrieve a generator code based on an URI.
type Loader interface {
	Load(ctx context.Context, url *url.URL) (*Generator, error)
}

type unsuportedSchemeError string

func (u unsuportedSchemeError) Error() string {
	return fmt.Sprintf("cannot load a generator with the scheme %q", string(u))
}

// SchemeProvider provides using a specific Provider based on  the URL scheme.
type schemeLoader map[string]Loader

func (s schemeLoader) Load(ctx context.Context, url *url.URL) (*Generator, error) {
	loader, ok := s[url.Scheme]
	if !ok {
		return nil, unsuportedSchemeError(url.Scheme)
	}

	return loader.Load(ctx, url)
}

func DefaultLoader() (Loader, error) {
	ociLoader, err := newOciLoader()
	if err != nil {
		return nil, err
	}

	return schemeLoader{
		fileScheme: &fileLoader{},
		ociScheme:  ociLoader,
	}, nil
}
