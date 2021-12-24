package host

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
)

var (
	ContainerLabel             = "mitum-contest"
	ContainerLabelMongodb      = "mongodb"
	ContainerLabelNode         = "node"
	ContainerLabelNodeAlias    = ContainerLabel + "-node"
	ContainerLabelNodeType     = ContainerLabel + "-type"
	ContainerLabelNodeInitType = "init"
	ContainerLabelNodeRunType  = "run"

	DefaultNodeImage    = "debian:testing-slim"
	DefaultMongodbImage = "mongo"
)

var ContainerLogIgnoreError = util.NewError("failed to read container logs; ignored")

func MongodbContainerName() string {
	return "contest-mongodb"
}

func NodeInitContainerName(alias string) string {
	return fmt.Sprintf("contest-node-init-%s", alias)
}

func NodeRunContainerName(alias string) string {
	return fmt.Sprintf("contest-node-run-%s", alias)
}

func NodeCustomContainerName(alias string) string {
	return fmt.Sprintf("contest-node-custom-%s-%s", util.UUID().String(), alias)
}

func TraverseContainers(
	ctx context.Context, client *dockerClient.Client, callback func(dockerTypes.Container,
	) (bool, error)) error {
	cs, err := client.ContainerList(
		ctx,
		dockerTypes.ContainerListOptions{
			All: true,
		},
	)
	if err != nil {
		return err
	}

	for i := range cs {
		c := cs[i]

		var found bool
		for k := range c.Labels {
			if strings.HasPrefix(k, ContainerLabel) {
				found = true

				break
			}
		}

		if !found {
			continue
		}

		if keep, err := callback(c); err != nil {
			return err
		} else if !keep {
			return nil
		}
	}

	return nil
}

func PullImages(client *dockerClient.Client, images []string, update bool) error {
	for _, image := range images {
		if err := PullImage(client, image, update); err != nil {
			return err
		}
	}

	return nil
}

func PullImage(client *dockerClient.Client, image string, update bool) error {
	if !update {
		if i, err := client.ImageList(
			context.Background(),
			dockerTypes.ImageListOptions{
				All: true,
				Filters: filters.NewArgs(filters.KeyValuePair{
					Key:   "reference",
					Value: image,
				}),
			},
		); err != nil {
			return err
		} else if len(i) > 0 {
			return nil
		}
	}

	if _, err := client.ImagePull(context.Background(), image, dockerTypes.ImagePullOptions{}); err != nil {
		return err
	}

	return nil
}

func ReadContainerLogs(
	ctx context.Context,
	client *dockerClient.Client,
	id string,
	options dockerTypes.ContainerLogsOptions,
	callback func(uint8, []byte),
) error {
	options.Timestamps = true

	var timestamp string
	for {
		options.Since = timestamp

		reader, err := client.ContainerLogs(ctx, id, options)
		if err != nil {
			return err
		}

		if t, err := readContainerLogs(ctx, reader, callback); err != nil {
			switch {
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				return nil
			case errors.Is(err, ContainerLogIgnoreError):
				<-time.After(time.Millisecond * 600)

				timestamp = t

				continue
			case errors.Is(err, io.EOF):
				return nil
			default:
				timestamp = t

				continue
			}
		}
	}
}

func readContainerLogs(ctx context.Context, reader io.Reader, callback func(uint8, []byte)) (string, error) {
	var timestamp, msg string
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			h := make([]byte, 8)
			if _, err := reader.Read(h); err != nil {
				return timestamp, err
			}

			count := binary.BigEndian.Uint32(h[4:])
			l := make([]byte, count)
			if _, err := reader.Read(l); err != nil {
				if bytes.Contains(l, []byte("Error grabbing logs")) {
					_, _ = fmt.Fprintf(os.Stderr, "grabbing error: %q\n", string(l))

					return timestamp, ContainerLogIgnoreError.Errorf("%s: %w", l, err)
				}

				return timestamp, errors.Wrapf(err, "failed to read logs body, %q", string(l))
			}

			s := strings.SplitN(string(l[:len(l)-1]), " ", 2)
			timestamp, msg = s[0], s[1]

			callback(h[0], []byte(msg))
		}
	}
}

func ContainerInspect(ctx context.Context, client *dockerClient.Client, id string) (dockerTypes.ContainerJSON, error) {
	return client.ContainerInspect(ctx, id)
}
