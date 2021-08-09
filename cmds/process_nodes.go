package cmds

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

const ProcessNameNodes = "nodes"

var ProcessorNodes pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameNodes,
		[]string{ProcessNameConfig, ProcessNameHosts},
		ProcessNodes,
	); err != nil {
		panic(err)
	} else {
		ProcessorNodes = i
	}
}

func ProcessNodes(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var design config.Design
	if err := config.LoadDesignContextValue(ctx, &design); err != nil {
		return ctx, err
	}

	var logDir string
	if err := config.LoadLogDirContextValue(ctx, &logDir); err != nil {
		return ctx, err
	}

	var hosts *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hosts); err != nil {
		return ctx, err
	}

	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return nil, err
	}

	log.Log().Debug().Msg("trying to prepare hosts")
	nodesConfig, err := generateNodesConfig(ctx, design, hosts)
	if err != nil {
		return ctx, errors.Wrap(err, "failed to generate nodes config")
	}

	if err := hosts.TraverseNodes(func(node *host.Node) (bool, error) {
		vars.Set(fmt.Sprintf("Design.Node.%s", node.Alias()), node.ConfigMap())

		err := saveNodeConfig(node.Alias(), logDir, node.ConfigData(), nodesConfig[node.Alias()])
		return err == nil, err
	}); err != nil {
		return ctx, err
	}

	log.Log().Debug().Msg("hosts and nodes prepared")

	return context.WithValue(ctx, config.ContextValueVars, vars), nil
}
