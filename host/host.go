package host

import (
	"context"
	"io"

	dockerClient "github.com/docker/docker/client"
	"github.com/spikeekips/contest/config"
)

type Host interface {
	Host() string
	DockerClient() *dockerClient.Client
	BaseDir() string
	Connect() error
	Close(context.Context) error
	Clean(context.Context, bool /* dry run */, bool /* if true, clean runnings */) error
	Prepare(string /* common node config */, *config.Vars) (map[string]interface{}, error)
	AvailablePort(string /* id */, string /* network */) (string, error)
	Nodes() map[ /* node alias */ string]*Node
	MongodbContainerID() string
	MongodbURI() string
	ShellExec(context.Context, string, []string) (io.ReadCloser /* stdout */, io.ReadCloser /* stderr */, error)
}
