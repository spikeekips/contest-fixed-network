package cmds

import (
	"context"
	"time"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"

	"github.com/spikeekips/contest/config"
	"github.com/spikeekips/contest/host"
)

const (
	ProcessNameMongodb   = "mongodb"
	HookNameCloseMongodb = "close_mongodb"
)

var ProcessorMongodb pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameMongodb,
		[]string{ProcessNameConfig},
		ProcessMongodb,
	); err != nil {
		panic(err)
	} else {
		ProcessorMongodb = i
	}
}

func ProcessMongodb(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var design config.Design
	if err := config.LoadDesignContextValue(ctx, &design); err != nil {
		return ctx, err
	}

	log.Debug().
		Str("mongodb", design.Storage.String()).
		Str("db", design.Storage.Database).
		Msg("trying to connect mongodb")

	connCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	mg := host.NewMongodb(design.Storage)
	if err := mg.Connect(connCtx); err != nil {
		return ctx, xerrors.Errorf("failed to connect mongodb: %w", err)
	} else if err := mg.Initialize(context.Background()); err != nil {
		return ctx, xerrors.Errorf("failed to initialize mongodb: %w", err)
	} else {
		log.Debug().Msg("mongodb connected")

		return context.WithValue(ctx, host.ContextValueMongodb, mg), nil
	}
}

func HookCloseMongodb(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var mg *host.Mongodb
	if err := host.LoadMongodbContextValue(ctx, &mg); err != nil {
		return ctx, nil
	}

	log.Debug().Msg("trying to close mongodb")

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := mg.Close(closeCtx); err != nil {
		return ctx, xerrors.Errorf("failed to close mongodb: %w", err)
	}

	log.Debug().Msg("mongodb closed")

	return ctx, nil
}
