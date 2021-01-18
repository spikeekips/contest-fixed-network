package cmds

import (
	"context"

	"github.com/spikeekips/contest/host"
)

const HookNameContestReady = "contest_ready"

func HookContestReady(ctx context.Context) (context.Context, error) {
	var ls *host.LogSaver
	if err := host.LoadLogSaverContextValue(ctx, &ls); err != nil {
		return ctx, err
	}

	ls.LogEntryChan() <- host.NewContestLogEntry([]byte("contest ready"), false)

	return ctx, nil
}
