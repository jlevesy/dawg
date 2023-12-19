package generator

import (
	"context"
	"errors"
	"net/url"
	"path"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	ociScheme = "oci"

	wasmLayer = "application/vnd.wasm.content.layer.v1+wasm"
)

type ociLoader struct {
}

func newOciLoader() (*ociLoader, error) {
	return &ociLoader{}, nil
}

func (o *ociLoader) Load(ctx context.Context, url *url.URL) (*Generator, error) {
	ref, err := registry.ParseReference(path.Join(url.Host, url.Path))
	if err != nil {
		return nil, err
	}

	repo, err := remote.NewRepository(path.Join(ref.Registry, ref.Repository))
	if err != nil {
		return nil, err
	}

	// TODO(jly): make this configurable.
	repo.PlainHTTP = true

	resolvedRef, err := repo.Resolve(ctx, ref.ReferenceOrDefault())
	if err != nil {
		return nil, err
	}

	successors, err := content.Successors(ctx, repo, resolvedRef)
	if err != nil {
		return nil, err
	}

	layer, ok := findWasmSuccessor(successors)
	if !ok {
		return nil, errors.New("no wasm layer")
	}

	desc, err := repo.Blobs().Resolve(ctx, layer.Digest.String())
	if err != nil {
		return nil, err
	}

	buf, err := content.FetchAll(ctx, repo, desc)
	if err != nil {
		return nil, err
	}

	return &Generator{Bin: buf}, nil
}

func findWasmSuccessor(successors []ocispec.Descriptor) (ocispec.Descriptor, bool) {
	for _, s := range successors {
		if s.MediaType == wasmLayer {
			return s, true
		}
	}

	return ocispec.Descriptor{}, false
}
