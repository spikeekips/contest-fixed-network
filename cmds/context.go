package cmds

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
)

var (
	ContextValueExitError util.ContextKey = "exit_error"
	ContextValueExitChan  util.ContextKey = "exit_chan"
)

func LoadExitErrorContextValue(ctx context.Context, l *error) error {
	err := util.LoadFromContextValue(ctx, ContextValueExitError, l)
	if err == nil {
		return nil
	}

	if errors.Is(err, util.ContextValueNotFoundError) {
		return nil
	}

	return err
}

func LoadExitChanContextValue(ctx context.Context, l *chan error) error {
	return util.LoadFromContextValue(ctx, ContextValueExitChan, l)
}
