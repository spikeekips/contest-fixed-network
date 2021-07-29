package cmds

import (
	"context"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

const (
	ProcessNameLogWatcher   = "log_watcher"
	HookNameStopLogHandlers = "stop_log_handlers"
)

var ProcessorLogWatcher pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameLogWatcher,
		[]string{ProcessNameNodes},
		ProcessLogWatcher,
	); err != nil {
		panic(err)
	} else {
		ProcessorLogWatcher = i
	}
}

func ProcessLogWatcher(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var design config.Design
	if err := config.LoadDesignContextValue(ctx, &design); err != nil {
		return ctx, err
	}

	var mg *host.Mongodb
	if err := host.LoadMongodbContextValue(ctx, &mg); err != nil {
		return ctx, err
	}

	var exitChan chan error
	if err := LoadExitChanContextValue(ctx, &exitChan); err != nil {
		return ctx, err
	}

	var vars *config.Vars
	if err := config.LoadVarsContextValue(ctx, &vars); err != nil {
		return ctx, err
	}

	sqs := make([]*host.Sequence, len(design.Sequences))
	for i := range design.Sequences {
		sq, err := parseSequence(ctx, design.Sequences[i])
		if err != nil {
			return ctx, err
		}
		sqs[i] = sq
	}

	lw, err := host.NewLogWatcher(mg, sqs, exitChan, vars)
	if err != nil {
		return ctx, err
	}

	_ = lw.SetLogger(log)

	return context.WithValue(ctx, host.ContextValueLogWatcher, lw), lw.Start()
}

func HookStopLogHandlers(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var ls *host.LogSaver
	if err := host.LoadLogSaverContextValue(ctx, &ls); err != nil {
		return ctx, err
	} else if err := ls.Stop(); err != nil {
		log.Error().Err(err).Msg("failed to stop log saver")

		return ctx, err
	}

	var lw *host.LogWatcher
	if err := host.LoadLogWatcherContextValue(ctx, &lw); err != nil {
		return ctx, err
	} else if err := lw.Stop(); err != nil {
		log.Error().Err(err).Msg("failed to stop log watcher")

		return ctx, nil
	}

	return ctx, nil
}
