package generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	registryScheme = "registry"

	mediaTypeWasmLayer    = "application/vnd.wasm.content.layer.v1+wasm"
	atrifactTypeGenerator = "application/vnd.dawg.generator.v1"
)

var (
	defaultRegistrySettings = registrySettings{
		PlainHTTP: false,
	}
	defaultRegistriesSettings = map[string]registrySettings{
		"dawg-dev.localhost": {
			PlainHTTP: true,
		},
		"localhost": {
			PlainHTTP: true,
		},
	}
)

type registrySettings struct {
	PlainHTTP bool
	// TODO handle registry auth.
}

type registryStore struct {
	localStore         oras.Target
	registriesSettings map[string]registrySettings
}

func newRegistryStore() *registryStore {
	// TODO(jly): configure store.
	// TODO(jly): use filesystem local store!?
	return &registryStore{
		localStore:         memory.New(),
		registriesSettings: defaultRegistriesSettings,
	}
}

func (st *registryStore) Store(ctx context.Context, url *url.URL, gen *Generator) error {
	blobDescriptor := newDescriptorFromGenerator(gen)

	if err := st.localStore.Push(ctx, blobDescriptor, bytes.NewReader(gen.Bin)); err != nil {
		return err
	}

	manifestDescriptor, err := oras.PackManifest(
		ctx,
		st.localStore,
		oras.PackManifestVersion1_1_RC4,
		atrifactTypeGenerator,
		oras.PackManifestOptions{
			Layers: []v1.Descriptor{blobDescriptor},
		},
	)
	if err != nil {
		return err
	}

	repo, err := st.repoWithSettings(url)
	if err != nil {
		return err
	}

	// TODO(jly): handle empty ref!!!!!

	if err := st.localStore.Tag(ctx, manifestDescriptor, repo.Reference.Reference); err != nil {
		return err
	}

	if _, err := oras.Copy(
		ctx,
		st.localStore,
		repo.Reference.Reference,
		repo,
		repo.Reference.Reference,
		oras.DefaultCopyOptions,
	); err != nil {
		return err
	}

	return nil
}

func (st *registryStore) Load(ctx context.Context, url *url.URL) (*Generator, error) {
	repo, err := st.repoWithSettings(url)
	if err != nil {
		return nil, err
	}

	manifestDescriptor, err := oras.Copy(
		ctx,
		repo,
		repo.Reference.ReferenceOrDefault(),
		st.localStore,
		repo.Reference.ReferenceOrDefault(),
		oras.DefaultCopyOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("could not pull generator from registry: %w", err)
	}

	successors, err := content.Successors(ctx, repo, manifestDescriptor)
	if err != nil {
		return nil, err
	}

	layer, ok := findWasmSuccessor(successors)
	if !ok {
		return nil, errors.New("no wasm layer")
	}

	content, err := st.localStore.Fetch(ctx, layer)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = content.Close()
	}()

	buf, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}

	return &Generator{Bin: buf}, nil
}

func (st *registryStore) repoWithSettings(url *url.URL) (*remote.Repository, error) {
	registrySettings, ok := st.registriesSettings[url.Hostname()]
	if !ok {
		registrySettings = defaultRegistrySettings
	}

	repo, err := remote.NewRepository(path.Join(url.Host, url.Path))
	if err != nil {
		return nil, err
	}

	repo.PlainHTTP = registrySettings.PlainHTTP

	return repo, nil
}

func newDescriptorFromGenerator(g *Generator) ocispec.Descriptor {
	return ocispec.Descriptor{
		MediaType: mediaTypeWasmLayer,
		Digest:    digest.FromBytes(g.Bin),
		Size:      int64(len(g.Bin)),
	}
}

func findWasmSuccessor(successors []ocispec.Descriptor) (ocispec.Descriptor, bool) {
	for _, s := range successors {
		if s.MediaType == mediaTypeWasmLayer {
			return s, true
		}
	}

	return ocispec.Descriptor{}, false
}
