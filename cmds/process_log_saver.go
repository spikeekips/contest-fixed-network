package cmds

import (
	"context"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

const ProcessNameLogSaver = "log_saver"

var ProcessorLogSaver pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameLogSaver,
		[]string{ProcessNameMongodb},
		ProcessLogSaver,
	); err != nil {
		panic(err)
	} else {
		ProcessorLogSaver = i
	}
}

func ProcessLogSaver(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
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

	nodes := make([]string, len(design.NodeConfig))
	var i int
	for alias := range design.NodeConfig {
		nodes[i] = alias
		i++
	}

	var logDir string
	if err := config.LoadLogDirContextValue(ctx, &logDir); err != nil {
		return ctx, err
	}

	lo, err := host.NewLogSaver(mg, logDir, nodes, exitChan, design.ExitOnError)
	if err != nil {
		return ctx, err
	}

	_ = lo.SetLogging(log)

	if err := lo.Start(); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, host.ContextValueLogSaver, lo), nil
}
