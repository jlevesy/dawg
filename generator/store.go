package generator

import (
	"context"
	"fmt"
	"net/url"
)

// Reader allows to retrieve a generator code based on an URL.
type Reader interface {
	Load(context.Context, *url.URL) (*Generator, error)
}

// Writer allows to store a generator at an URL.
type Writer interface {
	Store(context.Context, *url.URL, *Generator) error
}

// Store represents any storage allowing to load or store a generator.
type Store interface {
	Reader
	Writer
}

type unsupportedSchemeError string

func (e unsupportedSchemeError) Error() string {
	return fmt.Sprintf("unsupported scheme %q", string(e))
}

// schemeStore provides using a specific Provider based on  the URL scheme.
type schemeStore map[string]Store

func (s schemeStore) Load(ctx context.Context, url *url.URL) (*Generator, error) {
	st, ok := s[url.Scheme]
	if !ok {
		return nil, unsupportedSchemeError(url.Scheme)
	}

	return st.Load(ctx, url)
}

func (s schemeStore) Store(ctx context.Context, url *url.URL, g *Generator) error {
	st, ok := s[url.Scheme]
	if !ok {
		return unsupportedSchemeError(url.Scheme)
	}

	return st.Store(ctx, url, g)
}

func DefaultStore() (Store, error) {
	registryStore := newRegistryStore()
	return &schemeStore{
		fileScheme:     &fileStore{},
		registryScheme: registryStore,
	}, nil
}
