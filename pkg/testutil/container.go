package testutil

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
)

var (
	RegistryContainerConfig = ContainerConfig{
		Image:           "registry:2",
		Suffix:          "-registry",
		ExposedPort:     "5000/tcp",
		HealthcheckPath: "/",
	}

	KWOKContainerConfig = ContainerConfig{
		Suffix:          "kwok",
		Image:           "registry.k8s.io/kwok/cluster:v0.4.0-k8s.v1.28.0",
		ExposedPort:     "8080/tcp",
		HealthcheckPath: "/",
		PortBindings:    []string{"0:8080/tcp"},
	}
)

type ContainerInstance struct {
	docker      *dockerclient.Client
	ContainerID string
	Port        string
}

func (r *ContainerInstance) Shutdown(ctx context.Context) error {
	return r.docker.ContainerRemove(
		ctx,
		r.ContainerID,
		types.ContainerRemoveOptions{Force: true},
	)
}

type ContainerConfig struct {
	Suffix          string
	Image           string
	ExposedPort     string
	HealthcheckPath string
	PortBindings    []string
}

func RunContainer(t *testing.T, cfg ContainerConfig) *ContainerInstance {
	t.Helper()

	ctx := context.Background()

	docker, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	require.NoError(t, err)

	pullOutput, err := docker.ImagePull(ctx, cfg.Image, types.ImagePullOptions{})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, pullOutput.Close())
	}()

	_, err = io.Copy(io.Discard, pullOutput)
	require.NoError(t, err)

	exposedPorts, portBindings, err := nat.ParsePortSpecs(cfg.PortBindings)
	require.NoError(t, err)

	containerName := strings.ToLower(t.Name()) + "-" + cfg.Suffix

	container, err := docker.ContainerCreate(
		ctx,
		&container.Config{
			Hostname:     containerName,
			Image:        cfg.Image,
			ExposedPorts: nat.PortSet(exposedPorts),
		},
		&container.HostConfig{
			PublishAllPorts: true,
			PortBindings:    nat.PortMap(portBindings),
		},
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

	portInfo, ok := containerInfo.NetworkSettings.Ports[nat.Port(cfg.ExposedPort)]

	require.True(t, ok, "Container does not export port", cfg.ExposedPort)

	for _, port := range portInfo {
		hostPort = port.HostPort
		if hostPort != "" {
			break
		}
	}

	require.NotEmpty(t, hostPort, "No host port found for registry port", cfg.ExposedPort)

	// ping the container before running the test.
	for {
		if _, err := http.Head("http://localhost:" + hostPort + cfg.HealthcheckPath); err == nil {
			break
		}

		t.Log("Container is not accepting connections, will retry...")
		time.Sleep(500 * time.Millisecond)
	}

	return &ContainerInstance{
		docker:      docker,
		ContainerID: container.ID,
		Port:        hostPort,
	}
}
