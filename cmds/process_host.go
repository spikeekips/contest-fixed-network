package cmds

import (
	"context"
	"sort"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

const (
	ProcessNameHosts                   = "hosts"
	HookNameCleanStoppedNodeContainers = "clean_stopped_node_containers"
)

var (
	ProcessorHosts     pm.Process
	HookNameCloseHosts = "close_hosts"
)

func init() {
	if i, err := pm.NewProcess(
		ProcessNameHosts,
		[]string{ProcessNameConfig},
		ProcessHosts,
	); err != nil {
		panic(err)
	} else {
		ProcessorHosts = i
	}
}

func ProcessHosts(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var design config.Design
	if err := config.LoadDesignContextValue(ctx, &design); err != nil {
		return ctx, err
	}

	var lo *host.LogSaver
	if err := host.LoadLogSaverContextValue(ctx, &lo); err != nil {
		return ctx, err
	}

	nodes := make([]string, len(design.NodeConfig))
	var i int
	for k := range design.NodeConfig {
		nodes[i] = k
		i++
	}

	designHosts, selected := spreadNodes(design.Hosts, nodes)

	hosts := host.NewHosts(lo)
	_ = hosts.SetLogger(log)

	for i := range designHosts {
		de := designHosts[i]

		l := log.WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Str("host", de.Host)
		})

		if len(selected[i]) < 1 {
			l.Debug().Msg("no nodes; will be skipped")

			continue
		}

		nodeDesigns := map[string]string{}
		for _, alias := range selected[i] {
			nodeDesigns[alias] = design.NodeConfig[alias]
		}

		if h, err := newHost(ctx, designHosts[i], nodeDesigns); err != nil {
			return ctx, err
		} else if err := hosts.AddHost(h); err != nil {
			return ctx, err
		}

		l.Debug().Strs("nodes", selected[i]).Interface("design", de).Msg("host created")
	}

	log.Debug().Int("hosts", hosts.LenHosts()).Msg("hosts created")

	return context.WithValue(ctx, host.ContextValueHosts, hosts), nil
}

func spreadNodes(hosts []config.DesignHost, nodes []string) ([]config.DesignHost, [][]string) {
	byWeights := hosts
	sort.Slice(byWeights, func(i, j int) bool { return byWeights[i].Weight > byWeights[j].Weight })

	weights := make([]uint, len(byWeights))
	for i := range byWeights {
		weights[i] = byWeights[i].Weight
	}

	spread := calcSpreadNodes(uint(len(nodes)), weights)

	var selectedHosts []config.DesignHost // nolint
	var selectedNodes [][]string          // nolint
	var l uint
	for i := range byWeights {
		if spread[i] < 1 {
			continue
		}

		selectedHosts = append(selectedHosts, byWeights[i])
		selectedNodes = append(selectedNodes, nodes[l:l+spread[i]])

		l += spread[i]
	}

	return selectedHosts, selectedNodes
}

func HookCleanStoppedNodeContainers(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var flags map[string]interface{}
	if err := config.LoadFlagsContextValue(ctx, &flags); err != nil {
		return ctx, err
	}

	force := flags["Force"].(bool)

	var hosts *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hosts); err != nil {
		return ctx, err
	}

	log.Debug().Bool("force", force).Msg("trying to clean up stopped node containers")

	if err := hosts.TraverseHosts(func(h host.Host) (bool, error) {
		if err := h.Clean(context.Background(), true, force); err != nil {
			return false, err
		} else {
			return true, nil
		}
	}); err != nil {
		return ctx, err
	}

	if err := hosts.TraverseHosts(func(h host.Host) (bool, error) {
		if err := h.Clean(context.Background(), false, force); err != nil {
			return false, err
		} else {
			return true, nil
		}
	}); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func HookCloseHosts(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var hosts *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hosts); err != nil {
		return ctx, nil
	}

	log.Debug().Msg("trying to close hosts")

	return ctx, hosts.Close()
}
