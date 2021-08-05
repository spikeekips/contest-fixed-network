package cmds

import (
	"context"

	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

const HookNameCleanContainers = "clean_containers"

func HookCleanContainers(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var hosts *host.Hosts
	if err := host.LoadHostsContextValue(ctx, &hosts); err != nil {
		return ctx, err
	}

	var flags map[string]interface{}
	if err := config.LoadFlagsContextValue(ctx, &flags); err != nil {
		return ctx, err
	}

	if !flags["CleanAfter"].(bool) {
		return ctx, nil
	}

	log.Log().Debug().Msg("trying to clean containers")

	cleanCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := hosts.TraverseHosts(func(h host.Host) (bool, error) {
		return true, h.Clean(cleanCtx, false, true)
	}); err != nil {
		log.Log().Error().Err(err).Msg("failed to clean containers")

		return ctx, err
	}
	log.Log().Debug().Msg("containers cleaned")

	return ctx, nil
}
