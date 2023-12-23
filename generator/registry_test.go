package generator_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/jlevesy/dawg/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Registry(t *testing.T) {
	var (
		ctx = context.Background()
		gen = generator.Generator{
			Bin: []byte("coucou"),
		}
	)

	ts := runTestRegistry(t)
	t.Cleanup(func() {
		require.NoError(t, ts.Close())
	})

	genStore, err := generator.DefaultStore()
	require.NoError(t, err)

	genUrl, err := url.Parse("registry://localhost:" + ts.port + "/testgenerators/test:v0.0.1")
	require.NoError(t, err)

	err = genStore.Store(ctx, genUrl, &gen)
	require.NoError(t, err)

	gotGen, err := genStore.Load(ctx, genUrl)
	require.NoError(t, err)

	assert.Equal(t, &gen, gotGen)
}

type registryInstance struct {
	docker      *dockerclient.Client
	containerID string
	port        string
}

func (r *registryInstance) Close() error {
	return r.docker.ContainerRemove(
		context.Background(),
		r.containerID,
		types.ContainerRemoveOptions{Force: true},
	)
}

func runTestRegistry(t *testing.T) *registryInstance {
	t.Helper()

	ctx := context.Background()

	docker, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	require.NoError(t, err)

	pullOutput, err := docker.ImagePull(ctx, "registry:2", types.ImagePullOptions{})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, pullOutput.Close())
	}()

	_, err = io.Copy(io.Discard, pullOutput)
	require.NoError(t, err)

	containerName := strings.ToLower(t.Name()) + "-registry"

	container, err := docker.ContainerCreate(
		ctx,
		&container.Config{
			Hostname: containerName,
			Image:    "registry:2",
		},
		&container.HostConfig{PublishAllPorts: true},
		&network.NetworkingConfig{},
		nil,
		containerName,
	)
	require.NoError(t, err)

	err = docker.ContainerStart(ctx, container.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	containerInfo, err := docker.ContainerInspect(ctx, container.ID)
	require.NoError(t, err)

	var hostPort string

	portInfo, ok := containerInfo.NetworkSettings.Ports["5000/tcp"]
	require.True(t, ok, "Registry does not export 5000/tcp")

	for _, port := range portInfo {
		hostPort = port.HostPort
		if hostPort != "" {
			break
		}
	}

	require.NotEmpty(t, hostPort, "No host port found for registry port 5000")

	// ping the registry before running the test.
	for {
		if _, err := http.Head("http://localhost:" + hostPort + "/"); err == nil {
			break
		}

		t.Log("Registry is not accepting connections yet, retrying...")
		time.Sleep(500 * time.Millisecond)
	}

	return &registryInstance{
		docker:      docker,
		containerID: container.ID,
		port:        hostPort,
	}
}
