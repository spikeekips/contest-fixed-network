package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/contest/config"
)

type LocalHost struct {
	sync.RWMutex
	*logging.Logging
	design             config.DesignHost
	vars               *config.Vars
	nodeDesigns        map[string]string
	runner             string
	client             *dockerClient.Client
	baseDir            string
	ports              map[string]string
	nodes              map[string]*Node
	mongodbContainerID string
	mongodbURI         string
}

func NewLocalHost(
	design config.DesignHost,
	vars *config.Vars,
	nodeDesigns map[string]string,
	runner,
	baseDir string,
) *LocalHost {
	return &LocalHost{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "host").
				Str("host", design.Host)
		}),
		design:      design,
		vars:        vars,
		nodeDesigns: nodeDesigns,
		runner:      runner,
		baseDir:     baseDir,
		ports:       map[string]string{},
	}
}

func (ho *LocalHost) Host() string {
	return ho.design.Host
}

func (ho *LocalHost) DockerClient() *dockerClient.Client {
	return ho.client
}

func (ho *LocalHost) BaseDir() string {
	return ho.baseDir
}

func (ho *LocalHost) Connect() error {
	c, err := dockerClient.NewClientWithOpts(
		dockerClient.FromEnv,
	)
	if err != nil {
		return err
	}
	ho.client = c

	return ho.setRunner(ho.runner)
}

func (ho *LocalHost) Close(ctx context.Context) error {
	ho.Lock()
	defer ho.Unlock()

	var cs []dockerTypes.Container
	if err := TraverseContainers(ctx, ho.client, func(c dockerTypes.Container) (bool, error) {
		if c.State == "running" {
			cs = append(cs, c)
		}

		return true, nil
	}); err != nil {
		return err
	} else if len(cs) < 1 {
		return nil
	}

	if err := RunWaitGroup(len(cs), func(i int) error {
		return ho.client.ContainerStop(ctx, cs[i].ID, nil)
	}); err != nil {
		return err
	}

	return ho.client.Close()
}

// Clean cleans the stopped containers. If the containers are still running,
// returns error.
func (ho *LocalHost) Clean(ctx context.Context, dryrun, force bool) error {
	var cs []dockerTypes.Container
	if err := TraverseContainers(ctx, ho.client, func(c dockerTypes.Container) (bool, error) {
		if !force {
			if c.State == "running" {
				return false, errors.Errorf("founds still running node container, %q", c.ID)
			}
		}

		if !dryrun {
			cs = append(cs, c)
		}

		return true, nil
	}); err != nil {
		return err
	} else if len(cs) < 1 {
		return nil
	}

	return RunWaitGroup(len(cs), func(i int) error {
		return ho.client.ContainerRemove(ctx, cs[i].ID, dockerTypes.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         force,
		})
	})
}

func (ho *LocalHost) Prepare(common string, vars *config.Vars) (map[string]interface{}, error) {
	if vars == nil {
		return nil, errors.Errorf("empty vars")
	}

	if err := PullImages(ho.client, []string{DefaultMongodbImage, DefaultNodeImage}, false); err != nil {
		return nil, err
	} else if err := ho.launchMongodb(); err != nil {
		return nil, err
	}

	if _, err := os.Stat(filepath.Join(ho.baseDir, "runner")); os.IsNotExist(err) {
		return nil, errors.Errorf("runner does not exist, setRunner()")
	}

	if len(ho.nodeDesigns) < 1 {
		return nil, nil
	}

	shared := map[string]interface{}{}
	nodes := map[string]*Node{}

	var previousVars *config.Vars
	if vars != nil {
		previousVars = vars.Clone(nil)
	}

	for i := range ho.nodeDesigns {
		nvars := ho.newNodeVars(previousVars)
		if c, err := NewNode(i, ho); err != nil {
			return nil, err
		} else if s, err := c.Prepare(common, ho.nodeDesigns[i], nvars); err != nil {
			return nil, err
		} else {
			nodes[c.Alias()] = c

			for k := range s {
				if _, found := shared[k]; !found {
					shared[k] = map[string]interface{}{}
				}

				shared[k].(map[string]interface{})[c.Alias()] = s[k]
			}

			previousVars = nvars

			vars.Set(fmt.Sprintf("Runtime.Node.%s.Storage.URI", c.Alias()), ho.MongodbURI())
		}
	}

	ho.nodes = nodes

	for k := range previousVars.Map() {
		vars.Set(fmt.Sprintf("Design.Common.%s", k), previousVars.Map()[k])
	}

	return shared, nil
}

func (ho *LocalHost) newNodeVars(previous *config.Vars) *config.Vars {
	m := map[string]interface{}{}

	if previous != nil {
		for k := range previous.Map() {
			if k == "Self" {
				continue
			}

			m[k] = previous.Map()[k]
		}
	}

	vars := ho.vars.Clone(m)

	return vars
}

func (ho *LocalHost) AvailablePort(name, network string) (string, error) {
	ho.Lock()
	defer ho.Unlock()

	if port, found := ho.ports[name]; found {
		return port, nil
	}

	excludes := make([]string, len(ho.ports))

	var i int
	for k := range ho.ports {
		excludes[i] = ho.ports[k]
		i++
	}

	port, err := AvailablePort(network, excludes)
	if err != nil {
		return "", err
	}
	ho.ports[name] = port

	return port, nil
}

func (ho *LocalHost) Nodes() map[string]*Node {
	return ho.nodes
}

func (ho *LocalHost) MongodbContainerID() string {
	return ho.mongodbContainerID
}

func (ho *LocalHost) MongodbURI() string {
	if len(ho.mongodbURI) < 1 {
		ho.Log().Debug().Str("container_id", ho.mongodbContainerID).Msg("getting ip address of mongodb container")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if i, err := ContainerInspect(ctx, ho.client, ho.mongodbContainerID); err != nil {
			panic(err)
		} else {
			ho.mongodbURI = fmt.Sprintf("mongodb://%s:27017", i.NetworkSettings.IPAddress)

			ho.Log().Debug().Str("uri", ho.mongodbURI).Msg("mongodb uri")
		}
	}

	return ho.mongodbURI
}

func (*LocalHost) ShellExec(ctx context.Context, name string, args []string) (io.ReadCloser, io.ReadCloser, error) {
	nctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(nctx, name, args...) // nolint:gosec
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return ioutil.NopCloser(&stdout), ioutil.NopCloser(&stderr), err
}

func (ho *LocalHost) setRunner(f string) error {
	var source, dest *os.File
	var sourceStat os.FileInfo
	if s, err := os.Open(filepath.Clean(f)); err != nil {
		return errors.Wrap(err, "failed to read runner file")
	} else if fi, err := s.Stat(); err != nil {
		return errors.Wrap(err, "failed to read runner file")
	} else {
		source = s
		sourceStat = fi
	}

	dest, err := os.OpenFile(
		filepath.Join(ho.baseDir, "runner"),
		os.O_RDWR|os.O_CREATE, sourceStat.Mode(),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create new runner file")
	}

	buf := make([]byte, 1000000)
	for {
		n, err := source.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return errors.Wrap(err, "failed to copy runner")
		}
		if n == 0 {
			break
		}

		if _, err := dest.Write(buf[:n]); err != nil {
			return errors.Wrap(err, "failed to copy runner")
		}
	}

	_ = source.Close()
	_ = dest.Close()

	return nil
}

func (ho *LocalHost) launchMongodb() error {
	if err := ho.createMongodb(); err != nil {
		return err
	} else if err := ho.startMongodb(); err != nil {
		return err
	}

	return nil
}

func (ho *LocalHost) createMongodb() error {
	source, _ := nat.NewPort("tcp", "27017")

	r, err := ho.client.ContainerCreate(
		context.Background(),
		&container.Config{
			Tty:   false,
			Image: DefaultMongodbImage,
			Labels: map[string]string{
				ContainerLabel: ContainerLabelMongodb,
			},
			ExposedPorts: nat.PortSet{source: struct{}{}},
		},
		nil,
		nil,
		nil,
		MongodbContainerName(),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create mongodb container")
	}
	ho.mongodbContainerID = r.ID

	return nil
}

func (ho *LocalHost) startMongodb() error {
	if len(ho.mongodbContainerID) < 1 {
		return errors.Errorf("create mongodb container first")
	}

	return ho.client.ContainerStart(
		context.Background(),
		ho.mongodbContainerID,
		dockerTypes.ContainerStartOptions{},
	)
}
