package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/hpcloud/tail"
	"github.com/rs/zerolog"
	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
	"github.com/spikeekips/mitum/util/logging"
)

type LoadAction func(context.Context, config.DesignAction) (host.Action, error)

var ActionLoaders = map[string]LoadAction{
	"init-nodes":   initNodesActionFunc,
	"start-nodes":  startNodesActionFunc,
	"custom-nodes": customNodesActionFunc,
	"stop-nodes":   stopNodesActionFunc,
	"kill":         killActionFunc,
	"host-command": hostCommandActionFunc,
}

var initNodesActionFunc = func(ctx context.Context, design config.DesignAction) (host.Action, error) {
	var hs *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hs); err != nil {
		return nil, err
	}

	var nodes []string
	switch i, err := findNodesFromDesign(design); {
	case err != nil:
		return nil, err
	case len(i) < 1:
		return nil, errors.Errorf("empty nodes")
	default:
		nodes = i
	}

	return NewInitNodesAction(ctx, nodes)
}

var startNodesActionFunc = func(ctx context.Context, design config.DesignAction) (host.Action, error) {
	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	var hs *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hs); err != nil {
		return nil, err
	}

	var nodes []string
	switch i, err := findNodesFromDesign(design); {
	case err != nil:
		return nil, err
	case len(i) < 1:
		return nil, errors.Errorf("empty nodes")
	default:
		nodes = i
	}

	return NewStartNodesAction(ctx, nodes, design.Args)
}

var customNodesActionFunc = func(ctx context.Context, design config.DesignAction) (host.Action, error) {
	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	var hs *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hs); err != nil {
		return nil, err
	}

	var nodes []string
	switch i, err := findNodesFromDesign(design); {
	case err != nil:
		return nil, err
	case len(i) < 1:
		return nil, errors.Errorf("empty nodes")
	default:
		nodes = i
	}

	return NewCustomNodesAction(ctx, nodes, design.Args)
}

var stopNodesActionFunc = func(ctx context.Context, design config.DesignAction) (host.Action, error) {
	var hs *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hs); err != nil {
		return nil, err
	}

	var nodes []string
	switch i, err := findNodesFromDesign(design); {
	case err != nil:
		return nil, err
	case len(i) < 1:
		return nil, errors.Errorf("empty nodes")
	default:
		nodes = i
	}

	return NewStopNodesAction(ctx, nodes)
}

var killActionFunc = func(ctx context.Context, design config.DesignAction) (host.Action, error) {
	var exitChan chan error
	if err := LoadExitChanContextValue(ctx, &exitChan); err != nil {
		return nil, err
	}

	var err error
	if es, found := design.Extra["error"]; found {
		err = errors.Errorf(es.(string))
	}

	return KillAction{exitChan: exitChan, err: err}, nil
}

var hostCommandActionFunc = func(ctx context.Context, design config.DesignAction) (host.Action, error) {
	return NewHostCommandAction(ctx, design.Args)
}

type BaseNodesAction struct {
	*logging.Logging
	name  string
	nodes []*host.Node
	lo    *host.LogSaver
	vars  *config.Vars
	args  []string
}

func NewBaseNodesAction(ctx context.Context, name string, aliases []string, args []string) (*BaseNodesAction, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var lo *host.LogSaver
	if err := host.LoadLogSaverContextValue(ctx, &lo); err != nil {
		return nil, err
	}

	var hosts *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hosts); err != nil {
		return nil, err
	}

	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	nodes, err := filterNodes(hosts, aliases)
	if err != nil {
		return nil, err
	}

	action := &BaseNodesAction{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", fmt.Sprintf("%s-action", name)).Strs("nodes", aliases)
		}),
		name:  name,
		nodes: nodes,
		lo:    lo,
		vars:  vars,
		args:  args,
	}

	_ = action.SetLogging(log)

	return action, nil
}

func (ac *BaseNodesAction) Name() string {
	return ac.name
}

func (ac BaseNodesAction) Map() map[string]interface{} {
	nodes := make([]string, len(ac.nodes))
	for i := range ac.nodes {
		nodes[i] = ac.nodes[i].Alias()
	}

	return map[string]interface{}{
		"name":  ac.name,
		"nodes": nodes,
	}
}

func (ac *BaseNodesAction) createContainer(
	ctx context.Context,
	node *host.Node,
	commands []string,
	name,
	t string,
) (string, error) {
	hostConfig, err := ac.hostConfig(node)
	if err != nil {
		return "", err
	}

	r, err := node.Host().DockerClient().ContainerCreate(
		ctx,
		ac.mainConfig(node, commands, t),
		hostConfig,
		nil,
		nil,
		name,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create container")
	}
	return r.ID, nil
}

func (*BaseNodesAction) startContainer(ctx context.Context, node *host.Node, id string) error {
	client := node.Host().DockerClient()
	return client.ContainerStart(
		ctx,
		id,
		dockerTypes.ContainerStartOptions{},
	)
}

func (ac *BaseNodesAction) containerLogs(ctx context.Context, node *host.Node, id string) error {
	t, err := tail.TailFile(node.LogFile(), tail.Config{Follow: true})
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	go func() {
		for l := range t.Lines {
			if l.Err != nil {
				break
			}

			if e, err := host.NewNodeLogEntry(node.Alias(), []byte(l.Text), false); err != nil {
				ac.Log().Error().Err(err).Msg("failed to create LogEntry")
			} else {
				ac.lo.LogEntryChan() <- e
			}
		}
	}()

	go func() {
		options := dockerTypes.ContainerLogsOptions{
			ShowStdout: false,
			ShowStderr: true,
			Follow:     true,
			Tail:       "all",
		}

		err := host.ReadContainerLogs(ctx, node.Host().DockerClient(), id, options, func(status uint8, b []byte) {
			if e, err := host.NewNodeLogEntry(node.Alias(), b, status == 2); err != nil {
				ac.Log().Error().Err(err).Msg("failed to create LogEntry")
			} else {
				ac.lo.LogEntryChan() <- e
			}
		})
		if err != nil {
			ac.Log().Error().Err(err).Msg("failed to read container log")
		}
	}()

	return nil
}

func (*BaseNodesAction) waitContainer(
	ctx context.Context,
	node *host.Node,
	id string,
	condition container.WaitCondition,
) (host.NodeExistedMsg, error) {
	statusChan, errChan := node.Host().DockerClient().ContainerWait(ctx, id, condition)

	select {
	case err := <-errChan:
		return host.NodeExistedMsg{}, err
	case status := <-statusChan:
		var err error
		switch {
		case status.Error != nil:
			err = errors.Errorf("exited: %q", status.Error.Message)
		case status.StatusCode != 0:
			err = errors.Errorf("abnormally exited with status code, %v", status.StatusCode)
		}

		return host.NodeExistedMsg{StatusCode: status.StatusCode, Err: err}, nil
	}
}

func (*BaseNodesAction) mainConfig(node *host.Node, commands []string, t string) *container.Config {
	portSet := nat.PortSet{}
	for source := range node.PortMap() {
		portSet[source] = struct{}{}
	}

	return &container.Config{
		Cmd:        commands,
		WorkingDir: "/",
		Tty:        false,
		Image:      host.DefaultNodeImage,
		Labels: map[string]string{
			host.ContainerLabel:          host.ContainerLabelNode,
			host.ContainerLabelNodeAlias: node.Alias(),
			host.ContainerLabelNodeType:  t,
		},
		ExposedPorts: portSet,
	}
}

func (*BaseNodesAction) hostConfig(node *host.Node) (*container.HostConfig, error) {
	sharedDir := node.Host().BaseDir()
	dataDir := filepath.Join(sharedDir, node.Alias())
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return nil, errors.Errorf("failed to create data directory, %q", dataDir)
		}
	}

	return &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   node.ConfigFile(),
				Target:   "/config.yml",
				ReadOnly: true,
			},
			{
				Type:     mount.TypeBind,
				Source:   filepath.Join(node.Host().BaseDir(), "runner"),
				Target:   "/runner",
				ReadOnly: true,
			},
			{
				Type:     mount.TypeBind,
				Source:   dataDir,
				Target:   "/data",
				ReadOnly: false,
			},
			{
				Type:     mount.TypeBind,
				Source:   sharedDir,
				Target:   "/shared",
				ReadOnly: false,
			},
			{
				Type:   mount.TypeBind,
				Source: node.LogFile(),
				Target: "/log",
			},
		},
		PortBindings: node.PortMap(),
		Links: []string{
			node.Host().MongodbContainerID() + ":storage",
		},
	}, nil
}

func (ac BaseNodesAction) compileArgs() ([]string, error) {
	if len(ac.args) < 1 {
		return nil, nil
	}

	compiled := make([]string, len(ac.args))
	for i := range ac.args {
		c, err := ac.compileArg(ac.args[i])
		if err != nil {
			return nil, err
		}
		compiled[i] = c
	}

	return compiled, nil
}

func (ac BaseNodesAction) compileArg(s string) (string, error) {
	b, err := config.CompileTemplate(s, ac.vars)
	if err != nil {
		return "", errors.Wrapf(err, "failed to compile arg, %q", s)
	}

	return string(b), nil
}

type StartNodesAction struct {
	*BaseNodesAction
}

func NewStartNodesAction(ctx context.Context, aliases []string, args []string) (*StartNodesAction, error) {
	b, err := NewBaseNodesAction(ctx, "start-nodes", aliases, args)
	if err != nil {
		return nil, err
	}
	return &StartNodesAction{
		BaseNodesAction: b,
	}, nil
}

func (ac *StartNodesAction) Run(ctx context.Context) error {
	ids, err := filterRunningContainers(ctx, ac.nodes, true)
	if err != nil {
		return err
	}

	return host.RunWaitGroup(len(ac.nodes), func(i int) error {
		id, found := ids[ac.nodes[i].Alias()]
		if !found {
			return nil
		}

		return ac.run(ctx, ac.nodes[i], id)
	})
}

func (ac *StartNodesAction) run(ctx context.Context, node *host.Node, id string) error {
	args, err := ac.compileArgs()
	if err != nil {
		return err
	}

	cmds := make([]string, len(host.DefaultContainerCmdNodeRun)+len(args))
	copy(cmds[:len(host.DefaultContainerCmdNodeRun)], host.DefaultContainerCmdNodeRun)
	copy(cmds[len(host.DefaultContainerCmdNodeRun):], args)

	if len(id) < 1 {
		i, err := ac.createContainer(
			ctx,
			node,
			cmds,
			host.NodeRunContainerName(node.Alias()),
			"run",
		)
		if err != nil {
			return err
		}
		id = i
	}

	ac.Log().Debug().Strs("commands", cmds).Msg("trying to run node")

	if err := ac.startContainer(ctx, node, id); err != nil {
		return err
	}

	go func() {
		msg, err := ac.waitContainer(context.Background(), node, id, container.WaitConditionNotRunning)
		if err != nil {
			ac.Log().Error().Err(err).Msg("failed to wait container")
		}

		if msg.Err != nil {
			msg.Msg = "start node stopped with error"
		} else {
			msg.Msg = "start node stopped without error"
		}

		if e, err := host.NewNodeLogEntryWithInterface(node.Alias(), msg, msg.StatusCode != 0); err != nil {
			ac.Log().Error().Err(err).Msg("failed to make log entry")
		} else {
			ac.lo.LogEntryChan() <- e
		}
	}()

	return ac.containerLogs(ctx, node, id)
}

func (ac StartNodesAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ac.Map())
}

type StopNodesAction struct {
	*BaseNodesAction
}

func NewStopNodesAction(ctx context.Context, aliases []string) (*StopNodesAction, error) {
	b, err := NewBaseNodesAction(ctx, "stop-nodes", aliases, nil)
	if err != nil {
		return nil, err
	}
	return &StopNodesAction{
		BaseNodesAction: b,
	}, nil
}

func (ac *StopNodesAction) Run(ctx context.Context) error {
	ids, err := filterRunningContainers(ctx, ac.nodes, false)
	if err != nil {
		return err
	}

	return host.RunWaitGroup(len(ac.nodes), func(i int) error {
		node := ac.nodes[i]
		id, found := ids[node.Alias()]
		if !found {
			return nil
		}
		return node.Host().DockerClient().ContainerStop(ctx, id, nil)
	})
}

func (ac StopNodesAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ac.Map())
}

type InitNodesAction struct {
	*BaseNodesAction
}

func NewInitNodesAction(ctx context.Context, aliases []string) (*InitNodesAction, error) {
	b, err := NewBaseNodesAction(ctx, "init-nodes", aliases, nil)
	if err != nil {
		return nil, err
	}
	return &InitNodesAction{
		BaseNodesAction: b,
	}, nil
}

func (ac *InitNodesAction) Run(ctx context.Context) error {
	ids, err := filterRunningContainers(ctx, ac.nodes, true)
	if err != nil {
		return err
	}

	return host.RunWaitGroup(len(ac.nodes), func(i int) error {
		id, found := ids[ac.nodes[i].Alias()]
		if !found {
			return nil
		}
		return ac.run(ctx, ac.nodes[i], id)
	})
}

func (ac *InitNodesAction) run(ctx context.Context, node *host.Node, id string) error {
	if len(id) < 1 {
		i, err := ac.createContainer(
			ctx,
			node,
			host.DefaultContainerCmdNodeInit,
			host.NodeInitContainerName(node.Alias()),
			"init",
		)
		if err != nil {
			return err
		}
		id = i
	}

	if err := ac.startContainer(ctx, node, id); err != nil {
		return err
	}

	if err := ac.containerLogs(ctx, node, id); err != nil {
		return err
	}

	msg, err := ac.waitContainer(ctx, node, id, container.WaitConditionNotRunning)
	if err != nil {
		return err
	}

	if msg.Err != nil {
		msg.Msg = "init node stopped with error"
	} else {
		msg.Msg = "init node stopped without error"
	}

	e, err := host.NewNodeLogEntryWithInterface(node.Alias(), msg, msg.StatusCode != 0)
	if err != nil {
		return err
	}

	ac.lo.LogEntryChan() <- e

	return nil
}

func (ac InitNodesAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ac.Map())
}

type KillAction struct {
	exitChan chan error
	err      error
}

func (KillAction) Name() string {
	return "stop"
}

func (ac KillAction) Run(_ context.Context) error {
	go func() {
		ac.exitChan <- ac.err
	}()

	return nil
}

func (KillAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{"name": "stop"})
}

type CustomNodesAction struct {
	*BaseNodesAction
}

func NewCustomNodesAction(ctx context.Context, aliases []string, args []string) (*CustomNodesAction, error) {
	b, err := NewBaseNodesAction(ctx, "custom-nodes", aliases, args)
	if err != nil {
		return nil, err
	}
	return &CustomNodesAction{
		BaseNodesAction: b,
	}, nil
}

func (ac *CustomNodesAction) Run(ctx context.Context) error {
	ids, err := filterRunningContainers(ctx, ac.nodes, true)
	if err != nil {
		return err
	}

	return host.RunWaitGroup(len(ac.nodes), func(i int) error {
		id, found := ids[ac.nodes[i].Alias()]
		if !found {
			return nil
		}

		return ac.run(ctx, ac.nodes[i], id)
	})
}

func (ac *CustomNodesAction) run(ctx context.Context, node *host.Node, id string) error {
	args, err := ac.compileArgs()
	if err != nil {
		return err
	}

	cmds := host.CustomNodesActionContainerCmd(args)
	if len(id) < 1 {
		id, err = ac.createContainer(
			ctx,
			node,
			cmds,
			host.NodeCustomContainerName(node.Alias()),
			"custom",
		)
		if err != nil {
			return err
		}
	}

	ac.Log().Debug().Strs("commands", cmds).Msg("trying to run custom node")

	if err = ac.startContainer(ctx, node, id); err != nil {
		return err
	}

	if err = ac.containerLogs(ctx, node, id); err != nil {
		return err
	}

	msg, err := ac.waitContainer(ctx, node, id, container.WaitConditionNotRunning)
	if err != nil {
		ac.Log().Error().Err(err).Msg("failed to wait container")
	}

	if msg.Err != nil {
		msg.Msg = "custom node stopped with error"
	} else {
		msg.Msg = "custom node stopped without error"
	}

	if e, err := host.NewNodeLogEntryWithInterface(node.Alias(), msg, msg.StatusCode != 0); err != nil {
		ac.Log().Error().Err(err).Msg("failed to make log entry")
	} else {
		ac.lo.LogEntryChan() <- e
	}

	return nil
}

func (ac CustomNodesAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ac.Map())
}

type HostCommandAction struct {
	*logging.Logging
	command string
	vars    *config.Vars
	local   host.Host
}

func NewHostCommandAction(ctx context.Context, args []string) (host.Action, error) {
	if len(args) < 1 {
		return nil, errors.Errorf("empty command")
	}

	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var hosts *host.Hosts
	var local host.Host
	if err := host.LoadHostsContextValue(ctx, &hosts); err != nil {
		return nil, err
	} else if err := hosts.TraverseHosts(func(h host.Host) (bool, error) {
		// NOTE at this time, only local host is allowed exec command
		if i, ok := h.(*host.LocalHost); ok {
			local = i

			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, err
	} else if local == nil {
		return nil, errors.Errorf("local host not found for HostCommandAction")
	}

	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	action := &HostCommandAction{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			a := args[0]
			if len(a) > 20 {
				a = a[:20]
			}

			return c.Str("module", "host-command-action").Str("command", a)
		}),
		command: args[0],
		vars:    vars,
		local:   local,
	}

	_ = action.SetLogging(log)

	return action, nil
}

func (*HostCommandAction) Name() string {
	return "host-command"
}

func (ac *HostCommandAction) Map() map[string]interface{} {
	return map[string]interface{}{
		"name": ac.Name(),
		"args": []string{ac.command},
	}
}

func (ac *HostCommandAction) Run(ctx context.Context) error {
	_, _ = fmt.Fprintf(os.Stderr, "> input command: \n%s\n", ac.command)

	i, err := config.CompileTemplate(ac.command, ac.vars)
	if err != nil {
		return err
	}

	compiled := string(i)

	ac.Log().Debug().Str("command", compiled).Msg("command compiled")
	if ac.Log().GetLevel() <= zerolog.DebugLevel {
		_, _ = fmt.Fprintf(os.Stderr, "< compiled command: \n%s\n", compiled)
	}

	ac.Log().Debug().Msg("running command")

	nctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	stdout, stderr, err := ac.local.ShellExec(nctx, "/bin/sh", []string{"-c", compiled})
	stdoutOut, _ := ioutil.ReadAll(stdout)
	stderrOut, _ := ioutil.ReadAll(stderr)
	_, _ = fmt.Fprintf(os.Stderr, "= stdout: \n%s\n", string(stdoutOut))
	_, _ = fmt.Fprintf(os.Stderr, "= stderr: \n%s\n", string(stderrOut))

	if err != nil {
		l := ac.Log().Error().Err(err).Str("stderr", string(stderrOut))

		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			l = l.Int("exit_code", exitError.ExitCode())
		}

		l.Msg("failed to run command")

		return errors.Wrapf(err, "failed to run command, %q", string(stderrOut))
	}

	ac.Log().Debug().Msg("command finished")

	return nil
}

func (ac HostCommandAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(ac.Map())
}

func findNodesFromDesign(design config.DesignAction) ([]string, error) {
	i, found := design.Extra["nodes"]
	if !found {
		return nil, nil
	}

	j, ok := i.([]interface{})
	if !ok {
		return nil, errors.Errorf("nodes is not slice type, %T", i)
	}

	nodes := make([]string, len(j))
	for k := range j {
		m, ok := j[k].(string)
		if !ok {
			return nil, errors.Errorf("node is not string type, %T", j[k])
		}
		nodes[k] = m
	}

	return nodes, nil
}
