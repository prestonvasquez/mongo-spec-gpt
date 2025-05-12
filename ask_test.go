package mongospecgpt

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TODO

type atlasContainer struct {
	testcontainers.Container
	URI string
}

func setupAtlas(ctx context.Context) (*atlasContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongodb/mongodb-atlas-local",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(1 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	var atlasC *atlasContainer
	if container != nil {
		atlasC = &atlasContainer{Container: container}
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return atlasC, err
	}

	mappedPort, err := container.MappedPort(ctx, "27017")
	if err != nil {
		return atlasC, err
	}

	uri := &url.URL{
		Scheme:   "mongodb",
		Host:     net.JoinHostPort(ip, mappedPort.Port()),
		Path:     "/",
		RawQuery: "directConnection=true",
	}

	atlasC.URI = uri.String()

	return atlasC, nil
}
