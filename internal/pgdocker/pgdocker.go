// Package pgdocker creates one-off Postgres docker images to use so pggen can
// introspect the schema.
package pgdocker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/ports"
	"go.uber.org/zap"
	"io/ioutil"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

// Client is a client to control the running Postgres Docker container.
type Client struct {
	docker      *dockerClient.Client
	containerID string // container ID if started, empty otherwise
	l           *zap.SugaredLogger
	connString  string
}

// Start builds a Docker image and runs the image in a container.
func Start(ctx context.Context, l *zap.SugaredLogger) (client *Client, mErr error) {
	dockerCl, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	c := &Client{docker: dockerCl, l: l}
	imageID, err := c.buildImage(ctx)
	l.Debugf("build image ID: %s", imageID)
	if err != nil {
		return nil, fmt.Errorf("build image: %w", err)
	}
	containerID, port, err := c.runContainer(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("run container: %w", err)
	}
	defer func() {
		if client != nil && mErr != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := client.Stop(ctx); err != nil {
				c.l.Errorf("stop pgdocker client: %s", err)
			}
		}
	}()

	c.containerID = containerID
	c.connString = fmt.Sprintf("host=0.0.0.0 port=%d user=postgres", port)
	if err := c.waitIsReady(ctx); err != nil {
		return nil, fmt.Errorf("wait for postgres to be ready: %w", err)
	}
	return c, nil
}

func (c *Client) buildImage(ctx context.Context) (id string, mErr error) {
	// Create Dockerfile with template.
	dockerfileBuf := &bytes.Buffer{}
	tmpl, err := template.New("pgdocker").Parse(dockerfileTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	if err := tmpl.ExecuteTemplate(dockerfileBuf, "dockerfile", pgTemplate{}); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	c.l.Debug("wrote template into buffer")

	// Tar Dockerfile for build context.
	tarBuf := &bytes.Buffer{}
	tarW := tar.NewWriter(tarBuf)
	hdr := &tar.Header{Name: "Dockerfile", Size: int64(dockerfileBuf.Len())}
	if err := tarW.WriteHeader(hdr); err != nil {
		return "", fmt.Errorf("write dockerfile tar header: %w", err)
	}
	if _, err := tarW.Write(dockerfileBuf.Bytes()); err != nil {
		return "", fmt.Errorf("write dockerfile to tar: %w", err)
	}
	tarR := bytes.NewReader(tarBuf.Bytes())
	c.l.Debug("wrote tar dockerfile into buffer")

	// Send build request.
	opts := types.ImageBuildOptions{Dockerfile: "Dockerfile"}
	resp, err := c.docker.ImageBuild(ctx, tarR, opts)
	if err != nil {
		return "", fmt.Errorf("build postgres docker image: %w", err)
	}
	defer errs.Capture(&mErr, resp.Body.Close, "close image build response")
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read image build response: %w", err)
	}

	// Match image ID.
	imageIDRegexp := regexp.MustCompile(`Successfully built ([a-z0-9]+)`)
	matches := imageIDRegexp.FindSubmatch(response)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable find image ID in docker build output below:\n%s", string(response))
	}
	return string(matches[1]), nil
}

// runContainer creates and starts a new Postgres container using imageID.
// The postgres port is mapped to an available port on the host system.
func (c *Client) runContainer(ctx context.Context, imageID string) (string, ports.Port, error) {
	containerCfg := &container.Config{
		Image:        imageID,
		Env:          []string{"POSTGRES_HOST_AUTH_METHOD=trust"},
		ExposedPorts: nat.PortSet{"5432/tcp": struct{}{}},
	}
	port, err := ports.FindAvailable()
	if err != nil {
		return "", 0, fmt.Errorf("find available port: %w", err)
	}
	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			"5432/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(port)}},
		},
	}
	resp, err := c.docker.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return "", 0, fmt.Errorf("create container: %w", err)
	}
	containerID := resp.ID
	c.l.Debugf("created postgres container ID=%s port=%d", containerID, port)
	err = c.docker.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("start container: %w", err)
	}
	c.l.Debugf("started container ID %s", containerID)
	return containerID, port, nil
}

// waitIsReady waits until we can connect to the database.
func (c *Client) waitIsReady(ctx context.Context) error {
	connString, _ := c.ConnString()
	cfg, err := pgx.ParseConfig(connString + " connect_timeout=1")
	if err != nil {
		return fmt.Errorf("parse conn string: %w", err)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("postgres didn't start up with 10 seconds")
		case <-ctx.Done():
			return fmt.Errorf("postgres didn't start up before context expired")
		default:
			// continue
		}
		limit := time.After(200 * time.Millisecond) // debounce
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err == nil {
			if err := conn.Close(ctx); err != nil {
				c.l.Errorf("close postgres connection: %w", err)
			}
			return nil
		}
		c.l.Debugf("attempted connection: %s", err)
		<-limit
	}
}

// ConnString returns the connection string to connect to the started Postgres
// Docker container.
func (c *Client) ConnString() (string, error) {
	if c.connString == "" {
		return "", fmt.Errorf("conn string not set; did postgres start correctly")
	}
	return c.connString, nil
}

// Stop stops the running container, if any.
func (c *Client) Stop(ctx context.Context) error {
	if c.containerID == "" {
		return nil
	}
	if err := c.docker.ContainerStop(ctx, c.containerID, nil); err != nil {
		return fmt.Errorf("stop container %s: %w", c.containerID, err)
	}
	return nil
}